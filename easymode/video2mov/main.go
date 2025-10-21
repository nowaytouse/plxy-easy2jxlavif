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
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/karrick/godirwalk"
	"pixly/utils"
)

const (
	logFileName      = "video2mov.log"
	version     = "2.1.0"
	author           = "AI Assistant"
)

var (
	logger *log.Logger
	procSem chan struct{}
	fdSem   chan struct{}
)

type Options struct {
	Workers          int
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
	FilePath        string
	FileSize        int64
	FileType        string
	ProcessingTime  time.Duration
	ConversionMode  string
	Success         bool
	ErrorMsg        string
	SizeSaved       int64
	MetadataSuccess bool
}

// Stats 统计信息结构体
type Stats struct {
	sync.Mutex
	imagesProcessed     int64
	imagesFailed        int64
	othersSkipped       int64
	totalBytesBefore    int64
	totalBytesAfter     int64
	byExt               map[string]int
	detailedLogs        []FileProcessInfo
	processingStartTime time.Time
	totalProcessingTime time.Duration
}

func (s *Stats) addImageProcessed(sizeBefore, sizeAfter int64) {
	atomic.AddInt64(&s.imagesProcessed, 1)
	atomic.AddInt64(&s.totalBytesBefore, sizeBefore)
	atomic.AddInt64(&s.totalBytesAfter, sizeAfter)
}

func (s *Stats) addImageFailed() {
	atomic.AddInt64(&s.imagesFailed, 1)
}

func (s *Stats) addOtherSkipped() {
	atomic.AddInt64(&s.othersSkipped, 1)
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

func init() {
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)

	cpuCount := runtime.NumCPU()
	procLimit := cpuCount / 2
	if procLimit < 2 {
		procLimit = 2
	}
	if procLimit > 4 {
		procLimit = 4
	}
	procSem = make(chan struct{}, procLimit)
	fdSem = make(chan struct{}, procLimit*2)
}

func main() {
	logger.Printf("🎥 视频重新包装工具 v%s", version)
	logger.Println("✨ 作者:", author)
	logger.Println("🔧 开始初始化...")

	opts := parseFlags()

	if opts.InputDir == "" {
		logger.Fatal("❌ 错误: 必须指定输入目录")
	}

	if opts.OutputDir == "" {
		opts.OutputDir = opts.InputDir // Default to input directory if not specified
	}

	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		logger.Fatalf("❌ 错误: 输入目录不存在: %s", opts.InputDir)
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		logger.Fatalf("❌ 错误: 无法创建输出目录 %s: %v", opts.OutputDir, err)
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
		byExt: make(map[string]int),
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

	// 复制元数据
	if err := copyMetadata(filePath, outputPath); err != nil {
		logger.Printf("⚠️  元数据复制失败 %s: %v", filepath.Base(filePath), err)
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
	logger.Printf("   ---")
	logger.Printf("   期望剩余视频文件数 (输入目录): %d", expectedRemainingVideoCount)
	logger.Printf("   实际剩余视频文件数 (输入目录): %d", currentRemainingVideoCount)

	if currentMovCount == expectedMovCount && currentRemainingVideoCount == expectedRemainingVideoCount {
		logger.Printf("✅ 文件数量验证通过。")
	} else {
		logger.Printf("❌ 文件数量验证失败。")
		if currentMovCount != expectedMovCount {
			logger.Printf("   MOV文件数不匹配 (实际: %d, 期望: %d)", currentMovCount, expectedMovCount)
		}
		if currentRemainingVideoCount != expectedRemainingVideoCount {
			logger.Printf("   剩余视频文件数不匹配 (实际: %d, 期望: %d)", currentRemainingVideoCount, expectedRemainingVideoCount)
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
