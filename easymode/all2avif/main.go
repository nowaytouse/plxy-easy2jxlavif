// all2avif - 批量图像转AVIF格式工具
//
// 功能说明：
// - 支持多种图像格式批量转换为AVIF格式
// - 保留原始文件的元数据和系统时间戳
// - 支持动画图像和静态图像的无损转换
// - 提供详细的处理统计和进度报告
// - 支持并发处理以提高转换效率
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
	"sync/atomic"
	"syscall"
	"time"

	"pixly/utils"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"github.com/karrick/godirwalk"
	"github.com/panjf2000/ants/v2"
)

// 程序常量定义
const (
	logFileName = "all2avif.log" // 日志文件名
	version     = "2.1.0"        // 程序版本号
	author      = "AI Assistant" // 作者信息
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
	Workers          int    // 并发工作线程数，控制同时处理的文件数量
	Quality          int    // 图像质量参数 (1-100)，数值越高质量越好但文件越大
	Speed            int    // 编码速度参数 (0-6)，数值越高编码越快但压缩率可能降低
	SkipExist        bool   // 是否跳过已存在的AVIF文件
	DryRun           bool   // 试运行模式，只显示将要处理的文件而不实际转换
	TimeoutSeconds   int    // 单个文件处理的超时时间（秒）
	Retries          int    // 转换失败时的重试次数
	InputDir         string // 输入目录路径
	OutputDir        string // 输出目录路径，默认为输入目录
	ReplaceOriginals bool   // 是否在转换成功后删除原始文件
}

// FileProcessInfo 结构体用于记录单个文件在处理过程中的详细信息
// 这对于生成详细的处理报告和调试非常有用
type FileProcessInfo struct {
	FilePath       string        // 文件完整路径
	FileType       string        // 文件类型（扩展名）
	OriginalSize   int64         // 原始文件大小（字节）
	ConvertedSize  int64         // 转换后文件大小（字节）
	ProcessingTime time.Duration // 处理耗时
	Success        bool          // 是否处理成功
	Error          string        // 错误信息（如果处理失败）
}

// Stats 结构体用于在整个批处理过程中收集和管理统计数据
// 它使用互斥锁（sync.Mutex）来确保并发访问时的线程安全
type Stats struct {
	sync.Mutex                            // 互斥锁，确保并发安全
	successCount        int64             // 成功处理的文件数量
	failureCount        int64             // 处理失败的文件数量
	skippedCount        int64             // 跳过的文件数量
	videoSkippedCount   int64             // 跳过的视频文件数量
	linkSkippedCount    int64             // 跳过的符号链接数量
	otherSkippedCount   int64             // 跳过的其他文件数量
	totalOriginalSize   int64             // 原始文件总大小
	totalConvertedSize  int64             // 转换后文件总大小
	totalProcessingTime time.Duration     // 总处理时间
	detailedLogs        []FileProcessInfo // 详细的处理日志记录
}

// addSuccess 原子性地增加成功处理文件的计数
func (s *Stats) addSuccess() {
	atomic.AddInt64(&s.successCount, 1)
}

// addFailure 原子性地增加处理失败文件的计数
func (s *Stats) addFailure() {
	atomic.AddInt64(&s.failureCount, 1)
}

// addSkipped 原子性地增加跳过文件的计数
func (s *Stats) addSkipped() {
	atomic.AddInt64(&s.skippedCount, 1)
}

// addVideoSkipped 原子性地增加跳过视频文件的计数
func (s *Stats) addVideoSkipped() {
	atomic.AddInt64(&s.videoSkippedCount, 1)
}

// addLinkSkipped 原子性地增加跳过符号链接的计数
func (s *Stats) addLinkSkipped() {
	atomic.AddInt64(&s.linkSkippedCount, 1)
}

// addOtherSkipped 原子性地增加跳过其他文件的计数
func (s *Stats) addOtherSkipped() {
	atomic.AddInt64(&s.otherSkippedCount, 1)
}

