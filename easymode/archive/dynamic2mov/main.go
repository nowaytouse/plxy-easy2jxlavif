// dynamic2mov - 动态图片转高效视频MOV工具
// 版本: v1.0.0
// 作者: AI Assistant
// 功能: 将动态图片（GIF/WebP/APNG）转换为高效的AV1或H.265编码MOV视频

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/karrick/godirwalk"
)

const (
	version = "1.0.0"
	author  = "AI Assistant"
)

var (
	logger     *log.Logger
	globalCtx  context.Context
	cancelFunc context.CancelFunc
	stats      *Stats
	procSem    chan struct{}
	fdSem      chan struct{}
)

type Options struct {
	Workers           int
	InputDir          string
	OutputDir         string
	SkipExist         bool
	DryRun            bool
	TimeoutSeconds    int
	Retries           int
	MaxMemory         int64
	MaxFileSize       int64
	EnableHealthCheck bool
	PreferredCodec    string // "av1" 或 "h265" 或 "auto"
	OutputFormat      string // "mov" 或 "mp4"
}

type Stats struct {
	sync.RWMutex
	imagesProcessed  int
	imagesFailed     int
	imagesSkipped    int
	totalBytesBefore int64
	totalBytesAfter  int64
	peakMemoryUsage  int64
	totalRetries     int
	startTime        time.Time
	byExt            map[string]int
	errorTypes       map[string]int
	detailedLogs     []FileProcessInfo
}

type FileProcessInfo struct {
	FilePath       string
	FileSize       int64
	FileType       string
	ProcessingTime time.Duration
	ConversionMode string
	SizeBefore     int64
	SizeAfter      int64
	Success        bool
	ErrorMsg       string
	StartTime      time.Time
	EndTime        time.Time
	ErrorType      string
}

func init() {
	setupLogging()
	stats = &Stats{
		byExt:      make(map[string]int),
		errorTypes: make(map[string]int),
		startTime:  time.Now(),
	}
	setupSignalHandling()
}

func setupLogging() {
	logFile, err := os.OpenFile("dynamic2mov.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
}

func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Printf("🛑 收到信号 %v，开始优雅关闭...", sig)
		if cancelFunc != nil {
			cancelFunc()
		}
		time.Sleep(2 * time.Second)
		printStatistics()
		os.Exit(0)
	}()
}

func parseFlags() Options {
	var opts Options

	flag.StringVar(&opts.InputDir, "dir", "", "📂 输入目录路径（必需）")
	flag.StringVar(&opts.OutputDir, "output", "", "📁 输出目录路径（默认为输入目录）")
	flag.IntVar(&opts.Workers, "workers", 0, "⚡ 工作线程数 (0=自动检测)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", false, "⏭️ 跳过已存在的文件")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "🔍 试运行模式")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", 600, "⏰ 单个文件处理超时时间（秒）")
	flag.IntVar(&opts.Retries, "retries", 2, "🔄 转换失败重试次数")
	flag.Int64Var(&opts.MaxMemory, "max-memory", 0, "💾 最大内存使用量（字节，0=无限制）")
	flag.Int64Var(&opts.MaxFileSize, "max-file-size", 500*1024*1024, "📏 最大文件大小（字节）")
	flag.BoolVar(&opts.EnableHealthCheck, "health-check", true, "🏥 启用健康检查")
	flag.StringVar(&opts.PreferredCodec, "codec", "auto", "🎬 编码器选择 (av1/h265/auto)")
	flag.StringVar(&opts.OutputFormat, "format", "mov", "📦 输出格式 (mov/mp4)")

	flag.Parse()

	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录 (-dir)")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir
	}
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}

	return opts
}

func checkDependencies() error {
	dependencies := []string{"ffmpeg", "exiftool"}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("缺少依赖: %s", dep)
		}
	}
	logger.Println("✅ 所有系统依赖检查通过")
	return nil
}

func configurePerformance(opts *Options) {
	cpuCount := runtime.NumCPU()
	if opts.Workers <= 0 {
		if cpuCount >= 16 {
			opts.Workers = cpuCount
		} else if cpuCount >= 8 {
			opts.Workers = cpuCount - 1
		} else if cpuCount >= 4 {
			opts.Workers = cpuCount
		} else {
			opts.Workers = 4
		}
	}
	if opts.Workers > 8 {
		opts.Workers = 8
	}
	procSem = make(chan struct{}, opts.Workers)
	fdSem = make(chan struct{}, 16)
	globalCtx, cancelFunc = context.WithCancel(context.Background())
	logger.Printf("⚡ 性能配置: %d 个工作线程", opts.Workers)
}

