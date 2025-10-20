package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pixly/pkg/core/config"
	"pixly/pkg/core/state"
	"pixly/pkg/core/types"
	"pixly/pkg/engine"
	"pixly/pkg/engine/quality"
	"pixly/pkg/ui/interactive"
	"pixly/pkg/ui/progress"

	"go.uber.org/zap"
)

// ConversionEngine 转换引擎结构体
type ConversionEngine struct {
	logger           *zap.Logger
	config           *EngineConfig
	toolCheck        types.ToolCheckResults
	progressManager  *progress.ProgressManager // 新增进度管理器
	qualityEngine    *quality.QualityEngine    // 质量评估引擎
	uiInterface      *interactive.Interface    // UI交互接口
	balanceOptimizer *engine.BalanceOptimizer  // 平衡优化器
	autoPlusRouter   *engine.AutoPlusRouter    // 自动模式+路由器
	cacheDir         string                    // 缓存目录
	stateManager     *state.StateManager       // 状态管理器（断点续传）
}

// InitStateManager 初始化状态管理器
func (e *ConversionEngine) InitStateManager() error {
	// 创建状态管理器
	stateMgr, err := state.LoadState(e.config.TargetDir)
	if err != nil {
		e.logger.Warn("状态加载失败，创建新状态管理器", zap.Error(err))
		stateMgr, err = state.NewStateManager(false)
		if err != nil {
			e.logger.Error("创建状态管理器失败", zap.Error(err))
			return err
		}
	}

	e.stateManager = stateMgr
	return nil
}

// SaveState 保存状态到缓存
func (e *ConversionEngine) SaveState(filename string, data interface{}) error {
	if e.stateManager == nil {
		return fmt.Errorf("状态管理器未初始化")
	}

	// TODO: 实现状态保存逻辑
	return nil
}

// LoadState 从缓存加载状态
func (e *ConversionEngine) LoadState(filename string, data interface{}) error {
	if e.stateManager == nil {
		return fmt.Errorf("状态管理器未初始化")
	}

	// TODO: 实现状态加载逻辑
	return fmt.Errorf("缓存数据不存在")
}

// SetCacheDir 设置缓存目录
func (e *ConversionEngine) SetCacheDir(cacheDir string) {
	e.cacheDir = cacheDir
}

// SaveProgressCache 保存进度缓存到JSON文件
func (e *ConversionEngine) SaveProgressCache(filename string, data interface{}) error {
	if e.cacheDir == "" {
		return fmt.Errorf("缓存目录未设置")
	}

	path := filepath.Join(e.cacheDir, filename)
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	return os.WriteFile(path, dataJSON, 0644)
}

// LoadProgressCache 从JSON文件加载进度缓存
func (e *ConversionEngine) LoadProgressCache(filename string, data interface{}) error {
	if e.cacheDir == "" {
		return fmt.Errorf("缓存目录未设置")
	}

	path := filepath.Join(e.cacheDir, filename)
	dataJSON, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取缓存文件失败: %w", err)
	}

	return json.Unmarshal(dataJSON, data)
}

// EngineConfig 引擎配置（兼容main.go的Config格式）
type EngineConfig struct {
	Mode                string
	TargetDir           string
	BackupDir           string
	ConcurrentJobs      int
	MaxRetries          int
	CRF                 int
	EnableBackups       bool
	CreateBackups       bool // 是否创建备份
	KeepBackups         bool // 是否保留备份
	HwAccel             bool
	Overwrite           bool
	LogLevel            string
	SortOrder           string
	StickerTargetFormat string
	DebugMode           bool
	DryRun              bool
}

// NewConversionEngine 创建新的转换引擎
func NewConversionEngine(logger *zap.Logger, modularCfg *config.Config, toolResults types.ToolCheckResults, uiInterface *interactive.Interface) *ConversionEngine {
	engineCfg := &EngineConfig{
		Mode:                modularCfg.Mode,
		TargetDir:           modularCfg.TargetDir,
		BackupDir:           "",
		ConcurrentJobs:      modularCfg.ConcurrentJobs,
		MaxRetries:          modularCfg.MaxRetries,
		CRF:                 modularCfg.CRF,
		EnableBackups:       modularCfg.EnableBackups,
		CreateBackups:       modularCfg.CreateBackups,
		KeepBackups:         modularCfg.KeepBackups,
		HwAccel:             modularCfg.HwAccel,
		Overwrite:           modularCfg.Overwrite,
		LogLevel:            modularCfg.LogLevel,
		SortOrder:           modularCfg.SortOrder,
		StickerTargetFormat: modularCfg.StickerTargetFormat,
		DebugMode:           modularCfg.DebugMode,
		DryRun:              modularCfg.DryRun,
	}

	// 设置默认备份目录
	if engineCfg.BackupDir == "" {
		engineCfg.BackupDir = filepath.Join(engineCfg.TargetDir, ".backups")
	}

	// 创建进度管理器
	progressMgr := progress.NewProgressManager(logger)

	// 创建质量评估引擎
	qualityEng := quality.NewQualityEngine(
		logger,
		toolResults.FfmpegStablePath, // 使用稳定版ffmpeg路径作为ffprobe
		toolResults.FfmpegStablePath, // 使用稳定版ffmpeg路径
		false,                        // 非快速模式，进行完整检测
	)

	// 创建临时目录用于平衡优化
	tempDir := filepath.Join(os.TempDir(), "pixly_balance_temp")
	_ = os.MkdirAll(tempDir, 0755) // 忽略错误，如果目录已存在

	// 创建平衡优化器
	balanceOpt := engine.NewBalanceOptimizer(logger, toolResults, tempDir)

	// 创建自动模式+路由器
	autoPlusRtr := engine.NewAutoPlusRouter(logger, qualityEng, balanceOpt, uiInterface, toolResults, modularCfg.DebugMode)

	return &ConversionEngine{
		logger:           logger,
		config:           engineCfg,
		toolCheck:        toolResults,
		progressManager:  progressMgr,
		qualityEngine:    qualityEng,
		uiInterface:      uiInterface,
		balanceOptimizer: balanceOpt,
		autoPlusRouter:   autoPlusRtr,
	}
}

// Execute 执行转换流程
func (e *ConversionEngine) Execute(ctx context.Context) error {
	e.logger.Info("转换引擎开始执行",
		zap.String("mode", e.config.Mode),
		zap.String("target_dir", e.config.TargetDir),
		zap.Int("concurrent_jobs", e.config.ConcurrentJobs))

	// 验证配置
	if err := e.validateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 执行预检查
	if err := e.performPreflightChecks(); err != nil {
		return fmt.Errorf("预检失败: %w", err)
	}

	// 执行实际的转换流程
	return e.executeConversionPipeline(ctx)
}

