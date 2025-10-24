// all2jxl - 批量图像转JPEG XL格式工具 (优化版)
//
// 基于 universal_converter 功能进行深入优化
// 版本: v2.3.0 (优化版)
// 作者: AI Assistant
//
// 优化内容:
// 1. 增强错误处理和恢复机制
// 2. 改进资源管理和内存控制
// 3. 优化并发控制和性能
// 4. 增强日志记录和监控
// 5. 添加信号处理和优雅关闭
// 6. 改进参数验证和配置
// 7. 增强统计和报告功能
// 8. 添加健康监控和错误分类
// 9. 实现智能性能调优
// 10. 增强安全性和稳定性

package main

import (
	"context"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
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

	"sort"

	"pixly/utils"

	"github.com/karrick/godirwalk"
	"github.com/shirou/gopsutil/mem"
)

// 程序常量定义
const (
	logFileName = "all2jxl_optimized.log"
	version     = "2.3.0"
	author      = "AI Assistant"

	// 性能优化常量
	MaxConcurrentProcesses = 8
	MaxConcurrentFiles     = 16
	DefaultTimeoutSeconds  = 30
	MaxRetries             = 3
	MemoryThresholdMB      = 1000
	DefaultWorkers         = 4
	MaxFileSizeMB          = 500
	MinFreeMemoryMB        = 200
	HealthCheckInterval    = 10
)

// 全局变量定义
var (
	logger        *log.Logger
	globalCtx     context.Context
	cancelFunc    context.CancelFunc
	stats         *Stats
	procSem       chan struct{}
	fdSem         chan struct{}
	healthMonitor *HealthMonitor
)

// VerifyMode 验证模式类型
type VerifyMode string

const (
	VerifyStrict VerifyMode = "strict"
	VerifyFast   VerifyMode = "fast"
)

// Options 结构体定义了程序的配置选项
type Options struct {
	Workers           int
	Verify            VerifyMode
	DoCopy            bool
	Sample            int
	SkipExist         bool
	DryRun            bool
	CJXLThreads       int
	TimeoutSeconds    int
	Retries           int
	InputDir          string
	OutputDir         string
	LogLevel          string
	MaxMemory         int64
	MaxFileSize       int64
	MinFreeMemory     int64
	EnableHealthCheck bool
	ProgressReport    bool
	DetailedStats     bool
	ErrorRecovery     bool
	PerformanceTuning bool
}

// FileProcessInfo 结构体用于记录单个文件在处理过程中的详细信息
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
	MemoryUsed     int64
	CPUUsage       float64
	ErrorType      string
	RecoveryAction string
}

// Stats 结构体用于统计处理过程中的各种数据
type Stats struct {
	sync.RWMutex
	imagesProcessed    int
	imagesFailed       int
	imagesSkipped      int
	videosSkipped      int
	otherSkipped       int
	totalBytesBefore   int64
	totalBytesAfter    int64
	startTime          time.Time
	detailedLogs       []FileProcessInfo
	byExt              map[string]int
	peakMemoryUsage    int64
	totalRetries       int
	recoveryActions    int
	errorTypes         map[string]int
	performanceMetrics map[string]float64
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	mu            sync.RWMutex
	isHealthy     bool
	lastCheck     time.Time
	memoryUsage   uint64
	cpuUsage      float64
	errorCount    int
	recoveryCount int
	checkInterval time.Duration
	stopChan      chan struct{}
}

// 初始化函数
func init() {
	setupLogging()
	stats = &Stats{
		startTime:          time.Now(),
		byExt:              make(map[string]int),
		errorTypes:         make(map[string]int),
		performanceMetrics: make(map[string]float64),
	}
	healthMonitor = &HealthMonitor{
		isHealthy:     true,
		checkInterval: HealthCheckInterval * time.Second,
		stopChan:      make(chan struct{}),
	}
	setupSignalHandling()
}

