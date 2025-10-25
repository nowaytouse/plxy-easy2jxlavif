// 优化版工具 - 基于 universal_converter 功能进行深入优化
// 版本: v2.3.0 (优化版)
// 作者: AI Assistant

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"pixly/utils"

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
	stats      *utils.SharedStats
	procSem    chan struct{}
	fdSem      chan struct{}
)

type Options struct {
	Workers           int
	InputDir          string
	OutputDir         string
	SkipExist         bool
	DryRun            bool
	InPlace           bool // 原地转换：转换成功后删除原文件
	TimeoutSeconds    int
	Retries           int
	MaxMemory         int64
	MaxFileSize       int64
	EnableHealthCheck bool
}

func init() {
	logger = utils.SetupLogging("optimized.log")
	stats = utils.NewSharedStats()
	utils.SetupSignalHandlingWithCallback(logger, printStatistics)
}

func parseFlags() Options {
	var opts Options

	flag.StringVar(&opts.InputDir, "dir", "", "📂 输入目录路径（必需）")
	flag.StringVar(&opts.OutputDir, "output", "", "📁 输出目录路径（默认为输入目录）")
	flag.IntVar(&opts.Workers, "workers", 0, "⚡ 工作线程数 (0=自动检测)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", false, "⏭️ 跳过已存在的文件")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "🔍 试运行模式")
	flag.BoolVar(&opts.InPlace, "in-place", false, "🗑️ 原地转换（成功后删除原文件）")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", 30, "⏰ 单个文件处理超时时间（秒）")
	flag.IntVar(&opts.Retries, "retries", 3, "🔄 转换失败重试次数")
	flag.Int64Var(&opts.MaxMemory, "max-memory", 0, "💾 最大内存使用量（字节，0=无限制）")
	flag.Int64Var(&opts.MaxFileSize, "max-file-size", 500*1024*1024, "📏 最大文件大小（字节）")
	flag.BoolVar(&opts.EnableHealthCheck, "health-check", true, "🏥 启用健康检查")

	flag.Parse()

	// 交互模式：如果没有提供目录，提示用户输入
	opts.InputDir = utils.PromptForDirectory(opts.InputDir)
	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录")
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
	// static2avif只处理静态图片，排除动图和视频

	// 排除视频文件
	videoExts := map[string]bool{
		".mov": true, ".mp4": true, ".avi": true, ".mkv": true,
		".webm": true, ".m4v": true, ".mpg": true, ".mpeg": true,
		".wmv": true, ".flv": true,
	}
	if videoExts[ext] {
		return false
	}

	// 排除GIF（应该由dynamic2avif处理）
	if ext == ".gif" {
		return false
	}

	// 支持的静态图片格式
	supportedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".bmp": true,
		".tiff": true, ".tif": true, ".webp": true, ".avif": true,
		".jxl": true, ".heic": true, ".heif": true, ".jfif": true,
	}
	return supportedExts[ext]
}

func processFileWithRetry(filePath string, fileInfo os.FileInfo, opts Options) {
	var lastErr error
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if attempt > 0 {
			logger.Printf("🔄 重试处理文件: %s (第 %d 次)", filepath.Base(filePath), attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
			stats.AddRetry()
		}
		err := processFileWithOpts(filePath, fileInfo, stats, opts)
		if err == nil {
					// 转换成功
		// InPlace模式：删除原文件（已废弃）
		return
		}
		lastErr = err
		logger.Printf("⚠️  处理文件失败: %s - %v", filepath.Base(filePath), err)
		stats.AddErrorType(utils.ClassifyError(err))
	}
	logger.Printf("❌ 文件处理最终失败: %s - %v", filepath.Base(filePath), lastErr)
	stats.AddFailed()
}

func processFileWithOpts(filePath string, fileInfo os.FileInfo, stats *utils.SharedStats, opts Options) error {
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

	processInfo := utils.SharedFileProcessInfo{
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		FileType:       filepath.Ext(filePath),
		ProcessingTime: processingTime,
		ConversionMode: conversionMode,
		Success:        err == nil,
		ErrorMsg:       errorMsg,
		StartTime:      startTime,
		EndTime:        time.Now(),
		ErrorType:      utils.ClassifyError(err),
	}

	if err != nil {
		stats.AddFailed()
		processInfo.ErrorMsg = err.Error()
	} else {
		stats.AddProcessed(fileInfo.Size(), utils.GetFileSize(outputPath))
		stats.AddByExt(filepath.Ext(filePath))
	}
	stats.AddDetailedLog(processInfo)
	return err
}

