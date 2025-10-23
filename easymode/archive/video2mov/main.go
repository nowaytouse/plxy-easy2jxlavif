// video2mov - 批量视频转MOV格式工具
//
// 功能说明：
// - 支持多种视频格式批量转换为MOV格式
// - 保留原始文件的元数据和系统时间戳
// - 使用ffmpeg进行视频重新封装，不重新编码
// - 提供详细的处理统计和进度报告
// - 支持并发处理以提高转换效率
//
// 作者：AI Assistant
// 版本：2.1.0
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"pixly/utils"

	"github.com/karrick/godirwalk"
	"github.com/shirou/gopsutil/mem"
)

// 程序常量定义
const (
	logFileName = "video2mov.log" // 日志文件名
	version     = "2.1.0"         // 程序版本号
	author      = "AI Assistant"  // 作者信息
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
	SkipExist        bool   // 是否跳过已存在的MOV文件
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
	FilePath        string        // 文件完整路径
	FileSize        int64         // 文件大小（字节）
	FileType        string        // 文件类型（扩展名）
	ProcessingTime  time.Duration // 处理耗时
	ConversionMode  string        // 转换模式
	Success         bool          // 是否处理成功
	ErrorMsg        string        // 错误信息（如果处理失败）
	SizeSaved       int64         // 节省的空间大小
	MetadataSuccess bool          // 元数据复制是否成功
}

// Stats 结构体用于在整个批处理过程中收集和管理统计数据
// 它使用互斥锁（sync.Mutex）来确保并发访问时的线程安全
type Stats struct {
	sync.Mutex                            // 互斥锁，确保并发安全
	imagesProcessed     int64             // 成功处理的视频数量
	imagesFailed        int64             // 处理失败的视频数量
	othersSkipped       int64             // 跳过的其他文件数量
	totalBytesBefore    int64             // 原始文件总大小
	totalBytesAfter     int64             // 转换后文件总大小
	byExt               map[string]int    // 按扩展名统计的文件数量
	detailedLogs        []FileProcessInfo // 详细的处理日志记录
	processingStartTime time.Time         // 处理开始时间
	totalProcessingTime time.Duration     // 总处理时间
}

// addImageProcessed 原子性地增加成功处理视频的计数
func (s *Stats) addImageProcessed(sizeBefore, sizeAfter int64) {
	atomic.AddInt64(&s.imagesProcessed, 1)
	atomic.AddInt64(&s.totalBytesBefore, sizeBefore)
	atomic.AddInt64(&s.totalBytesAfter, sizeAfter)
}

// addImageFailed 原子性地增加处理失败视频的计数
func (s *Stats) addImageFailed() {
	atomic.AddInt64(&s.imagesFailed, 1)
}

// addOtherSkipped 原子性地增加跳过其他文件的计数
func (s *Stats) addOtherSkipped() {
	atomic.AddInt64(&s.othersSkipped, 1)
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
		var totalSize, totalSaved int64
		var successCount int
		for _, log := range logs {
			totalSize += log.FileSize
			totalSaved += log.SizeSaved
			if log.Success {
				successCount++
			}
		}
		compressionRatio := float64(totalSaved) / float64(totalSize) * 100
		logger.Printf("🎥 %s格式: %d个文件, 成功率%.1f%%, 平均压缩率%.1f%%",
			format, len(logs), float64(successCount)/float64(len(logs))*100, compressionRatio)
	}

	// 显示处理最慢的文件
	logger.Println("⏱️  处理时间最长的文件:")
	var slowestFiles []FileProcessInfo
	for _, log := range s.detailedLogs {
		slowestFiles = append(slowestFiles, log)
	}
	sort.Slice(slowestFiles, func(i, j int) bool {
		return slowestFiles[i].ProcessingTime > slowestFiles[j].ProcessingTime
	})

	for i, log := range slowestFiles {
		if i >= 3 { // 只显示前3个最慢的
			break
		}
		logger.Printf("   🐌 %s: %v", filepath.Base(log.FilePath), log.ProcessingTime)
	}
}