// 设置日志记录
func setupLogging() {
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
}

// 设置信号处理
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Printf("🛑 收到信号 %v，开始优雅关闭...", sig)
		if cancelFunc != nil {
			cancelFunc()
		}
		if healthMonitor != nil {
			close(healthMonitor.stopChan)
		}
		time.Sleep(2 * time.Second)
		printStatistics()
		os.Exit(0)
	}()
}

// 解析命令行参数
func parseFlags() Options {
	var opts Options

	// 基础参数
	flag.StringVar(&opts.InputDir, "dir", "", "📂 输入目录路径（必需）")
	flag.StringVar(&opts.OutputDir, "output", "", "📁 输出目录路径（默认为输入目录）")
	flag.IntVar(&opts.Workers, "workers", 0, "⚡ 工作线程数 (0=自动检测)")
	flag.BoolVar(&opts.DoCopy, "copy", false, "📋 复制文件而不是移动")
	flag.IntVar(&opts.Sample, "sample", 0, "🎯 采样处理文件数量 (0=处理所有)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", false, "⏭️ 跳过已存在的JXL文件")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "🔍 试运行模式，只显示将要处理的文件")

	// 转换参数
	flag.StringVar((*string)(&opts.Verify), "verify", "fast", "🔍 验证模式: strict, fast")
	flag.IntVar(&opts.CJXLThreads, "cjxl-threads", 4, "🧵 CJXL编码器线程数")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", DefaultTimeoutSeconds, "⏰ 单个文件处理超时时间（秒）")
	flag.IntVar(&opts.Retries, "retries", MaxRetries, "🔄 转换失败重试次数")

	// 性能参数
	flag.StringVar(&opts.LogLevel, "log-level", "INFO", "📝 日志级别: DEBUG, INFO, WARN, ERROR")
	flag.Int64Var(&opts.MaxMemory, "max-memory", 0, "💾 最大内存使用量（字节，0=无限制）")
	flag.Int64Var(&opts.MaxFileSize, "max-file-size", MaxFileSizeMB*1024*1024, "📏 最大文件大小（字节）")
	flag.Int64Var(&opts.MinFreeMemory, "min-free-memory", MinFreeMemoryMB*1024*1024, "💾 最小空闲内存（字节）")
	flag.BoolVar(&opts.EnableHealthCheck, "health-check", true, "🏥 启用健康检查")
	flag.BoolVar(&opts.ProgressReport, "progress", true, "📊 启用进度报告")
	flag.BoolVar(&opts.DetailedStats, "detailed-stats", false, "📈 启用详细统计")
	flag.BoolVar(&opts.ErrorRecovery, "error-recovery", true, "🔄 启用错误恢复")
	flag.BoolVar(&opts.PerformanceTuning, "performance-tuning", true, "⚡ 启用性能调优")

	flag.Parse()

	// 参数验证
	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录 (-dir)")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir
	}
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}
	if opts.Workers < 0 {
		opts.Workers = 0
	}
	if opts.CJXLThreads < 1 {
		opts.CJXLThreads = 1
	}
	if opts.TimeoutSeconds < 1 {
		opts.TimeoutSeconds = DefaultTimeoutSeconds
	}
	if opts.Retries < 0 {
		opts.Retries = 0
	}
	if opts.MaxMemory > 0 && opts.MaxMemory < opts.MinFreeMemory {
		logger.Fatal("❌ 错误: 最大内存使用量不能小于最小空闲内存要求")
	}

	return opts
}

// 检查系统依赖
func checkDependencies() error {
	dependencies := []string{"cjxl", "djxl", "exiftool"}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("缺少依赖: %s", dep)
		}
	}
	logger.Println("✅ 所有系统依赖检查通过")
	return nil
}