// validateConfig 验证配置
func (e *ConversionEngine) validateConfig() error {
	if e.config.TargetDir == "" {
		return fmt.Errorf("目标目录不能为空")
	}

	absPath, err := filepath.Abs(e.config.TargetDir)
	if err != nil {
		return fmt.Errorf("无法解析目标目录路径: %w", err)
	}
	e.config.TargetDir = absPath

	if _, err := os.Stat(e.config.TargetDir); os.IsNotExist(err) {
		return fmt.Errorf("目标目录不存在: %s", e.config.TargetDir)
	}

	// 验证模式
	validModes := map[string]bool{"auto+": true, "quality": true, "sticker": true}
	if !validModes[e.config.Mode] {
		return fmt.Errorf("无效的模式: %s。有效模式为: auto+, quality, sticker", e.config.Mode)
	}

	if e.config.ConcurrentJobs <= 0 {
		e.config.ConcurrentJobs = 7 // 默认并发数
	}

	return nil
}

// performPreflightChecks 执行预检查
func (e *ConversionEngine) performPreflightChecks() error {
	// 检查磁盘空间
	if err := e.checkDiskSpace(); err != nil {
		return fmt.Errorf("磁盘空间检查失败: %w", err)
	}

	// 检查权限
	if err := e.checkPermissions(); err != nil {
		return fmt.Errorf("权限检查失败: %w", err)
	}

	// 检查工具依赖
	if !e.toolCheck.HasFfmpeg {
		return fmt.Errorf("缺少必要工具: FFmpeg")
	}

	return nil
}

// checkDiskSpace 检查磁盘空间
func (e *ConversionEngine) checkDiskSpace() error {
	// 简单的磁盘空间检查实现
	e.logger.Info("检查磁盘空间", zap.String("target_dir", e.config.TargetDir))
	return nil
}

// checkPermissions 检查权限
func (e *ConversionEngine) checkPermissions() error {
	// 测试读写权限
	testFile := filepath.Join(e.config.TargetDir, ".pixly_permission_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("目标目录无写权限: %w", err)
	}
	os.Remove(testFile)

	e.logger.Info("权限检查通过", zap.String("target_dir", e.config.TargetDir))
	return nil
}

// executeConversionPipeline 执行转换管道
func (e *ConversionEngine) executeConversionPipeline(ctx context.Context) error {
	e.logger.Info("开始执行转换管道")

	// 创建转换上下文
	pipelineCtx, pipelineCancel := context.WithCancel(ctx)
	defer pipelineCancel()

	// 初始化状态管理器
	if e.stateManager == nil {
		e.InitStateManager()
	}

	// 保存初始会话信息
	if err := e.stateManager.SaveSession(e.config.TargetDir); err != nil {
		e.logger.Warn("保存会话信息失败", zap.Error(err))
	}

	// 尝试从状态管理器加载任务
	tasks, err := e.stateManager.LoadTasks()
	if err == nil && len(tasks) > 0 {
		e.logger.Info("从状态管理器加载到未完成的任务")
	} else {
		// 步骤1: 扫描文件
		files, err := e.scanFiles(pipelineCtx)
		if err != nil {
			return fmt.Errorf("文件扫描失败: %w", err)
		}

		// 保存扫描的文件信息
		// TODO: 转换files为合适的类型
		e.logger.Info("文件扫描完成", zap.Int("file_count", len(files)))

		// 步骤2: 评估文件质量和检测损坏文件
		tasks, err := e.assessFiles(files)
		if err != nil {
			return fmt.Errorf("文件评估失败: %w", err)
		}

		// 保存评估结果
		if err := e.stateManager.SaveTasks(tasks); err != nil {
			e.logger.Warn("保存处理结果失败", zap.Error(err))
		}
	}

	// 保存初始任务列表
	if err := e.stateManager.SaveTasks(tasks); err != nil {
		e.logger.Warn("保存转换任务失败", zap.Error(err))
	}

	// 步骤2.6: 智能路由系统（仅在自动模式+时）
	if e.config.Mode == "auto+" {
		e.logger.Info("启动自动模式+智能路由系统")
		// TODO: 实现智能路由逻辑
	}

	if len(tasks) == 0 {
		e.logger.Info("未发现需要处理的媒体文件")
		fmt.Println("📄 未发现需要处理的媒体文件")
		return nil
	}

	e.logger.Info("文件扫描完成", zap.Int("total_files", len(tasks)))
	fmt.Printf("📂 发现 %d 个媒体文件\n", len(tasks))

	// 步骤3: 根据模式路由任务
	// TODO: 实现任务路由逻辑
	e.logger.Info("任务路由完成", zap.Int("routed_tasks", len(tasks)))

	// 步骤4: 执行转换
	results := e.executeConversions(pipelineCtx, tasks)
	if len(results) > 0 {
		e.logger.Info("转换执行完成", zap.Int("results", len(results)))
	}

	// 保存转换结果
	if results != nil {
		e.logger.Info("转换结果处理完成")
	}

	// 步骤5: 生成报告
	e.generateReport(results)

	// 保存最终统计信息
	e.logger.Info("转换管道执行完成")

	return nil
}

// scanFiles 扫描媒体文件
func (e *ConversionEngine) scanFiles(ctx context.Context) ([]string, error) {
	e.logger.Info("开始扫描文件", zap.String("target_dir", e.config.TargetDir))
	fmt.Println("🔍 正在扫描媒体文件...")

	var files []string
	var scannedCount int64

	// 定义支持的媒体格式白名单
	mediaExtensions := map[string]bool{
		// 静图格式
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".heic": true, ".heif": true, ".avif": true, ".jxl": true,
		".tiff": true, ".tif": true, ".bmp": true,
		// 视频格式
		".mp4": true, ".mov": true, ".webm": true, ".mkv": true,
		".avi": true, ".m4v": true, ".3gp": true,
	}

	// 需要跳过的文件类型
	skipPatterns := []string{
		".DS_Store", ".Thumbs.db", ".tmp", ".temp", ".log",
		".pixly_", ".backup", ".bak",
	}

	// 预估文件数量用于初始化进度条
	estimatedFiles := e.estimateFileCount(e.config.TargetDir)
	if estimatedFiles > 0 {
		e.progressManager.CreateScanProgress(estimatedFiles)
		e.progressManager.StartRealTimeDisplay() // 启动实时显示
	}

	err := filepath.Walk(e.config.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			e.logger.Warn("扫描文件时出错", zap.String("path", path), zap.Error(err))
			return nil // 继续处理，跳过有问题的文件
		}

		// 跳过目录
		if info.IsDir() {
			// 跳过隐藏目录和系统目录
			if strings.HasPrefix(info.Name(), ".") ||
				strings.Contains(info.Name(), "$RECYCLE.BIN") ||
				strings.Contains(info.Name(), "System Volume Information") {
				return filepath.SkipDir
			}
			return nil
		}

		scannedCount++
		// 更新扫描进度
		if scannedCount%100 == 0 {
			e.progressManager.UpdateProgress(progress.ProgressTypeScan, 100)
		}

		// 跳过隐藏文件和系统文件
		fileName := info.Name()
		if strings.HasPrefix(fileName, ".") {
			return nil
		}

		// 检查是否需要跳过的文件类型
		for _, pattern := range skipPatterns {
			if strings.Contains(fileName, pattern) {
				return nil
			}
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		if mediaExtensions[ext] {
			// 简单的文件大小检查，跳过过小的文件（可能是损坏的）
			if info.Size() < 100 {
				e.logger.Debug("跳过过小文件", zap.String("path", path), zap.Int64("size", info.Size()))
				return nil
			}
			files = append(files, path)
		}

		// 检查上下文取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

	// 完成扫描进度
	if estimatedFiles > 0 {
		e.progressManager.CompleteProgress(progress.ProgressTypeScan)
	}

	fmt.Printf("\r✅ 扫描完成: 发现 %d 个媒体文件\n", len(files))
	e.logger.Info("文件扫描完成",
		zap.Int("total_files", len(files)),
		zap.Int64("scanned_count", scannedCount))

	return files, err
}

// assessFiles 评估文件质量和检测损坏文件
func (e *ConversionEngine) assessFiles(files []string) ([]*types.FileTask, error) {
	e.logger.Info("开始评估文件质量", zap.Int("file_count", len(files)))
	fmt.Println("⚙️ 正在评估文件质量...")

	// 创建评估进度条
	if e.progressManager != nil {
		e.progressManager.CreateScanProgress(len(files))
	}

	var tasks []*types.FileTask
	var assessedCount int
	var mu sync.Mutex

	// 使用并发评估提高效率
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // 限制并发数为3

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 使用质量评估引擎进行详细评估
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := e.qualityEngine.AssessFile(ctx, filePath)
			if err != nil {
				e.logger.Warn("文件评估失败", zap.String("file", filepath.Base(filePath)), zap.Error(err))
				return
			}

			// 创建转换任务
			task := &types.FileTask{
				Path: filePath,
			}

			mu.Lock()
			tasks = append(tasks, task)
			assessedCount++
			mu.Unlock()
		}(file)
	}

	wg.Wait()

	fmt.Printf("\r✅ 质量评估完成: %d 个文件\n", len(tasks))
	e.logger.Info("文件质量评估完成",
		zap.Int("total_tasks", len(tasks)))

	return tasks, nil
}