// addSize 原子性地增加文件大小统计
// original: 原始文件大小
// converted: 转换后文件大小
func (s *Stats) addSize(original, converted int64) {
	atomic.AddInt64(&s.totalOriginalSize, original)
	atomic.AddInt64(&s.totalConvertedSize, converted)
}

// addProcessingTime 原子性地增加处理时间统计
func (s *Stats) addProcessingTime(duration time.Duration) {
	atomic.AddInt64((*int64)(&s.totalProcessingTime), int64(duration))
}

// addDetailedLog 线程安全地向详细日志中添加一条处理记录
func (s *Stats) addDetailedLog(info FileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

// logDetailedSummary 输出详细的处理摘要信息
// 包括按格式统计的处理结果、处理时间最长的文件等信息
func (s *Stats) logDetailedSummary() {
	s.Lock()
	defer s.Unlock()

	logger.Println("🎯 ===== 详细处理摘要 =====")
	logger.Printf("📊 总处理时间: %v", s.totalProcessingTime)
	if len(s.detailedLogs) > 0 {
		logger.Printf("📈 平均处理时间: %v", s.totalProcessingTime/time.Duration(len(s.detailedLogs)))
	} else {
		logger.Printf("📈 平均处理时间: 无处理文件")
	}

	// 按格式统计处理结果
	formatStats := make(map[string][]FileProcessInfo)
	for _, log := range s.detailedLogs {
		formatStats[log.FileType] = append(formatStats[log.FileType], log)
	}

	for format, logs := range formatStats {
		successCount := 0
		totalOriginalSize := int64(0)
		totalConvertedSize := int64(0)
		for _, log := range logs {
			if log.Success {
				successCount++
				totalOriginalSize += log.OriginalSize
				totalConvertedSize += log.ConvertedSize
			}
		}
		successRate := float64(successCount) / float64(len(logs)) * 100
		compressionRate := float64(totalConvertedSize) / float64(totalOriginalSize) * 100
		logger.Printf("🖼️  %s格式: %d个文件, 成功率%.1f%%, 平均压缩率%.1f%%", format, len(logs), successRate, compressionRate)
	}

	// 找出处理时间最长的文件
	if len(s.detailedLogs) > 0 {
		logger.Println("⏱️  处理时间最长的文件:")
		sortedLogs := make([]FileProcessInfo, len(s.detailedLogs))
		copy(sortedLogs, s.detailedLogs)
		// 简单排序（按处理时间降序）
		for i := 0; i < len(sortedLogs)-1; i++ {
			for j := i + 1; j < len(sortedLogs); j++ {
				if sortedLogs[i].ProcessingTime < sortedLogs[j].ProcessingTime {
					sortedLogs[i], sortedLogs[j] = sortedLogs[j], sortedLogs[i]
				}
			}
		}
		// 显示前3个
		for i := 0; i < 3 && i < len(sortedLogs); i++ {
			log := sortedLogs[i]
			logger.Printf("   🐌 %s: %v", filepath.Base(log.FilePath), log.ProcessingTime)
		}
	}
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

// main 函数是程序的入口点
func main() {
	logger.Printf("🎨 AVIF 批量转换工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 检查系统依赖工具是否可用
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	// 解析命令行参数
	opts := parseFlags()
	logger.Printf("📁 准备处理目录...")

	// 验证输入目录
	if opts.InputDir == "" {
		logger.Fatalf("❌ 必须指定输入目录")
	}

	// 检查输入目录是否存在
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 输入目录不存在: %s", opts.InputDir)
	}

	// 设置输出目录，默认为输入目录
	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir
	}

	logger.Printf("📂 直接处理目录: %s", opts.InputDir)

	// 扫描目录中的候选文件
	candidateFiles, err := scanCandidateFiles(opts.InputDir)
	if err != nil {
		logger.Fatalf("❌ 扫描文件失败: %v", err)
	}

	if len(candidateFiles) == 0 {
		logger.Println("ℹ️  未找到可处理的文件")
		return
	}

	logger.Printf("📊 发现 %d 个候选文件", len(candidateFiles))

	// 配置处理性能参数
	logger.Printf("⚡ 配置处理性能...")
	logger.Printf("🚀 性能配置: %d个工作线程, %d个进程限制, %d个文件句柄限制", opts.Workers, cap(procSem), cap(fdSem))
	logger.Printf("💻 系统信息: %d个CPU核心", runtime.NumCPU())

	// 开始并行处理文件
	logger.Printf("🚀 开始并行处理 - 目录: %s, 工作线程: %d, 文件数: %d", opts.InputDir, opts.Workers, len(candidateFiles))

	// 设置信号处理，支持优雅中断
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	logger.Printf("🛑 设置信号处理...")

	// 添加全局超时保护，防止系统卡死
	globalTimeout := time.Duration(len(candidateFiles)) * 30 * time.Second // 每个文件最多30秒
	if globalTimeout > 2*time.Hour {
		globalTimeout = 2 * time.Hour // 最大2小时
	}
	logger.Printf("⏰ 设置全局超时保护: %v", globalTimeout)

	// 创建超时上下文，用于全局超时控制
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), globalTimeout)
	defer timeoutCancel()

	// 创建统计对象用于收集处理结果
	stats := &Stats{}

	// 使用ants库创建goroutine池，提高并发处理效率
	pool, err := ants.NewPool(opts.Workers)
	if err != nil {
		logger.Fatalf("❌ 创建goroutine池失败: %v", err)
	}
	defer pool.Release()

	// 创建WaitGroup等待所有任务完成
	var wg sync.WaitGroup

	// 处理文件
	startTime := time.Now()
	for _, filePath := range candidateFiles {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			select {
			case <-timeoutCtx.Done():
				// ⏰ 超时保护
				logger.Printf("⚠️  全局超时，跳过文件: %s", filepath.Base(filePath))
				return
			default:
				processFileWithOpts(filePath, opts, stats)
			}
		})
	}

	// 等待所有任务完成
	wg.Wait()
	totalTime := time.Since(startTime)

	logger.Printf("⏱️  总处理时间: %v", totalTime)

	// 输出详细统计信息
	stats.logDetailedSummary()

	// 输出简单统计摘要
	logger.Println("🎯 ===== 处理摘要 =====")
	logger.Printf("✅ 成功处理图像: %d", atomic.LoadInt64(&stats.successCount))
	logger.Printf("❌ 转换失败图像: %d", atomic.LoadInt64(&stats.failureCount))
	logger.Printf("🎬 跳过视频文件: %d", atomic.LoadInt64(&stats.videoSkippedCount))
	logger.Printf("🔗 跳过符号链接: %d", atomic.LoadInt64(&stats.linkSkippedCount))
	logger.Printf("📄 跳过其他文件: %d", atomic.LoadInt64(&stats.otherSkippedCount))

	// 计算文件大小统计
	originalSize := atomic.LoadInt64(&stats.totalOriginalSize)
	convertedSize := atomic.LoadInt64(&stats.totalConvertedSize)

	// 计算节省的空间，如果转换后文件更大则显示为0
	savedSize := originalSize - convertedSize
	if savedSize < 0 {
		savedSize = 0
	}

	// 计算压缩率（如果转换后文件更大则显示大于100%）
	compressionRate := float64(convertedSize) / float64(originalSize) * 100

	logger.Println("📊 ===== 大小统计 =====")
	logger.Printf("📥 原始总大小: %.2f MB", float64(originalSize)/(1024*1024))
	logger.Printf("📤 转换后大小: %.2f MB", float64(convertedSize)/(1024*1024))
	logger.Printf("💾 节省空间: %.2f MB (压缩率: %.1f%%)", float64(savedSize)/(1024*1024), compressionRate)

	// 按格式统计处理结果
	formatCounts := make(map[string]int)
	for _, log := range stats.detailedLogs {
		formatCounts[log.FileType]++
	}

	logger.Println("📋 ===== 格式统计 =====")
	for format, count := range formatCounts {
		logger.Printf("  🖼️  %s: %d个文件", format, count)
	}

	// 文件数量验证，确保处理结果正确
	logger.Println("🔍 验证处理结果...")
	validateFileCount(opts.InputDir, len(candidateFiles), stats)

	logger.Println("🎉 ===== 处理完成 =====")
}