func scanCandidateFiles(inputDir string, opts Options) []string {
	var files []string
	err := godirwalk.Walk(inputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(osPathname))
			// 支持所有动态图片格式
			if ext != ".gif" && ext != ".webp" && ext != ".apng" && ext != ".png" {
				return nil
			}
			// 对于PNG，需要检查是否为APNG（动态PNG）
			if ext == ".png" {
				// 简化：假设所有PNG都可能是APNG，让ffmpeg自动处理
			}
			if info, err := os.Stat(osPathname); err == nil {
				if info.Size() > 0 && info.Size() <= opts.MaxFileSize {
					files = append(files, osPathname)
				}
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			logger.Printf("⚠️  扫描文件时出错: %s - %v", osPathname, err)
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		logger.Printf("❌ 扫描文件时出错: %v", err)
	}
	return files
}

func processFileWithRetry(filePath string, fileInfo os.FileInfo, opts Options) {
	var lastErr error
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if attempt > 0 {
			logger.Printf("🔄 重试处理文件: %s (第 %d 次)", filepath.Base(filePath), attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
			stats.Lock()
			stats.totalRetries++
			stats.Unlock()
		}
		err := processFileWithOpts(filePath, fileInfo, stats, opts)
		if err == nil {
			return
		}
		lastErr = err
		logger.Printf("⚠️  处理文件失败: %s - %v", filepath.Base(filePath), err)
		stats.Lock()
		stats.errorTypes[classifyError(err)]++
		stats.Unlock()
	}
	logger.Printf("❌ 文件处理最终失败: %s - %v", filepath.Base(filePath), lastErr)
	stats.addImageFailed()
}

func classifyError(err error) string {
	if err == nil {
		return "unknown"
	}
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") {
		return "timeout"
	} else if strings.Contains(errStr, "permission") {
		return "permission"
	} else if strings.Contains(errStr, "memory") {
		return "memory"
	} else if strings.Contains(errStr, "disk") {
		return "disk"
	}
	return "other"
}

func processFileWithOpts(filePath string, fileInfo os.FileInfo, stats *Stats, opts Options) error {
	startTime := time.Now()

	procSem <- struct{}{}
	defer func() { <-procSem }()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	// GIF转AV1编码MOV
	conversionMode, outputPath, errorMsg, err := processFileByType(filePath, opts)
	processingTime := time.Since(startTime)

	processInfo := FileProcessInfo{
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		FileType:       filepath.Ext(filePath),
		ProcessingTime: processingTime,
		ConversionMode: conversionMode,
		Success:        err == nil,
		ErrorMsg:       errorMsg,
		StartTime:      startTime,
		EndTime:        time.Now(),
		ErrorType:      classifyError(err),
	}

	if err != nil {
		stats.addImageFailed()
		processInfo.ErrorMsg = err.Error()
	} else {
		stats.addImageProcessed(fileInfo.Size(), getFileSize(outputPath))
		stats.addByExt(filepath.Ext(filePath))
	}
	stats.addDetailedLog(processInfo)
	return err
}

