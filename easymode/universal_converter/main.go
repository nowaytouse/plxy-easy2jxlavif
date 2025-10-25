// universal_converter - 通用媒体转换工具
//
// 功能特性：
// - 支持AVIF、JXL、MOV格式转换
// - 支持静态、动态、视频文件处理
// - 集成8层验证系统确保转换质量
// - 智能性能优化和资源管理
// - 完整的元数据保留和错误处理
// - 支持批量处理和进度监控
//
// 作者: AI Assistant
// 版本: v2.3.2
// 更新: 2025-10-24

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"pixly/utils"

	"github.com/karrick/godirwalk"
)

// 全局变量定义
var (
	logger     *log.Logger        // 日志记录器，用于输出处理信息
	stats      *ProcessingStats   // 处理统计信息，记录转换进度和结果
	procSem    chan struct{}      // 处理信号量，控制并发处理数量
	fdSem      chan struct{}      // 文件描述符信号量，防止文件句柄耗尽
	globalCtx  context.Context    // 全局上下文，用于取消操作
	cancelFunc context.CancelFunc // 取消函数，用于优雅停止处理
)

// ProcessingStats 处理统计信息结构体
// 用于记录和跟踪媒体文件转换过程中的各种统计数据和性能指标
type ProcessingStats struct {
	mu              sync.RWMutex      // 读写锁，保护并发访问
	processed       int               // 成功处理的文件数量
	failed          int               // 处理失败的文件数量
	skipped         int               // 跳过的文件总数
	videoSkipped    int               // 跳过的视频文件数量
	otherSkipped    int               // 跳过的其他类型文件数量
	totalSizeBefore int64             // 处理前总文件大小（字节）
	totalSizeAfter  int64             // 处理后总文件大小（字节）
	byExt           map[string]int    // 按文件扩展名统计处理数量
	detailedLogs    []FileProcessInfo // 详细的文件处理日志
	startTime       time.Time         // 处理开始时间
}

// FileProcessInfo 文件处理信息结构体
// 记录单个文件在转换过程中的详细信息，用于日志记录和性能分析
type FileProcessInfo struct {
	FileName       string        // 文件名（不含路径）
	FilePath       string        // 完整文件路径
	FileType       string        // 文件类型（如：jpg, png, gif等）
	IsAnimated     bool          // 是否为动画文件
	Success        bool          // 处理是否成功
	ErrorMsg       string        // 错误信息（如果处理失败）
	ProcessingTime time.Duration // 处理耗时
	SizeBefore     int64         // 处理前文件大小（字节）
	SizeAfter      int64         // 处理后文件大小（字节）
	ConversionMode string        // 转换模式（static/dynamic/video）
}

// 统计方法
func (s *ProcessingStats) addProcessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processed++
}

func (s *ProcessingStats) addFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed++
}

func (s *ProcessingStats) addSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skipped++
}

func (s *ProcessingStats) addVideoSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.videoSkipped++
}

func (s *ProcessingStats) addOtherSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otherSkipped++
}

func (s *ProcessingStats) addImageProcessed(sizeBefore, sizeAfter int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalSizeBefore += sizeBefore
	s.totalSizeAfter += sizeAfter
}

func (s *ProcessingStats) addDetailedLog(info FileProcessInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detailedLogs = append(s.detailedLogs, info)
}

func (s *ProcessingStats) addByExt(ext string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byExt[ext]++
}

