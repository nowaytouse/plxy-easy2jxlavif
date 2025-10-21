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

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"github.com/karrick/godirwalk"
	"github.com/panjf2000/ants/v2"
)

const (
	logFileName = "all2avif.log"
	version     = "2.0.0"
	author      = "AI Assistant"
)

var (
	logger *log.Logger
	// 限制外部进程与文件句柄并发，避免过载
	procSem chan struct{}
	fdSem   chan struct{}
)

type Options struct {
	Workers          int
	Quality          int
	Speed            int
	SkipExist        bool
	DryRun           bool
	TimeoutSeconds   int
	Retries          int
	InputDir         string
	OutputDir        string
	ReplaceOriginals bool
}

// FileProcessInfo 记录单个文件的处理信息
type FileProcessInfo struct {
	FilePath       string
	FileType       string
	OriginalSize   int64
	ConvertedSize  int64
	ProcessingTime time.Duration
	Success        bool
	Error          string
}

// Stats 统计信息
type Stats struct {
	sync.Mutex
	successCount        int64
	failureCount        int64
	skippedCount        int64
	videoSkippedCount   int64
	linkSkippedCount    int64
	otherSkippedCount   int64
	totalOriginalSize   int64
	totalConvertedSize  int64
	totalProcessingTime time.Duration
	detailedLogs        []FileProcessInfo
}

func (s *Stats) addSuccess() {
	atomic.AddInt64(&s.successCount, 1)
}

func (s *Stats) addFailure() {
	atomic.AddInt64(&s.failureCount, 1)
}

func (s *Stats) addSkipped() {
	atomic.AddInt64(&s.skippedCount, 1)
}

func (s *Stats) addVideoSkipped() {
	atomic.AddInt64(&s.videoSkippedCount, 1)
}

func (s *Stats) addLinkSkipped() {
	atomic.AddInt64(&s.linkSkippedCount, 1)
}

func (s *Stats) addOtherSkipped() {
	atomic.AddInt64(&s.otherSkippedCount, 1)
}

func (s *Stats) addSize(original, converted int64) {
	atomic.AddInt64(&s.totalOriginalSize, original)
	atomic.AddInt64(&s.totalConvertedSize, converted)
}

func (s *Stats) addProcessingTime(duration time.Duration) {
	atomic.AddInt64((*int64)(&s.totalProcessingTime), int64(duration))
}

func (s *Stats) addDetailedLog(info FileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

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

	// 按格式统计
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

func init() {
	// 设置日志
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)

	// 初始化信号量 - 更保守的设置防止系统卡死
	cpuCount := runtime.NumCPU()
	procLimit := cpuCount / 2
	if procLimit < 2 {
		procLimit = 2
	}
	if procLimit > 4 {
		procLimit = 4 // 更严格的进程限制
	}
	procSem = make(chan struct{}, procLimit)
	fdSem = make(chan struct{}, procLimit*2)
}