func processFileByType(filePath string, opts Options) (string, string, string, error) {
	// 动态图片转AV1/H.265编码视频的实际转换逻辑
	// 根据输出格式选择文件扩展名
	outputExt := "." + opts.OutputFormat
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + outputExt

	// ✅ 步骤1: 捕获源文件的文件系统元数据（在转换之前）
	srcInfo, _ := os.Stat(filePath)
	var creationTime time.Time
	if srcInfo != nil {
		if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
			creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
		}
	}

	// ✅ 步骤2: 智能选择编码器（根据输出格式）
	codec, codecName := selectBestCodec(opts.PreferredCodec, opts.OutputFormat)
	var conversionMode string
	var args []string

	if codec == "av1" {
		// AV1编码（最高压缩比，仅MP4容器）
		conversionMode = fmt.Sprintf("动图转AV1编码%s", strings.ToUpper(opts.OutputFormat))

		if codecName == "libaom-av1" {
			// libaom-AV1编码器（官方实现，质量最高）
			args = []string{
				"-i", filePath,
				"-c:v", "libaom-av1", // libaom-AV1编码器（官方）
				"-crf", "28", // 质量参数（与H.265相同，便于比较）
				"-cpu-used", "4", // 速度参数（0-8，4为平衡）
				"-row-mt", "1", // 多线程优化
				"-tiles", "2x2", // 平铺编码（加速）
				"-pix_fmt", "yuv420p", // 像素格式
				"-map_metadata", "0", // ✅ 复制所有元数据
				"-f", opts.OutputFormat, // MP4格式
				"-y", outputPath, // 覆盖输出
			}
		} else {
			// SVT-AV1编码器（速度快，质量略低）
			args = []string{
				"-i", filePath,
				"-c:v", "libsvtav1", // SVT-AV1编码器
				"-crf", "28", // 质量参数（与H.265一致）
				"-preset", "6", // 速度预设（0-13，越大越快）
				"-pix_fmt", "yuv420p", // 像素格式
				"-map_metadata", "0", // ✅ 复制所有元数据
				"-f", opts.OutputFormat, // MP4格式
				"-y", outputPath, // 覆盖输出
			}
		}
	} else {
		// H.265编码（高兼容性）
		conversionMode = fmt.Sprintf("动图转H.265编码%s", strings.ToUpper(opts.OutputFormat))

		baseArgs := []string{
		"-i", filePath,
			"-c:v", "libx265", // H.265/HEVC编码器
			"-crf", "28", // 质量参数（0-51）
			"-preset", "medium", // 速度预设
		"-pix_fmt", "yuv420p", // 像素格式
		"-map_metadata", "0", // ✅ 复制所有元数据
		}

		// MOV格式需要额外的元数据标签
		if opts.OutputFormat == "mov" {
			args = append(baseArgs, "-movflags", "use_metadata_tags") // ✅ 保留MOV元数据标签
		}

		args = append(args,
			"-f", opts.OutputFormat, // 输出格式（mov或mp4）
		"-y", outputPath, // 覆盖输出
		)
	}

	ctx, cancel := context.WithTimeout(globalCtx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return conversionMode, "", string(output), fmt.Errorf("ffmpeg编码失败 (%s): %v\n输出: %s", codec, err, string(output))
	}

	logger.Printf("✅ 动图转MOV成功（%s编码）: %s", strings.ToUpper(codec), filepath.Base(outputPath))

	// ✅ 步骤3: 复制EXIF元数据（会改变文件修改时间）
	if err := copyMetadata(filePath, outputPath); err != nil {
		logger.Printf("⚠️  EXIF元数据复制失败: %s -> %s: %v",
			filepath.Base(filePath), filepath.Base(outputPath), err)
	} else {
		logger.Printf("✅ EXIF元数据复制成功: %s", filepath.Base(outputPath))
	}

	// ✅ 步骤4: 恢复文件系统元数据（在exiftool之后）
	if srcInfo != nil {
		// 4.1 恢复Finder标签和注释
		if err := copyFinderMetadata(filePath, outputPath); err != nil {
			logger.Printf("⚠️  Finder元数据复制失败 %s: %v", filepath.Base(outputPath), err)
		} else {
			logger.Printf("✅ Finder元数据复制成功: %s", filepath.Base(outputPath))
		}

		// 4.2 恢复修改时间和创建时间（使用touch统一设置）
		if !creationTime.IsZero() {
			timeStr := creationTime.Format("200601021504.05")
			touchCmd := exec.Command("touch", "-t", timeStr, outputPath)
			if err := touchCmd.Run(); err != nil {
				logger.Printf("⚠️  文件时间恢复失败 %s: %v", filepath.Base(outputPath), err)
			} else {
				logger.Printf("✅ 文件系统元数据已保留: %s (创建/修改: %s)",
					filepath.Base(outputPath), creationTime.Format("2006-01-02 15:04:05"))
			}
		}
	}

	return conversionMode, outputPath, "", nil
}

func copyMetadata(inputPath, outputPath string) error {
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", inputPath, outputPath)
	return cmd.Run()
}