// main 主函数
// 程序入口点，负责初始化、参数解析、依赖检查和启动转换流程
func main() {
	// 初始化轮转日志系统
	// 日志文件大小限制为50MB，超过后自动轮转
	rl, lf, err := utils.NewRotatingLogger("universal_converter.log", 50*1024*1024)
	if err != nil {
		log.Fatalf("无法初始化轮转日志: %v", err)
	}
	logger = rl
	_ = lf
	logger.Printf("🎨 通用媒体转换工具 v2.3.2")
	logger.Printf("✨ 作者: AI Assistant")
	logger.Printf("🔧 开始初始化...")

	// 解析命令行参数，获取用户配置
	opts := utils.ParseUniversalFlags()
	logger.Printf("📋 配置: %s", opts.GetDescription())

	// 检查系统依赖工具是否可用
	// 包括cjxl、djxl、ffmpeg、exiftool等必要工具
	if err := checkDependencies(opts); err != nil {
		logger.Fatalf("❌ 系统依赖检查失败: %v", err)
	}

	// 初始化统计
	stats = &ProcessingStats{
		byExt:     make(map[string]int),
		startTime: time.Now(),
	}

	// 设置信号处理
	setupSignalHandling()

	// 设置全局超时
	globalCtx, cancelFunc = context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancelFunc()

	// 初始化信号量
	procSem = make(chan struct{}, opts.ProcessLimit)
	fdSem = make(chan struct{}, opts.FileLimit)

	// 扫描文件
	files, err := scanFiles(opts)
	if err != nil {
		logger.Fatalf("❌ 文件扫描失败: %v", err)
	}

	if len(files) == 0 {
		logger.Printf("📊 没有找到需要处理的文件")
		return
	}

	logger.Printf("📊 发现 %d 个文件需要处理", len(files))

	// 验证扫描到的文件是否真实存在(防止并发/文件系统缓存问题)
	validatedFiles := make([]string, 0, len(files))
	invalidCount := 0
	for _, filePath := range files {
		if _, err := os.Stat(filePath); err == nil {
			validatedFiles = append(validatedFiles, filePath)
		} else {
			logger.Printf("⚠️  扫描验证: 文件不存在或无法访问: %s", filePath)
			invalidCount++
		}
	}

	if invalidCount > 0 {
		logger.Printf("⚠️  扫描验证: 发现 %d 个无效文件,已过滤", invalidCount)
	}

	files = validatedFiles
	logger.Printf("✅ 验证完成: %d 个有效文件准备处理", len(files))

	// 开始处理
	processedPairs := processFiles(files, opts)

	// 输出统计信息
	printStatistics()

	// 转换后验证（如果不是试运行模式）
	if !opts.DryRun && len(processedPairs) > 0 {
		logger.Println("")
		logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Println("🔍 开始转换后验证...")
		logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		performPostValidation(processedPairs)
	}
}

// checkDependencies 检查系统依赖，如果缺失则尝试自动安装
func checkDependencies(opts utils.UniversalOptions) error {
	// 依赖映射: 命令名 -> brew包名
	installMap := map[string]string{
		"exiftool": "exiftool",
		"avifenc":  "libavif",
		"ffmpeg":   "ffmpeg",
		"cjxl":     "jpeg-xl",
		"djxl":     "jpeg-xl",
	}

	dependencies := []string{"exiftool"}

	switch opts.ConversionType {
	case utils.ConvertToAVIF:
		dependencies = append(dependencies, "avifenc", "ffmpeg") // avifenc用于静态, ffmpeg用于动态
	case utils.ConvertToJXL:
		dependencies = append(dependencies, "cjxl", "djxl")
	case utils.ConvertToMOV:
		dependencies = append(dependencies, "ffmpeg")
	}

	for _, dep := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			logger.Printf("⚠️  缺失依赖: %s，正在尝试自动安装...", dep)
			installCmd := installMap[dep]
			if installCmd == "" {
				return fmt.Errorf("无法自动安装 %s，请手动安装", dep)
			}
			// 尝试使用brew安装 (macOS)
			cmd := exec.Command("brew", "install", installCmd)
			output, installErr := cmd.CombinedOutput()
			if installErr != nil {
				return fmt.Errorf("自动安装 %s 失败: %v\n输出: %s\n请手动运行 'brew install %s'", dep, installErr, string(output), installCmd)
			}
			logger.Printf("✅ 成功安装 %s", dep)
			// 安装后重新检查
			if _, err := exec.LookPath(dep); err != nil {
				return fmt.Errorf("%s 安装后仍不可用: %v", dep, err)
			}
		}
		logger.Printf("✅ %s 已就绪", dep)
	}

	return nil
}

// fileInfo 文件信息结构（用于排序）
type fileInfo struct {
	path string
	size int64
}