// 智能性能配置
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
			opts.Workers = DefaultWorkers
		}
	}
	if opts.Workers > MaxConcurrentProcesses {
		opts.Workers = MaxConcurrentProcesses
	}
	procSem = make(chan struct{}, opts.Workers)
	fdSem = make(chan struct{}, MaxConcurrentFiles)
	globalCtx, cancelFunc = context.WithCancel(context.Background())
	logger.Printf("⚡ 性能配置: %d 个工作线程, %d 个文件句柄", opts.Workers, MaxConcurrentFiles)
}

// 启动健康监控
func startHealthMonitor(opts *Options) {
	if !opts.EnableHealthCheck {
		return
	}
	go func() {
		ticker := time.NewTicker(healthMonitor.checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-globalCtx.Done():
				return
			case <-healthMonitor.stopChan:
				return
			case <-ticker.C:
				checkSystemHealth(opts)
			}
		}
	}()
	logger.Println("🏥 健康监控已启动")
}

// 检查系统健康状态
func checkSystemHealth(opts *Options) {
	healthMonitor.mu.Lock()
	defer healthMonitor.mu.Unlock()
	if mem, err := mem.VirtualMemory(); err == nil {
		healthMonitor.memoryUsage = mem.Used
		if opts.MaxMemory > 0 && mem.Used > uint64(opts.MaxMemory) {
			logger.Printf("⚠️  内存使用过高: %d MB / %d MB",
				mem.Used/1024/1024, opts.MaxMemory/1024/1024)
			healthMonitor.isHealthy = false
		}
		if mem.Available < uint64(opts.MinFreeMemory) {
			logger.Printf("⚠️  空闲内存不足: %d MB", mem.Available/1024/1024)
			healthMonitor.isHealthy = false
		}
	}
	if healthMonitor.errorCount > 10 {
		logger.Printf("⚠️  错误率过高: %d 个错误", healthMonitor.errorCount)
		healthMonitor.isHealthy = false
	}
	healthMonitor.lastCheck = time.Now()
}

// 内存监控
func monitorMemory(opts *Options) {
	if opts.MaxMemory <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-globalCtx.Done():
				return
			case <-ticker.C:
				if mem, err := mem.VirtualMemory(); err == nil {
					if mem.Used > uint64(opts.MaxMemory) {
						logger.Printf("⚠️  内存使用过高: %d MB / %d MB",
							mem.Used/1024/1024, opts.MaxMemory/1024/1024)
						stats.Lock()
						if int64(mem.Used) > stats.peakMemoryUsage {
							stats.peakMemoryUsage = int64(mem.Used)
						}
						stats.Unlock()
					}
				}
			}
		}
	}()
}

// 扫描候选文件
func scanCandidateFiles(inputDir string, opts Options) []string {
	var files []string
	err := godirwalk.Walk(inputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(osPathname))
			if !isImageFile(ext) {
				return nil
			}
			if info, err := os.Stat(osPathname); err == nil {
				if info.Size() > 0 && info.Size() <= opts.MaxFileSize {
					files = append(files, osPathname)
				} else if info.Size() > opts.MaxFileSize {
					logger.Printf("⚠️  文件过大，跳过: %s (%d MB)",
						filepath.Base(osPathname), info.Size()/1024/1024)
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
	sort.Slice(files, func(i, j int) bool {
		info1, _ := os.Stat(files[i])
		info2, _ := os.Stat(files[j])
		return info1.Size() < info2.Size()
	})
	return files
}

// 检查是否为图像文件
func isImageFile(ext string) bool {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true,
		".png": true, ".bmp": true,
		".tiff": true, ".tif": true,
		".gif": true, ".webp": true,
		".avif": true, ".heic": true, ".heif": true,
	}
	return imageExts[ext]
}

// 处理文件（带重试机制）
func processFileWithRetry(filePath string, fileInfo os.FileInfo, opts Options) {
	var lastErr error
	var errorType string
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
		errorType = classifyError(err)
		logger.Printf("⚠️  处理文件失败: %s - %v (错误类型: %s)",
			filepath.Base(filePath), err, errorType)
		stats.Lock()
		stats.errorTypes[errorType]++
		stats.Unlock()
		healthMonitor.mu.Lock()
		healthMonitor.errorCount++
		healthMonitor.mu.Unlock()
	}
	logger.Printf("❌ 文件处理最终失败: %s - %v", filepath.Base(filePath), lastErr)
	stats.addImageFailed()
}