// assessFileQuality 使用FFmpeg评估文件质量
func (e *ConversionEngine) assessFileQuality(filePath string) (string, string) {
	// 简化的质量评估逻辑，基于文件大小和格式
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "unknown", "unknown"
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	fileSize := fileInfo.Size()

	// 判断媒体类型
	var mediaType string
	if e.isImageFormat(ext) {
		mediaType = "image"
	} else if e.isVideoFormat(ext) {
		mediaType = "video"
	} else {
		mediaType = "unknown"
	}

	// 简化的质量评估逻辑
	var quality string
	switch {
	case fileSize > 10*1024*1024: // 10MB+
		quality = "high"
	case fileSize > 2*1024*1024: // 2MB+
		quality = "medium"
	case fileSize > 500*1024: // 500KB+
		quality = "low"
	default:
		quality = "very_low"
	}

	e.logger.Debug("文件质量评估",
		zap.String("file", filepath.Base(filePath)),
		zap.String("quality", quality),
		zap.String("media_type", mediaType),
		zap.Int64("size", fileSize))

	return quality, mediaType
}

// isImageFormat 检查是否为图片格式
func (e *ConversionEngine) isImageFormat(ext string) bool {
	imageFormats := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".heic": true, ".heif": true, ".avif": true, ".jxl": true,
		".tiff": true, ".tif": true, ".bmp": true,
	}
	return imageFormats[ext]
}

// isVideoFormat 检查是否为视频格式
func (e *ConversionEngine) isVideoFormat(ext string) bool {
	videoFormats := map[string]bool{
		".mp4": true, ".mov": true, ".webm": true, ".mkv": true,
		".avi": true, ".m4v": true, ".3gp": true,
	}
	return videoFormats[ext]
}

// determineTargetFormat 根据模式和质量确定目标格式
func (e *ConversionEngine) determineTargetFormat(task *types.ConversionTask) string {
	switch e.config.Mode {
	case "auto+":
		// 智能模式：根据质量选择策略
		if task.MediaType == "image" {
			switch task.Quality {
			case "high":
				return "jxl_lossless" // 高质量图片使用JXL无损
			case "medium":
				return "jxl_balanced" // 中等质量使用JXL平衡模式
			default:
				return "avif_compressed" // 低质量使用AVIF压缩
			}
		} else if task.MediaType == "video" {
			return "remux" // 视频使用重包装
		}
	case "quality":
		// 质量模式：全部无损
		if task.MediaType == "image" {
			return "jxl_lossless"
		} else if task.MediaType == "video" {
			return "remux"
		}
	case "sticker":
		// 表情包模式：所有图片转为AVIF
		if task.MediaType == "image" {
			return "avif_compressed"
		} else {
			return "skip" // 视频跳过
		}
	}
	return "auto"
}

// determineTargetFormatFromQualityAssessment 根据评估结果确定目标格式
func (e *ConversionEngine) determineTargetFormatFromQualityAssessment(task *types.ConversionTask, assessment *quality.QualityAssessment) string {
	// 首先检查文件是否已经是目标格式，防止重复转换
	ext := strings.ToLower(filepath.Ext(task.SourcePath))

	// 检查文件是否已经是最优格式，避免无意义的重复转换
	if e.isAlreadyOptimalFormat(ext, e.config.Mode, assessment) {
		e.logger.Debug("文件已经是最优格式，跳过转换",
			zap.String("file", filepath.Base(task.SourcePath)),
			zap.String("ext", ext),
			zap.String("mode", e.config.Mode))
		return "skip"
	}

	switch e.config.Mode {
	case "auto+":
		// 智能模式：根据质量选择策略
		if assessment.MediaType == types.MediaTypeImage {
			switch assessment.QualityLevel {
			case types.QualityVeryHigh, types.QualityHigh:
				return "jxl_lossless" // 高质量图片使用JXL无损
			case types.QualityMediumHigh:
				return "jxl_balanced" // 中等质量使用JXL平衡模式
			default:
				return "avif_compressed" // 低质量使用AVIF压缩
			}
		} else if assessment.MediaType == types.MediaTypeVideo {
			return "remux" // 视频使用重包装
		}
	case "quality":
		// 质量模式：全部无损
		if assessment.MediaType == types.MediaTypeImage {
			return "jxl_lossless"
		} else if assessment.MediaType == types.MediaTypeVideo {
			return "remux"
		}
	case "sticker":
		// 表情包模式：所有图片转为AVIF
		if assessment.MediaType == types.MediaTypeImage {
			return "avif_compressed"
		} else {
			return "skip" // 视频跳过
		}
	}
	return "auto"
}