// scanFiles 扫描文件，并按文件大小升序排序以优化感官速度
func scanFiles(opts utils.UniversalOptions) ([]string, error) {
	var fileInfos []fileInfo
	scannedCount := 0
	skippedCount := 0
	skipExistCount := 0

	logger.Printf("📁 开始扫描目录: %s", opts.InputDir)
	logger.Printf("🔍 转换类型: %v, 处理模式: %v", opts.ConversionType, opts.ProcessingMode)

	err := godirwalk.Walk(opts.InputDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			// 跳过目录
			if de.IsDir() {
				// 排除垃圾箱和临时目录
				dirName := filepath.Base(osPathname)
				if dirName == ".trash" || dirName == ".Trash" || dirName == "Trash" {
					return filepath.SkipDir
				}
				return nil
			}

			scannedCount++

			// 检查是否为支持的格式
			if !opts.IsSupportedInputFormat(osPathname) {
				skippedCount++
				return nil
			}

			// 检查是否跳过已存在的文件
			if opts.SkipExist {
				ext := opts.GetOutputExtensionForFile(osPathname)
				outputPath := strings.TrimSuffix(osPathname, filepath.Ext(osPathname)) + ext
				if _, err := os.Stat(outputPath); err == nil {
					skipExistCount++
					logger.Printf("⏩ 跳过已存在: %s", filepath.Base(osPathname))
					return nil
				}
			}

			// 获取文件大小用于排序
			info, err := os.Stat(osPathname)
			if err != nil {
				return err
			}
			fileInfos = append(fileInfos, fileInfo{path: osPathname, size: info.Size()})
			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, err
	}

	// 输出扫描统计
	logger.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Printf("📊 扫描完成统计:")
	logger.Printf("  • 总扫描文件: %d", scannedCount)
	logger.Printf("  • 格式不支持: %d", skippedCount)
	logger.Printf("  • 已存在跳过: %d", skipExistCount)
	logger.Printf("  • 待处理文件: %d", len(fileInfos))
	logger.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 按文件大小升序排序（小文件先处理，优化感官速度）
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].size < fileInfos[j].size
	})

	logger.Printf("🔄 文件已按大小排序(小→大),优化感官速度")

	// 提取文件路径
	var files []string
	totalSize := int64(0)
	for _, fi := range fileInfos {
		files = append(files, fi.path)
		totalSize += fi.size
	}

	logger.Printf("💾 待处理总大小: %.2f MB", float64(totalSize)/(1024*1024))

	return files, nil
}

// processFiles 处理文件，返回成功处理的文件对
func processFiles(files []string, opts utils.UniversalOptions) []utils.FilePair {
	logger.Printf("🚀 开始并行处理 - 目录: %s, 工作线程: %d, 文件数: %d",
		opts.InputDir, opts.Workers, len(files))

	// 创建等待组
	var wg sync.WaitGroup

	// 创建文件通道
	fileChan := make(chan string, len(files))

	// 用于收集成功处理的文件对
	pairsChan := make(chan utils.FilePair, len(files))
	var pairs []utils.FilePair

	// 进度计数器
	processedCount := int32(0)
	totalFiles := int32(len(files))

	// 启动工作协程
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filePath := range fileChan {
				outputPath := processFile(filePath, opts)

				// 更新进度
				current := atomic.AddInt32(&processedCount, 1)
				percentage := float64(current) / float64(totalFiles) * 100

				if current%10 == 0 || current == totalFiles {
					logger.Printf("⏳ 处理进度: %d/%d (%.1f%%)", current, totalFiles, percentage)
				}

				if outputPath != "" {
					// 成功处理，记录文件对
					pairsChan <- utils.FilePair{
						OriginalPath:  filePath,
						ConvertedPath: outputPath,
					}
				}
			}
		}(i)
	}

	// 收集结果的协程
	go func() {
		for pair := range pairsChan {
			pairs = append(pairs, pair)
		}
	}()

	// 发送文件到通道
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// 等待所有工作协程完成
	wg.Wait()
	close(pairsChan)

	// 给收集协程一点时间完成
	time.Sleep(100 * time.Millisecond)

	return pairs
}