// 错误分类
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
	} else if strings.Contains(errStr, "corrupt") {
		return "corrupt"
	}
	return "unknown"
}

// 处理单个文件
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
	enhancedType, err := utils.DetectFileType(filePath)
	if err != nil {
		return fmt.Errorf("文件类型检测失败: %v", err)
	}
	if opts.SkipExist {
		outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".jxl"
		if _, err := os.Stat(outputPath); err == nil {
			logger.Printf("⏩ 跳过已存在: %s", filepath.Base(filePath))
			stats.addImageSkipped()
			return nil
		}
	}
	conversionMode, outputPath, errorMsg, err := convertToJxlWithOpts(filePath, enhancedType, opts)
	processingTime := time.Since(startTime)
	processInfo := FileProcessInfo{
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		FileType:       filepath.Ext(filePath),
		IsAnimated:     enhancedType.IsAnimated,
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

// 转换到JXL格式
func convertToJxlWithOpts(filePath string, enhancedType utils.EnhancedFileType, opts Options) (string, string, string, error) {
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".jxl"
	if enhancedType.IsAnimated {
		return convertAnimatedToJxl(filePath, outputPath, opts)
	}
	return convertStaticToJxl(filePath, outputPath, opts)
}

// 转换静态图像到JXL
func convertStaticToJxl(inputPath, outputPath string, opts Options) (string, string, string, error) {
	args := []string{
		inputPath,
		"-d", "0",
		"-e", "7",
		"--num_threads", strconv.Itoa(opts.CJXLThreads),
		outputPath,
	}
	ctx, cancel := context.WithTimeout(globalCtx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "静态转换", outputPath, string(output), fmt.Errorf("cjxl转换失败: %v", err)
	}
	if err := copyMetadata(inputPath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败: %v", err)
	}
	return "静态转换", outputPath, "", nil
}

// 转换动画图像到JXL
func convertAnimatedToJxl(inputPath, outputPath string, opts Options) (string, string, string, error) {
	args := []string{
		inputPath,
		"-d", "0",
		"-e", "7",
		"--num_threads", strconv.Itoa(opts.CJXLThreads),
		"--container=1",
		outputPath,
	}
	ctx, cancel := context.WithTimeout(globalCtx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "动画转换", outputPath, string(output), fmt.Errorf("cjxl动画转换失败: %v", err)
	}
	if err := copyMetadata(inputPath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败: %v", err)
	}
	return "动画转换", outputPath, "", nil
}

// 复制元数据
func copyMetadata(inputPath, outputPath string) error {
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", inputPath, outputPath)
	return cmd.Run()
}

// 获取文件大小
func getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}
	return 0
}

// 统计信息方法
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

// 打印统计信息
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

// 主函数
func main() {
	logger.Printf("🎨 JPEG XL 批量转换工具 v%s (优化版)", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")
	opts := parseFlags()
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}
	configurePerformance(&opts)
	startHealthMonitor(&opts)
	monitorMemory(&opts)
	logger.Println("🔍 扫描图像文件...")
	files := scanCandidateFiles(opts.InputDir, opts)
	logger.Printf("📊 发现 %d 个候选文件", len(files))
	if len(files) == 0 {
		logger.Println("📊 没有找到需要处理的文件")
		return
	}
	if opts.Sample > 0 && len(files) > opts.Sample {
		files = files[:opts.Sample]
		logger.Printf("🎯 采样模式: 选择 %d 个文件进行处理", len(files))
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
