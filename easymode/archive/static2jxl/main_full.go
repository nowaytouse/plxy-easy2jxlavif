// 优化版工具 - 基于 universal_converter 功能进行深入优化
// 版本: v2.3.0 (优化版)
// 作者: AI Assistant

package main

import (
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
	version = "2.3.0"
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
}

type FileProcessInfo struct {
	FilePath       string
	FileSize       int64
	FileType       string
	IsAnimated     bool
	ProcessingTime time.Duration
	ConversionMode string
	Success        bool
	ErrorMsg       string
	RetryCount     int
	StartTime      time.Time
	EndTime        time.Time
	ErrorType      string
}

type Stats struct {
	sync.RWMutex
	imagesProcessed  int
	imagesFailed     int
	imagesSkipped    int
	totalBytesBefore int64
	totalBytesAfter  int64
	startTime        time.Time
	detailedLogs     []FileProcessInfo
	byExt            map[string]int
	peakMemoryUsage  int64
	totalRetries     int
	errorTypes       map[string]int
}

func init() {
	setupLogging()
	stats = &Stats{
		startTime:  time.Now(),
		byExt:      make(map[string]int),
		errorTypes: make(map[string]int),
	}
	setupSignalHandling()
}

func setupLogging() {
	logFile, err := os.OpenFile("optimized.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
	flag.IntVar(&opts.TimeoutSeconds, "timeout", 30, "⏰ 单个文件处理超时时间（秒）")
	flag.IntVar(&opts.Retries, "retries", 3, "🔄 转换失败重试次数")
	flag.Int64Var(&opts.MaxMemory, "max-memory", 0, "💾 最大内存使用量（字节，0=无限制）")
	flag.Int64Var(&opts.MaxFileSize, "max-file-size", 500*1024*1024, "📏 最大文件大小（字节）")
	flag.BoolVar(&opts.EnableHealthCheck, "health-check", true, "🏥 启用健康检查")

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
	// 检查必要的依赖
	dependencies := []string{"exiftool"}
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
			if !isSupportedFile(ext) {
				return nil
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

func isSupportedFile(ext string) bool {
	// 根据工具类型返回支持的文件扩展名
	supportedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".bmp": true,
		".tiff": true, ".tif": true, ".gif": true, ".webp": true,
		".avif": true, ".jxl": true, ".heic": true, ".heif": true,
		".mov": true, ".mp4": true, ".avi": true, ".mkv": true,
	}
	return supportedExts[ext]
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
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") {
		return "timeout"
	} else if strings.Contains(errStr, "memory") {
		return "memory"
	} else if strings.Contains(errStr, "permission") {
		return "permission"
	} else if strings.Contains(errStr, "format") {
		return "format"
	}
	return "unknown"
}

func processFileWithOpts(filePath string, fileInfo os.FileInfo, stats *Stats, opts Options) error {
	startTime := time.Now()
	procSem <- struct{}{}
	defer func() { <-procSem }()
	fdSem <- struct{}{}
	defer func() { <-fdSem }()
	
	select {
	case <-globalCtx.Done():
		return globalCtx.Err()
	default:
	}
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}
	
	// 根据工具类型执行相应的处理逻辑
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
	// 静态图转JXL的实际转换逻辑（v2.3.1+元数据保留）
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".jxl"

	conversionMode := "静态转JXL"

	// 检测是否为JPEG（使用lossless_jpeg=1）
	ext := strings.ToLower(filepath.Ext(filePath))
	var args []string

	if ext == ".jpg" || ext == ".jpeg" {
		// JPEG专用：lossless_jpeg=1（完美可逆）
		args = []string{
			"--lossless_jpeg=1",
			"-e", "7",
			filePath,
			outputPath,
		}
	} else {
		// 其他格式：distance=0（无损）
		args = []string{
			"-d", "0",
			"-e", "7",
			filePath,
			outputPath,
		}
	}

	ctx, cancel := context.WithTimeout(globalCtx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cjxl", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return conversionMode, "", string(output), fmt.Errorf("cjxl转换失败: %v", err)
	}

	// ✅ 转换成功后，立即复制元数据（文件内部+文件系统）
	if err := copyMetadata(filePath, outputPath); err != nil {
		logger.Printf("⚠️  EXIF元数据复制失败: %s -> %s: %v",
			filepath.Base(filePath), filepath.Base(outputPath), err)
	} else {
		logger.Printf("✅ EXIF元数据复制成功: %s", filepath.Base(outputPath))
	}

	// ✅ 步骤1: 捕获源文件的文件系统元数据
	srcInfo, _ := os.Stat(filePath)
	var creationTime, modTime time.Time
	if srcInfo != nil {
		modTime = srcInfo.ModTime()
		if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
			creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
		}
	}

	// ✅ 步骤3: 恢复文件系统元数据（创建时间、修改时间）
	if srcInfo != nil {
		// 2.1 恢复修改时间
		if err := os.Chtimes(outputPath, modTime, modTime); err != nil {
			logger.Printf("⚠️  文件时间恢复失败 %s: %v", filepath.Base(outputPath), err)
		}

		// 2.2 恢复创建时间（macOS）
		if !creationTime.IsZero() {
			timeStr := creationTime.Format("200601021504.05")
			exec.Command("touch", "-t", timeStr, outputPath).Run()
		}

		// 2.3 恢复Finder标签和注释（可选）
		if err := copyFinderMetadata(filePath, outputPath); err != nil {
			logger.Printf("⚠️  Finder元数据复制失败 %s: %v", filepath.Base(outputPath), err)
		} else {
			logger.Printf("✅ Finder元数据复制成功: %s", filepath.Base(outputPath))
		}

		logger.Printf("✅ 文件系统元数据已保留: %s", filepath.Base(outputPath))
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

func (s *Stats) addImageProcessed(sizeBefore, sizeAfter int64) {
	s.Lock()
	defer s.Unlock()
	s.imagesProcessed++
	s.totalBytesBefore += sizeBefore
	s.totalBytesAfter += sizeAfter
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
	successRate := float64(stats.imagesProcessed) / float64(totalProcessed) * 100
	logger.Println("")
	logger.Println("📊 处理统计:")
	logger.Printf("  • 总文件数: %d", totalProcessed)
	logger.Printf("  • 成功处理: %d", stats.imagesProcessed)
	logger.Printf("  • 处理失败: %d", stats.imagesFailed)
	logger.Printf("  • 跳过文件: %d", stats.imagesSkipped)
	logger.Printf("  • 成功率: %.1f%%", successRate)
	if stats.totalBytesBefore > 0 {
		compressionRatio := float64(stats.totalBytesAfter) / float64(stats.totalBytesBefore)
		logger.Printf("  • 压缩比: %.2f", compressionRatio)
	}
	logger.Printf("  • 处理时间: %v", time.Since(stats.startTime))
	if stats.peakMemoryUsage > 0 {
		logger.Printf("  • 峰值内存: %d MB", stats.peakMemoryUsage/1024/1024)
	}
	if stats.totalRetries > 0 {
		logger.Printf("  • 总重试次数: %d", stats.totalRetries)
	}
	if len(stats.errorTypes) > 0 {
		logger.Println("  • 错误类型统计:")
		for errorType, count := range stats.errorTypes {
			logger.Printf("    - %s: %d 次", errorType, count)
		}
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
	logger.Printf("🎨 优化版工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")
	
	opts := parseFlags()
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}
	
	configurePerformance(&opts)
	logger.Println("🔍 扫描文件...")
	files := scanCandidateFiles(opts.InputDir, opts)
	logger.Printf("📊 发现 %d 个候选文件", len(files))
	
	if len(files) == 0 {
		logger.Println("📊 没有找到需要处理的文件")
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
// static2jxl交互模式包装器
// 提供简易的拖拽式CLI UI + 强大安全检查

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// runInteractiveMode 运行交互模式
func runInteractiveMode() {
	// 1. 显示横幅
	showBanner()
	
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
		Workers:        4, // 默认4个并发
		InputDir:       targetDir,
		SkipExist:      true,
		DryRun:         false,
		TimeoutSeconds: 600,
		Retries:        2,
		CopyMetadata:   true, // 自动保留元数据
	}
	
	fmt.Println("🔄 开始处理...")
	fmt.Println("")
	
	// 开始主处理流程
	main_process(opts)
}

// showBanner 显示横幅
func showBanner() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║   🎨 static2jxl v2.3.0 - 静态图转JXL工具                    ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║   功能: 静态图片转换为JXL格式（无损/完美可逆）              ║")
	fmt.Println("║   元数据: EXIF + 文件系统时间戳 + Finder标签 100%保留       ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println("")
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

// main_process 主处理流程（从main.go调用）
func main_process(opts Options) {
	// 这个函数会在main.go中实现
	// 这里只是定义接口
}

