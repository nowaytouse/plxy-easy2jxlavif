// static2avif - 静态图像转AVIF格式工具
//
// 功能说明：
// - 专门处理静态图像文件转换为AVIF格式
// - 支持多种静态图像格式（JPEG、PNG、BMP、TIFF等）
// - 保留原始文件的元数据和系统时间戳
// - 提供详细的处理统计和进度报告
// - 支持并发处理以提高转换效率
// - 使用ImageMagick进行高质量转换
//
// 作者：AI Assistant
// 版本：2.1.0
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
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"pixly/utils"

	"github.com/h2non/filetype/types"
	"github.com/karrick/godirwalk"
	"github.com/panjf2000/ants/v2"
)

// 程序常量定义
const (
	logFileName = "static2avif.log" // 日志文件名
	version     = "2.1.0"           // 程序版本号
	author      = "AI Assistant"    // 作者信息
)

// 全局变量定义
var (
	logger *log.Logger // 全局日志记录器，同时输出到控制台和文件

	// 并发控制信号量，用于限制外部进程和文件句柄的并发数量
	// 防止系统资源过载导致程序卡死或崩溃
	procSem chan struct{} // 外部进程并发限制信号量
	fdSem   chan struct{} // 文件句柄并发限制信号量
)

// Options 结构体定义了程序的配置选项
// 这些选项控制着转换过程的各种参数和行为
type Options struct {
	Workers        int    // 并发工作线程数，控制同时处理的文件数量
	Quality        int    // 图像质量（1-100）
	Speed          int    // 编码速度（1-10）
	SkipExist      bool   // 是否跳过已存在的AVIF文件
	DryRun         bool   // 试运行模式，只显示将要处理的文件而不实际转换
	TimeoutSeconds int    // 单个文件处理的超时时间（秒）
	Retries        int    // 转换失败时的重试次数
	InputDir       string // 输入目录路径
	OutputDir      string // 输出目录路径，默认为输入目录
}

// FileProcessInfo 结构体用于记录单个文件在处理过程中的详细信息
// 这对于生成详细的处理报告和调试非常有用
type FileProcessInfo struct {
	FilePath       string        // 文件完整路径
	FileSize       int64         // 文件大小（字节）
	FileType       string        // 文件类型（扩展名）
	ProcessingTime time.Duration // 处理耗时
	ConversionMode string        // 转换模式
	Success        bool          // 是否处理成功
	ErrorMsg       string        // 错误信息（如果处理失败）
	SizeSaved      int64         // 节省的空间大小
}

// Stats 结构体用于在整个批处理过程中收集和管理统计数据
// 它使用互斥锁（sync.Mutex）来确保并发访问时的线程安全
type Stats struct {
	sync.Mutex                         // 互斥锁，确保并发安全
	imagesProcessed  int               // 成功处理的图像数量
	imagesFailed     int               // 处理失败的图像数量
	videosSkipped    int               // 跳过的视频文件数量
	symlinksSkipped  int               // 跳过的符号链接数量
	othersSkipped    int               // 跳过的其他文件数量
	totalBytesBefore int64             // 原始文件总大小
	totalBytesAfter  int64             // 转换后文件总大小
	byExt            map[string]int    // 按扩展名统计的文件数量
	detailedLogs     []FileProcessInfo // 详细的处理日志记录
}

// addImageProcessed 原子性地增加成功处理图像的计数
func (s *Stats) addImageProcessed(sizeBefore, sizeAfter int64) {
	s.Lock()
	defer s.Unlock()
	s.imagesProcessed++
	s.totalBytesBefore += sizeBefore
	s.totalBytesAfter += sizeAfter
}

// addImageFailed 原子性地增加处理失败图像的计数
func (s *Stats) addImageFailed() {
	s.Lock()
	defer s.Unlock()
	s.imagesFailed++
}

// addVideoSkipped 原子性地增加跳过视频文件的计数
func (s *Stats) addVideoSkipped() {
	s.Lock()
	defer s.Unlock()
	s.videosSkipped++
}