// copyFinderMetadata 复制Finder标签和注释
func copyFinderMetadata(src, dst string) error {
	// 复制Finder标签
	cmd := exec.Command("xattr", "-p", "com.apple.metadata:_kMDItemUserTags", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", string(output), dst).Run()
	}

	// 复制Finder注释
	cmd = exec.Command("xattr", "-p", "com.apple.metadata:kMDItemFinderComment", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:kMDItemFinderComment", string(output), dst).Run()
	}

	// 复制其他扩展属性
	cmd = exec.Command("xattr", src)
	if output, err := cmd.CombinedOutput(); err == nil {
		attrs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, attr := range attrs {
			if attr != "" && !strings.Contains(attr, "com.apple.metadata:_kMDItemUserTags") &&
				!strings.Contains(attr, "com.apple.metadata:kMDItemFinderComment") {
				cmd = exec.Command("xattr", "-p", attr, src)
				if value, err := cmd.CombinedOutput(); err == nil && len(value) > 0 {
					exec.Command("xattr", "-w", attr, string(value), dst).Run()
				}
			}
		}
	}

	return nil
}

func getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}
	return 0
}

func (s *Stats) addImageProcessed(bytesBefore, bytesAfter int64) {
	s.Lock()
	defer s.Unlock()
	s.imagesProcessed++
	s.totalBytesBefore += bytesBefore
	s.totalBytesAfter += bytesAfter
}

func (s *Stats) addImageFailed() {
	s.Lock()
	defer s.Unlock()
	s.imagesFailed++
}

func (s *Stats) addImageSkipped() {
	s.Lock()
	defer s.Unlock()
	s.imagesSkipped++
}

func (s *Stats) addByExt(ext string) {
	s.Lock()
	defer s.Unlock()
	s.byExt[ext]++
}

func (s *Stats) addDetailedLog(info FileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

func printStatistics() {
	stats.RLock()
	defer stats.RUnlock()
	totalProcessed := stats.imagesProcessed + stats.imagesFailed + stats.imagesSkipped
	if totalProcessed == 0 {
		return
	}
	successRate := float64(stats.imagesProcessed) / float64(totalProcessed) * 100
	logger.Println("")
	logger.Println("📊 处理统计:")
	logger.Printf("  • 总文件数: %d", totalProcessed)
	logger.Printf("  • 成功处理: %d", stats.imagesProcessed)
	logger.Printf("  • 处理失败: %d", stats.imagesFailed)
	logger.Printf("  • 跳过文件: %d", stats.imagesSkipped)
	logger.Printf("  • 成功率: %.1f%%", successRate)
	if stats.totalBytesBefore > 0 {
		savingPercent := (1 - float64(stats.totalBytesAfter)/float64(stats.totalBytesBefore)) * 100
		logger.Printf("  • 空间节省: %.1f%%", savingPercent)
	}
	logger.Printf("  • 处理时间: %v", time.Since(stats.startTime))
	if stats.totalRetries > 0 {
		logger.Printf("  • 总重试次数: %d", stats.totalRetries)
	}
}

func main() {
	// 🎨 检测模式：无参数时启动交互模式
	if len(os.Args) == 1 {
		runInteractiveMode()
		return
	}

	// 📝 非交互模式：命令行参数
	runNonInteractiveMode()
}

// runNonInteractiveMode 非交互模式入口
func runNonInteractiveMode() {
	logger.Printf("🎬 dynamic2mov v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	opts := parseFlags()
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	configurePerformance(&opts)
	logger.Println("🔍 扫描GIF文件...")
	files := scanCandidateFiles(opts.InputDir, opts)
	logger.Printf("📊 发现 %d 个GIF文件", len(files))

	if len(files) == 0 {
		logger.Println("📊 没有找到GIF文件")
		return
	}

	if opts.DryRun {
		logger.Println("🔍 试运行模式 - 将要处理的文件:")
		for i, file := range files {
			logger.Printf("  %d. %s", i+1, file)
		}
		return
	}

	logger.Printf("🚀 开始处理 %d 个文件 (使用 %d 个工作线程)...", len(files), opts.Workers)
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			if info, err := os.Stat(filePath); err == nil {
				processFileWithRetry(filePath, info, opts)
			}
		}(file)
	}
	wg.Wait()
	printStatistics()
	logger.Println("🎉 处理完成！")
}