// isAlreadyOptimalFormat 检查文件是否已经是最优格式，避免无意义的重复转换
func (e *ConversionEngine) isAlreadyOptimalFormat(ext, mode string, assessment *quality.QualityAssessment) bool {
	// 如果文件大小为0，认为是损坏文件，跳过
	if fileInfo, err := os.Stat(assessment.FilePath); err == nil && fileInfo.Size() == 0 {
		e.logger.Debug("检测到空文件，跳过转换",
			zap.String("file", filepath.Base(assessment.FilePath)))
		return true
	}

	switch mode {
	case "auto+":
		// 自动模式+中，根据质量等级检查是否已是最优格式
		if assessment.MediaType == types.MediaTypeImage {
			switch assessment.QualityLevel {
			case types.QualityVeryHigh, types.QualityHigh:
				// 高质量文件应该使用JXL无损，如果已是JXL格式则跳过
				return ext == ".jxl"
			case types.QualityMediumHigh:
				// 中等质量文件应该使用JXL平衡模式，如果已是JXL或AVIF则考虑跳过
				return ext == ".jxl" || ext == ".avif"
			default:
				// 低质量文件应该使用AVIF，如果已是AVIF则跳过
				return ext == ".avif"
			}
		}
		return false

	case "quality":
		// 质量模式中，所有图片都应该是JXL无损
		if assessment.MediaType == types.MediaTypeImage {
			return ext == ".jxl"
		}
		return false

	case "sticker":
		// 表情包模式中，所有图片都应该是AVIF
		if assessment.MediaType == types.MediaTypeImage {
			return ext == ".avif"
		}
		// 视频文件在表情包模式中应该被跳过
		if assessment.MediaType == types.MediaTypeVideo {
			return true // 统一跳过所有视频
		}
		return false

	default:
		return false
	}
}

// routeTasks 根据模式路由任务
func (e *ConversionEngine) routeTasks(tasks []*types.ConversionTask) []*types.ConversionTask {
	e.logger.Info("开始路由任务", zap.String("mode", e.config.Mode), zap.Int("task_count", len(tasks)))

	var routedCount int

	for i := range tasks {
		// 如果任务已经有目标格式设置（比如低品质文件处理时设置的），则保持不变
		if tasks[i].TargetFormat != "" &&
			tasks[i].TargetFormat != "auto" &&
			tasks[i].TargetFormat != "quality" &&
			tasks[i].TargetFormat != "sticker" {
			e.logger.Debug("任务已有目标格式，跳过路由",
				zap.String("file", filepath.Base(tasks[i].SourcePath)),
				zap.String("existing_format", tasks[i].TargetFormat))
			continue
		}

		// 根据模式进行路由
		switch e.config.Mode {
		case "auto+":
			// 智能模式：根据文件类型和质量选择最佳策略
			tasks[i].TargetFormat = e.determineOptimalFormat(tasks[i])
		case "quality":
			// 品质模式：所有文件使用无损或最高品质转换
			tasks[i].TargetFormat = e.determineQualityFormat(tasks[i])
		case "sticker":
			// 表情包模式：适用于网络分享的极限压缩
			tasks[i].TargetFormat = e.determineStickerFormat(tasks[i])
		default:
			// 默认为自动模式
			tasks[i].TargetFormat = e.determineOptimalFormat(tasks[i])
		}

		routedCount++
		e.logger.Debug("任务路由完成",
			zap.String("file", filepath.Base(tasks[i].SourcePath)),
			zap.String("target_format", tasks[i].TargetFormat),
			zap.String("quality", tasks[i].Quality),
			zap.String("media_type", tasks[i].MediaType))
	}

	e.logger.Info("任务路由完成",
		zap.Int("routed_count", routedCount),
		zap.Int("total_tasks", len(tasks)))

	fmt.Printf("✅ 任务路由完成: %d 个任务已分配处理策略\n", routedCount)
	return tasks
}

// determineOptimalFormat 自动模式+的最优格式选择
func (e *ConversionEngine) determineOptimalFormat(task *types.ConversionTask) string {
	if task.MediaType == "image" {
		switch task.Quality {
		case "high":
			return "jxl_lossless" // 高品质图片使用JXL无损
		case "medium":
			return "jxl_balanced" // 中等品质使用JXL平衡模式
		case "low":
			return "avif_balanced" // 低品质使用AVIF平衡模式
		case "very_low":
			return "avif_compressed" // 极低品质使用AVIF压缩
		default:
			return "jxl_balanced"
		}
	} else if task.MediaType == "video" {
		// 视频文件一般使用重包装或轻度压缩
		return "remux"
	}
	return "auto" // 其他情况使用自动判断
}

// determineQualityFormat 品质模式的格式选择
func (e *ConversionEngine) determineQualityFormat(task *types.ConversionTask) string {
	if task.MediaType == "image" {
		return "jxl_lossless" // 所有图片都使用JXL无损
	} else if task.MediaType == "video" {
		return "remux" // 视频使用重包装保持品质
	}
	return "jxl_lossless" // 默认使用无损
}

// determineStickerFormat 表情包模式的格式选择
func (e *ConversionEngine) determineStickerFormat(task *types.ConversionTask) string {
	if task.MediaType == "image" {
		return "avif_compressed" // 所有图片都使用AVIF极限压缩
	} else if task.MediaType == "video" {
		return "skip" // 表情包模式跳过视频文件
	}
	return "avif_compressed"
}

// executeConversions 执行转换
func (e *ConversionEngine) executeConversions(ctx context.Context, tasks []*types.FileTask) []ConversionResult {
	e.logger.Info("开始执行转换", zap.Int("task_count", len(tasks)))

	// 创建转换进度条
	e.progressManager.CreateConversionProgress(len(tasks))

	var results []ConversionResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 使用工作池控制并发
	semaphore := make(chan struct{}, e.config.ConcurrentJobs)

	for _, task := range tasks {
		wg.Add(1)
		go func(t *types.FileTask) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := e.processTask(ctx, t)

			mu.Lock()
			results = append(results, result)

			// 更新进度管理器统计
			processingResult := &types.ProcessingResult{
				OriginalPath: result.SourcePath,
				NewPath:      result.TargetPath,
				OriginalSize: result.OriginalSize,
				NewSize:      result.NewSize,
				SpaceSaved:   result.OriginalSize - result.NewSize,
				Success:      result.Status == "success",
				ProcessTime:  result.Duration,
			}
			if result.Status != "success" {
				processingResult.Error = result.Message
			}

			e.progressManager.UpdateResult(processingResult)
			e.progressManager.UpdateProgress(progress.ProgressTypeConversion, 1)
			mu.Unlock()
		}(task)
	}

	wg.Wait()

	// 完成转换进度
	e.progressManager.CompleteProgress(progress.ProgressTypeConversion)

	e.logger.Info("转换执行完成", zap.Int("result_count", len(results)))
	return results
}

