// dynamic2jxl - 动态图像转JPEG XL格式工具
//
// 功能说明：
// - 专门处理动态图像文件转换为JPEG XL格式
// - 支持多种动态图像格式（GIF、APNG、WebP、AVIF、HEIF等）
// - 保留原始文件的元数据和系统时间戳
// - 提供详细的处理统计和进度报告
// - 支持并发处理以提高转换效率
// - 使用CJXL编码器进行高质量转换
//
// 作者：AI Assistant
// 版本：2.1.0
package main

import (
	"context"
	"flag"
	"fmt"
	"image/gif"
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

	"github.com/karrick/godirwalk"
	"github.com/panjf2000/ants/v2"
)

// 程序常量定义
const (
	logFileName = "dynamic2jxl.log" // 日志文件名
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
	SkipExist      bool   // 是否跳过已存在的JXL文件
	DryRun         bool   // 试运行模式，只显示将要处理的文件而不实际转换
	CJXLThreads    int    // CJXL编码器使用的线程数
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
	IsAnimated     bool          // 是否为动画图像
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
	// 设置日志记录器，同时输出到控制台和文件
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)

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

func main() {
	// 🚀 程序启动
	logger.Printf("🎨 动态图片转JXL工具 v%s", version)
	logger.Println("✨ 作者:", author)
	logger.Println("🔧 开始初始化...")

	// 解析命令行参数
	opts := parseFlags()

	// 检查输入目录
	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录")
	}

	// 检查输出目录
	if opts.OutputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输出目录")
	}

	// 确保输出目录存在
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		logger.Fatalf("❌ 错误: 无法创建输出目录 %s: %v", opts.OutputDir, err)
	}

	// 检查输入目录是否存在
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}

	// 检查系统依赖工具是否可用
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Printf("❌ 系统依赖检查失败: %v", err)
		return
	}

	// 注册信号处理函数以实现优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("\n🛑 收到中断信号，正在优雅退出...")
		cancel()
	}()

	// 执行转换
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
	dependencies := []string{"cjxl", "exiftool"}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("缺少依赖: %s", dep)
		}
	}
	logger.Printf("✅ cjxl 已就绪")
	logger.Printf("✅ exiftool 已就绪")
	return nil
}

func parseFlags() *Options {
	opts := &Options{
		Workers:        0, // 默认值将在后续设置
		SkipExist:      true,
		DryRun:         false,
		CJXLThreads:    1,
		TimeoutSeconds: 300, // 默认5分钟超时
		Retries:        2,   // 默认重试2次
	}

	flag.IntVar(&opts.Workers, "workers", opts.Workers, "并发工作线程数 (默认: CPU核心数)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", opts.SkipExist, "跳过已存在的文件")
	flag.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "试运行模式，只打印将要处理的文件")
	flag.IntVar(&opts.CJXLThreads, "cjxl-threads", opts.CJXLThreads, "每个转换任务的线程数")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", opts.TimeoutSeconds, "单个文件处理超时秒数")
	flag.IntVar(&opts.Retries, "retries", opts.Retries, "失败重试次数")
	flag.StringVar(&opts.InputDir, "input", "", "输入目录 (必需)")
	flag.StringVar(&opts.OutputDir, "output", "", "输出目录 (必需)")

	flag.Parse()

	return opts
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

			// 检查文件类型
			ext := strings.ToLower(filepath.Ext(osPathname))
			if isSupportedDynamicType(ext) {
				files = append(files, osPathname)
			}

			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, fmt.Errorf("目录扫描失败: %w", err)
	}

	logger.Printf("✅ 找到 %d 个支持的动态图像文件", len(files))

	if len(files) == 0 {
		logger.Println("⚠️  没有找到支持的动态图像文件")
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
		workers = cpuCount
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
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".jxl"

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

	// 检测是否为动画图像
	isAnimated, animErr := isAnimatedImage(filePath)
	if animErr != nil {
		logger.Printf("⚠️  动画检测失败 %s: %v", fileName, animErr)
		isAnimated = false
	}
	info.IsAnimated = isAnimated

	if isAnimated {
		logger.Printf("🎬 检测到动画图像: %s", fileName)
		info.ConversionMode = "Animated Conversion"
	} else {
		logger.Printf("🖼️  静态图像: %s", fileName)
		info.ConversionMode = "Static Conversion"
	}

	// 🔄 执行转换（支持重试）
	var success bool
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		logger.Printf("🔄 开始转换 %s (尝试 %d/%d)", fileName, attempt+1, opts.Retries+1)
		err = convertToJxlWithOpts(filePath, outputPath, opts)
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

	// 获取新文件大小以计算节省的空间
	if stat, err := os.Stat(outputPath); err == nil {
		info.SizeSaved = info.FileSize - stat.Size()
	}

	info.ProcessingTime = time.Since(startTime)
	return info
}