// checkDependencies 检查系统依赖工具是否可用
// 返回错误如果任何必需的依赖工具不可用
func checkDependencies() error {
	dependencies := []string{"ffmpeg", "exiftool"}
	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("缺少依赖: %s", dep)
		}
	}
	logger.Printf("✅ ffmpeg 已就绪")
	logger.Printf("✅ exiftool 已就绪")
	return nil
}

// parseFlags 解析命令行参数并返回配置选项
func parseFlags() Options {
	var (
		workers          = flag.Int("workers", 10, "🚀 工作线程数")
		quality          = flag.Int("quality", 80, "🎨 图像质量 (1-100)")
		speed            = flag.Int("speed", 4, "⚡ 编码速度 (0-6)")
		skipExist        = flag.Bool("skip-exist", true, "⏭️  跳过已存在的 .avif 文件")
		dryRun           = flag.Bool("dry-run", false, "🔍 试运行模式（不实际转换）")
		timeoutSec       = flag.Int("timeout", 300, "⏰ 单个文件超时时间（秒）")
		retries          = flag.Int("retries", 1, "🔄 重试次数")
		dir              = flag.String("dir", "", "📁 输入目录")
		outputDir        = flag.String("output", "", "📁 输出目录（默认为输入目录）")
		replaceOriginals = flag.Bool("replace", true, "🗑️  转换后删除原始文件")
	)

	flag.Parse()

	return Options{
		Workers:          *workers,
		Quality:          *quality,
		Speed:            *speed,
		SkipExist:        *skipExist,
		DryRun:           *dryRun,
		TimeoutSeconds:   *timeoutSec,
		Retries:          *retries,
		InputDir:         *dir,
		OutputDir:        *outputDir,
		ReplaceOriginals: *replaceOriginals,
	}
}

var supportedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".apng": true, ".webp": true,
	".avif": true, ".heic": true, ".heif": true, ".jfif": true, ".jpe": true, ".bmp": true,
	".tiff": true, ".tif": true, ".ico": true, ".cur": true,
}

// scanCandidateFiles 扫描目录中的候选文件
// 返回所有支持格式的文件路径列表
func scanCandidateFiles(inputDir string) ([]string, error) {
	var files []string
	logger.Printf("🔍 扫描媒体文件...")
	err := godirwalk.Walk(inputDir, &godirwalk.Options{
		Unsorted: true,
		Callback: func(p string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}
			info, err := os.Lstat(p)
			if err != nil {
				return nil
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if supportedExtensions[ext] {
				files = append(files, p)
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			logger.Printf("⚠️  扫描文件时出错 %s: %v", osPathname, err)
			return godirwalk.SkipNode
		},
	})
	return files, err
}

// isSupportedImageType 检查文件扩展名是否为支持的图像格式
func isSupportedImageType(ext string) bool {
	return supportedExtensions[ext]
}

// isVideoType 检查文件扩展名是否为视频格式
func isVideoType(ext string) bool {
	videoTypes := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mov":  true,
		".mkv":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
		".3gp":  true,
	}
	return videoTypes[ext]
}