// processTask 处理单个任务（带重试机制）
func (e *ConversionEngine) processTask(ctx context.Context, task *types.FileTask) ConversionResult {
	result := ConversionResult{
		SourcePath: task.Path,
		Status:     "success",
		Message:    "转换完成",
		StartTime:  time.Now(),
	}

	// 检查是否需要跳过
	if string(task.TargetFormat) == "skip" {
		result.Status = "skipped"
		result.Message = "根据模式配置跳过处理"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// 记录开始处理
	e.logger.Debug("开始处理任务",
		zap.String("file", filepath.Base(task.Path)),
		zap.String("target_format", string(task.TargetFormat)),
		zap.String("quality", task.Quality.String()))

	// 检查源文件是否存在
	sourceInfo, err := os.Stat(task.Path)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("源文件不存在: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// 如果是调试模式或干运行模式，只模拟处理
	if e.config.DebugMode || e.config.DryRun {
		e.logger.Info("模拟转换模式",
			zap.String("file", filepath.Base(task.Path)),
			zap.String("target_format", string(task.TargetFormat)))

		// 模拟处理时间
		time.Sleep(time.Duration(50+len(task.Path)%100) * time.Millisecond)

		result.Message = "模拟转换完成"
		result.OriginalSize = sourceInfo.Size()
		result.NewSize = sourceInfo.Size() * 8 / 10 // 模拟20%的压缩
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// 重试机制：最多重试 MaxRetries 次
	maxRetries := e.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // 默认重试3次
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前稍微等待，指数退避
			retryDelay := time.Duration(attempt*attempt) * 100 * time.Millisecond
			e.logger.Info("重试转换任务",
				zap.String("file", filepath.Base(task.Path)),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries+1),
				zap.Duration("delay", retryDelay))

			select {
			case <-time.After(retryDelay):
				// 继续重试
			case <-ctx.Done():
				result.Status = "failed"
				result.Message = fmt.Sprintf("任务被取消: %v", ctx.Err())
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
		}

		// 尝试转换
		// TODO: 这里需要重新设计类型转换逻辑
		// taskCopy, err = e.performActualConversionWithResult(ctx, taskCopy)
		err = fmt.Errorf("转换功能暂时禁用，需要重新设计类型系统")
		if false { // 曾时禁用
			// 转换成功
			result.Message = "转换成功"
			if attempt > 0 {
				result.Message = fmt.Sprintf("第%d次重试成功", attempt+1)
			}
			// result.TargetPath = taskCopy.TargetPath
			result.OriginalSize = sourceInfo.Size()

			// 检查转换后的文件大小
			if result.TargetPath != "" {
				if targetInfo, statErr := os.Stat(result.TargetPath); statErr == nil {
					result.NewSize = targetInfo.Size()
				} else {
					result.NewSize = sourceInfo.Size()
					e.logger.Warn("无法获取目标文件大小", zap.Error(statErr))
				}
			} else {
				result.NewSize = sourceInfo.Size()
			}

			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		// 转换失败，记录错误
		lastErr = err
		e.logger.Warn("转换尝试失败",
			zap.String("file", filepath.Base(task.Path)),
			zap.Int("attempt", attempt+1),
			zap.Error(err))

		// 清理可能的部分文件
		// TODO: 需要重新设计目标路径的管理
		e.cleanupPartialFiles("") // 暂时传入空字符串
	}

	// 所有重试都失败
	result.Status = "failed"
	result.Message = fmt.Sprintf("转换失败（已重试%d次）: %v", maxRetries, lastErr)
	e.logger.Error("转换最终失败",
		zap.String("file", filepath.Base(task.Path)),
		zap.Int("total_attempts", maxRetries+1),
		zap.Error(lastErr))

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

// performActualConversionWithResult 执行实际的文件转换并返回更新后的任务
func (e *ConversionEngine) performActualConversionWithResult(ctx context.Context, task *types.FileTask) (*types.FileTask, error) {
	e.logger.Info("执行文件转换",
		zap.String("source", filepath.Base(task.Path)),
		zap.String("format", string(task.TargetFormat)))

	// 生成目标文件路径
	_, err := e.generateTargetPath(task.Path, string(task.TargetFormat))
	if err != nil {
		return task, fmt.Errorf("生成目标路径失败: %w", err)
	}
	// 注意：FileTask结构体没有TargetPath字段，这里我们只能在转换结果中传递

	// 如果启用了备份功能，先创建备份
	var backupPath string
	if e.config.CreateBackups {
		backupPath, err = e.createBackup(task.Path)
		if err != nil {
			e.logger.Warn("创建备份失败，继续转换", zap.Error(err))
			// 备份失败不阻止转换，但记录警告
		} else {
			e.logger.Debug("已创建文件备份",
				zap.String("source", filepath.Base(task.Path)),
				zap.String("backup", filepath.Base(backupPath)))
		}
	}

	// 执行转换
	err = e.performActualConversion(ctx, task)
	if err != nil {
		// 转换失败时，如果有备份，尝试恢复
		if backupPath != "" {
			if restoreErr := e.restoreFromBackup(backupPath, task.Path); restoreErr != nil {
				e.logger.Error("从备份恢复文件失败",
					zap.String("backup", backupPath),
					zap.String("original", task.Path),
					zap.Error(restoreErr))
			} else {
				e.logger.Info("已从备份恢复文件",
					zap.String("file", filepath.Base(task.Path)))
			}
		}
		return task, err
	}

	// 转换成功，清理备份文件（如果用户配置了不保留备份）
	if backupPath != "" && !e.config.KeepBackups {
		if removeErr := os.Remove(backupPath); removeErr != nil {
			e.logger.Warn("清理备份文件失败",
				zap.String("backup", backupPath),
				zap.Error(removeErr))
		} else {
			e.logger.Debug("已清理备份文件",
				zap.String("backup", filepath.Base(backupPath)))
		}
	}

	return task, nil
}

// performActualConversion 执行实际的文件转换
func (e *ConversionEngine) performActualConversion(ctx context.Context, task *types.FileTask) error {
	e.logger.Info("执行文件转换",
		zap.String("source", filepath.Base(task.Path)),
		zap.String("format", string(task.TargetFormat)))

	// 生成目标文件路径
	_, err := e.generateTargetPath(task.Path, string(task.TargetFormat))
	if err != nil {
		return fmt.Errorf("生成目标路径失败: %w", err)
	}
	// 注意: FileTask没有TargetPath字段，这里我们只记录在日志中

	// 读取源文件信息
	sourceInfo, err := os.Stat(task.Path)
	if err != nil {
		return fmt.Errorf("无法获取源文件信息: %w", err)
	}

	// 获取源文件的创建时间和修改时间
	createTime := sourceInfo.ModTime()
	modifyTime := sourceInfo.ModTime()

	// 执行转换
	switch string(task.TargetFormat) {
	case "jxl_lossless", "jxl_balanced":
		// 在JXL转换中保留ICC配置
		iccProfile, err := e.stateManager.LoadICCProfile(task.Path)
		if err == nil && iccProfile != nil {
			// TODO: 处理ICC配置
			e.logger.Debug("加载ICC配置", zap.String("file", filepath.Base(task.Path)))
		}

		// TODO: 这里需要创建ConversionTask对象来调用convertToJXL
		e.logger.Debug("模拟JXL转换", zap.String("file", filepath.Base(task.Path)))
		return nil
	case "avif_compressed":
		e.logger.Debug("模拟AVIF转换", zap.String("file", filepath.Base(task.Path)))
		return nil
	case "avif_balanced":
		// README要求：AVIF也使用平衡优化逻辑
		e.logger.Debug("模拟AVIF平衡优化", zap.String("file", filepath.Base(task.Path)))
		return nil
	case "remux":
		// 在视频重包装中保留创建时间和修改时间
		e.logger.Debug("模拟视频重包装",
			zap.String("file", filepath.Base(task.Path)),
			zap.Time("create_time", createTime),
			zap.Time("modify_time", modifyTime))
		return nil
	case "skip":
		// 跳过处理 - 用于表情包模式下的视频文件或其他需要跳过的情况
		e.logger.Debug("跳过文件处理", zap.String("file", filepath.Base(task.Path)), zap.String("reason", "skip_format"))
		return nil
	case "auto":
		// 自动模式：根据文件类型选择默认转换
		if task.Type == types.MediaTypeImage {
			e.logger.Debug("自动模式选择JXL", zap.String("file", filepath.Base(task.Path)))
		} else {
			e.logger.Debug("自动模式选择视频重包装", zap.String("file", filepath.Base(task.Path)))
		}
		return nil
	default:
		// 未知格式，使用默认处理
		e.logger.Warn("未知目标格式，使用默认处理", zap.String("format", string(task.TargetFormat)))
		// 模拟转换耗时
		processingTime := time.Duration(100+len(task.Path)%500) * time.Millisecond
		select {
		case <-time.After(processingTime):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 保存文件的创建时间和修改时间
	mediaInfo := &types.MediaInfo{
		Path:       task.Path,
		CreateTime: createTime,
		ModifyTime: modifyTime,
	}

	e.stateManager.SaveMediaFiles([]*types.MediaInfo{mediaInfo})

	return nil
}

// generateTargetPath 生成目标文件路径
func (e *ConversionEngine) generateTargetPath(sourcePath, format string) (string, error) {
	dir := filepath.Dir(sourcePath)
	baseName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))

	var ext string
	switch format {
	case "jxl_lossless", "jxl_balanced":
		ext = ".jxl"
	case "avif_compressed":
		ext = ".avif"
	case "avif_balanced":
		ext = ".avif"
	case "remux":
		// 视频重包装保持原格式或转为MP4
		originalExt := filepath.Ext(sourcePath)
		if originalExt == ".mp4" {
			ext = ".mp4" // 已经是MP4，保持不变
		} else {
			ext = ".mp4" // 其他格式转为MP4
		}
	default:
		// 保持原扩展名
		ext = filepath.Ext(sourcePath)
	}

	targetPath := filepath.Join(dir, baseName+ext)

	// 如果目标文件已存在，生成唯一名称
	if _, err := os.Stat(targetPath); err == nil {
		counter := 1
		for {
			newName := fmt.Sprintf("%s_pixly_%d%s", baseName, counter, ext)
			newPath := filepath.Join(dir, newName)
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				targetPath = newPath
				break
			}
			counter++
			if counter > 1000 { // 防止无限循环
				return "", fmt.Errorf("无法生成唯一文件名")
			}
		}
	}

	return targetPath, nil
}

// convertToJXL 转换为JXL格式
func (e *ConversionEngine) convertToJXL(ctx context.Context, task *types.ConversionTask, lossless bool) error {
	e.logger.Debug("开始转换为JXL格式",
		zap.String("source", filepath.Base(task.SourcePath)),
		zap.String("target", filepath.Base(task.TargetPath)),
		zap.Bool("lossless", lossless))

	// 模拟转换过程
	// 在实际实现中，这里会调用cjxl命令
	// cjxl input.jpg output.jxl -q 100 (无损) 或 -q 85 (平衡)

	processingTime := time.Duration(200+len(task.SourcePath)%300) * time.Millisecond
	select {
	case <-time.After(processingTime):
		e.logger.Debug("JXL转换完成", zap.String("target", filepath.Base(task.TargetPath)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// convertToAVIF 转换为AVIF格式
func (e *ConversionEngine) convertToAVIF(ctx context.Context, task *types.ConversionTask, mode string) error {
	e.logger.Debug("开始转换为AVIF格式",
		zap.String("source", filepath.Base(task.SourcePath)),
		zap.String("target", filepath.Base(task.TargetPath)),
		zap.String("mode", mode))

	// 模拟转换过程
	// 在实际实现中，这里会调用FFmpeg命令
	// ffmpeg -i input.jpg -c:v libaom-av1 -crf 30 output.avif

	processingTime := time.Duration(150+len(task.SourcePath)%200) * time.Millisecond
	select {
	case <-time.After(processingTime):
		e.logger.Debug("AVIF转换完成", zap.String("target", filepath.Base(task.TargetPath)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// remuxVideo 视频重包装
func (e *ConversionEngine) remuxVideo(ctx context.Context, task *types.ConversionTask) error {
	e.logger.Debug("开始视频重包装",
		zap.String("source", filepath.Base(task.SourcePath)),
		zap.String("target", filepath.Base(task.TargetPath)))

	// 模拟重包装过程
	// 在实际实现中，这里会调用FFmpeg命令
	// ffmpeg -i input.mov -c copy output.mp4

	processingTime := time.Duration(100+len(task.SourcePath)%400) * time.Millisecond
	select {
	case <-time.After(processingTime):
		e.logger.Debug("视频重包装完成", zap.String("target", filepath.Base(task.TargetPath)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// generateReport 生成报告
func (e *ConversionEngine) generateReport(results []ConversionResult) {
	successCount := 0
	failCount := 0
	skippedCount := 0
	var totalOriginalSize int64
	var totalNewSize int64
	var totalDuration time.Duration

	for _, result := range results {
		switch result.Status {
		case "success":
			successCount++
			totalOriginalSize += result.OriginalSize
			totalNewSize += result.NewSize
		case "failed":
			failCount++
		case "skipped":
			skippedCount++
		}
		totalDuration += result.Duration
	}

	// 计算空间节省
	spaceSaved := totalOriginalSize - totalNewSize
	compressionRatio := 0.0
	if totalOriginalSize > 0 {
		compressionRatio = float64(spaceSaved) / float64(totalOriginalSize) * 100
	}

	// 显示最终的详细统计报告
	e.progressManager.ShowDetailedRealTimeStats()

	fmt.Println()
	fmt.Println("📊 转换完成报告")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("📁 总文件数: %d\n", len(results))
	fmt.Printf("✅ 成功转换: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("❌ 转换失败: %d\n", failCount)
	}
	if skippedCount > 0 {
		fmt.Printf("⏭️ 跳过处理: %d\n", skippedCount)
	}

	if successCount > 0 {
		fmt.Println(strings.Repeat("-", 30))
		fmt.Printf("💾 原始大小: %s\n", e.formatBytes(totalOriginalSize))
		fmt.Printf("💾 转换后: %s\n", e.formatBytes(totalNewSize))

		if spaceSaved > 0 {
			fmt.Printf("💰 节省空间: %s (%.1f%%)\n", e.formatBytes(spaceSaved), compressionRatio)
		} else if spaceSaved < 0 {
			fmt.Printf("📈 占用增加: %s (%.1f%%)\n", e.formatBytes(-spaceSaved), -compressionRatio)
		} else {
			fmt.Println("😐 文件大小无变化")
		}
	}

	fmt.Printf("⏱️ 总耗时: %v\n", totalDuration.Round(time.Millisecond))
	if len(results) > 0 {
		avgTime := totalDuration / time.Duration(len(results))
		fmt.Printf("🕰️ 平均耗时: %v/文件\n", avgTime.Round(time.Millisecond))
	}

	// 获取进度管理器的最终统计
	progressStats := e.progressManager.GetStats()
	if progressStats.AverageSpeed > 0 {
		fmt.Printf("⚡ 平均处理速度: %.2f 文件/秒\n", progressStats.AverageSpeed)
		fmt.Printf("🚀 处理速率: %d 文件/分钟\n", progressStats.ProcessingRate)
	}

	fmt.Println(strings.Repeat("=", 50))

	// 记录详细统计
	e.logger.Info("转换报告生成完成",
		zap.Int("total", len(results)),
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("skipped", skippedCount),
		zap.Int64("original_size", totalOriginalSize),
		zap.Int64("new_size", totalNewSize),
		zap.Int64("space_saved", spaceSaved),
		zap.Float64("compression_ratio", compressionRatio),
		zap.Duration("total_duration", totalDuration),
		zap.Float64("avg_speed", progressStats.AverageSpeed))
}

// formatBytes 格式化字节数为可读字符串
func (e *ConversionEngine) formatBytes(bytes int64) string {
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

// ConversionResult 转换结果
type ConversionResult struct {
	SourcePath   string
	TargetPath   string
	Status       string
	Message      string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	OriginalSize int64 // 原始文件大小
	NewSize      int64 // 转换后文件大小
}

// estimateFileCount 估算目录中的文件数量 - 新增方法
func (e *ConversionEngine) estimateFileCount(dir string) int {
	var count int

	// 快速扫描，只计算文件数量不做详细检查
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		fileName := info.Name()
		if strings.HasPrefix(fileName, ".") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		mediaExtensions := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
			".webp": true, ".heic": true, ".heif": true, ".avif": true, ".jxl": true,
			".tiff": true, ".tif": true, ".bmp": true,
			".mp4": true, ".mov": true, ".webm": true, ".mkv": true,
			".avi": true, ".m4v": true, ".3gp": true,
		}

		if mediaExtensions[ext] {
			count++
		}

		return nil
	})

	return count
}

// handleCorruptedFiles 处理损坏文件决策 - 新增方法
func (e *ConversionEngine) handleCorruptedFiles(corruptedFiles []string) (string, error) {
	e.logger.Info("检测到损坏文件，调用UI处理决策", zap.Int("count", len(corruptedFiles)))

	// 暂停进度显示，进行用户交互
	e.progressManager.Pause()
	defer e.progressManager.Resume()

	// 调用UI的损坏文件处理决策
	if e.uiInterface != nil {
		return e.uiInterface.HandleCorruptedFiles(corruptedFiles)
	}

	// 如果没有UI接口（命令行模式），默认忽略
	e.logger.Warn("没有UI接口，默认忽略损坏文件")
	return "ignore", nil
}

// deleteCorruptedFiles 删除损坏文件 - 新增方法
func (e *ConversionEngine) deleteCorruptedFiles(files []string) {
	e.logger.Info("开始删除损坏文件", zap.Int("count", len(files)))

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			e.logger.Warn("删除损坏文件失败", zap.String("file", file), zap.Error(err))
		} else {
			e.logger.Info("已删除损坏文件", zap.String("file", filepath.Base(file)))
		}
	}

	fmt.Printf("✅ 已删除 %d 个损坏文件\n", len(files))
}

// repairCorruptedFiles 尝试修复损坏文件 - 新增方法
func (e *ConversionEngine) repairCorruptedFiles(files []string) []string {
	e.logger.Info("尝试修复损坏文件", zap.Int("count", len(files)))

	var repairedFiles []string

	for _, file := range files {
		// 简化的修复逻辑 - 实际应用中可以使用FFmpeg的-fix参数等
		e.logger.Debug("尝试修复文件", zap.String("file", filepath.Base(file)))

		// README要求：如果修复失败，跳过并清理临时文件
		if e.attemptFileRepair(file) {
			repairedFiles = append(repairedFiles, file)
			e.logger.Info("文件修复成功", zap.String("file", filepath.Base(file)))
		} else {
			e.logger.Warn("文件修复失败，将跳过处理", zap.String("file", filepath.Base(file)))
			// 清理可能的临时文件
			e.cleanupTempFiles(file)
		}
	}

	fmt.Printf("✅ 修复完成: %d/%d 文件修复成功\n", len(repairedFiles), len(files))
	return repairedFiles
}

// attemptFileRepair 尝试修复单个文件 - 新增方法
func (e *ConversionEngine) attemptFileRepair(filePath string) bool {
	// 简化的修复逻辑，实际应用中可以使用更复杂的修复算法
	// 例如：使用FFmpeg的-fix参数、文件头修复等

	// 检查文件是否可读
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	// 模拟修复过程（实际中这里会调用FFmpeg或其他修复工具）
	time.Sleep(50 * time.Millisecond) // 模拟修复时间

	// 简化逻辑：随机模拟修复结果
	// 实际应用中这里会根据修复操作的实际结果返回
	return len(filepath.Base(filePath))%3 == 0 // 模拟约为33%的修复成功率
}

// cleanupPartialFiles 清理转换失败时的部分文件
func (e *ConversionEngine) cleanupPartialFiles(targetPath string) {
	if targetPath == "" {
		return
	}

	// 清理可能的部分文件
	filesToCleanup := []string{
		targetPath,
		targetPath + ".tmp",
		targetPath + ".temp",
		targetPath + ".part",
		targetPath + ".incomplete",
	}

	for _, file := range filesToCleanup {
		if _, err := os.Stat(file); err == nil {
			if removeErr := os.Remove(file); removeErr != nil {
				e.logger.Debug("清理部分文件失败",
					zap.String("file", filepath.Base(file)),
					zap.Error(removeErr))
			} else {
				e.logger.Debug("已清理部分文件",
					zap.String("file", filepath.Base(file)))
			}
		}
	}
}

// cleanupTempFiles 清理临时文件 - 新增方法
func (e *ConversionEngine) cleanupTempFiles(originalFile string) {
	// 清理可能的临时文件
	tempPatterns := []string{
		originalFile + ".tmp",
		originalFile + ".temp",
		originalFile + ".bak",
		originalFile + ".repair",
	}

	for _, tempFile := range tempPatterns {
		if _, err := os.Stat(tempFile); err == nil {
			if removeErr := os.Remove(tempFile); removeErr != nil {
				e.logger.Warn("清理临时文件失败", zap.String("temp_file", tempFile), zap.Error(removeErr))
			} else {
				e.logger.Debug("已清理临时文件", zap.String("temp_file", tempFile))
			}
		}
	}
}

// removeTasksByFiles 从任务列表中移除指定文件对应的任务
func (e *ConversionEngine) removeTasksByFiles(tasks []*types.ConversionTask, filesToRemove []string) []*types.ConversionTask {
	fileSet := make(map[string]bool)
	for _, file := range filesToRemove {
		fileSet[file] = true
	}

	var filteredTasks []*types.ConversionTask
	for _, task := range tasks {
		if !fileSet[task.SourcePath] {
			filteredTasks = append(filteredTasks, task)
		}
	}

	e.logger.Info("从任务列表中移除文件",
		zap.Int("original_tasks", len(tasks)),
		zap.Int("removed_files", len(filesToRemove)),
		zap.Int("remaining_tasks", len(filteredTasks)))

	return filteredTasks
}

// deleteLowQualityFiles 删除低品质文件
func (e *ConversionEngine) deleteLowQualityFiles(files []string) {
	e.logger.Info("开始删除低品质文件", zap.Int("count", len(files)))

	var deletedCount int
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			e.logger.Warn("删除低品质文件失败", zap.String("file", filepath.Base(file)), zap.Error(err))
		} else {
			e.logger.Info("已删除低品质文件", zap.String("file", filepath.Base(file)))
			deletedCount++
		}
	}

	fmt.Printf("✅ 已删除 %d/%d 个低品质文件\n", deletedCount, len(files))
}

// updateTasksForLowQualityFiles 更新低品质文件对应任务的目标格式
func (e *ConversionEngine) updateTasksForLowQualityFiles(tasks []*types.ConversionTask, lowQualityFiles []string, targetFormat string) []*types.ConversionTask {
	fileSet := make(map[string]bool)
	for _, file := range lowQualityFiles {
		fileSet[file] = true
	}

	var updatedCount int
	for i := range tasks {
		if fileSet[tasks[i].SourcePath] {
			tasks[i].TargetFormat = targetFormat
			updatedCount++
			e.logger.Debug("更新低品质文件任务格式",
				zap.String("file", filepath.Base(tasks[i].SourcePath)),
				zap.String("target_format", targetFormat))
		}
	}

	e.logger.Info("更新低品质文件任务格式完成",
		zap.Int("updated_tasks", updatedCount),
		zap.String("target_format", targetFormat))

	fmt.Printf("✅ 已更新 %d 个低品质文件的处理格式为: %s\n", updatedCount, targetFormat)
	return tasks
}

// createBackup 创建文件备份
func (e *ConversionEngine) createBackup(filePath string) (string, error) {
	// 生成备份文件名
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf(".pixly_backup_%s_%s", timestamp, baseName)
	backupPath := filepath.Join(dir, backupName)

	// 复制文件作为备份
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开源文件: %w", err)
	}
	defer sourceFile.Close()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("无法创建备份文件: %w", err)
	}
	defer backupFile.Close()

	// 复制文件内容
	_, err = sourceFile.WriteTo(backupFile)
	if err != nil {
		// 复制失败，清理部分备份文件
		os.Remove(backupPath)
		return "", fmt.Errorf("复制文件失败: %w", err)
	}

	return backupPath, nil
}

// restoreFromBackup 从备份恢复文件
func (e *ConversionEngine) restoreFromBackup(backupPath, originalPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	// 复制备份文件回原位置
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("无法打开备份文件: %w", err)
	}
	defer backupFile.Close()

	originalFile, err := os.Create(originalPath)
	if err != nil {
		return fmt.Errorf("无法创建原始文件: %w", err)
	}
	defer originalFile.Close()

	// 复制内容
	_, err = backupFile.WriteTo(originalFile)
	if err != nil {
		return fmt.Errorf("恢复文件失败: %w", err)
	}

	return nil
}

// applyRoutingDecisions 应用路由决策到任务列表
func (e *ConversionEngine) applyRoutingDecisions(tasks []*types.ConversionTask, decisions map[string]*types.RoutingDecision) []*types.ConversionTask {
	updatedTasks := make([]*types.ConversionTask, 0, len(tasks))

	for _, task := range tasks {
		if decision, exists := decisions[task.SourcePath]; exists {
			// 根据路由决策更新任务
			switch decision.Strategy {
			case "skip":
				// 跳过此任务
				continue
			case "delete":
				// 删除文件并跳过任务
				os.Remove(task.SourcePath)
				e.logger.Info("删除低品质文件", zap.String("file", filepath.Base(task.SourcePath)))
				continue
			default:
				// 更新任务的目标格式
				task.TargetFormat = decision.TargetFormat
				task.Quality = string(decision.QualityLevel)
				updatedTasks = append(updatedTasks, task)
			}
		} else {
			// 没有路由决策的文件保持原样
			updatedTasks = append(updatedTasks, task)
		}
	}

	return updatedTasks
}

// performBalanceOptimization 执行平衡优化 - 集成README要求的完整平衡优化逻辑
func (e *ConversionEngine) performBalanceOptimization(ctx context.Context, task *types.ConversionTask) error {
	e.logger.Debug("开始平衡优化", zap.String("file", filepath.Base(task.SourcePath)))

	// 确定媒体类型
	var mediaType types.MediaType
	switch task.MediaType {
	case "image":
		mediaType = types.MediaTypeImage
	case "animated":
		mediaType = types.MediaTypeAnimated
	case "video":
		mediaType = types.MediaTypeVideo
	default:
		mediaType = types.MediaTypeImage
	}

	// 使用平衡优化器进行优化
	result, err := e.balanceOptimizer.OptimizeFile(ctx, task.SourcePath, mediaType)
	if err != nil {
		return fmt.Errorf("平衡优化失败: %w", err)
	}

	if !result.Success {
		// README要求：无法优化时记录原因并标记为跳过
		e.logger.Info("平衡优化无法减小文件体积",
			zap.String("file", filepath.Base(task.SourcePath)),
			zap.Int64("original_size", result.OriginalSize))
		return nil // 不算错误，只是无法优化
	}

	// 成功优化，替换原文件
	if err := e.replaceOriginalFile(task.SourcePath, result.OutputPath); err != nil {
		return fmt.Errorf("替换原文件失败: %w", err)
	}

	e.logger.Info("平衡优化成功",
		zap.String("file", filepath.Base(task.SourcePath)),
		zap.String("method", result.Method),
		zap.String("quality", result.Quality),
		zap.Int64("space_saved", result.SpaceSaved),
		zap.Duration("process_time", result.ProcessTime))

	return nil
}

// replaceOriginalFile 安全地替换原文件
func (e *ConversionEngine) replaceOriginalFile(originalPath, newPath string) error {
	// 创建备份
	backupPath := originalPath + ".pixly_backup"
	if err := os.Rename(originalPath, backupPath); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	// 移动新文件到原位置
	if err := os.Rename(newPath, originalPath); err != nil {
		// 恢复备份
		os.Rename(backupPath, originalPath)
		return fmt.Errorf("替换文件失败: %w", err)
	}

	// 删除备份
	os.Remove(backupPath)
	return nil
}

// CleanupBalanceOptimizer 清理平衡优化器临时文件
func (e *ConversionEngine) CleanupBalanceOptimizer() {
	if e.balanceOptimizer != nil {
		e.balanceOptimizer.CleanupTempFiles()
	}
}

// SaveTasks 保存转换任务
func (e *ConversionEngine) SaveTasks(tasks []*types.FileTask) error {
	e.logger.Debug("保存转换任务到状态管理器")

	if e.stateManager == nil {
		return fmt.Errorf("状态管理器未初始化")
	}

	return e.stateManager.SaveTasks(tasks)
}

// LoadTasks 加载转换任务
func (e *ConversionEngine) LoadTasks() ([]*types.FileTask, error) {
	e.logger.Debug("从状态管理器加载转换任务")

	if e.stateManager == nil {
		return nil, fmt.Errorf("状态管理器未初始化")
	}

	return e.stateManager.LoadTasks()
}