func convertToJxlWithOpts(filePath, outputPath string, opts *Options) error {
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

	// 使用cjxl进行转换
	args := []string{
		filePath,
		outputPath,
		"-d", "0", // 无损压缩
		"-e", "9", // 最高效率
		"--num_threads", fmt.Sprintf("%d", opts.CJXLThreads),
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	// 限制并发进程数
	procSem <- struct{}{}
	defer func() { <-procSem }()

	// 执行cjxl命令
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	output, err := cmd.CombinedOutput()
	if tempPngPath != "" {
		os.Remove(tempPngPath)
	}
	if tempRelaxedPngPath != "" {
		os.Remove(tempRelaxedPngPath)
	}
	if err != nil {
		return fmt.Errorf("cjxl执行失败: %w\n输出: %s", err, string(output))
	}

	// 复制元数据
	if err := copyMetadata(originalFilePath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败 %s: %v", filepath.Base(originalFilePath), err)
	}

	return nil
}

var supportedDynamicExtensions = map[string]bool{
	".gif":  true,
	".webp": true,
	".apng": true,
	".heic": true,
	".heif": true,
}

func isSupportedDynamicType(ext string) bool {
	return supportedDynamicExtensions[ext]
}

// isAnimatedImage 检测是否为真实的动画图像
func isAnimatedImage(filePath string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".gif":
		return isAnimatedGIF(filePath)
	case ".apng":
		return isAnimatedPNG(filePath)
	case ".webp":
		return isAnimatedWebP(filePath)
	case ".heic", ".heif":
		return isAnimatedHEIF(filePath)
	default:
		return false, nil
	}
}

// isAnimatedGIF 检测GIF是否为动画
func isAnimatedGIF(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	g, err := gif.DecodeAll(file)
	if err != nil {
		return false, err
	}

	return len(g.Image) > 1, nil
}

// isAnimatedPNG 检测PNG是否为动画 (APNG)
func isAnimatedPNG(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// 读取PNG文件头
	header := make([]byte, 8)
	if _, err := file.Read(header); err != nil {
		return false, err
	}

	// 检查PNG签名
	if string(header[:8]) != "\x89PNG\r\n\x1a\n" {
		return false, nil
	}

	// 查找acTL块
	buffer := make([]byte, 8192)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return false, err
		}
		if n == 0 {
			break
		}

		// 在缓冲区中查找acTL
		if strings.Contains(string(buffer[:n]), "acTL") {
			return true, nil
		}
	}

	return false, nil
}

// isAnimatedWebP 检测WebP是否为动画
func isAnimatedWebP(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// 读取WebP文件头
	header := make([]byte, 12)
	if _, err := file.Read(header); err != nil {
		return false, err
	}

	// WebP文件头检查
	if len(header) < 12 || string(header[:4]) != "RIFF" || string(header[8:12]) != "WEBP" {
		return false, nil
	}

	// 查找ANIM块
	buffer := make([]byte, 8192)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return false, err
		}
		if n == 0 {
			break
		}

		// 在缓冲区中查找ANIM
		if strings.Contains(string(buffer[:n]), "ANIM") {
			return true, nil
		}
	}

	return false, nil
}

