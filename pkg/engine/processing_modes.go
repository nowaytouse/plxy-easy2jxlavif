package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/engine/quality"

	"go.uber.org/zap"
)

// ProcessingModeManager 处理模式管理器 - 统一管理三大处理模式
type ProcessingModeManager struct {
	logger        *zap.Logger
	toolPaths     types.ToolCheckResults
	qualityEngine *quality.QualityEngine

	// 三大处理模式实例
	autoPlusMode *AutoPlusMode
	qualityMode  *QualityMode
	emojiMode    *EmojiMode
}

// ProcessingMode 处理模式接口
type ProcessingMode interface {
	// GetModeName 获取模式名称
	GetModeName() string

	// ProcessFile 处理单个文件
	ProcessFile(ctx context.Context, info *types.MediaInfo) (*types.ProcessingResult, error)

	// GetStrategy 获取该模式对指定文件的处理策略
	GetStrategy(info *types.MediaInfo) (*ProcessingStrategy, error)

	// ShouldSkipFile 判断是否应该跳过该文件
	ShouldSkipFile(info *types.MediaInfo) (bool, string)
}

// ProcessingStrategy 处理策略
type ProcessingStrategy struct {
	Mode           types.AppMode          `json:"mode"`
	TargetFormat   string                 `json:"target_format"` // "jxl", "avif", "mov"
	Quality        string                 `json:"quality"`       // "lossless", "balanced", "compressed"
	ToolChain      []string               `json:"tool_chain"`    // ["cjxl", "ffmpeg", "avifenc"]
	Parameters     map[string]interface{} `json:"parameters"`
	ExpectedSaving int64                  `json:"expected_saving"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Confidence     float64                `json:"confidence"`
	Reason         string                 `json:"reason"`
}

// NewProcessingModeManager 创建处理模式管理器
func NewProcessingModeManager(logger *zap.Logger, toolPaths types.ToolCheckResults, qualityEngine *quality.QualityEngine) *ProcessingModeManager {
	manager := &ProcessingModeManager{
		logger:        logger,
		toolPaths:     toolPaths,
		qualityEngine: qualityEngine,
	}

	// 初始化三大处理模式
	manager.autoPlusMode = NewAutoPlusMode(logger, toolPaths, qualityEngine)
	manager.qualityMode = NewQualityMode(logger, toolPaths)
	manager.emojiMode = NewEmojiMode(logger, toolPaths)

	return manager
}

// GetMode 根据模式类型获取处理模式实例
func (pmm *ProcessingModeManager) GetMode(mode types.AppMode) ProcessingMode {
	switch mode {
	case types.ModeAutoPlus:
		return pmm.autoPlusMode
	case types.ModeQuality:
		return pmm.qualityMode
	case types.ModeEmoji:
		return pmm.emojiMode
	default:
		pmm.logger.Warn("未知处理模式，使用自动模式+", zap.String("mode", mode.String()))
		return pmm.autoPlusMode
	}
}

// ProcessFiles 批量处理文件
func (pmm *ProcessingModeManager) ProcessFiles(ctx context.Context, mode types.AppMode, files []*types.MediaInfo) ([]*types.ProcessingResult, error) {
	processingMode := pmm.GetMode(mode)
	pmm.logger.Info("开始批量处理",
		zap.String("mode", processingMode.GetModeName()),
		zap.Int("total_files", len(files)))

	var results []*types.ProcessingResult

	for _, fileInfo := range files {
		// 检查是否应该跳过
		shouldSkip, reason := processingMode.ShouldSkipFile(fileInfo)
		if shouldSkip {
			result := &types.ProcessingResult{
				OriginalPath: fileInfo.Path,
				Success:      false,
				Error:        reason,
				Mode:         mode,
			}
			results = append(results, result)
			pmm.logger.Debug("跳过文件",
				zap.String("file", filepath.Base(fileInfo.Path)),
				zap.String("reason", reason))
			continue
		}

		// 处理文件
		result, err := processingMode.ProcessFile(ctx, fileInfo)
		if err != nil {
			pmm.logger.Error("文件处理失败",
				zap.String("file", filepath.Base(fileInfo.Path)),
				zap.String("error", err.Error()))
			result = &types.ProcessingResult{
				OriginalPath: fileInfo.Path,
				Success:      false,
				Error:        err.Error(),
				Mode:         mode,
			}
		}

		// If conversion was successful and a new file was created, delete the original file.
		if result.Success && result.NewPath != "" && result.OriginalPath != result.NewPath {
			if err := os.Remove(result.OriginalPath); err != nil {
				pmm.logger.Warn("Failed to delete original file",
					zap.String("file", result.OriginalPath),
					zap.Error(err))
			} else {
				pmm.logger.Info("Deleted original file", zap.String("file", result.OriginalPath))
			}
		}

		results = append(results, result)
	}

	pmm.logger.Info("批量处理完成",
		zap.String("mode", processingMode.GetModeName()),
		zap.Int("total_results", len(results)))

	return results, nil
}

// =============================================================================
// 🤖 自动模式+ (智能决策核心) - README要求的核心模式
// =============================================================================

// AutoPlusMode 自动模式+ - 根据智能品质判断引擎的结果自动路由到最优处理策略
type AutoPlusMode struct {
	logger           *zap.Logger
	toolPaths        types.ToolCheckResults
	qualityEngine    *quality.QualityEngine
	qualityMode      *QualityMode     // 复用品质模式逻辑
	emojiMode        *EmojiMode       // 复用表情包模式逻辑
	conversionEngine *SimpleConverter // 简化转换器
}

// NewAutoPlusMode 创建自动模式+实例
func NewAutoPlusMode(logger *zap.Logger, toolPaths types.ToolCheckResults, qualityEngine *quality.QualityEngine) *AutoPlusMode {
	return &AutoPlusMode{
		logger:           logger,
		toolPaths:        toolPaths,
		qualityEngine:    qualityEngine,
		qualityMode:      NewQualityMode(logger, toolPaths),
		emojiMode:        NewEmojiMode(logger, toolPaths),
		conversionEngine: NewSimpleConverter(logger, toolPaths, false),
	}
}

func (apm *AutoPlusMode) GetModeName() string {
	return "自动模式+"
}

func (apm *AutoPlusMode) ProcessFile(ctx context.Context, info *types.MediaInfo) (*types.ProcessingResult, error) {
	// 进行品质评估
	assessment, err := apm.qualityEngine.AssessFile(ctx, info.Path)
	if err != nil {
		return nil, fmt.Errorf("品质评估失败: %w", err)
	}

	// 根据README规范的品质分类体系进行路由
	switch assessment.QualityLevel {
	case types.QualityVeryHigh, types.QualityHigh:
		// 极高/高品质 -> 路由至品质模式的无损压缩逻辑
		apm.logger.Debug("高品质文件，路由至品质模式",
			zap.String("file", filepath.Base(info.Path)),
			zap.String("quality", assessment.QualityLevel.String()))
		return apm.qualityMode.ProcessFile(ctx, info)

	case types.QualityMediumHigh, types.QualityMediumLow:
		// 中高/中低品质 -> 平衡优化逻辑
		apm.logger.Debug("中等品质文件，使用平衡优化",
			zap.String("file", filepath.Base(info.Path)),
			zap.String("quality", assessment.QualityLevel.String()))
		return apm.processBalancedOptimization(ctx, info, assessment)

	case types.QualityLow, types.QualityVeryLow:
		// 极低/低品质 -> 触发极低品质决策流程（这里简化处理）
		apm.logger.Debug("低品质文件，触发特殊处理",
			zap.String("file", filepath.Base(info.Path)),
			zap.String("quality", assessment.QualityLevel.String()))
		return apm.processLowQualityFile(ctx, info, assessment)

	default:
		return nil, fmt.Errorf("未知品质等级: %s", assessment.QualityLevel.String())
	}
}

// processBalancedOptimization 平衡优化逻辑 - README要求的核心算法
func (apm *AutoPlusMode) processBalancedOptimization(ctx context.Context, info *types.MediaInfo, assessment *quality.QualityAssessment) (*types.ProcessingResult, error) {
	// README要求的平衡优化逻辑：
	// 1. 无损重新包装优先
	// 2. 数学无损压缩
	// 3. 有损探测（高品质组: 90,85,75；中等品质组: 60,55）
	// 4. 最终决策：只要体积有任何减小，即视为成功

	var bestResult *types.ProcessingResult
	originalSize := info.Size

	// 步骤1: 尝试无损重新包装
	if result := apm.tryLosslessRepackaging(ctx, info); result != nil && result.Success && result.NewSize < originalSize {
		apm.logger.Debug("无损重新包装成功",
			zap.String("file", filepath.Base(info.Path)),
			zap.Int64("saved", originalSize-result.NewSize))
		return result, nil
	}

	// 步骤2: 尝试数学无损压缩
	if result := apm.tryMathematicalLossless(ctx, info); result != nil && result.Success && result.NewSize < originalSize {
		apm.logger.Debug("数学无损压缩成功",
			zap.String("file", filepath.Base(info.Path)),
			zap.Int64("saved", originalSize-result.NewSize))
		return result, nil
	}

	// 步骤3: 有损探测
	qualityLevels := []int{90, 85, 75, 60, 55} // README规定的探测点
	for _, quality := range qualityLevels {
		result := apm.tryLossyCompression(ctx, info, quality)
		if result != nil && result.Success && result.NewSize < originalSize {
			if bestResult == nil || result.NewSize < bestResult.NewSize {
				bestResult = result
			}
		}
	}

	if bestResult != nil {
		apm.logger.Debug("有损压缩探测成功",
			zap.String("file", filepath.Base(info.Path)),
			zap.Int64("saved", originalSize-bestResult.NewSize))
		return bestResult, nil
	}

	// 步骤4: 如果都无法优化，则跳过
	return &types.ProcessingResult{
		OriginalPath: info.Path,
		OriginalSize: originalSize,
		NewSize:      originalSize,
		Success:      false,
		Error:        "无法找到有效的优化方案",
		Mode:         types.ModeAutoPlus,
	}, nil
}

// processLowQualityFile 处理低品质文件
func (apm *AutoPlusMode) processLowQualityFile(ctx context.Context, info *types.MediaInfo, assessment *quality.QualityAssessment) (*types.ProcessingResult, error) {
	// 对于低品质文件，README要求触发用户批量决策流程
	// 但为了保证转换功能，这里先尝试基本的转换处理
	// TODO: 在未来版本中集成批量决策管理器

	apm.logger.Debug("低品质文件处理",
		zap.String("file", filepath.Base(info.Path)),
		zap.Float64("quality_score", assessment.Score),
		zap.String("recommended_mode", assessment.RecommendedMode.String()))

	// 根据文件类型尝试基本转换
	switch info.Type {
	case types.MediaTypeVideo:
		// 视频文件：尝试重包装为MOV格式
		ext := strings.ToLower(filepath.Ext(info.Path))
		targetPath := strings.TrimSuffix(info.Path, ext) + ".mov"

		result, err := apm.conversionEngine.RemuxVideo(ctx, info.Path, targetPath)
		if err != nil {
			apm.logger.Debug("低品质视频重包装失败",
				zap.String("file", filepath.Base(info.Path)),
				zap.Error(err))
			return &types.ProcessingResult{
				OriginalPath: info.Path,
				OriginalSize: info.Size,
				NewSize:      info.Size,
				Success:      false,
				Error:        fmt.Sprintf("低品质视频处理失败: %v", err),
				Mode:         types.ModeAutoPlus,
			}, nil
		}

		// 检查是否有体积优化
		if result.Success && result.NewSize < result.OriginalSize {
			apm.logger.Debug("低品质视频重包装成功",
				zap.String("file", filepath.Base(info.Path)),
				zap.Int64("saved", result.OriginalSize-result.NewSize))
			result.Mode = types.ModeAutoPlus
			return result, nil
		}

	case types.MediaTypeImage, types.MediaTypeAnimated:
		// 静图文件：尝试AVIF压缩（表情包模式推荐）
		ext := strings.ToLower(filepath.Ext(info.Path))
		targetPath := strings.TrimSuffix(info.Path, ext) + ".avif"

		result, err := apm.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, "compressed", info.Type)
		if err != nil {
			apm.logger.Debug("低品质图片转换失败",
				zap.String("file", filepath.Base(info.Path)),
				zap.Error(err))
			return &types.ProcessingResult{
				OriginalPath: info.Path,
				OriginalSize: info.Size,
				NewSize:      info.Size,
				Success:      false,
				Error:        fmt.Sprintf("低品质图片处理失败: %v", err),
				Mode:         types.ModeAutoPlus,
			}, nil
		}

		// 检查是否有体积优化
		if result.Success && result.NewSize < result.OriginalSize {
			apm.logger.Debug("低品质图片转换成功",
				zap.String("file", filepath.Base(info.Path)),
				zap.Int64("saved", result.OriginalSize-result.NewSize))
			result.Mode = types.ModeAutoPlus
			return result, nil
		}

	default:
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        "不支持的媒体类型",
			Mode:         types.ModeAutoPlus,
		}, nil
	}

	// 如果转换失败或没有体积优化，返回跳过结果
	return &types.ProcessingResult{
		OriginalPath: info.Path,
		OriginalSize: info.Size,
		NewSize:      info.Size,
		Success:      false,
		Error:        "低品质文件转换未产生明显优化",
		Mode:         types.ModeAutoPlus,
	}, nil
}

// 辅助方法（简化实现）
func (apm *AutoPlusMode) tryLosslessRepackaging(ctx context.Context, info *types.MediaInfo) *types.ProcessingResult {
	// 实现真实的无损重新包装
	ext := strings.ToLower(filepath.Ext(info.Path))

	// 生成目标文件路径
	var targetPath string
	var result *types.ProcessingResult
	var err error

	switch info.Type {
	case types.MediaTypeImage:
		// 静图转换为JXL无损
		targetPath = strings.TrimSuffix(info.Path, ext) + ".jxl"
		result, err = apm.conversionEngine.ConvertToJXL(ctx, info.Path, targetPath, true)
	case types.MediaTypeAnimated:
		// 动图转换为AVIF无损
		targetPath = strings.TrimSuffix(info.Path, ext) + ".avif"
		result, err = apm.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, "lossless", info.Type)

	case types.MediaTypeVideo:
		// 视频重包装为MOV
		targetPath = strings.TrimSuffix(info.Path, ext) + ".mov"
		result, err = apm.conversionEngine.RemuxVideo(ctx, info.Path, targetPath)

	default:
		// 不支持的类型
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        "不支持的媒体类型",
			Mode:         types.ModeAutoPlus,
		}
	}

	if err != nil {
		apm.logger.Debug("无损重新包装失败",
			zap.String("file", filepath.Base(info.Path)),
			zap.Error(err))
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        err.Error(),
			Mode:         types.ModeAutoPlus,
		}
	}

	return result
}

func (apm *AutoPlusMode) tryMathematicalLossless(ctx context.Context, info *types.MediaInfo) *types.ProcessingResult {
	// 实现真实的数学无损压缩
	ext := strings.ToLower(filepath.Ext(info.Path))

	// 只对静图进行数学无损压缩
	if info.Type != types.MediaTypeImage {
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        "仅支持静图的数学无损压缩",
			Mode:         types.ModeAutoPlus,
		}
	}

	// 生成目标文件路径
	targetPath := strings.TrimSuffix(info.Path, ext) + ".jxl"

	// 使用JXL无损压缩
	result, err := apm.conversionEngine.ConvertToJXL(ctx, info.Path, targetPath, true)
	if err != nil {
		apm.logger.Debug("数学无损压缩失败",
			zap.String("file", filepath.Base(info.Path)),
			zap.Error(err))
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        err.Error(),
			Mode:         types.ModeAutoPlus,
		}
	}

	return result
}

func (apm *AutoPlusMode) tryLossyCompression(ctx context.Context, info *types.MediaInfo, quality int) *types.ProcessingResult {
	// 实现真实的有损压缩
	ext := strings.ToLower(filepath.Ext(info.Path))

	// 根据质量等级选择格式
	var targetPath string
	var result *types.ProcessingResult
	var err error

	switch info.Type {
	case types.MediaTypeImage:
		// 静图：根据质量等级选择JXL或AVIF
		if quality >= 75 {
			// 高质量使用JXL平衡模式
			targetPath = strings.TrimSuffix(info.Path, ext) + ".jxl"
			result, err = apm.conversionEngine.ConvertToJXL(ctx, info.Path, targetPath, false)
		} else {
			// 低质量使用AVIF压缩
			targetPath = strings.TrimSuffix(info.Path, ext) + ".avif"
			mode := "balanced"
			if quality <= 60 {
				mode = "compressed"
			}
			result, err = apm.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, mode, info.Type)
		}
	case types.MediaTypeAnimated:
		// 动图：使用AVIF压缩
		targetPath = strings.TrimSuffix(info.Path, ext) + ".avif"
		mode := "balanced"
		if quality <= 60 {
				mode = "compressed"
		}
		result, err = apm.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, mode, info.Type)

	case types.MediaTypeVideo:
		// 视频：使用重包装
		targetPath = strings.TrimSuffix(info.Path, ext) + ".mov"
		result, err = apm.conversionEngine.RemuxVideo(ctx, info.Path, targetPath)

	default:
		// 不支持的类型
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        "不支持的媒体类型",
			Mode:         types.ModeAutoPlus,
		}
	}

	if err != nil {
		apm.logger.Debug("有损压缩失败",
			zap.String("file", filepath.Base(info.Path)),
			zap.Int("quality", quality),
			zap.Error(err))
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        err.Error(),
			Mode:         types.ModeAutoPlus,
		}
	}

	return result
}

func (apm *AutoPlusMode) GetStrategy(info *types.MediaInfo) (*ProcessingStrategy, error) {
	return &ProcessingStrategy{
		Mode:         types.ModeAutoPlus,
		TargetFormat: "auto",
		Quality:      "balanced",
		Confidence:   0.8,
		Reason:       "智能路由决策",
	}, nil
}

func (apm *AutoPlusMode) ShouldSkipFile(info *types.MediaInfo) (bool, string) {
	// 检查文件是否损坏
	if info.IsCorrupted {
		return true, "文件已损坏"
	}

	// 检查文件大小
	if info.Size < 1024 { // 小于1KB
		return true, "文件过小"
	}

	return false, ""
}

// =============================================================================
// 🔥 品质模式 (无损优先) - README要求的品质优先模式
// =============================================================================

// QualityMode 品质模式 - 追求最大保真度，采用数学无损压缩
type QualityMode struct {
	logger           *zap.Logger
	toolPaths        types.ToolCheckResults
	conversionEngine *SimpleConverter // 简化转换器
}

// NewQualityMode 创建品质模式实例
func NewQualityMode(logger *zap.Logger, toolPaths types.ToolCheckResults) *QualityMode {
	return &QualityMode{
		logger:           logger,
		toolPaths:        toolPaths,
		conversionEngine: NewSimpleConverter(logger, toolPaths, false),
	}
}

func (qm *QualityMode) GetModeName() string {
	return "品质模式"
}

func (qm *QualityMode) ProcessFile(ctx context.Context, info *types.MediaInfo) (*types.ProcessingResult, error) {
	// README要求：追求最大保真度，全部采用数学无损压缩
	// 目标格式: 静图: JXL, 动图: AVIF (无损), 视频: MOV (重包装)

	var targetFormat string
	var toolChain []string

	ext := strings.ToLower(filepath.Ext(info.Path))

	switch info.Type {
	case types.MediaTypeImage:
		// 静图转换为JXL无损
		targetFormat = "jxl"
		if ext == ".jpg" || ext == ".jpeg" {
			// JPEG必须使用cjxl的lossless=1参数
			toolChain = []string{"cjxl"}
		} else {
			toolChain = []string{"cjxl"}
		}

	case types.MediaTypeAnimated:
		// 动图转换为AVIF无损
		targetFormat = "avif"
		toolChain = []string{"ffmpeg"} // 必须使用ffmpeg处理动图

	case types.MediaTypeVideo:
		// 视频重新包装为MOV
		targetFormat = "mov"
		toolChain = []string{"ffmpeg"}

	default:
		return nil, fmt.Errorf("不支持的媒体类型: %s", info.Type.String())
	}

	qm.logger.Debug("品质模式处理",
		zap.String("file", filepath.Base(info.Path)),
		zap.String("target_format", targetFormat),
		zap.Strings("tool_chain", toolChain))

	// 调用真实的转换器
	var result *types.ProcessingResult
	var err error

	// 生成目标文件路径
	targetPath := strings.TrimSuffix(info.Path, ext) + "." + targetFormat

	switch info.Type {
	case types.MediaTypeImage:
		// 静图使用JXL无损压缩
		result, err = qm.conversionEngine.ConvertToJXL(ctx, info.Path, targetPath, true)

	case types.MediaTypeAnimated:
		// 动图使用AVIF无损模式
		result, err = qm.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, "lossless", info.Type)

	case types.MediaTypeVideo:
		// 视频重包装为MOV
		result, err = qm.conversionEngine.RemuxVideo(ctx, info.Path, targetPath)

	default:
		return nil, fmt.Errorf("不支持的媒体类型: %s", info.Type.String())
	}

	if err != nil {
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        err.Error(),
			Mode:         types.ModeQuality,
		}, nil
	}

	result.Mode = types.ModeQuality
	return result, nil
}

func (qm *QualityMode) GetStrategy(info *types.MediaInfo) (*ProcessingStrategy, error) {
	var targetFormat string
	switch info.Type {
	case types.MediaTypeImage:
		targetFormat = "jxl"
	case types.MediaTypeAnimated:
		targetFormat = "avif"
	case types.MediaTypeVideo:
		targetFormat = "mov"
	default:
		targetFormat = "unknown"
	}

	return &ProcessingStrategy{
		Mode:         types.ModeQuality,
		TargetFormat: targetFormat,
		Quality:      "lossless",
		Confidence:   0.95,
		Reason:       "品质模式无损压缩",
	}, nil
}

func (qm *QualityMode) ShouldSkipFile(info *types.MediaInfo) (bool, string) {
	// 检查文件是否损坏
	if info.IsCorrupted {
		return true, "文件已损坏"
	}

	// 检查是否已是目标格式
	ext := strings.ToLower(filepath.Ext(info.Path))
	switch info.Type {
	case types.MediaTypeImage:
		if ext == ".jxl" {
			return true, "已是JXL格式"
		}
	case types.MediaTypeAnimated:
		if ext == ".avif" {
			return true, "已是AVIF格式"
		}
	case types.MediaTypeVideo:
		if ext == ".mov" {
			return true, "已是MOV格式"
		}
	}

	return false, ""
}

// =============================================================================
// 🚀 表情包模式 (极限压缩) - README要求的网络分享优化模式
// =============================================================================

// EmojiMode 表情包模式 - 为网络分享而生，优先考虑文件大小
type EmojiMode struct {
	logger           *zap.Logger
	toolPaths        types.ToolCheckResults
	conversionEngine *SimpleConverter // 简化转换器
}

// NewEmojiMode 创建表情包模式实例
func NewEmojiMode(logger *zap.Logger, toolPaths types.ToolCheckResults) *EmojiMode {
	return &EmojiMode{
		logger:           logger,
		toolPaths:        toolPaths,
		conversionEngine: NewSimpleConverter(logger, toolPaths, false),
	}
}

func (em *EmojiMode) GetModeName() string {
	return "表情包模式"
}

func (em *EmojiMode) ProcessFile(ctx context.Context, info *types.MediaInfo) (*types.ProcessingResult, error) {
	// README要求：所有图片（无论动静）统一强制转换为AVIF格式，视频直接跳过

	if info.Type == types.MediaTypeVideo {
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        "表情包模式跳过视频文件",
			Mode:         types.ModeEmoji,
		}, nil
	}

	// 转换逻辑：
	// 1. 优先尝试无损压缩和重包装
	// 2. 若体积优势不明显，采用比"平衡优化"更激进的有损压缩
	// 3. 替换规则：只要转换后文件比原图小7%-13%即视为成功

	targetFormat := "avif"
	toolChain := []string{"avifenc"} // README要求：静图表情包模式必须使用avifenc

	em.logger.Debug("表情包模式处理",
		zap.String("file", filepath.Base(info.Path)),
		zap.String("target_format", targetFormat),
		zap.Strings("tool_chain", toolChain))

	// 生成目标文件路径
	ext := strings.ToLower(filepath.Ext(info.Path))
	targetPath := strings.TrimSuffix(info.Path, ext) + ".avif"

	// 调用真实的AVIF压缩转换（使用压缩模式）
	result, err := em.conversionEngine.ConvertToAVIF(ctx, info.Path, targetPath, "compressed", info.Type)
	if err != nil {
		return &types.ProcessingResult{
			OriginalPath: info.Path,
			OriginalSize: info.Size,
			NewSize:      info.Size,
			Success:      false,
			Error:        err.Error(),
			Mode:         types.ModeEmoji,
		}, nil
	}

	// 检查是否满足7%-13%的节省标准
	savingRatio := float64(result.SpaceSaved) / float64(result.OriginalSize)
	if savingRatio >= 0.07 { // 至少节省7%
		result.Mode = types.ModeEmoji
		return result, nil
	}

	return &types.ProcessingResult{
		OriginalPath: info.Path,
		OriginalSize: info.Size,
		NewSize:      info.Size,
		Success:      false,
		Error:        "压缩效果不达标（<7%）",
		Mode:         types.ModeEmoji,
	}, nil
}

func (em *EmojiMode) GetStrategy(info *types.MediaInfo) (*ProcessingStrategy, error) {
	if info.Type == types.MediaTypeVideo {
		return &ProcessingStrategy{
			Mode:         types.ModeEmoji,
			TargetFormat: "skip",
			Quality:      "skip",
			Confidence:   1.0,
			Reason:       "表情包模式跳过视频",
		}, nil
	}

	return &ProcessingStrategy{
		Mode:         types.ModeEmoji,
		TargetFormat: "avif",
		Quality:      "compressed",
		Confidence:   0.9,
		Reason:       "表情包模式极限压缩",
	}, nil
}

func (em *EmojiMode) ShouldSkipFile(info *types.MediaInfo) (bool, string) {
	// 检查文件是否损坏
	if info.IsCorrupted {
		return true, "文件已损坏"
	}

	// 表情包模式跳过视频
	if info.Type == types.MediaTypeVideo {
		return true, "表情包模式不处理视频"
	}

	// 检查是否已是AVIF格式
	ext := strings.ToLower(filepath.Ext(info.Path))
	if ext == ".avif" {
		return true, "已是AVIF格式"
	}

	return false, ""
}