// processFile 处理单个文件，返回输出路径（如果成功）
func processFile(filePath string, opts utils.UniversalOptions) string {
	startTime := time.Now()
	fileName := filepath.Base(filePath)

	// 创建处理信息
	processInfo := FileProcessInfo{
		FileName:       fileName,
		FilePath:       filePath,
		ProcessingTime: time.Since(startTime),
	}

	// 检查文件是否存在(带重试机制,应对并发/文件系统延迟问题)
	// 增加重试次数和延迟,彻底解决文件系统缓存/NFS延迟问题
	maxRetries := 5
	var statErr error
	for retry := 0; retry < maxRetries; retry++ {
		if _, statErr = os.Stat(filePath); statErr == nil {
			break
		}
		if retry < maxRetries-1 {
			// 指数退避: 200ms, 400ms, 800ms, 1600ms
			delay := time.Duration(200*(1<<retry)) * time.Millisecond
			logger.Printf("⚠️  文件暂时不可访问,等待%v后重试(%d/%d): %s", delay, retry+1, maxRetries, fileName)
			time.Sleep(delay)
		}
	}
	if statErr != nil {
		logger.Printf("❌ 致命错误: 文件在%d次重试后仍不存在: %s", maxRetries, filePath)
		processInfo.ErrorMsg = fmt.Sprintf("文件不存在(重试%d次,路径: %s)", maxRetries, filePath)
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addDetailedLog(processInfo)
		stats.addOtherSkipped()
		return ""
	}

	// 检测文件类型
	enhancedType, err := utils.DetectFileType(filePath)
	if err != nil {
		logger.Printf("⏭️  文件类型检测失败: %s (路径: %s) - %v", fileName, filePath, err)
		processInfo.ErrorMsg = fmt.Sprintf("文件类型检测失败: %v", err)
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addDetailedLog(processInfo)
		stats.addOtherSkipped()
		return ""
	}

	processInfo.FileType = enhancedType.Extension
	processInfo.IsAnimated = enhancedType.IsAnimated

	// 检查是否为支持的格式
	if !opts.IsSupportedInputFormat(filePath) {
		if enhancedType.IsVideo {
			logger.Printf("🎬 跳过视频文件: %s (类型: %s, 路径: %s)", fileName, enhancedType.Extension, filePath)
			processInfo.ErrorMsg = fmt.Sprintf("视频文件 (类型: %s)", enhancedType.Extension)
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			stats.addVideoSkipped()
		} else {
			logger.Printf("📄 跳过非支持文件: %s (类型: %s, 路径: %s)", fileName, enhancedType.Extension, filePath)
			processInfo.ErrorMsg = fmt.Sprintf("非支持文件 (类型: %s)", enhancedType.Extension)
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			stats.addOtherSkipped()
		}
		return ""
	}

	logger.Printf("✅ 识别为支持格式: %s (%s)", fileName, enhancedType.Extension)

	// 试运行模式
	if opts.DryRun {
		logger.Printf("🔍 试运行模式: 将转换 %s", fileName)
		processInfo.Success = true
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addDetailedLog(processInfo)
		stats.addProcessed()
		return ""
	}

	// 执行转换
	conversionMode, outputPath, err := convertFile(filePath, opts, enhancedType)
	if err != nil {
		logger.Printf("❌ 转换失败 %s: %v", fileName, err)
		processInfo.ErrorMsg = fmt.Sprintf("转换失败: %v", err)
		processInfo.ProcessingTime = time.Since(startTime)
		stats.addDetailedLog(processInfo)
		stats.addFailed()
		return ""
	}

	processInfo.ConversionMode = conversionMode

	// 验证转换结果
	if opts.StrictMode {
		validator := utils.NewEightLayerValidator(utils.ValidationOptions{
			TimeoutSeconds: opts.TimeoutSeconds,
			CJXLThreads:    opts.CJXLThreads,
			StrictMode:     opts.StrictMode,
			AllowTolerance: opts.AllowTolerance,
		})

		result, err := validator.ValidateConversion(filePath, outputPath, enhancedType)
		if err != nil {
			logger.Printf("❌ 验证失败 %s: %v", fileName, err)
			processInfo.ErrorMsg = fmt.Sprintf("验证失败: %v", err)
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			stats.addFailed()
			os.Remove(outputPath)
			return ""
		}

		if !result.Success {
			logger.Printf("❌ 验证失败 %s: %s (第%d层: %s)", fileName, result.Message, result.Layer, result.LayerName)
			processInfo.ErrorMsg = fmt.Sprintf("验证失败: %s", result.Message)
			processInfo.ProcessingTime = time.Since(startTime)
			stats.addDetailedLog(processInfo)
			stats.addFailed()
			os.Remove(outputPath)
			return ""
		}

		logger.Printf("✅ 验证通过: %s (%s)", fileName, result.Message)
	}

	// 复制元数据（文件内部+文件系统）- 最佳执行顺序
	if opts.CopyMetadata {
		// ✅ 步骤1: 捕获源文件的文件系统元数据（在exiftool之前）
		srcInfo, _ := os.Stat(filePath)
		var creationTime time.Time
		if srcInfo != nil {
			if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
				creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
			}
		}

		// ✅ 步骤2: 复制文件内部元数据（EXIF/XMP）- 会改变文件修改时间
		if err := copyMetadata(filePath, outputPath); err != nil {
			logger.Printf("⚠️  EXIF元数据复制失败 %s (非致命): %v", fileName, err)
		} else {
			logger.Printf("✅ EXIF元数据复制成功: %s", fileName)
		}

		// ✅ 步骤3: 恢复Finder扩展属性（标签、注释）
		if srcInfo != nil {
			if err := copyFinderMetadata(filePath, outputPath); err != nil {
				logger.Printf("⚠️  Finder元数据复制失败 %s: %v", fileName, err)
			} else {
				logger.Printf("✅ Finder元数据复制成功: %s", fileName)
			}
		}

		// ✅ 步骤4: 最后恢复文件系统时间戳（覆盖exiftool的修改）
		if srcInfo != nil && !creationTime.IsZero() {
			timeStr := creationTime.Format("200601021504.05")
			touchCmd := exec.Command("touch", "-t", timeStr, outputPath)
			if err := touchCmd.Run(); err != nil {
				logger.Printf("⚠️  文件时间恢复失败 %s: %v", fileName, err)
			} else {
				logger.Printf("✅ 文件系统元数据已保留: %s (创建/修改: %s)",
					fileName, creationTime.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// 获取文件大小
	originalInfo, _ := os.Stat(filePath)
	outputInfo, _ := os.Stat(outputPath)
	processInfo.SizeBefore = originalInfo.Size()
	processInfo.SizeAfter = outputInfo.Size()

	// 删除原始文件
	if err := utils.SafeDelete(filePath, outputPath, logger.Printf); err != nil {
		logger.Printf("⚠️  删除原始文件失败 %s: %v", fileName, err)
	} else {
		logger.Printf("🗑️  已安全删除原始文件: %s", fileName)
	}

	// 更新统计
	processInfo.Success = true
	processInfo.ProcessingTime = time.Since(startTime)
	stats.addDetailedLog(processInfo)
	stats.addProcessed()
	stats.addImageProcessed(processInfo.SizeBefore, processInfo.SizeAfter)
	stats.addByExt(enhancedType.Extension)

	logger.Printf("🎉 处理成功: %s", fileName)

	return outputPath
}

// convertFile 转换文件
func convertFile(filePath string, opts utils.UniversalOptions, fileType utils.EnhancedFileType) (string, string, error) {
	// 生成输出路径
	ext := opts.GetOutputExtensionForFile(filePath)
	outputPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ext

	// 对于 AVIF/HEIC/HEIF → JXL 转换，需要先转换为中间格式
	actualInputPath := filePath
	if opts.ConversionType == utils.ConvertToJXL {
		inputExt := strings.ToLower(filepath.Ext(filePath))
		if inputExt == ".avif" || inputExt == ".heic" || inputExt == ".heif" {
			// 创建唯一的临时文件基础名（避免特殊字符和并发冲突）
			tempBase := filepath.Join(os.TempDir(), fmt.Sprintf("conv_%d", time.Now().UnixNano()))
			tempPath, err := utils.ToPNGOrTIFF(filePath, tempBase, false)
			if err != nil {
				return "", "", fmt.Errorf("预处理 %s 失败: %v", inputExt, err)
			}
			defer os.Remove(tempPath)

			// 使用临时文件作为输入
			actualInputPath = tempPath
		}
	}

	// 获取转换命令
	cmdName, args, err := opts.GetConversionCommand(actualInputPath, outputPath)
	if err != nil {
		return "", "", err
	}

	// 智能超时机制: 根据文件大小动态调整超时时间
	timeout := getSmartTimeout(filePath, opts.TimeoutSeconds)
	ctx, cancel := context.WithTimeout(globalCtx, timeout)
	defer cancel()

	procSem <- struct{}{}
	defer func() { <-procSem }()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("转换命令执行失败: %v\n输出: %s", err, string(output))
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outputPath); err != nil {
		return "", "", fmt.Errorf("输出文件未生成: %v", err)
	}

	return fmt.Sprintf("%s转换", strings.ToUpper(ext[1:])), outputPath, nil
}

// getSmartTimeout 根据文件大小智能计算超时时间
// 避免大文件因超时被杀死
func getSmartTimeout(filePath string, baseTimeout int) time.Duration {
	stat, err := os.Stat(filePath)
	if err != nil {
		return time.Duration(baseTimeout) * time.Second
	}

	fileSize := stat.Size()

	// 动态超时策略:
	// < 500KB:   30秒
	// < 2MB:     60秒
	// < 5MB:     120秒 (2分钟)
	// < 10MB:    300秒 (5分钟)
	// >= 10MB:   600秒 (10分钟)
	var timeout time.Duration
	switch {
	case fileSize < 500*1024:
		timeout = 30 * time.Second
	case fileSize < 2*1024*1024:
		timeout = 60 * time.Second
	case fileSize < 5*1024*1024:
		timeout = 120 * time.Second
	case fileSize < 10*1024*1024:
		timeout = 300 * time.Second
	default:
		timeout = 600 * time.Second
	}

	// 至少使用基础超时时间
	baseTimeoutDuration := time.Duration(baseTimeout) * time.Second
	if timeout < baseTimeoutDuration {
		timeout = baseTimeoutDuration
	}

	return timeout
}

// copyMetadata 复制元数据（EXIF/XMP）
func copyMetadata(originalPath, outputPath string) error {
	ctx, cancel := context.WithTimeout(globalCtx, 30*time.Second)
	defer cancel()

	// 使用exiftool复制元数据
	cmd := exec.CommandContext(ctx, "exiftool", "-overwrite_original", "-TagsFromFile", originalPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool执行失败: %v\n输出: %s", err, string(output))
	}

	return nil
}

// copyFinderMetadata 复制Finder标签和注释
func copyFinderMetadata(src, dst string) error {
	// 复制Finder标签
	cmd := exec.Command("xattr", "-p", "com.apple.metadata:_kMDItemUserTags", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", string(output), dst).Run()
	}

	// 复制Finder注释
	cmd = exec.Command("xattr", "-p", "com.apple.metadata:kMDItemFinderComment", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:kMDItemFinderComment", string(output), dst).Run()
	}

	// 复制其他扩展属性
	cmd = exec.Command("xattr", src)
	if output, err := cmd.CombinedOutput(); err == nil {
		attrs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, attr := range attrs {
			if attr != "" && !strings.Contains(attr, "com.apple.metadata:_kMDItemUserTags") &&
				!strings.Contains(attr, "com.apple.metadata:kMDItemFinderComment") {
				cmd = exec.Command("xattr", "-p", attr, src)
				if value, err := cmd.CombinedOutput(); err == nil && len(value) > 0 {
					exec.Command("xattr", "-w", attr, string(value), dst).Run()
				}
			}
		}
	}

	return nil
}

// setupSignalHandling 设置信号处理
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Printf("🛑 收到信号 %v，正在优雅关闭...", sig)
		if cancelFunc != nil {
			cancelFunc()
		}
	}()
}