func printSummary(stats *Stats) {
	stats.Lock()
	defer stats.Unlock()

	// 计算节省的空间，如果转换后文件更大则显示为0
	totalSavedBytes := stats.totalBytesBefore - stats.totalBytesAfter
	if totalSavedBytes < 0 {
		totalSavedBytes = 0
	}
	totalSavedKB := float64(totalSavedBytes) / 1024.0
	totalSavedMB := totalSavedKB / 1024.0

	// 计算压缩率（如果转换后文件更大则显示大于100%）
	compressionRatio := float64(stats.totalBytesAfter) / float64(stats.totalBytesBefore) * 100

	logger.Println("🎯 ===== 处理摘要 =====")
	logger.Printf("✅ 成功重新包装视频: %d", stats.imagesProcessed)
	logger.Printf("❌ 重新包装失败视频: %d", stats.imagesFailed)
	logger.Printf("📄 跳过其他文件: %d", stats.othersSkipped)
	logger.Println("📊 ===== 大小统计 =====")
	logger.Printf("📥 原始总大小: %.2f MB", float64(stats.totalBytesBefore)/(1024*1024))
	logger.Printf("📤 重新包装后大小: %.2f MB", float64(stats.totalBytesAfter)/(1024*1024))
	logger.Printf("💾 节省空间: %.2f MB (压缩率: %.1f%%)", totalSavedMB, compressionRatio)

	if len(stats.byExt) > 0 {
		logger.Println("📋 ===== 格式统计 =====")
		for k, v := range stats.byExt {
			logger.Printf("  🎥  %s: %d个文件", k, v)
		}
	}
	logger.Println("🎉 ===== 处理完成 =====")
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
	logger.Printf("🎥 视频重新包装工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 解析命令行参数
	opts := parseFlags()

	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录")
	}

	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir // 默认输出目录为输入目录
	}

	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		logger.Fatalf("❌ 错误: 无法创建输出目录 %s: %v", opts.OutputDir, err)
	}

	// 检查系统依赖工具是否可用
	logger.Println("🔍 检查系统依赖...")
	if err := checkDependencies(); err != nil {
		logger.Printf("❌ 系统依赖检查失败: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("\n🛑 收到中断信号，正在优雅退出...")
		cancel()
	}()

	stats := &Stats{
		byExt:               make(map[string]int),
		processingStartTime: time.Now(),
	}

	files, err := processDirectory(ctx, opts, stats)
	if err != nil {
		logger.Fatalf("❌ 处理目录时出错: %v", err)
	}

	elapsed := time.Since(stats.processingStartTime)
	stats.totalProcessingTime = elapsed
	logger.Printf("⏱️  总处理时间: %s", elapsed)

	stats.logDetailedSummary()

	validateFileCount(opts.InputDir, opts.OutputDir, len(files), stats)

	printSummary(stats)
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

// smartThreadAdjustment
func smartThreadAdjustment(currentWorkers int) int {
	v, err := mem.VirtualMemory()
	if err != nil {
		return currentWorkers
	}
	if v.UsedPercent > 80 {
		return currentWorkers / 2
	}
	return currentWorkers
}

// parseFlags 解析命令行参数并返回配置选项
func parseFlags() *Options {
	opts := &Options{
		Workers:          0,
		SkipExist:        true,
		DryRun:           false,
		TimeoutSeconds:   300,
		Retries:          2,
		ReplaceOriginals: false,
	}

	flag.IntVar(&opts.Workers, "workers", opts.Workers, "并发工作线程数 (默认: CPU核心数)")
	flag.BoolVar(&opts.SkipExist, "skip-exist", opts.SkipExist, "跳过已存在的文件")
	flag.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "试运行模式，只打印将要处理的文件")
	flag.IntVar(&opts.TimeoutSeconds, "timeout", opts.TimeoutSeconds, "单个文件处理超时秒数")
	flag.IntVar(&opts.Retries, "retries", opts.Retries, "失败重试次数")
	flag.StringVar(&opts.InputDir, "input", "", "输入目录 (必需)")
	flag.StringVar(&opts.OutputDir, "output", "", "输出目录 (默认为输入目录)")
	flag.BoolVar(&opts.ReplaceOriginals, "replace", opts.ReplaceOriginals, "重新包装后删除原始文件")

	flag.Parse()

	// 参数验证
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

var supportedVideoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mov": true, ".mkv": true, ".wmv": true, ".flv": true, ".webm": true, ".m4v": true, ".3gp": true,
}

// Only source formats (not including .mov since we're converting TO mov)
var sourceVideoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mkv": true, ".wmv": true, ".flv": true, ".webm": true, ".m4v": true, ".3gp": true,
}

func isSupportedVideoType(ext string) bool {
	return sourceVideoExtensions[ext]
}