// processFileWithOpts 处理单个文件，根据选项进行转换
// 这是文件处理的核心函数，负责协调整个转换流程
func processFileWithOpts(filePath string, opts Options, stats *Stats) {
	startTime := time.Now()
	fileName := filepath.Base(filePath)

	processInfo := FileProcessInfo{
		FilePath: filePath,
		FileType: filepath.Ext(filePath),
	}

	// Get original file info for modification time and creation time
	var originalModTime time.Time
	var originalCreateTime time.Time
	if stat, err := os.Stat(filePath); err == nil {
		processInfo.OriginalSize = stat.Size()
		originalModTime = stat.ModTime()
		if ctime, _, ok := getFileTimesDarwin(filePath); ok {
			originalCreateTime = ctime
		}
	}

	logger.Printf("🔄 开始处理: %s", fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Printf("⚠️  文件不存在: %s", filepath.Base(filePath))
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 检查是否为符号链接
	if info, err := os.Lstat(filePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		logger.Printf("🔗 跳过符号链接: %s", filepath.Base(filePath))
		stats.addLinkSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 检查文件类型
	file, err := os.Open(filePath)
	if err != nil {
		logger.Printf("⚠️  无法打开文件 %s: %v", filepath.Base(filePath), err)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}
	defer file.Close()

	// 读取文件头
	head := make([]byte, 261)
	_, err = file.Read(head)
	if err != nil {
		logger.Printf("⚠️  无法读取文件头 %s: %v", filepath.Base(filePath), err)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 检测文件类型
	kind, err := filetype.Match(head)
	if err != nil {
		logger.Printf("⚠️  无法检测文件类型 %s: %v", filepath.Base(filePath), err)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 检查是否为视频文件
	if isVideoType(kind.Extension) {
		logger.Printf("🎬 跳过视频文件: %s", filepath.Base(filePath))
		stats.addVideoSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 检查是否为支持的图像类型
	ext := strings.ToLower(filepath.Ext(filePath))
	if !isSupportedImageType(ext) {
		logger.Printf("📄 跳过不支持的文件类型: %s (%s)", filepath.Base(filePath), ext)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	logger.Printf("🔄 开始处理: %s", filepath.Base(filePath))

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logger.Printf("⚠️  无法获取文件信息 %s: %v", filepath.Base(filePath), err)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	// 设置原始文件大小
	processInfo.OriginalSize = fileInfo.Size()

	// 检查是否已存在AVIF文件
	avifPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".avif"
	if opts.SkipExist {
		if _, err := os.Stat(avifPath); err == nil {
			logger.Printf("⏭️  跳过已存在: %s", filepath.Base(avifPath))
			stats.addSkipped()
			processInfo.Success = true
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			return
		}
	}

	// 苹果Live Photo检测
	if kind.Extension == "heic" || kind.Extension == "heif" {
		baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
		movPath := baseName + ".mov"
		if _, err := os.Stat(movPath); err == nil {
			logger.Printf("🏞️  检测到苹果Live Photo，跳过HEIC转换: %s", filepath.Base(filePath))
			stats.addOtherSkipped()
			processInfo.Error = "跳过Live Photo"
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			return
		}
	}

	// 检测是否为动画图像
	isAnimated := false
	if kind.Extension == "gif" {
		if gifFile, err := os.Open(filePath); err == nil {
			if gifImage, err := gif.DecodeConfig(gifFile); err == nil {
				// 检查GIF是否有多个图像帧
				isAnimated = gifImage.Width > 0 && gifImage.Height > 0
				// 进一步检查是否真的是动画
				if isAnimated {
					// 尝试解码GIF来检查帧数
					if gifData, err := gif.DecodeAll(gifFile); err == nil {
						isAnimated = len(gifData.Image) > 1
					}
				}
			}
			gifFile.Close()
		}
	}

	if isAnimated {
		logger.Printf("🎬 检测到动画图像: %s", filepath.Base(filePath))
	} else {
		logger.Printf("🖼️  静态图像: %s", filepath.Base(filePath))
	}

	// 执行转换
	if opts.DryRun {
		logger.Printf("🔍 试运行模式: 跳过实际转换 %s", filepath.Base(filePath))
		stats.addSkipped()
		processInfo.Success = true
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addDetailedLog(processInfo)
		return
	}

	// 转换文件
	convertedSize, err := convertToAvif(filePath, kind, isAnimated, opts)
	if err != nil {
		logger.Printf("❌ 转换失败 %s: %v", filepath.Base(filePath), err)
		stats.addFailure()
		processInfo.ConvertedSize = 0
		processInfo.ProcessingTime = time.Since(startTime)
		processInfo.Error = err.Error()
		stats.addDetailedLog(processInfo)
		return
	}

	// 更新统计
	stats.addSuccess()
	stats.addSize(processInfo.OriginalSize, convertedSize)
	processInfo.ConvertedSize = convertedSize
	processInfo.ProcessingTime = time.Since(startTime)
	processInfo.Success = true
	stats.addDetailedLog(processInfo)

	// 计算压缩率
	compressionRate := float64(convertedSize) / float64(processInfo.OriginalSize) * 100
	savedSize := processInfo.OriginalSize - convertedSize

	logger.Printf("🎉 处理成功: %s", filepath.Base(filePath))
	logger.Printf("📊 大小变化: %.2f KB -> %.2f KB (节省: %.2f KB, 压缩率: %.1f%%)",
		float64(processInfo.OriginalSize)/1024, float64(convertedSize)/1024, float64(savedSize)/1024, compressionRate)

	// 设置修改时间
	err = os.Chtimes(avifPath, originalModTime, originalModTime)
	if err != nil {
		logger.Printf("WARN: Failed to set modification time for %s: %v", avifPath, err)
	}
	// 在 macOS 上尽量同步 Finder 可见的创建/修改日期
	if runtime.GOOS == "darwin" && !originalCreateTime.IsZero() {
		if e := setFinderDates(avifPath, originalCreateTime, originalModTime); e != nil {
			logger.Printf("WARN: Failed to set Finder dates for %s: %v", avifPath, e)
		}
	}

	// 安全删除原始文件
	if opts.ReplaceOriginals {
		avifPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".avif"
		if err := utils.SafeDelete(filePath, avifPath, func(format string, v ...interface{}) {
			logger.Printf(format, v...)
		}); err != nil {
			logger.Printf("⚠️  安全删除失败 %s: %v", filepath.Base(filePath), err)
		}
	}
}

// convertToAvif 将图像文件转换为AVIF格式
// 这是转换的核心函数，处理不同格式的图像转换
func convertToAvif(filePath string, kind types.Type, isAnimated bool, opts Options) (int64, error) {
	avifPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".avif"
	originalFilePath := filePath // 保留原始文件路径用于元数据复制
	var tempPngPath string
	var tempRelaxedPngPath string

	// HEIC/HEIF转换使用增强的magick转换为更稳定的PNG中间格式
	if kind.Extension == "heic" || kind.Extension == "heif" {
		tempPngPath = avifPath + ".png"
		logger.Printf("INFO: [HEIC] Converting to PNG intermediate: %s", filepath.Base(tempPngPath))

		// 方法1：使用ImageMagick增加限制转换为PNG
		cmd := exec.Command("magick", "-define", "heic:limit-num-tiles=0", "-define", "heic:max-image-size=0", "-define", "heic:use-embedded-profile=false", filePath, tempPngPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Printf("WARN: ImageMagick failed for %s: %v. Output: %s. Trying alternative method.", filepath.Base(filePath), err, string(output))

			// 方法2：使用ffmpeg作为后备方案转换HEIC到PNG
			// 首先获取HEIC文件的实际尺寸以确保提取完整分辨率
			var ffmpegOutput []byte
			var ffmpegErr error
			dimCmd := exec.Command("exiftool", "-s", "-S", "-ImageWidth", "-ImageHeight", filePath)
			dimOutput, dimErr := dimCmd.CombinedOutput()

			if dimErr != nil {
				// 如果exiftool失败，回退到默认方法
				logger.Printf("WARN: Exiftool dimension detection failed for %s: %v. Falling back to default method.", filepath.Base(filePath), dimErr)
				cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
				ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
				if ffmpegErr != nil {
					// 如果失败，尝试不同参数
					logger.Printf("WARN: Default ffmpeg method failed for %s: %v. Output: %s. Trying enhanced approach.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-vcodec", "png", "-frames:v", "1", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
					if ffmpegErr != nil {
						logger.Printf("WARN: Second ffmpeg attempt failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
					}
				}
			} else {
				// 从exiftool输出解析尺寸
				lines := strings.Split(strings.TrimSpace(string(dimOutput)), "\n")
				var width, height int

				// 处理exiftool的数字格式
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}

					// 尝试简单数字格式（只是数字）
					if intValue, err := strconv.Atoi(line); err == nil {
						// 假设第一个数字是宽度，第二个是高度
						if width == 0 {
							width = intValue
						} else if height == 0 {
							height = intValue
						}
					}
				}

				// 如果有有效尺寸，使用它们进行适当缩放
				if width > 0 && height > 0 {
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-vf", fmt.Sprintf("scale=%d:%d", width, height), "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
					if ffmpegErr != nil {
						logger.Printf("WARN: Scaled ffmpeg method failed for %s: %v. Output: %s. Trying unscaled approach.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
						// 如果失败，尝试不缩放
						cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
						ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
						if ffmpegErr != nil {
							logger.Printf("WARN: Unscaled ffmpeg method also failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
						}
					}
				} else {
					// 如果尺寸无效，回退到默认方法
					logger.Printf("WARN: Invalid dimensions detected for %s (width: %d, height: %d). Falling back to default method.", filepath.Base(filePath), width, height)
					cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", filePath, "-pix_fmt", "rgb24", "-frames:v", "1", "-c:v", "png", tempPngPath)
					ffmpegOutput, ffmpegErr = cmd.CombinedOutput()
				}
			}

			// 只有当ffmpeg和ImageMagick方法都失败时，才尝试更宽松限制的ImageMagick
			if ffmpegErr != nil {
				logger.Printf("WARN: Ffmpeg failed for %s: %v. Output: %s. Trying ImageMagick with more relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))

				// 方法3：尝试使用更宽松策略的ImageMagick
				tempRelaxedPngPath = avifPath + ".relaxed.png"
				cmd = exec.Command("magick", "-define", "heic:limit-num-tiles=0", "-define", "heic:max-image-size=0", "-define", "heic:use-embedded-profile=false", "-define", "heic:decode-effort=0", "-depth", "16", filePath, tempRelaxedPngPath)
				output, err = cmd.CombinedOutput()
				if err != nil {
					logger.Printf("WARN: All HEIC conversion methods failed for %s. ImageMagick, ffmpeg, and relaxed ImageMagick all failed. Output ImageMagick: %s, ffmpeg: %s, relaxed ImageMagick: %s",
						filepath.Base(filePath), string(output), string(ffmpegOutput), string(output))
					return 0, fmt.Errorf("所有HEIC转换方法都失败了: ImageMagick错误: %v, ffmpeg错误: %v", err, ffmpegErr)
				}
				// 使用宽松ImageMagick的输出
				filePath = tempRelaxedPngPath
			} else {
				// 使用ffmpeg成功转换，现在使用PNG作为输入
				filePath = tempPngPath
			}
		} else {
			// 使用原始ImageMagick方法成功转换
			filePath = tempPngPath
		}
	}

	// 构建ffmpeg命令
	var cmd *exec.Cmd

	// 计算CRF值，确保在有效范围内
	crf := 63 - opts.Quality
	if crf < 0 {
		crf = 0
	}
	if crf > 63 {
		crf = 63
	}

	if isAnimated {
		// 动画图像使用不同的参数
		cmd = exec.Command("ffmpeg",
			"-i", filePath,
			"-c:v", "libaom-av1",
			"-crf", strconv.Itoa(crf),
			"-cpu-used", strconv.Itoa(opts.Speed),
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-y", // 覆盖输出文件
			avifPath)
	} else {
		// 静态图像
		cmd = exec.Command("ffmpeg",
			"-i", filePath,
			"-c:v", "libaom-av1",
			"-crf", strconv.Itoa(crf),
			"-cpu-used", strconv.Itoa(opts.Speed),
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-y", // 覆盖输出文件
			avifPath)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if tempPngPath != "" {
		os.Remove(tempPngPath)
	}
	if tempRelaxedPngPath != "" {
		os.Remove(tempRelaxedPngPath)
	}
	if err != nil {
		return 0, fmt.Errorf("ffmpeg执行失败: %s\n输出: %s", err, string(output))
	}

	// 获取转换后文件大小
	info, err := os.Stat(avifPath)
	if err != nil {
		return 0, fmt.Errorf("无法获取转换后文件信息: %v", err)
	}

	// 复制元数据
	if err := copyMetadata(originalFilePath, avifPath); err != nil {
		logger.Printf("⚠️  元数据复制失败 %s: %v", filepath.Base(originalFilePath), err)
	}

	return info.Size(), nil
}

// copyMetadata 使用exiftool复制元数据从源文件到目标文件
func copyMetadata(sourcePath, targetPath string) error {
	// 使用exiftool复制元数据
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", sourcePath, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool失败: %s\n输出: %s", err, string(output))
	}
	logger.Printf("📋 元数据复制成功: %s", filepath.Base(sourcePath))
	return nil
}

// withTimeout 创建一个带超时的上下文
func withTimeout(ctx context.Context, opts Options) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(opts.TimeoutSeconds)*time.Second)
}

// validateFileCount 验证处理前后的文件数量
// 确保处理结果正确，统计各种文件类型的数量
func validateFileCount(workDir string, originalMediaCount int, stats *Stats) {
	currentMediaCount := 0
	currentAvifCount := 0
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				ext := strings.ToLower(filepath.Ext(osPathname))
				if supportedExtensions[ext] {
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

	successCount := int(atomic.LoadInt64(&stats.successCount))
	failureCount := int(atomic.LoadInt64(&stats.failureCount))
	videoSkippedCount := int(atomic.LoadInt64(&stats.videoSkippedCount))
	otherSkippedCount := int(atomic.LoadInt64(&stats.otherSkippedCount))

	expectedAvifCount := successCount
	expectedMediaCount := originalMediaCount - successCount

	logger.Printf("📊 文件数量验证:")
	logger.Printf("   原始媒体文件数: %d", originalMediaCount)
	logger.Printf("   成功处理图像: %d", successCount)
	logger.Printf("   转换失败/跳过: %d", failureCount+videoSkippedCount+otherSkippedCount)
	logger.Printf("   ---")
	logger.Printf("   期望AVIF文件数: %d", expectedAvifCount)
	logger.Printf("   实际AVIF文件数: %d", currentAvifCount)
	logger.Printf("   ---")
	logger.Printf("   期望剩余媒体文件数: %d", expectedMediaCount)
	logger.Printf("   实际剩余媒体文件数: %d", currentMediaCount)

	if currentAvifCount == expectedAvifCount && currentMediaCount == expectedMediaCount {
		logger.Printf("✅ 文件数量验证通过。")
	} else {
		logger.Printf("❌ 文件数量验证失败。")
		if currentAvifCount != expectedAvifCount {
			logger.Printf("   AVIF文件数不匹配 (实际: %d, 期望: %d)", currentAvifCount, expectedAvifCount)
		}
		if currentMediaCount != expectedMediaCount {
			logger.Printf("   剩余媒体文件数不匹配 (实际: %d, 期望: %d)", currentMediaCount, expectedMediaCount)
		}

		// 查找可能的临时文件
		tempFiles := findTempFiles(workDir)
		if len(tempFiles) > 0 {
			logger.Printf("🗑️  发现 %d 个临时文件，正在清理...", len(tempFiles))
			cleanupTempFiles(tempFiles)
			logger.Printf("✅ 临时文件清理完成")
		}
	}
}

// findTempFiles 查找临时文件
// 扫描工作目录中的临时文件，用于清理
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

// cleanupTempFiles 清理临时文件
// 删除指定的临时文件列表
func cleanupTempFiles(tempFiles []string) {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			logger.Printf("⚠️  删除临时文件失败 %s: %v", filepath.Base(file), err)
		} else {
			logger.Printf("🗑️  已删除临时文件: %s", filepath.Base(file))
		}
	}
}

// getFileTimesDarwin 尝试获取文件的创建/修改时间（macOS）
// 使用mdls命令获取文件的创建和修改时间
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
// 在macOS上设置文件的创建和修改时间，使其在Finder中正确显示
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
		return fmt.Errorf("exiftool设置Finder日期失败: %v, 输出=%s", err, string(out))
	}
	return nil
}