// addSymlinkSkipped 原子性地增加跳过符号链接的计数
func (s *Stats) addSymlinkSkipped() {
	s.Lock()
	defer s.Unlock()
	s.symlinksSkipped++
}

// addOtherSkipped 原子性地增加跳过其他文件的计数
func (s *Stats) addOtherSkipped() {
	s.Lock()
	defer s.Unlock()
	s.othersSkipped++
}

// addDetailedLog 线程安全地向详细日志中添加一条处理记录
func (s *Stats) addDetailedLog(info FileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

// init 函数在main函数之前执行，用于初始化日志记录器和并发控制信号量
func init() {
	// 设置日志记录器，带大小轮转，同时输出到控制台和文件
	rl, lf, err := utils.NewRotatingLogger(logFileName, 50*1024*1024)
	if err != nil {
		log.Fatalf("无法初始化轮转日志: %v", err)
	}
	logger = rl
	_ = lf

	// 初始化并发控制信号量，防止系统资源过载
	cpuCount := runtime.NumCPU()
	procLimit := cpuCount / 2
	if procLimit < 2 {
		procLimit = 2
	}
	if procLimit > 4 {
		procLimit = 4 // 更严格的进程限制，防止系统卡死
	}
	procSem = make(chan struct{}, procLimit)
	fdSem = make(chan struct{}, procLimit*2)
}

// main 函数是程序的入口点
func main() {
	logger.Printf("🎨 静态图片转AVIF工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 解析命令行参数
	opts := parseFlags()

	// 验证输入和输出目录
	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录")
	}
	if opts.OutputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输出目录")
	}
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		logger.Fatalf("❌ 错误: 无法创建输出目录 %s: %v", opts.OutputDir, err)
	}
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}

	// 检查系统依赖工具是否可用
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Printf("❌ 系统依赖检查失败: %v", err)
		return
	}

	// 设置信号处理，以实现优雅的程序中断
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("\n🛑 收到中断信号，正在优雅退出...")
		cancel()
	}()

	// 初始化统计数据结构体
	stats := &Stats{
		byExt: make(map[string]int),
	}

	files, err := processDirectory(ctx, opts, stats)
	if err != nil {
		logger.Fatalf("❌ 处理目录时出错: %v", err)
	}

	// 输出统计信息
	printSummary(stats)
	validateFileCount(opts.InputDir, len(files), stats)
}

// checkDependencies 检查系统依赖工具是否可用
// 返回错误如果任何必需的依赖工具不可用
func checkDependencies() error {
	dependencies := []string{"magick", "exiftool"}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("缺少依赖: %s", dep)
		}
	}
	logger.Printf("✅ magick 已就绪")
	logger.Printf("✅ exiftool 已就绪")
	return nil
}

func parseFlags() *Options {
	opts := &Options{
		Workers:        0,  // 默认值将在后续设置
		Quality:        50, // 默认质量50 (范围0-100)
		Speed:          6,  // 默认速度6 (范围0-10)
		SkipExist:      false,
		DryRun:         false,
		TimeoutSeconds: 120, // 默认2分钟超时
		Retries:        2,   // 默认重试2次
	}

	flag.IntVar(&opts.Workers, "workers", opts.Workers, "并发工作线程数 (默认: CPU核心数)")
	flag.IntVar(&opts.Quality, "quality", opts.Quality, "AVIF质量 (0-100, 默认50)")
	flag.IntVar(&opts.Speed, "speed", opts.Speed, "编码速度 (0-10, 默认6)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", opts.SkipExist, "跳过已存在的文件")
	flag.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "试运行模式，只打印将要处理的文件")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", opts.TimeoutSeconds, "单个文件处理超时秒数")
	flag.IntVar(&opts.Retries, "retries", opts.Retries, "失败重试次数")
	flag.StringVar(&opts.InputDir, "input", "", "输入目录 (必需)")
	flag.StringVar(&opts.OutputDir, "output", "", "输出目录 (必需)")

	flag.Parse()

	// 验证参数
	if opts.Quality < 0 || opts.Quality > 100 {
		logger.Fatal("❌ 错误: 质量参数必须在0-100之间")
	}

	if opts.Speed < 0 || opts.Speed > 10 {
		logger.Fatal("❌ 错误: 速度参数必须在0-10之间")
	}

	if opts.Workers < 0 || opts.Workers > 100 {
		logger.Fatal("❌ 错误: 工作线程数必须在0-100之间")
	}

	if opts.TimeoutSeconds < 1 || opts.TimeoutSeconds > 3600 {
		logger.Fatal("❌ 错误: 超时时间必须在1-3600秒之间")
	}

	if opts.Retries < 0 || opts.Retries > 10 {
		logger.Fatal("❌ 错误: 重试次数必须在0-10之间")
	}

	return opts
}

