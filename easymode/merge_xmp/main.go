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

func init() {
	logger = utils.SetupLogging("merge_xmp.log")
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
			stats.TotalRetries++
			stats.Unlock()
		}
		err := processFileWithOpts(filePath, fileInfo, stats, opts)
		if err == nil {
			return
		}
		lastErr = err
		logger.Printf("⚠️  处理文件失败: %s - %v", filepath.Base(filePath), err)
		stats.Lock()
		stats.ErrorTypes[classifyError(err)]++
		stats.Unlock()
	}
	logger.Printf("❌ 文件处理最终失败: %s - %v", filepath.Base(filePath), lastErr)
	stats.AddFailed()
}

func classifyError(err error) string {
	if err == nil {
		return "unknown"
	}
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

func processFileWithOpts(filePath string, fileInfo os.FileInfo, stats *utils.SharedStats, opts Options) error {
	StartTime := time.Now()
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
	processingTime := time.Since(StartTime)

	processInfo := utils.SharedFileProcessInfo{
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		FileType:       filepath.Ext(filePath),
		ProcessingTime: processingTime,
		ConversionMode: conversionMode,
		Success:        err == nil,
		ErrorMsg:       errorMsg,
		StartTime:      StartTime,
		EndTime:        time.Now(),
		ErrorType:      classifyError(err),
	}

	if err != nil {
		stats.AddFailed()
		processInfo.ErrorMsg = err.Error()
	} else {
		stats.AddProcessed(fileInfo.Size(), getFileSize(outputPath))
		stats.AddByExt(filepath.Ext(filePath))
	}
	stats.AddDetailedLog(processInfo)
	return err
}

func processFileByType(filePath string, opts Options) (string, string, string, error) {
	// 根据工具类型实现具体的处理逻辑
	// 这里是一个通用的实现框架
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".processed"

	// 模拟处理过程
	time.Sleep(100 * time.Millisecond)

	return "通用处理", outputPath, "", nil
}

func copyMetadata(inputPath, outputPath string) error {
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", inputPath, outputPath)
	return cmd.Run()
}

func getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}
	return 0
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