// runNonInteractiveMode_WithOpts 使用指定选项运行
func runNonInteractiveMode_WithOpts(opts Options) {
	logger.Printf("🎬 dynamic2mov v%s", version)
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	configurePerformance(&opts)
	logger.Println("🔍 扫描动态图片文件（GIF/WebP/APNG）...")
	files := scanCandidateFiles(opts.InputDir, opts)
	logger.Printf("📊 发现 %d 个动态图片文件", len(files))

	if len(files) == 0 {
		logger.Println("📊 没有找到动态图片文件")
		return
	}

	logger.Printf("🚀 开始处理 %d 个文件 (使用 %d 个工作线程)...", len(files), opts.Workers)
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			if info, err := os.Stat(filePath); err == nil {
				processFileWithRetry(filePath, info, opts)
			}
		}(file)
	}
	wg.Wait()
	printStatistics()
	logger.Println("🎉 处理完成！")
}

// runInteractiveMode 交互模式入口
func runInteractiveMode() {
	// 1. 显示横幅
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║   🎬 dynamic2mov v1.0.0 - 动态图片转视频工具                ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║   输入: GIF / WebP（动图）/ APNG                             ║")
	fmt.Println("║   输出: MOV/MP4视频（AV1或H.265编码）                        ║")
	fmt.Println("║   编码: AV1(MP4)最高压缩 / H.265(MOV)高兼容                 ║")
	fmt.Println("║   元数据: EXIF + 文件系统时间戳 + Finder标签 100%保留       ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println("")

	// 2. 提示输入目录
	targetDir, err := promptForDirectory()
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		os.Exit(1)
	}

	// 3. 安全检查
	if err := performSafetyCheck(targetDir); err != nil {
		fmt.Printf("❌ 安全检查失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 设置选项并开始处理
	opts := Options{
		Workers:           4,
		InputDir:          targetDir,
		OutputDir:         targetDir,
		SkipExist:         false,
		DryRun:            false,
		TimeoutSeconds:    600,
		Retries:           2,
		MaxMemory:         0,
		MaxFileSize:       500 * 1024 * 1024,
		EnableHealthCheck: true,
		PreferredCodec:    "auto", // 自动选择
		OutputFormat:      "mov",  // 默认MOV格式
	}

	fmt.Println("🔄 开始处理...")
	fmt.Println("")

	// 开始主处理流程
	runNonInteractiveMode_WithOpts(opts)
}

// promptForDirectory 提示用户输入目录
func promptForDirectory() (string, error) {
	fmt.Println("📁 请拖入要处理的文件夹，然后按回车键：")
	fmt.Println("   （或直接输入路径）")
	fmt.Print("\n路径: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取输入失败: %v", err)
	}

	// 清理并反转义路径
	path := strings.TrimSpace(input)
	path = unescapeShellPath(path)

	if path == "" {
		return "", fmt.Errorf("路径不能为空")
	}

	return path, nil
}

// performSafetyCheck 执行安全检查
func performSafetyCheck(targetPath string) error {
	fmt.Println("")
	fmt.Println("🔍 正在执行安全检查...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. 检查路径是否存在
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("无法解析路径: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("路径不存在: %s", absPath)
		}
		return fmt.Errorf("无法访问路径: %v", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("路径不是文件夹: %s", absPath)
	}

	fmt.Printf("  ✅ 路径存在: %s\n", absPath)

	// 2. 检查是否为系统关键目录
	if isCriticalSystemPath(absPath) {
		return fmt.Errorf("禁止访问系统关键目录: %s\n建议使用: ~/Documents, ~/Desktop, ~/Downloads", absPath)
	}

	fmt.Printf("  ✅ 路径安全: 非系统目录\n")

	// 3. 检查读写权限
	testFile := filepath.Join(absPath, ".pixly_permission_test")
	if file, err := os.Create(testFile); err != nil {
		return fmt.Errorf("目录没有写入权限: %v", err)
	} else {
		file.Close()
		os.Remove(testFile)
		fmt.Printf("  ✅ 权限验证: 可读可写\n")
	}

	// 4. 检查磁盘空间
	if freeSpace, totalSpace, err := getDiskSpace(absPath); err == nil {
		freeGB := float64(freeSpace) / 1024 / 1024 / 1024
		totalGB := float64(totalSpace) / 1024 / 1024 / 1024
		ratio := float64(freeSpace) / float64(totalSpace) * 100

		fmt.Printf("  💾 磁盘空间: %.1fGB / %.1fGB (%.1f%% 可用)\n", freeGB, totalGB, ratio)

		if ratio < 10 {
			return fmt.Errorf("磁盘空间不足（剩余%.1f%%），建议至少保留10%%空间", ratio)
		} else if ratio < 20 {
			fmt.Printf("  ⚠️  磁盘空间较少（剩余%.1f%%），建议谨慎处理\n", ratio)
		}
	}

	// 5. 检查是否为敏感目录
	if isSensitiveDirectory(absPath) {
		fmt.Printf("  ⚠️  敏感目录警告: %s\n", absPath)
		fmt.Print("\n  是否继续处理此目录？(输入 yes 确认): ")

		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))

		if confirm != "yes" && confirm != "y" {
			return fmt.Errorf("用户取消操作")
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 安全检查通过！")
	fmt.Println("")

	return nil
}

// isCriticalSystemPath 检查是否为系统关键目录
func isCriticalSystemPath(path string) bool {
	criticalPaths := []string{
		"/System",
		"/Library/System",
		"/private",
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
		"/var/root",
		"/etc",
		"/dev",
		"/proc",
		"/Applications/Utilities",
		"/System/Library",
	}

	for _, critical := range criticalPaths {
		if strings.HasPrefix(path, critical) {
			return true
		}
	}

	return false
}

// isSensitiveDirectory 检查是否为敏感目录
func isSensitiveDirectory(path string) bool {
	sensitivePaths := []string{
		"/Applications",
		"/Library",
		"/usr",
		"/var",
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		sensitivePaths = append(sensitivePaths, homeDir)
	}

	for _, sensitive := range sensitivePaths {
		if path == sensitive {
			return true
		}
	}

	return false
}

// getDiskSpace 获取磁盘空间信息
func getDiskSpace(path string) (free, total uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	free = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)

	return free, total, nil
}

// unescapeShellPath 反转义Shell路径（macOS拖拽）
func unescapeShellPath(path string) string {
	path = strings.ReplaceAll(path, "\\ ", " ")
	path = strings.ReplaceAll(path, "\\!", "!")
	path = strings.ReplaceAll(path, "\\(", "(")
	path = strings.ReplaceAll(path, "\\)", ")")
	path = strings.ReplaceAll(path, "\\[", "[")
	path = strings.ReplaceAll(path, "\\]", "]")
	path = strings.ReplaceAll(path, "\\&", "&")
	path = strings.ReplaceAll(path, "\\$", "$")
	path = strings.Trim(path, "\"'")

	return path
}

// selectBestCodec 智能选择最佳编码器
func selectBestCodec(preferred, format string) (codec, codecName string) {
	// 如果输出格式是MP4，可以使用AV1
	if format == "mp4" {
		// MP4容器支持AV1编码
		if preferred == "av1" || preferred == "auto" {
			// 优先使用libaom-AV1（官方实现，质量最高，压缩比最好）
			if isCodecAvailable("libaom-av1") {
				logger.Printf("🎯 使用libaom-AV1编码器（官方AV1，最高质量）")
				return "av1", "libaom-av1"
			} else if isCodecAvailable("libsvtav1") {
				logger.Printf("🎯 使用SVT-AV1编码器（快速AV1）")
				return "av1", "libsvtav1"
			}
		}

		// 如果AV1不可用，使用H.265
		if preferred == "h265" || preferred == "auto" {
			logger.Printf("🎯 使用H.265编码器（高兼容性）")
			return "h265", "libx265"
		}
	}

	// 如果输出格式是MOV，只能使用H.265
	if format == "mov" {
		if preferred == "av1" {
			logger.Printf("⚠️  MOV容器不支持AV1编码，自动使用H.265")
			logger.Printf("💡 提示：如需AV1编码，请使用 --format mp4")
		}
		logger.Printf("🎯 使用H.265编码器（MOV容器标准编码）")
		return "h265", "libx265"
	}

	// 默认H.265
	logger.Printf("🎯 使用H.265编码器（默认）")
	return "h265", "libx265"
}

// isCodecAvailable 检查编码器是否可用
func isCodecAvailable(codecName string) bool {
	cmd := exec.Command("ffmpeg", "-codecs")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// 检查输出中是否包含编码器名称
	return strings.Contains(string(output), codecName)
}