func main() {
	logger.Printf("🎨 AVIF 批量转换工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 检查系统依赖
	if err := checkDependencies(); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	// 解析命令行参数
	opts := parseFlags()
	logger.Printf("📁 准备处理目录...")

	// 处理输入目录
	if opts.InputDir == "" {
		logger.Fatalf("❌ 必须指定输入目录")
	}

	// 检查输入目录是否存在
	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 输入目录不存在: %s", opts.InputDir)
	}

	// 设置输出目录
	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir
	}

	logger.Printf("📂 直接处理目录: %s", opts.InputDir)

	// 扫描文件
	candidateFiles, err := scanCandidateFiles(opts.InputDir)
	if err != nil {
		logger.Fatalf("❌ 扫描文件失败: %v", err)
	}

	if len(candidateFiles) == 0 {
		logger.Println("ℹ️  未找到可处理的文件")
		return
	}

	logger.Printf("📊 发现 %d 个候选文件", len(candidateFiles))

	// 配置处理性能
	logger.Printf("⚡ 配置处理性能...")
	logger.Printf("🚀 性能配置: %d个工作线程, %d个进程限制, %d个文件句柄限制", opts.Workers, cap(procSem), cap(fdSem))
	logger.Printf("💻 系统信息: %d个CPU核心", runtime.NumCPU())

	// 开始处理
	logger.Printf("🚀 开始并行处理 - 目录: %s, 工作线程: %d, 文件数: %d", opts.InputDir, opts.Workers, len(candidateFiles))

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	logger.Printf("🛑 设置信号处理...")

	// 添加全局超时保护，防止系统卡死
	globalTimeout := time.Duration(len(candidateFiles)) * 30 * time.Second // 每个文件最多30秒
	if globalTimeout > 2*time.Hour {
		globalTimeout = 2 * time.Hour // 最大2小时
	}
	logger.Printf("⏰ 设置全局超时保护: %v", globalTimeout)

	// 创建超时上下文
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), globalTimeout)
	defer timeoutCancel()

	// 创建统计对象
	stats := &Stats{}

	// 使用ants创建goroutine池
	pool, err := ants.NewPool(opts.Workers)
	if err != nil {
		logger.Fatalf("❌ 创建goroutine池失败: %v", err)
	}
	defer pool.Release()

	// 创建WaitGroup
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

	// 输出详细统计
	stats.logDetailedSummary()

	// 输出简单统计
	logger.Println("🎯 ===== 处理摘要 =====")
	logger.Printf("✅ 成功处理图像: %d", atomic.LoadInt64(&stats.successCount))
	logger.Printf("❌ 转换失败图像: %d", atomic.LoadInt64(&stats.failureCount))
	logger.Printf("🎬 跳过视频文件: %d", atomic.LoadInt64(&stats.videoSkippedCount))
	logger.Printf("🔗 跳过符号链接: %d", atomic.LoadInt64(&stats.linkSkippedCount))
	logger.Printf("📄 跳过其他文件: %d", atomic.LoadInt64(&stats.otherSkippedCount))

	// 大小统计
	originalSize := atomic.LoadInt64(&stats.totalOriginalSize)
	convertedSize := atomic.LoadInt64(&stats.totalConvertedSize)
	savedSize := originalSize - convertedSize
	compressionRate := float64(convertedSize) / float64(originalSize) * 100

	logger.Println("📊 ===== 大小统计 =====")
	logger.Printf("📥 原始总大小: %.2f MB", float64(originalSize)/(1024*1024))
	logger.Printf("📤 转换后大小: %.2f MB", float64(convertedSize)/(1024*1024))
	logger.Printf("💾 节省空间: %.2f MB (压缩率: %.1f%%)", float64(savedSize)/(1024*1024), compressionRate)

	// 格式统计
	formatCounts := make(map[string]int)
	for _, log := range stats.detailedLogs {
		formatCounts[log.FileType]++
	}

	logger.Println("📋 ===== 格式统计 =====")
	for format, count := range formatCounts {
		logger.Printf("  🖼️  %s: %d个文件", format, count)
	}

	// 🔍 文件数量验证
	logger.Println("🔍 验证处理结果...")
	validateFileCount(opts.InputDir, len(candidateFiles), stats)

	logger.Println("🎉 ===== 处理完成 =====")
}

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
	".jpg":  true, ".jpeg": true, ".png":  true, ".gif":  true, ".apng": true, ".webp": true,
	".avif": true, ".heic": true, ".heif": true, ".jfif": true, ".jpe":  true, ".bmp":  true,
	".tiff": true, ".tif":  true, ".ico":  true, ".cur":  true, ".psd":  true, ".xcf":  true,
	".ora":  true, ".kra":  true, ".svg":  true, ".eps":  true, ".ai":   true,
}

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

func isSupportedImageType(ext string) bool {
	return supportedExtensions[ext]
}

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

	// 删除原始文件
	if opts.ReplaceOriginals {
		if err := os.Remove(filePath); err != nil {
			logger.Printf("⚠️  删除原始文件失败 %s: %v", filepath.Base(filePath), err)
		} else {
			logger.Printf("🗑️  已删除原始文件: %s", filepath.Base(filePath))
		}
	}
}

func convertToAvif(filePath string, kind types.Type, isAnimated bool, opts Options) (int64, error) {
	avifPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".avif"
	originalFilePath := filePath // Preserve original file path for metadata copy

	// HEIC/HEIF conversion using enhanced magick to a more stable PNG intermediate
	if kind.Extension == "heic" || kind.Extension == "heif" {
		tempPngPath := avifPath + ".png"
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
				tempRelaxedPngPath := avifPath + ".relaxed.png"
				cmd = exec.Command("magick", "-define", "heic:limit-num-tiles=0", "-define", "heic:max-image-size=0", "-define", "heic:use-embedded-profile=false", "-define", "heic:decode-effort=0", "-depth", "16", filePath, tempRelaxedPngPath)
				output, err = cmd.CombinedOutput()
				if err != nil {
					logger.Printf("WARN: All HEIC conversion methods failed for %s. ImageMagick, ffmpeg, and relaxed ImageMagick all failed. Output ImageMagick: %s, ffmpeg: %s, relaxed ImageMagick: %s", 
						filepath.Base(filePath), string(output), string(ffmpegOutput), string(output))
					return 0, fmt.Errorf("all HEIC conversion methods failed: ImageMagick error: %v, ffmpeg error: %v", err, ffmpegErr)
				}
				// Use the relaxed ImageMagick output
				defer os.Remove(tempRelaxedPngPath)
				filePath = tempRelaxedPngPath
			} else {
				// Successfully converted with ffmpeg, now use PNG as input
				defer os.Remove(tempPngPath)
				filePath = tempPngPath
			}
		} else {
			// Successfully converted with original ImageMagick approach
			defer os.Remove(tempPngPath)
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
	if err != nil {
		return 0, fmt.Errorf("ffmpeg execution failed: %s\nOutput: %s", err, string(output))
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

func withTimeout(ctx context.Context, opts Options) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(opts.TimeoutSeconds)*time.Second)
}

// validateFileCount 验证处理前后的文件数量
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