// printStatistics 打印统计信息
func printStatistics() {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	totalTime := time.Since(stats.startTime)
	avgTime := time.Duration(0)
	if stats.processed > 0 {
		avgTime = totalTime / time.Duration(stats.processed)
	}

	logger.Printf("⏱️  总处理时间: %v", totalTime)
	logger.Printf("📈 平均处理时间: %v", avgTime)
	logger.Printf("✅ 成功处理: %d", stats.processed)
	logger.Printf("❌ 转换失败: %d", stats.failed)
	logger.Printf("🎬 跳过视频: %d", stats.videoSkipped)
	logger.Printf("📄 跳过其他: %d", stats.otherSkipped)

	if stats.totalSizeBefore > 0 {
		saved := stats.totalSizeBefore - stats.totalSizeAfter
		ratio := float64(stats.totalSizeAfter) / float64(stats.totalSizeBefore) * 100
		logger.Printf("📊 大小变化: %s -> %s (节省: %s, 压缩率: %.1f%%)",
			formatBytes(stats.totalSizeBefore),
			formatBytes(stats.totalSizeAfter),
			formatBytes(saved),
			ratio)
	}

	// 按格式统计
	if len(stats.byExt) > 0 {
		logger.Printf("📋 格式统计:")
		for ext, count := range stats.byExt {
			logger.Printf("  %s: %d个文件", ext, count)
		}
	}

	// 处理完成后的整体扫描
	logger.Printf("🔍 正在进行处理完成后的整体扫描...")
	outputFiles, err := scanOutputDirectory()
	if err != nil {
		logger.Printf("⚠️  输出目录扫描失败: %v", err)
	} else {
		logger.Printf("📊 输出目录中共有 %d 个文件", len(outputFiles))
	}
}

