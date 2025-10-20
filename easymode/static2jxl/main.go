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
)

const (
	logFileName = "static2jxl.log"
	version     = "2.0.1"
	author      = "AI Assistant"
)

var (
	logger *log.Logger
	// 限制外部进程与文件句柄并发，避免过载
	procSem chan struct{}
	fdSem   chan struct{}
)

type Options struct {
	Workers        int
	SkipExist      bool
	DryRun         bool
	CJXLThreads    int
	TimeoutSeconds int
	Retries        int
	InputDir       string
	OutputDir      string
}

// FileProcessInfo 记录单个文件的处理信息
type FileProcessInfo struct {
	FilePath       string
	FileSize       int64
	FileType       string
	ProcessingTime time.Duration
	ConversionMode string
	Success        bool
	ErrorMsg       string
	SizeSaved      int64
}

// Stats 统计信息结构体
type Stats struct {
	sync.Mutex
	imagesProcessed  int
	imagesFailed     int
	othersSkipped    int
	totalBytesBefore int64
	totalBytesAfter  int64
	byExt            map[string]int
	detailedLogs     []FileProcessInfo
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

func (s *Stats) addOtherSkipped() {
	s.Lock()
	defer s.Unlock()
	s.othersSkipped++
}

func (s *Stats) addDetailedLog(info FileProcessInfo) {
	s.Lock()
	defer s.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

func init() {
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
}

func main() {
	// 🚀 程序启动
	logger.Printf("🎨 静态图片转JXL工具 v%s", version)
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
			if isSupportedStaticType(ext) {
				files = append(files, osPathname)
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
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".heic" || ext == ".heif" {
		// Use multiple approaches to convert HEIC to a format that cjxl can handle
		// Approach 1: Use magick with increased limits to convert to tiff first
		tempTiffPath := outputPath + ".tiff"
		cmd := exec.Command("magick", filePath, tempTiffPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Printf("WARN: ImageMagick failed for %s: %v. Output: %s. Trying alternative method.", filepath.Base(filePath), err, string(output))
			
			// Approach 2: Use ffmpeg as fallback to convert HEIC to PNG
			tempPngPath := outputPath + ".png"
			cmd = exec.Command("ffmpeg", "-i", filePath, "-c:v", "png", tempPngPath)
			ffmpegOutput, ffmpegErr := cmd.CombinedOutput()
			if ffmpegErr != nil {
				logger.Printf("WARN: Ffmpeg failed for %s: %v. Output: %s. Trying ImageMagick with relaxed limits.", filepath.Base(filePath), ffmpegErr, string(ffmpegOutput))
				
				// Approach 3: Try using ImageMagick with relaxed policy
				tempRelaxedTiffPath := outputPath + ".relaxed.tiff"
				cmd = exec.Command("magick", filePath, "-define", "heic:limit-num-tiles=0", tempRelaxedTiffPath)
				output, err = cmd.CombinedOutput()
				if err != nil {
					logger.Printf("WARN: All HEIC conversion methods failed for %s. ImageMagick, ffmpeg, and relaxed ImageMagick all failed. Output ImageMagick: %s, ffmpeg: %s, relaxed ImageMagick: %s", 
						filepath.Base(filePath), string(output), string(ffmpegOutput), string(output))
					return fmt.Errorf("all HEIC conversion methods failed: ImageMagick error: %v, ffmpeg error: %v", err, ffmpegErr)
				}
				// Use the relaxed ImageMagick output
				defer os.Remove(tempRelaxedTiffPath)
				filePath = tempRelaxedTiffPath
			} else {
				// Successfully converted with ffmpeg, now use PNG as input
				defer os.Remove(tempPngPath)
				filePath = tempPngPath
			}
		} else {
			// Successfully converted with original ImageMagick approach
			defer os.Remove(tempTiffPath)
			filePath = tempTiffPath
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
	if err != nil {
		return fmt.Errorf("cjxl执行失败: %w\n输出: %s", err, string(output))
	}

	return nil
}

var supportedStaticExtensions = map[string]bool{
	".jpg":  true, ".jpeg": true, ".png":  true, ".bmp":  true,
	".tiff": true, ".tif":  true, ".heic": true, ".heif": true,
    ".jfif": true, ".jpe": true,
}

func isSupportedStaticType(ext string) bool {
	return supportedStaticExtensions[ext]
}

func printSummary(stats *Stats) {
	stats.Lock()
	defer stats.Unlock()

	totalSavedKB := float64(stats.totalBytesBefore-stats.totalBytesAfter) / 1024.0
	totalSavedMB := totalSavedKB / 1024.0
	compressionRatio := float64(stats.totalBytesAfter) / float64(stats.totalBytesBefore) * 100

	logger.Println("🎯 ===== 处理摘要 =====")
	logger.Printf("✅ 成功处理图像: %d", stats.imagesProcessed)
	logger.Printf("❌ 转换失败图像: %d", stats.imagesFailed)
	logger.Printf("📄 跳过其他文件: %d", stats.othersSkipped)
	logger.Println("📊 ===== 大小统计 =====")
	logger.Printf("📥 原始总大小: %.2f MB", float64(stats.totalBytesBefore)/(1024*1024))
	logger.Printf("📤 转换后大小: %.2f MB", float64(stats.totalBytesAfter)/(1024*1024))
	logger.Printf("💾 节省空间: %.2f MB (压缩率: %.1f%%)", totalSavedMB, compressionRatio)

	if len(stats.byExt) > 0 {
		logger.Println("📋 ===== 格式统计 =====")
		for k, v := range stats.byExt {
			logger.Printf("  🖼️  %s: %d个文件", k, v)
		}
	}
	logger.Println("🎉 ===== 处理完成 =====")
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