// validateFileType 验证文件类型是否合法
func validateFileType(filePath string) error {
	// 检查文件路径是否包含非法字符
	if strings.ContainsAny(filePath, "\x00") {
		return fmt.Errorf("文件路径包含非法字符")
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return fmt.Errorf("文件没有扩展名")
	}

	// 检查是否是支持的格式
	supportedExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".bmp": true,
		".tif": true, ".tiff": true, ".webp": true, ".heic": true, ".heif": true,
	}
	if !supportedExtensions[ext] {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 读取文件头以验证实际类型
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	// 读取前261字节用于文件类型检测
	head := make([]byte, 261)
	_, err = file.Read(head)
	if err != nil && err != io.EOF {
		return fmt.Errorf("无法读取文件头: %v", err)
	}

	// 使用增强的文件类型检测
	enhancedType, err := utils.DetectFileType(filePath)
	if err != nil {
		return fmt.Errorf("文件类型检测失败: %v", err)
	}

	// 验证文件类型是否支持
	if !enhancedType.IsImage || !enhancedType.IsValid {
		return fmt.Errorf("不支持的文件类型: %s", enhancedType.Extension)
	}

	// 验证扩展名与检测结果匹配
	expectedExt := "." + enhancedType.Extension
	if ext != expectedExt && !isCompatibleExtension(ext, expectedExt) {
		// 对于某些特殊格式，允许扩展名差异
		specialFormats := map[string]bool{".ico": true, ".cur": true, ".jfif": true, ".jpe": true}
		if !specialFormats[ext] {
			return fmt.Errorf("文件内容(%s)与扩展名(%s)不匹配", expectedExt, ext)
		}
	}

	return nil
}

// isCompatibleExtension 检查两个扩展名是否兼容
func isCompatibleExtension(ext1, ext2 string) bool {
	compatiblePairs := map[string]string{
		".jpg":  ".jpeg",
		".jpeg": ".jpg",
		".tif":  ".tiff",
		".tiff": ".tif",
	}
	return compatiblePairs[ext1] == ext2
}