// scanOutputDirectory 扫描输出目录统计文件数量
func scanOutputDirectory() ([]string, error) {
	var files []string
	// 扫描处理目录中的所有文件（假设在原地转换）
	err := godirwalk.Walk(".", &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				files = append(files, osPathname)
			}
			return nil
		},
		Unsorted: true,
	})
	return files, err
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// performPostValidation 执行转换后验证
func performPostValidation(pairs []utils.FilePair) {
	if len(pairs) == 0 {
		logger.Println("⚠️  没有文件需要验证")
		return
	}

	// 创建验证器（10%抽样率，最少5个，最多20个）
	validator := utils.NewPostValidator(0.1, 5, 20)

	logger.Printf("📊 总计 %d 个成功转换的文件", len(pairs))

	// 执行验证
	result := validator.ValidateConversions(pairs)

	logger.Println("")
	logger.Printf("📋 验证摘要: %s", result.Summary)
	logger.Println("")

	// 输出详细结果
	for i, item := range result.ValidationItems {
		fileName := filepath.Base(item.OriginalPath)
		convFileName := filepath.Base(item.ConvertedPath)

		if item.Passed {
			logger.Printf("  ✅ [%d/%d] %s → %s (%s)",
				i+1, len(result.ValidationItems),
				fileName, convFileName, item.FileType)
		} else {
			logger.Printf("  ❌ [%d/%d] %s → %s (%s)",
				i+1, len(result.ValidationItems),
				fileName, convFileName, item.FileType)
			for _, issue := range item.Issues {
				logger.Printf("      ⚠️  %s", issue)
			}
		}
	}

	logger.Println("")

	// 最终判断
	passRate := float64(result.PassedFiles) / float64(result.SampledFiles) * 100
	if passRate >= 95.0 {
		logger.Printf("🎉 验证通过！通过率: %.1f%% (%d/%d)",
			passRate, result.PassedFiles, result.SampledFiles)
	} else if passRate >= 80.0 {
		logger.Printf("⚠️  验证警告！通过率: %.1f%% (%d/%d) - 建议检查失败的文件",
			passRate, result.PassedFiles, result.SampledFiles)
	} else {
		logger.Printf("❌ 验证失败！通过率: %.1f%% (%d/%d) - 转换可能存在问题",
			passRate, result.PassedFiles, result.SampledFiles)
	}

	logger.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