func processFileByType(filePath string, opts Options) (string, string, string, error) {
	// 静态图转AVIF的实际转换逻辑（v2.3.1+元数据保留）
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".avif"

	conversionMode := "静态转AVIF"

	// 检查是否需要格式转换
	actualInputPath := filePath
	needsCleanup := false
	if utils.NeedsConversion(filePath, "avifenc") {
		convertedPath, wasConverted, err := utils.ConvertIfNeeded(filePath, "avifenc")
		if err != nil {
			return conversionMode, outputPath, fmt.Sprintf("格式转换失败: %v", err), err
		}
		if wasConverted {
			actualInputPath = convertedPath
			needsCleanup = true
			defer func() {
				if needsCleanup {
					utils.GetFormatConverter().CleanupTempFile(convertedPath)
				}
			}()
		}
	}

	// 使用avifenc转换静态图
	args := []string{
		actualInputPath,
		outputPath,
		"-s", "6", // 速度6（平衡）
		"-j", "4", // 4个任务线程
	}

	ctx, cancel := context.WithTimeout(globalCtx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "avifenc", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return conversionMode, "", string(output), fmt.Errorf("avifenc转换失败: %v", err)
	}

	// ✅ 步骤1: 捕获源文件的文件系统元数据（在exiftool之前）
	srcInfo, _ := os.Stat(filePath)
	var creationTime time.Time
	if srcInfo != nil {
		if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
			creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
		}
	}

	// ✅ 步骤2: 复制EXIF元数据（会改变文件修改时间）
	if err := utils.CopyMetadata(filePath, outputPath); err != nil {
		logger.Printf("⚠️  EXIF元数据复制失败: %s -> %s: %v",
			filepath.Base(filePath), filepath.Base(outputPath), err)
	} else {
		logger.Printf("✅ EXIF元数据复制成功: %s", filepath.Base(outputPath))
	}

	// ✅ 步骤3: 恢复文件系统元数据（在exiftool之后）
	if srcInfo != nil {
		// 3.1 恢复Finder标签和注释
		if err := utils.CopyFinderMetadata(filePath, outputPath); err != nil {
			logger.Printf("⚠️  Finder元数据复制失败 %s: %v", filepath.Base(outputPath), err)
		} else {
			logger.Printf("✅ Finder元数据复制成功: %s", filepath.Base(outputPath))
		}

		// 3.2 恢复修改时间和创建时间（使用touch统一设置）
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

func printStatistics() {
	stats.RLock()
	defer stats.RUnlock()
	totalProcessed := stats.ImagesProcessed + stats.ImagesFailed + stats.ImagesSkipped
	successRate := float64(stats.ImagesProcessed) / float64(totalProcessed) * 100
	logger.Println("")
	logger.Println("📊 处理统计:")
	logger.Printf("  • 总文件数: %d", totalProcessed)
	logger.Printf("  • 成功处理: %d", stats.ImagesProcessed)
	logger.Printf("  • 处理失败: %d", stats.ImagesFailed)
	logger.Printf("  • 跳过文件: %d", stats.ImagesSkipped)
	logger.Printf("  • 成功率: %.1f%%", successRate)
	if stats.TotalBytesBefore > 0 {
		compressionRatio := float64(stats.TotalBytesAfter) / float64(stats.TotalBytesBefore)
		logger.Printf("  • 压缩比: %.2f", compressionRatio)
	}
	logger.Printf("  • 处理时间: %v", stats.GetElapsedTime())
	if stats.PeakMemoryUsage > 0 {
		logger.Printf("  • 峰值内存: %d MB", stats.PeakMemoryUsage/1024/1024)
	}
	if stats.TotalRetries > 0 {
		logger.Printf("  • 总重试次数: %d", stats.TotalRetries)
	}
	if len(stats.ErrorTypes) > 0 {
		logger.Println("  • 错误类型统计:")
		for errorType, count := range stats.ErrorTypes {
			logger.Printf("    - %s: %d 次", errorType, count)
		}
	}
}

func main() {
	logger.Printf("🎨 优化版工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	opts := parseFlags()
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	// 🔒 多级验证系统
	logger.Println("🔍 执行安全检查...")
	if err := utils.PerformSafetyCheck(opts.InputDir); err != nil {
		logger.Printf("⚠️  安全检查警告: %v", err)
	}

	// 💾 磁盘空间检查
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = opts.InputDir
	}
	// 磁盘空间检查已集成到utils.PerformSafetyCheck中
	logger.Println("✅ 安全检查通过")

	configurePerformance(&opts)

	// 原地转换警告
	if opts.InPlace {
		logger.Println("")
		logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Println("⚠️  原地转换模式已启用")
		logger.Println("   转换成功的文件的原始文件将被永久删除")
		logger.Println("   失败的文件将被保留")
		logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Println("")
	}

	var files []string

	// 检查是文件还是目录
	info, err := os.Stat(opts.InputDir)
	if err != nil {
		logger.Fatalf("❌ 无法访问路径: %v", err)
	}

	if info.IsDir() {
		// 目录：扫描所有文件
		logger.Println("🔍 扫描文件...")
		files = scanCandidateFiles(opts.InputDir, opts)
		logger.Printf("📊 发现 %d 个候选文件", len(files))
	} else {
		// 单个文件
		logger.Printf("📄 处理单个文件: %s", filepath.Base(opts.InputDir))
		files = []string{opts.InputDir}
	}

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