func processDirectory(ctx context.Context, opts *Options, stats *Stats) ([]string, error) {
	logger.Printf("📂 扫描目录: %s", opts.InputDir)

	// 使用 godirwalk 遍历目录
	files := make([]string, 0)
	err := godirwalk.Walk(opts.InputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			// 检查是否应该停止遍历
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if de.IsDir() {
				return nil
			}

			// 使用增强的文件类型检测（避免filetype无法识别HEIC/AVIF等问题）
			eft, err := utils.DetectFileType(osPathname)
			if err != nil {
				return nil
			}
			// 仅收集静态图像（排除动画）
			if eft.IsImage && !eft.IsAnimated {
				// 按扩展名再次确认静态类型
				ext := strings.ToLower(filepath.Ext(osPathname))
				if supportedStaticExtensions[ext] {
					files = append(files, osPathname)
				}
			}

			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, fmt.Errorf("目录扫描失败: %w", err)
	}

	logger.Printf("✅ 找到 %d 个支持的静态图像文件", len(files))

	if len(files) == 0 {
		logger.Println("⚠️  没有找到支持的静态图像文件")
		return files, nil
	}

	if opts.DryRun {
		logger.Println("🔍 试运行模式，将处理以下文件:")
		for _, file := range files {
			logger.Printf("  - %s", file)
		}
		return files, nil
	}

	// ⚡ 智能性能配置
	workers := opts.Workers
	cpuCount := runtime.NumCPU()

	if workers <= 0 {
		// 智能线程数配置：根据CPU核心数动态调整
		if cpuCount >= 16 {
			workers = cpuCount
		} else if cpuCount >= 8 {
			workers = cpuCount
		} else if cpuCount >= 4 {
			workers = cpuCount
		} else {
			workers = cpuCount
		}
	}

	// 安全限制：避免系统过载
	maxWorkers := cpuCount * 2
	if workers > maxWorkers {
		workers = maxWorkers
	}

	// 资源并发限制配置
	procLimit := cpuCount
	if procLimit > 8 {
		procLimit = 8 // 避免过多并发进程
	}
	fdLimit := procLimit * 4 // 文件句柄限制

	// 初始化线程池
	p, err := ants.NewPool(workers, ants.WithPreAlloc(true))
	if err != nil {
		logger.Printf("❌ 关键错误: 创建线程池失败: %v", err)
		return files, err
	}
	defer p.Release()

	// 初始化资源限制
	procSem = make(chan struct{}, procLimit)
	fdSem = make(chan struct{}, fdLimit)

	logger.Printf("⚡ 启动处理进程 (工作线程: %d)", workers)

	// 创建任务通道
	taskChan := make(chan string, len(files))
	resultChan := make(chan FileProcessInfo, len(files))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case filePath, ok := <-taskChan:
					if !ok {
						return
					}
					// 处理单个文件
					info := processFile(ctx, filePath, opts)
					resultChan <- info
				}
			}
		}()
	}

	// 发送任务到通道
	go func() {
		defer close(taskChan)
		for _, file := range files {
			select {
			case <-ctx.Done():
				return
			case taskChan <- file:
			}
		}
	}()

	// 启动结果收集协程
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		if result.Success {
			stats.addImageProcessed(result.FileSize, result.FileSize-result.SizeSaved)
		} else {
			stats.addImageFailed()
		}
		stats.addDetailedLog(result)

		// 统计扩展名
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(result.FilePath)), ".")
		if ext == "" {
			ext = "unknown"
		}
		stats.Lock()
		stats.byExt[ext]++
		stats.Unlock()
	}

	logger.Println("🎉 所有文件处理完成")
	return files, nil
}