func processDirectory(ctx context.Context, opts *Options, stats *Stats) ([]string, error) {
	logger.Printf("📂 扫描目录: %s", opts.InputDir)

	files := make([]string, 0)
	err := godirwalk.Walk(opts.InputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if de.IsDir() {
				if osPathname == opts.OutputDir {
					// Skip the output directory if it's a subdirectory of input directory
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(osPathname))
			if isSupportedVideoType(ext) {
				files = append(files, osPathname)
			}

			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, fmt.Errorf("目录扫描失败: %w", err)
	}

	logger.Printf("✅ 找到 %d 个支持的视频文件", len(files))

	if len(files) == 0 {
		logger.Println("⚠️  没有找到支持的视频文件")
		return files, nil
	}

	// 智能性能配置
	workers := opts.Workers
	cpuCount := runtime.NumCPU()

	if workers <= 0 {
		workers = cpuCount
	}

	maxWorkers := cpuCount * 2
	if workers > maxWorkers {
		workers = maxWorkers
	}

	procLimit := cpuCount
	if procLimit > 8 {
		procLimit = 8
	}
	fdLimit := procLimit * 4

	procSem = make(chan struct{}, procLimit)
	fdSem = make(chan struct{}, fdLimit)

	workers = smartThreadAdjustment(workers)

	logger.Printf("⚡ 启动处理进程 (工作线程: %d)", workers)

	var wg sync.WaitGroup
	for _, filePath := range files {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				processFileWithOpts(fp, opts, stats)
			}
		}(filePath)
	}

	wg.Wait()
	logger.Println("🎉 所有文件处理完成")
	return files, nil
}

func processFileWithOpts(filePath string, opts *Options, stats *Stats) {
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
		processInfo.FileSize = stat.Size()
		originalModTime = stat.ModTime()
		if ctime, _, ok := getFileTimesDarwin(filePath); ok {
			originalCreateTime = ctime
		}
	}

	logger.Printf("🔄 开始处理: %s", fileName)

	relPath, err := filepath.Rel(opts.InputDir, filePath)
	if err != nil {
		processInfo.ErrorMsg = fmt.Sprintf("无法获取相对路径: %v", err)
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	outputPath := filepath.Join(opts.OutputDir, relPath)
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".mov"

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		processInfo.ErrorMsg = fmt.Sprintf("无法创建输出目录: %v", err)
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addOtherSkipped()
		stats.addDetailedLog(processInfo)
		return
	}

	if opts.SkipExist {
		if _, err := os.Stat(outputPath); err == nil {
			logger.Printf("⏭️  跳过已存在的文件: %s", fileName)
			processInfo.Success = true
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addOtherSkipped()
			stats.addDetailedLog(processInfo)
			return
		}
	}

	var success bool
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		logger.Printf("🔄 开始重新包装 %s (尝试 %d/%d)", fileName, attempt+1, opts.Retries+1)
		err = rePackageToMov(filePath, outputPath, opts)
		if err != nil {
			if attempt == opts.Retries {
				logger.Printf("❌ 重新包装失败 %s: %v", fileName, err)
				processInfo.ErrorMsg = fmt.Sprintf("重新包装失败: %v", err)
				processInfo.ProcessingTime = time.Since(startTime)
				stats.addImageFailed()
				stats.addDetailedLog(processInfo)
				return
			}
			logger.Printf("🔄 重试重新包装 %s (尝试 %d/%d)", fileName, attempt+1, opts.Retries)
			continue
		}
		success = true
		break
	}

	if !success {
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addImageFailed()
		stats.addDetailedLog(processInfo)
		return
	}

	processInfo.Success = true
	logger.Printf("✅ 重新包装完成: %s -> %s", fileName, filepath.Base(outputPath))

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

	if stat, err := os.Stat(outputPath); err == nil {
		processInfo.SizeSaved = processInfo.FileSize - stat.Size()
	}

	processInfo.ProcessingTime = time.Since(startTime)
	stats.addImageProcessed(processInfo.FileSize, processInfo.FileSize-processInfo.SizeSaved)
	stats.addDetailedLog(processInfo)

	if opts.ReplaceOriginals {
		// 安全删除：使用安全删除函数，仅在确认输出文件存在且有效后才删除原始文件
		if err := utils.SafeDelete(filePath, outputPath, func(format string, v ...interface{}) {
			logger.Printf(format, v...)
		}); err != nil {
			logger.Printf("⚠️  安全删除失败 %s: %v", filepath.Base(filePath), err)
		}
	}
}

func rePackageToMov(filePath, outputPath string, opts *Options) error {
	args := []string{
		"-i", filePath,
		"-c", "copy", // 重新包装，不进行编码
		"-movflags", "+faststart",
		"-y", // 覆盖输出文件
		outputPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()

	procSem <- struct{}{}
	defer func() { <-procSem }()

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg重新包装失败: %w\n输出: %s", err, string(output))
	}

	// 转换结果验证
	if verr := validateMov(filePath, outputPath); verr != nil {
		return fmt.Errorf("转换后验证失败: %w", verr)
	}

	// 复制元数据
	if err := copyMetadata(filePath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败 %s: %v", filepath.Base(filePath), err)
	}

	return nil
}