// isAnimatedHEIF 检测HEIF是否为动画
func isAnimatedHEIF(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// 读取HEIF文件头
	header := make([]byte, 12)
	if _, err := file.Read(header); err != nil {
		return false, err
	}

	// HEIF文件头检查
	if len(header) < 12 || string(header[:4]) != "ftyp" {
		return false, nil
	}

	// 查找动画相关标识
	buffer := make([]byte, 8192)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return false, err
		}
		if n == 0 {
			break
		}

		// 在缓冲区中查找动画标识
		if strings.Contains(string(buffer[:n]), "avis") ||
			strings.Contains(string(buffer[:n]), "anim") ||
			strings.Contains(string(buffer[:n]), "idat") {
			return true, nil
		}
	}

	return false, nil
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
	s := strings.TrimSpace(string(out))
	// 示例: 2024-10-02 22:33:44 +0000
	t, perr := time.Parse("2006-01-02 15:04:05 -0700", s)
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

// validateFileCount 验证处理前后的文件数量
func validateFileCount(workDir string, originalMediaCount int, stats *Stats) {
	currentMediaCount := 0
	currentJxlCount := 0
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				ext := strings.ToLower(filepath.Ext(osPathname))
				if supportedDynamicExtensions[ext] {
					currentMediaCount++
				} else if ext == ".jxl" {
					currentJxlCount++
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

	expectedJxlCount := stats.imagesProcessed
	expectedMediaCount := originalMediaCount - stats.imagesProcessed

	logger.Printf("📊 文件数量验证:")
	logger.Printf("   原始媒体文件数: %d", originalMediaCount)
	logger.Printf("   成功处理图像: %d", stats.imagesProcessed)
	logger.Printf("   转换失败/跳过: %d", stats.imagesFailed+stats.othersSkipped)
	logger.Printf("   ---")
	logger.Printf("   期望JXL文件数: %d", expectedJxlCount)
	logger.Printf("   实际JXL文件数: %d", currentJxlCount)
	logger.Printf("   ---")
	logger.Printf("   期望剩余媒体文件数: %d", expectedMediaCount)
	logger.Printf("   实际剩余媒体文件数: %d", currentMediaCount)

	if currentJxlCount == expectedJxlCount && currentMediaCount == expectedMediaCount {
		logger.Printf("✅ 文件数量验证通过。\n")
	} else {
		logger.Printf("❌ 文件数量验证失败。\n")
		if currentJxlCount != expectedJxlCount {
			logger.Printf("   JXL文件数不匹配 (实际: %d, 期望: %d)\n", currentJxlCount, expectedJxlCount)
		}
		if currentMediaCount != expectedMediaCount {
			logger.Printf("   剩余媒体文件数不匹配 (实际: %d, 期望: %d)\n", currentMediaCount, expectedMediaCount)
		}

		// 查找可能的临时文件
		tempFiles := findTempFiles(workDir)
		if len(tempFiles) > 0 {
			logger.Printf("🗑️  发现 %d 个临时文件，正在清理...\n", len(tempFiles))
			cleanupTempFiles(tempFiles)
			logger.Printf("✅ 临时文件清理完成\n")
		}
	}
}

// findTempFiles 查找临时文件
func findTempFiles(workDir string) []string {
	var tempFiles []string
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				// 查找临时文件模式
				if strings.Contains(filepath.Base(osPathname), ".jxl.tmp") ||
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

// cleanupTempFiles 清理临时文件
func cleanupTempFiles(tempFiles []string) {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			logger.Printf("⚠️  删除临时文件失败 %s: %v", filepath.Base(file), err)
		} else {
			logger.Printf("🗑️  已删除临时文件: %s", filepath.Base(file))
		}
	}
}