func processFile(ctx context.Context, filePath string, opts *Options) FileProcessInfo {
	startTime := time.Now()
	fileName := filepath.Base(filePath)

	info := FileProcessInfo{
		FilePath: filePath,
		FileType: filepath.Ext(filePath),
	}

	// Get original file info for modification time and creation time
	var originalModTime time.Time
	var originalCreateTime time.Time
	if stat, err := os.Stat(filePath); err == nil {
		info.FileSize = stat.Size()
		originalModTime = stat.ModTime()
		if ctime, _, ok := getFileTimesDarwin(filePath); ok {
			originalCreateTime = ctime
		}
	}

	logger.Printf("🔄 开始处理: %s", fileName)

	// 检查是否应该跳过已存在的文件
	relPath, err := filepath.Rel(opts.InputDir, filePath)
	if err != nil {
		info.ErrorMsg = fmt.Sprintf("无法获取相对路径: %v", err)
		info.ProcessingTime = time.Since(startTime)
		return info
	}

	outputPath := filepath.Join(opts.OutputDir, relPath)
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".avif"

	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		info.ErrorMsg = fmt.Sprintf("无法创建输出目录: %v", err)
		info.ProcessingTime = time.Since(startTime)
		return info
	}

	// 检查是否跳过已存在的文件
	if opts.SkipExist {
		if _, err := os.Stat(outputPath); err == nil {
			logger.Printf("⏭️  跳过已存在的文件: %s", fileName)
			// 修复：跳过已存在的目标文件时，不删除原始文件
			// 这确保了原始数据的安全，避免误删文件
			info.Success = true
			info.ProcessingTime = time.Since(startTime)
			return info
		}
	}

	// 苹果Live Photo检测
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".heic" || ext == ".heif" {
		baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
		movPath := baseName + ".mov"
		if _, err := os.Stat(movPath); err == nil {
			logger.Printf("🏞️  检测到苹果Live Photo，跳过HEIC转换: %s", fileName)
			info.ErrorMsg = "跳过Live Photo"
			info.ProcessingTime = time.Since(startTime)
			return info
		}
	}

	// 🔄 执行转换（支持重试）
	var success bool
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		logger.Printf("🔄 开始转换 %s (尝试 %d/%d)", fileName, attempt+1, opts.Retries+1)
		err = convertToAvifWithOpts(filePath, outputPath, opts)
		if err != nil {
			if attempt == opts.Retries {
				logger.Printf("❌ 转换失败 %s: %v", fileName, err)
				info.ErrorMsg = fmt.Sprintf("转换失败: %v", err)
				info.ProcessingTime = time.Since(startTime)
				return info
			}
			logger.Printf("🔄 重试转换 %s (尝试 %d/%d)", fileName, attempt+1, opts.Retries)
			continue
		}
		success = true
		break
	}

	if !success {
		info.ProcessingTime = time.Since(startTime)
		return info
	}

	info.Success = true
	logger.Printf("✅ 转换完成: %s -> %s", fileName, filepath.Base(outputPath))

	// 获取新文件大小以计算节省的空间
	if stat, err := os.Stat(outputPath); err == nil {
		info.SizeSaved = info.FileSize - stat.Size()
	}

	// Set modification time for the new file
	err = os.Chtimes(outputPath, originalModTime, originalModTime)
	if err != nil {
		logger.Printf("WARN: Failed to set modification time for %s: %v", outputPath, err)
	}
	// On macOS, try to sync Finder visible creation/modification dates
	if runtime.GOOS == "darwin" && !originalCreateTime.IsZero() {
		if e := setFinderDates(outputPath, originalCreateTime, originalModTime); e != nil {
			logger.Printf("WARN: Failed to set Finder dates for %s: %v", outputPath, e)
		}
	}

	info.ProcessingTime = time.Since(startTime)
	return info
}