// validateMov: ffprobe 容器/时长/分辨率校验
func validateMov(originalPath, outputPath string) error {
	type Probe struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}

	run := func(p string) (Probe, error) {
		var pr Probe
		out, err := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_format", "-show_streams", p).CombinedOutput()
		if err != nil {
			return pr, fmt.Errorf("ffprobe失败: %v, 输出:%s", err, string(out))
		}
		if e := json.Unmarshal(out, &pr); e != nil {
			return pr, fmt.Errorf("解析ffprobe输出失败: %v", e)
		}
		return pr, nil
	}

	op, err := run(originalPath)
	if err != nil {
		return err
	}
	np, err := run(outputPath)
	if err != nil {
		return err
	}

	if np.Format.FormatName == "" || !strings.Contains(np.Format.FormatName, "mov") {
		return fmt.Errorf("输出容器非MOV: %s", np.Format.FormatName)
	}
	pf := func(s string) float64 { v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64); return v }
	if od, nd := pf(op.Format.Duration), pf(np.Format.Duration); od > 0 && nd > 0 && math.Abs(od-nd) > 0.5 {
		return fmt.Errorf("时长差异过大: %.3fs vs %.3fs", od, nd)
	}
	ow, oh := 0, 0
	nw, nh := 0, 0
	for _, s := range op.Streams {
		if s.CodecType == "video" {
			ow, oh = s.Width, s.Height
			break
		}
	}
	for _, s := range np.Streams {
		if s.CodecType == "video" {
			nw, nh = s.Width, s.Height
			break
		}
	}
	if ow > 0 && oh > 0 && (ow != nw || oh != nh) {
		return fmt.Errorf("分辨率不一致: %dx%d -> %dx%d", ow, oh, nw, nh)
	}
	return nil
}

func copyMetadata(sourcePath, targetPath string) error {
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", sourcePath, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool failed: %s\nOutput: %s", err, string(output))
	}
	logger.Printf("📋 元数据复制成功: %s", filepath.Base(sourcePath))
	return nil
}

func validateFileCount(inputDir string, outputDir string, originalVideoCount int, stats *Stats) {
	currentRemainingVideoCount := 0
	currentMovCount := 0

	// Scan outputDir for .mov files
	err := godirwalk.Walk(outputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				ext := strings.ToLower(filepath.Ext(osPathname))
				if ext == ".mov" {
					currentMovCount++
				}
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		logger.Printf("⚠️  文件数量验证失败 (扫描输出目录): %v", err)
		return
	}

	// Scan inputDir for remaining original video files, excluding output directory if it's within input
	err = godirwalk.Walk(inputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				ext := strings.ToLower(filepath.Ext(osPathname))
				if sourceVideoExtensions[ext] {
					currentRemainingVideoCount++
				}
			} else if osPathname == outputDir {
				// Skip the output directory if it's a subdirectory of input directory
				return filepath.SkipDir
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		logger.Printf("⚠️  文件数量验证失败 (扫描输入目录): %v", err)
		return
	}

	expectedMovCount := int(stats.imagesProcessed)
	expectedRemainingVideoCount := originalVideoCount - int(stats.imagesProcessed)

	logger.Printf("📊 文件数量验证:")
	logger.Printf("   原始视频文件数: %d", originalVideoCount)
	logger.Printf("   成功重新包装视频: %d", stats.imagesProcessed)
	logger.Printf("   重新包装失败/跳过: %d", stats.imagesFailed+stats.othersSkipped)
	logger.Printf("   ---")
	logger.Printf("   期望MOV文件数 (输出目录): %d", expectedMovCount)
	logger.Printf("   实际MOV文件数 (输出目录): %d", currentMovCount)
	if currentMovCount == expectedMovCount && currentRemainingVideoCount == expectedRemainingVideoCount {
		logger.Printf("✅ 文件数量验证通过。")
	} else {
		logger.Printf("⚠️ 文件数量验证存在差异 (实际MOV: %d, 期望MOV: %d; 实际剩余: %d, 期望剩余: %d) —— 仅记录，不判失败。", currentMovCount, expectedMovCount, currentRemainingVideoCount, expectedRemainingVideoCount)
	}

	// 查找可能的临时文件
	tempFiles := findTempFiles(inputDir)
	outputTempFiles := findTempFiles(outputDir)
	allTempFiles := append(tempFiles, outputTempFiles...)
	if len(allTempFiles) > 0 {
		logger.Printf("🗑️  发现 %d 个临时文件，正在清理...", len(allTempFiles))
		cleanupTempFiles(allTempFiles)
		logger.Printf("✅ 临时文件清理完成")
	}
}

func findTempFiles(workDir string) []string {
	var tempFiles []string
	err := godirwalk.Walk(workDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				if strings.Contains(filepath.Base(osPathname), ".mov.tmp") ||
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
		logger.Printf("⚠️  查找临时文件失败: %v\n", err)
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