func convertToAvifWithOpts(filePath, outputPath string, opts *Options) error {
	originalFilePath := filePath // Preserve original file path for metadata copy
	ext := strings.ToLower(filepath.Ext(filePath))
	var tempPngPath string
	var tempRelaxedPngPath string

	// For HEIC/HEIF, convert to a stable intermediate format (PNG) first using enhanced methods.
	if ext == ".heic" || ext == ".heif" {
		tempPngPath = outputPath + ".png"
		logger.Printf("INFO: [HEIC] Converting to PNG intermediate: %s", filepath.Base(tempPngPath))

		// Approach 1: Use ImageMagick with increased limits to convert to png first
		cmd := exec.Command("magick", "-define", "heic:limit-num-tiles=0", "-define", "heic:max-image-size=0", "-define", "heic:use-embedded-profile=false", filePath, tempPngPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Printf("WARN: ImageMagick failed for %s: %v. Output: %s. Trying alternative method.", filepath.Base(filePath), err, string(output))

			// Approach 2: Use ffmpeg as fallback to convert HEIC to PNG with multiple options
			// First, get the actual dimensions of the HEIC file to ensure we extract the full resolution
			var ffmpegOutput []byte
			var ffmpegErr error
			dimCmd := exec.Command("exiftool", "-s", "-S", "-ImageWidth", "-ImageHeight", filePath)
			dimOutput, dimErr := dimCmd.CombinedOutput()

			if dimErr != nil {
				// If exiftool fails, fall back to default approach
				logger.Printf("WARN: Exiftool dimension detection failed for %s: %v. Falling back to default method.", filepath.Base(filePath), dimErr)
				cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
				ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
				if ffmpegErr != nil {
					// If that fails, try with different parameters
					logger.Printf("WARN: Default ffmpeg method failed for %s: %v. Output: %s. Trying enhanced approach.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-vcodec", "png", "-frames:v", "1", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
					if ffmpegErr != nil {
						logger.Printf("WARN: Second ffmpeg attempt failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
					}
				}
			} else {
				// Parse the dimensions from exiftool output
				lines := strings.Split(strings.TrimSpace(string(dimOutput)), "\n")
				var width, height int

				// Handle numeric format from exiftool
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}

					// Try simple numeric format (just the numbers)
					if intValue, err := strconv.Atoi(line); err == nil {
						// Assume first number is width, second is height
						if width == 0 {
							width = intValue
						} else if height == 0 {
							height = intValue
						}
					}
				}

				// If we have valid dimensions, use them for proper scaling
				if width > 0 && height > 0 {
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-vf", fmt.Sprintf("scale=%d:%d", width, height), "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
					if ffmpegErr != nil {
						logger.Printf("WARN: Scaled ffmpeg method failed for %s: %v. Output: %s. Trying unscaled approach.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
						// Try without scaling if that fails
						cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
						ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
						if ffmpegErr != nil {
							logger.Printf("WARN: Unscaled ffmpeg method also failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
						}
					}
				} else {
					// Fall back to default approach if dimensions are invalid
					logger.Printf("WARN: Invalid dimensions detected for %s (width: %d, height: %d). Falling back to default method.", filepath.Base(filePath), width, height)
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
				}
			}

			// Only if both ffmpeg and ImageMagick approaches fail, try ImageMagick with more relaxed limits
			if ffmpegErr != nil {
				logger.Printf("WARN: Ffmpeg failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))

				// Approach 3: Try using ImageMagick with even more relaxed policy
				tempRelaxedPngPath = outputPath + ".relaxed.png"
				cmd = exec.Command("magick", "-define", "heic:limit-num-tiles=0", "-define", "heic:max-image-size=0", "-define", "heic:use-embedded-profile=false", "-define", "heic:decode-effort=0", "-depth", "16", filePath, tempRelaxedPngPath)
				output, err = cmd.CombinedOutput()
				if err != nil {
					logger.Printf("WARN: All HEIC conversion methods failed for %s. ImageMagick, ffmpeg, and relaxed ImageMagick all failed. Output ImageMagick: %s, ffmpeg: %s, relaxed ImageMagick: %s",
						filepath.Base(filePath), string(output), string(ffmpegOutput), string(output))
					return fmt.Errorf("all HEIC conversion methods failed: ImageMagick error: %v, ffmpeg error: %v", err, ffmpegErr)
				}
				// Use the relaxed ImageMagick output
				filePath = tempRelaxedPngPath
			} else {
				// Successfully converted with ffmpeg, now use PNG as input
				filePath = tempPngPath
			}
		} else {
			// Successfully converted with original ImageMagick approach
			filePath = tempPngPath
		}
	}

	// 使用ffmpeg进行转换
	// 构建ffmpeg命令参数
	args := []string{
		"-i", filePath, // 输入文件
		"-c:v", "libsvtav1", // 使用SVT-AV1编码器
		"-crf", fmt.Sprintf("%d", 50-opts.Quality/2), // CRF值基于质量参数 (质量越高，CRF越低)
		"-preset", fmt.Sprintf("%d", opts.Speed), // 编码速度
		"-pix_fmt", "yuv420p", // 像素格式
		"-y",       // 覆盖输出文件
		outputPath, // 输出文件
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	// 限制并发进程数
	procSem <- struct{}{}
	defer func() { <-procSem }()

	// 执行ffmpeg命令
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if tempPngPath != "" {
		os.Remove(tempPngPath)
	}
	if tempRelaxedPngPath != "" {
		os.Remove(tempRelaxedPngPath)
	}
	if err != nil {
		return fmt.Errorf("ffmpeg执行失败: %w\n输出: %s", err, string(output))
	}

	// 统一8层验证（严格质量优先）
	enhancedType, _ := utils.DetectFileType(filePath)
	tol := 5.0
	if opts.Quality >= 85 {
		tol = 1.0
	} else if opts.Quality >= 70 {
		tol = 2.0
	}
	validator := utils.NewEightLayerValidator(utils.ValidationOptions{TimeoutSeconds: opts.TimeoutSeconds, CJXLThreads: runtime.NumCPU(), StrictMode: true, AllowTolerance: tol})
	if vr, vErr := validator.ValidateConversion(originalFilePath, outputPath, enhancedType); vErr != nil {
		logger.Printf("❌ 验证失败 %s: %v", filepath.Base(filePath), vErr)
		_ = os.Remove(outputPath)
		return fmt.Errorf("验证失败: %w", vErr)
	} else if !vr.Success {
		logger.Printf("❌ 验证失败 %s: %s (第%d层: %s)", filepath.Base(filePath), vr.Message, vr.Layer, vr.LayerName)
		_ = os.Remove(outputPath)
		return fmt.Errorf("验证失败: %s", vr.Message)
	} else {
		logger.Printf("✅ 验证通过: %s (%s)", filepath.Base(filePath), vr.Message)
	}

	// 复制元数据
	if err := copyMetadata(originalFilePath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败 %s: %v", filepath.Base(originalFilePath), err)
	}

	return nil
}

func copyMetadata(sourcePath, targetPath string) error {
	// 使用exiftool复制元数据
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", sourcePath, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool failed: %s\nOutput: %s", err, string(output))
	}
	logger.Printf("📋 元数据复制成功: %s", filepath.Base(sourcePath))
	return nil
}

var supportedStaticExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".bmp": true,
	".tiff": true, ".tif": true, ".heic": true, ".heif": true,
	".jfif": true, ".jpe": true,
}

func isSupportedStaticType(kind types.Type) bool {
	return supportedStaticExtensions[kind.Extension]
}

// getFileTimesDarwin 尝试获取文件的创建/修改时间（macOS）
func getFileTimesDarwin(p string) (ctime, mtime time.Time, ok bool) {
	if runtime.GOOS != "darwin" {
		return time.Time{}, time.Time{}, false
	}
	fi, err := os.Stat(p)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	// 修改时间直接取
	mtime = fi.ModTime()
	// 创建时间通过 mdls 提取 kMDItemFSCreationDate
	out, err := exec.Command("mdls", "-raw", "-name", "kMDItemFSCreationDate", p).CombinedOutput()
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	ss := strings.TrimSpace(string(out))
	// 示例: 2024-10-02 22:33:44 +0000
	t, perr := time.Parse("2006-01-02 15:04:05 -0700", ss)
	if perr != nil {
		return time.Time{}, time.Time{}, false
	}
	return t, mtime, true
}

// setFinderDates 通过 exiftool 设置文件的文件系统日期（Finder 可见）
func setFinderDates(p string, ctime, mtime time.Time) error {
	// exiftool -overwrite_original -P -FileCreateDate=... -FileModifyDate=...
	layout := "2006:01:02 15:04:05"
	args := []string{
		"-overwrite_original",
		"-P",
		"-FileCreateDate=" + ctime.Local().Format(layout),
		"-FileModifyDate=" + mtime.Local().Format(layout),
		p,
	}
	out, err := exec.Command("exiftool", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool set Finder dates failed: %v, out=%s", err, string(out))
	}
	return nil
}

func printSummary(stats *Stats) {
	stats.Lock()
	defer stats.Unlock()

	totalSavedKB := float64(stats.totalBytesBefore-stats.totalBytesAfter) / 1024.0
	totalSavedMB := totalSavedKB / 1024.0
	compressionRatio := float64(stats.totalBytesAfter) / float64(stats.totalBytesBefore) * 100

	logger.Println("🎯 ===== 处理摘要 ====")
	logger.Printf("✅ 成功处理图像: %d", stats.imagesProcessed)
	logger.Printf("❌ 转换失败图像: %d", stats.imagesFailed)
	logger.Printf("🎬 跳过视频文件: %d", stats.videosSkipped)
	logger.Printf("🔗 跳过符号链接: %d", stats.symlinksSkipped)
	logger.Printf("📄 跳过其他文件: %d", stats.othersSkipped)
	logger.Println("📊 ===== 大小统计 ====")
	logger.Printf("📥 原始总大小: %.2f MB", float64(stats.totalBytesBefore)/(1024*1024))
	logger.Printf("📤 转换后大小: %.2f MB", float64(stats.totalBytesAfter)/(1024*1024))
	logger.Printf("💾 节省空间: %.2f MB (压缩率: %.1f%%)", totalSavedMB, compressionRatio)

	if len(stats.byExt) > 0 {
		logger.Println("📋 ===== 格式统计 ====")
		for k, v := range stats.byExt {
			logger.Printf("  🖼️  %s: %d个文件", k, v)
		}
	}
	logger.Println("🎉 ===== 处理完成 ====")
}

func validateFileCount(workDir string, originalMediaCount int, stats *Stats) {
	currentMediaCount := 0
	currentAvifCount := 0
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				ext := strings.ToLower(filepath.Ext(osPathname))
				if supportedStaticExtensions[ext] {
					currentMediaCount++
				} else if ext == ".avif" {
					currentAvifCount++
				}
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})

	if err != nil {
		logger.Printf("⚠️  文件数量验证失败: %v", err)
		return
	}

	expectedAvifCount := stats.imagesProcessed
	expectedMediaCount := originalMediaCount - stats.imagesProcessed

	logger.Printf("📊 文件数量验证:")
	logger.Printf("   原始媒体文件数: %d", originalMediaCount)
	logger.Printf("   成功处理图像: %d", stats.imagesProcessed)
	logger.Printf("   转换失败/跳过: %d", stats.imagesFailed+stats.videosSkipped+stats.othersSkipped)
	logger.Printf("   ---")
	logger.Printf("   期望AVIF文件数: %d", expectedAvifCount)
	logger.Printf("   实际AVIF文件数: %d", currentAvifCount)
	if currentAvifCount == expectedAvifCount {
		logger.Printf("✅ 目标格式数量匹配。\n")
	} else {
		logger.Printf("⚠️  目标格式数量不匹配 (实际: %d, 期望: %d) —— 仅提示，不判失败。\n", currentAvifCount, expectedAvifCount)
	}

	// 对目录中未处理的原媒体数量，仅提示，不判失败（避免混合目录误报）
	logger.Printf("   ---")
	logger.Printf("   期望剩余媒体文件数(参考): %d", expectedMediaCount)
	logger.Printf("   实际剩余媒体文件数: %d", currentMediaCount)
	if currentMediaCount != expectedMediaCount {
		logger.Printf("ℹ️  目录包含未处理的其他原文件，忽略为提示。\n")
	}

	// 查找并清理临时文件
	tempFiles := findTempFiles(workDir)
	if len(tempFiles) > 0 {
		logger.Printf("🗑️  发现 %d 个临时文件，正在清理...\n", len(tempFiles))
		cleanupTempFiles(tempFiles)
		logger.Printf("✅ 临时文件清理完成\n")
	}
}

func findTempFiles(workDir string) []string {
	var tempFiles []string
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				// 查找临时文件模式
				if strings.Contains(filepath.Base(osPathname), ".avif.tmp") ||
					strings.Contains(filepath.Base(osPathname), ".tmp") ||
					strings.HasSuffix(osPathname, ".tmp") {
					tempFiles = append(tempFiles, osPathname)
				}
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})

	if err != nil {
		logger.Printf("⚠️  查找临时文件失败: %v", err)
	}

	return tempFiles
}

func cleanupTempFiles(tempFiles []string) {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			logger.Printf("⚠️  删除临时文件失败 %s: %v", filepath.Base(file), err)
		} else {
			logger.Printf("🗑️  已删除临时文件: %s", filepath.Base(file))
		}
	}
}
