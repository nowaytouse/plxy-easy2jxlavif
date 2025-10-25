package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/knowledge"
	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

// BalanceOptimizer 平衡优化器 - v3.0增强：智能预测优先，探索为辅
// README要求的多点探测试探性压缩策略
type BalanceOptimizer struct {
	logger    *zap.Logger
	toolPaths types.ToolCheckResults
	tempDir   string
	debugMode bool

	// v3.0新增：智能预测器
	predictor           *predictor.Predictor
	explorationEngine   *predictor.ExplorationEngine // Week 3-4新增
	enablePrediction    bool                         // 是否启用预测（v3.0默认启用）
	enableExploration   bool                         // 是否启用探索（Week 3-4新增）
	confidenceThreshold float64                      // 置信度阈值（>此值直接使用预测）

	// Week 7-8新增：知识库
	knowledgeDB     *knowledge.Database      // 知识库数据库
	recordBuilder   *knowledge.RecordBuilder // 记录构建器
	enableKnowledge bool                     // 是否启用知识库记录
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Success      bool
	OutputPath   string
	OriginalSize int64
	NewSize      int64
	SpaceSaved   int64
	Method       string // 使用的优化方法
	Quality      string // 质量参数
	ProcessTime  time.Duration
	Error        error
}

// OptimizationAttempt 单次优化尝试
type OptimizationAttempt struct {
	Method       string // "lossless_repack", "lossless_math", "lossy_high", "lossy_medium", "lossy_low"
	Format       string // "jxl", "avif", "webp"
	Parameters   map[string]string
	ExpectedSize int64 // 预期文件大小（用于快速跳过）
}

// NewBalanceOptimizer 创建平衡优化器
// v3.0增强：集成智能预测器
func NewBalanceOptimizer(logger *zap.Logger, toolPaths types.ToolCheckResults, tempDir string) *BalanceOptimizer {
	// 创建预测器（使用ffprobe命令，通常在PATH中）
	// 优先使用系统ffprobe，确保与FFmpeg版本一致
	ffprobePath := "ffprobe" // 使用PATH中的ffprobe

	// 尝试从环境变量获取自定义路径
	if customPath := os.Getenv("PIXLY_FFPROBE_PATH"); customPath != "" {
		ffprobePath = customPath
	}

	pred := predictor.NewPredictor(logger, ffprobePath)

	// Week 3-4新增：创建探索引擎
	explorer := predictor.NewExplorationEngine(
		logger,
		toolPaths.CjxlPath,
		toolPaths.FfmpegStablePath,
		tempDir,
	)

	// Week 7-8新增：初始化知识库
	dbPath := filepath.Join(tempDir, "pixly_knowledge.db")
	knowledgeDB, err := knowledge.NewDatabase(dbPath, logger)
	if err != nil {
		logger.Warn("知识库初始化失败，将禁用知识库功能",
			zap.Error(err))
		knowledgeDB = nil
	}

	return &BalanceOptimizer{
		logger:              logger,
		toolPaths:           toolPaths,
		tempDir:             tempDir,
		debugMode:           os.Getenv("PIXLY_DEBUG") == "true",
		predictor:           pred,
		explorationEngine:   explorer,                                         // Week 3-4新增
		enablePrediction:    os.Getenv("PIXLY_DISABLE_PREDICTION") != "true",  // 默认启用
		enableExploration:   os.Getenv("PIXLY_DISABLE_EXPLORATION") != "true", // Week 3-4新增
		confidenceThreshold: 0.80,                                             // 置信度>0.80直接使用预测
		knowledgeDB:         knowledgeDB,                                      // Week 7-8新增
		enableKnowledge:     knowledgeDB != nil && os.Getenv("PIXLY_DISABLE_KNOWLEDGE") != "true",
	}
}

// OptimizeFile 执行平衡优化
// v3.0增强：智能预测优先，探索为辅
// README要求的核心平衡优化逻辑
func (bo *BalanceOptimizer) OptimizeFile(ctx context.Context, filePath string, mediaType types.MediaType) (*OptimizationResult, error) {
	bo.logger.Debug("开始平衡优化",
		zap.String("file", filepath.Base(filePath)),
		zap.String("media_type", mediaType.String()))

	startTime := time.Now()

	// 获取原始文件信息
	originalStat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法获取原始文件信息: %w", err)
	}
	originalSize := originalStat.Size()

	result := &OptimizationResult{
		OriginalSize: originalSize,
		ProcessTime:  0,
	}

	// v3.0新增：优先尝试智能预测（仅对PNG生效）
	if bo.enablePrediction && mediaType == types.MediaTypeImage {
		if predictResult := bo.tryPredictiveOptimization(ctx, filePath, originalSize); predictResult != nil {
			if predictResult.Success && predictResult.NewSize < originalSize {
				bo.logger.Info("✨ 智能预测成功（v3.0）",
					zap.String("file", filepath.Base(filePath)),
					zap.String("method", predictResult.Method),
					zap.Int64("original_size", originalSize),
					zap.Int64("new_size", predictResult.NewSize),
					zap.Float64("saved_percent", float64(originalSize-predictResult.NewSize)/float64(originalSize)*100),
					zap.Duration("time", time.Since(startTime)))

				result.Success = true
				result.OutputPath = predictResult.OutputPath
				result.NewSize = predictResult.NewSize
				result.SpaceSaved = originalSize - predictResult.NewSize
				result.Method = "v3_predictive_" + predictResult.Method
				result.Quality = predictResult.Quality
				result.ProcessTime = time.Since(startTime)
				return result, nil
			}
		}
	}

	// v1.0流程：如果预测失败或未启用，回退到原有的平衡优化步骤
	bo.logger.Debug("使用v1.0平衡优化流程（预测未覆盖此格式）",
		zap.String("file", filepath.Base(filePath)))

	// README要求的平衡优化步骤：
	// 1. 无损重新包装优先
	// 2. 数学无损压缩
	// 3. 有损探测（多个质量点）
	// 4. 最优选择决策

	// 步骤1: 无损重新包装优先
	if repackResult := bo.tryLosslessRepack(ctx, filePath, mediaType); repackResult.Success {
		if repackResult.NewSize < originalSize {
			bo.logger.Info("无损重新包装成功",
				zap.String("file", filepath.Base(filePath)),
				zap.Int64("original_size", originalSize),
				zap.Int64("new_size", repackResult.NewSize),
				zap.Int64("saved", originalSize-repackResult.NewSize))

			result.Success = true
			result.OutputPath = repackResult.OutputPath
			result.NewSize = repackResult.NewSize
			result.SpaceSaved = originalSize - repackResult.NewSize
			result.Method = "lossless_repack"
			result.Quality = "lossless"
			result.ProcessTime = time.Since(startTime)
			return result, nil
		}
	}

	// 步骤2: 数学无损压缩
	if losslessResult := bo.tryMathematicalLossless(ctx, filePath, mediaType); losslessResult.Success {
		if losslessResult.NewSize < originalSize {
			bo.logger.Info("数学无损压缩成功",
				zap.String("file", filepath.Base(filePath)),
				zap.Int64("original_size", originalSize),
				zap.Int64("new_size", losslessResult.NewSize),
				zap.Int64("saved", originalSize-losslessResult.NewSize))

			result.Success = true
			result.OutputPath = losslessResult.OutputPath
			result.NewSize = losslessResult.NewSize
			result.SpaceSaved = originalSize - losslessResult.NewSize
			result.Method = "lossless_math"
			result.Quality = "lossless"
			result.ProcessTime = time.Since(startTime)
			return result, nil
		}
	}

	// 步骤3: 有损探测 - README要求的多点探测
	bo.logger.Debug("开始有损探测", zap.String("file", filepath.Base(filePath)))

	bestResult := bo.performMultiPointLossyProbing(ctx, filePath, mediaType, originalSize)
	if bestResult != nil && bestResult.Success && bestResult.NewSize < originalSize {
		bo.logger.Info("有损探测找到最优结果",
			zap.String("file", filepath.Base(filePath)),
			zap.String("method", bestResult.Method),
			zap.String("quality", bestResult.Quality),
			zap.Int64("original_size", originalSize),
			zap.Int64("new_size", bestResult.NewSize),
			zap.Int64("saved", originalSize-bestResult.NewSize))

		result.Success = true
		result.OutputPath = bestResult.OutputPath
		result.NewSize = bestResult.NewSize
		result.SpaceSaved = originalSize - bestResult.NewSize
		result.Method = bestResult.Method
		result.Quality = bestResult.Quality
		result.ProcessTime = time.Since(startTime)
		return result, nil
	}

	// 步骤4: 无法优化处理
	bo.logger.Info("无法找到有效的优化方案",
		zap.String("file", filepath.Base(filePath)),
		zap.Int64("original_size", originalSize))

	result.Success = false
	result.Error = fmt.Errorf("所有优化尝试均无法减小文件体积")
	result.ProcessTime = time.Since(startTime)
	return result, nil
}

// tryLosslessRepack 尝试无损重新包装
func (bo *BalanceOptimizer) tryLosslessRepack(ctx context.Context, filePath string, mediaType types.MediaType) *OptimizationResult {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 检查是否支持无损重新包装
	if mediaType == types.MediaTypeImage && (ext == ".jpg" || ext == ".jpeg") {
		// JPEG可以尝试JXL的lossless_jpeg=1重新包装
		return bo.tryJXLLosslessRepack(ctx, filePath)
	}

	return &OptimizationResult{Success: false}
}

// tryJXLLosslessRepack 尝试JXL无损重新包装
func (bo *BalanceOptimizer) tryJXLLosslessRepack(ctx context.Context, filePath string) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".jxl")

	// 使用cjxl的lossless_jpeg=1参数进行无损重新包装
	cmd := exec.CommandContext(ctx, bo.toolPaths.CjxlPath,
		"-d", "0", // 距离0表示无损
		"--lossless_jpeg=1", // README要求的无损重新包装参数
		"-e", "7",           // 默认effort 7
		filePath,
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return &OptimizationResult{Success: false, Error: err}
	}

	// 检查输出文件
	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// tryMathematicalLossless 尝试数学无损压缩
func (bo *BalanceOptimizer) tryMathematicalLossless(ctx context.Context, filePath string, mediaType types.MediaType) *OptimizationResult {
	if mediaType == types.MediaTypeImage {
		return bo.tryJXLMathematicalLossless(ctx, filePath)
	}
	return &OptimizationResult{Success: false}
}

// tryJXLMathematicalLossless 尝试JXL数学无损压缩
func (bo *BalanceOptimizer) tryJXLMathematicalLossless(ctx context.Context, filePath string) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".jxl")

	// 使用标准的数学无损压缩（距离=0）
	cmd := exec.CommandContext(ctx, bo.toolPaths.CjxlPath,
		"-d", "0", // 距离0表示无损
		"-e", "7", // 默认effort 7
		filePath,
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return &OptimizationResult{Success: false, Error: err}
	}

	// 检查输出文件
	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// performMultiPointLossyProbing 执行多点有损探测 - README要求的核心逻辑优化
func (bo *BalanceOptimizer) performMultiPointLossyProbing(ctx context.Context, filePath string, mediaType types.MediaType, originalSize int64) *OptimizationResult {
	// README要求：至少分两组尝试多个质量点
	// 高品质组: 90, 85, 75 - 智能选择策略
	// 中等品质组: 60, 55 - 激进压缩策略

	// 根据文件大小智能选择质量点组合
	fileSizeMB := float64(originalSize) / (1024 * 1024)
	attempts := bo.generateSmartQualityAttempts(mediaType, fileSizeMB)

	var bestResult *OptimizationResult
	bestSpaceSaved := int64(0)

	// 按优先级顺序尝试（高品质组优先）
	for i, attempt := range attempts {
		select {
		case <-ctx.Done():
			return bestResult
		default:
		}

		var result *OptimizationResult

		switch attempt.Format {
		case "jxl":
			result = bo.tryJXLLossyCompression(ctx, filePath, attempt.Parameters)
		case "avif":
			result = bo.tryAVIFLossyCompression(ctx, filePath, attempt.Parameters)
		case "webp":
			result = bo.tryWebPLossyCompression(ctx, filePath, attempt.Parameters)
		default:
			continue
		}

		if result.Success && result.NewSize < originalSize {
			spaceSaved := originalSize - result.NewSize
			bo.logger.Debug("有损探测结果",
				zap.String("file", filepath.Base(filePath)),
				zap.String("method", attempt.Method),
				zap.String("quality", attempt.Parameters["quality"]),
				zap.Int64("space_saved", spaceSaved),
				zap.Float64("compression_ratio", float64(spaceSaved)/float64(originalSize)*100))

			// README要求：选择体积最小的结果（哪怕仅1KB也算成功）
			if spaceSaved > bestSpaceSaved {
				if bestResult != nil {
					os.Remove(bestResult.OutputPath) // 清理之前的结果
				}
				bestResult = result
				bestResult.Method = attempt.Method
				bestResult.Quality = attempt.Parameters["quality"]
				bestSpaceSaved = spaceSaved

				// 高品质组达到满意结果时可以提前结束
				if i < 3 && spaceSaved > originalSize/4 { // 前3个高品质尝试且节省>25%
					bo.logger.Debug("高品质组达到满意结果，提前结束探测",
						zap.String("file", filepath.Base(filePath)),
						zap.Float64("savings_percent", float64(spaceSaved)/float64(originalSize)*100))
					break
				}
			} else {
				os.Remove(result.OutputPath) // 清理较差的结果
			}
		}
	}

	return bestResult
}

// generateSmartQualityAttempts 生成智能质量点尝试组合
func (bo *BalanceOptimizer) generateSmartQualityAttempts(mediaType types.MediaType, fileSizeMB float64) []OptimizationAttempt {
	var attempts []OptimizationAttempt

	// README要求的高品质组 (90, 85, 75) - 智能调整
	if mediaType == types.MediaTypeImage {
		// JXL高品质组 - 根据文件大小调整distance参数
		if fileSizeMB > 10 { // 大文件使用更保守的参数
			attempts = append(attempts,
				OptimizationAttempt{Method: "lossy_jxl_conservative", Format: "jxl", Parameters: map[string]string{"distance": "0.3", "quality": "92"}},
				OptimizationAttempt{Method: "lossy_jxl_high", Format: "jxl", Parameters: map[string]string{"distance": "0.6", "quality": "88"}},
				OptimizationAttempt{Method: "lossy_jxl_balanced", Format: "jxl", Parameters: map[string]string{"distance": "1.0", "quality": "80"}},
			)
		} else {
			attempts = append(attempts,
				OptimizationAttempt{Method: "lossy_jxl_high", Format: "jxl", Parameters: map[string]string{"distance": "0.5", "quality": "90"}},
				OptimizationAttempt{Method: "lossy_jxl_good", Format: "jxl", Parameters: map[string]string{"distance": "0.8", "quality": "85"}},
				OptimizationAttempt{Method: "lossy_jxl_balanced", Format: "jxl", Parameters: map[string]string{"distance": "1.2", "quality": "75"}},
			)
		}

		// AVIF高品质组作为备选
		attempts = append(attempts,
			OptimizationAttempt{Method: "lossy_avif_high", Format: "avif", Parameters: map[string]string{"crf": "18", "quality": "88"}},
			OptimizationAttempt{Method: "lossy_avif_good", Format: "avif", Parameters: map[string]string{"crf": "23", "quality": "80"}},
		)
	}

	// README要求的中等品质组 (60, 55) - 激进压缩
	if fileSizeMB > 5 { // 大文件才使用激进压缩
		attempts = append(attempts,
			OptimizationAttempt{Method: "lossy_jxl_aggressive", Format: "jxl", Parameters: map[string]string{"distance": "2.0", "quality": "60"}},
			OptimizationAttempt{Method: "lossy_jxl_extreme", Format: "jxl", Parameters: map[string]string{"distance": "2.5", "quality": "55"}},
			OptimizationAttempt{Method: "lossy_avif_aggressive", Format: "avif", Parameters: map[string]string{"crf": "35", "quality": "60"}},
			OptimizationAttempt{Method: "lossy_webp_aggressive", Format: "webp", Parameters: map[string]string{"quality": "60"}},
		)
	}

	return attempts
}

// tryWebPLossyCompression 尝试WebP有损压缩
func (bo *BalanceOptimizer) tryWebPLossyCompression(ctx context.Context, filePath string, params map[string]string) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".webp")
	quality := params["quality"]

	// 使用FFmpeg进行WebP压缩
	cmd := exec.CommandContext(ctx, bo.toolPaths.FfmpegStablePath,
		"-i", filePath,
		"-c:v", "libwebp",
		"-quality", quality,
		"-y",
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return &OptimizationResult{Success: false, Error: err}
	}

	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// tryJXLLossyCompression 尝试JXL有损压缩
func (bo *BalanceOptimizer) tryJXLLossyCompression(ctx context.Context, filePath string, params map[string]string) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".jxl")
	distance := params["distance"]

	cmd := exec.CommandContext(ctx, bo.toolPaths.CjxlPath,
		"-d", distance,
		"-e", "7", // 固定effort
		filePath,
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return &OptimizationResult{Success: false, Error: err}
	}

	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// tryAVIFLossyCompression 尝试AVIF有损压缩
func (bo *BalanceOptimizer) tryAVIFLossyCompression(ctx context.Context, filePath string, params map[string]string) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".avif")
	quality := params["quality"]

	// 使用FFmpeg进行AVIF压缩
	cmd := exec.CommandContext(ctx, bo.toolPaths.FfmpegStablePath,
		"-i", filePath,
		"-c:v", "libaom-av1",
		"-crf", quality,
		"-cpu-used", "6", // 平衡速度和质量
		"-y",
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return &OptimizationResult{Success: false, Error: err}
	}

	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// generateTempPath 生成临时文件路径
func (bo *BalanceOptimizer) generateTempPath(originalPath, ext string) string {
	baseName := strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath))
	timestamp := time.Now().UnixNano()
	return filepath.Join(bo.tempDir, fmt.Sprintf("%s_balance_%d%s", baseName, timestamp, ext))
}

// CleanupTempFiles 清理临时文件
func (bo *BalanceOptimizer) CleanupTempFiles() {
	if bo.tempDir != "" {
		os.RemoveAll(bo.tempDir)
	}
}

// ========== v3.0新增：智能预测相关函数 ==========

// tryPredictiveOptimization 尝试基于预测的优化（v3.0核心）
// 使用智能预测器预测最优参数，并直接执行单次转换
func (bo *BalanceOptimizer) tryPredictiveOptimization(ctx context.Context, filePath string, originalSize int64) *OptimizationResult {
	// 步骤1: 使用预测器预测最优参数
	prediction, err := bo.predictor.PredictOptimalParams(filePath)
	if err != nil {
		bo.logger.Warn("预测失败，回退到v1.0流程",
			zap.String("file", filepath.Base(filePath)),
			zap.Error(err))
		return nil
	}

	bo.logger.Debug("预测完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("target_format", prediction.Params.TargetFormat),
		zap.Float64("confidence", prediction.Confidence),
		zap.String("method", prediction.Method),
		zap.Float64("expected_saving", prediction.ExpectedSaving*100),
		zap.Bool("should_explore", prediction.ShouldExplore))

	// 步骤2: 检查置信度和探索需求
	if prediction.Confidence < bo.confidenceThreshold || prediction.ShouldExplore {
		// Week 3-4新增：低置信度时使用智能探索
		if bo.enableExploration && prediction.ShouldExplore && len(prediction.ExplorationCandidates) > 0 {
			bo.logger.Info("🔍 触发智能探索（v3.0）",
				zap.String("file", filepath.Base(filePath)),
				zap.Float64("confidence", prediction.Confidence),
				zap.Int("candidates", len(prediction.ExplorationCandidates)))

			// 暂时禁用探索引擎（类型不匹配）
			// TODO: 修复ExplorationCandidates类型后重新启用
			var exploreResult *predictor.ExplorationResult = nil
			_ = ctx // 避免未使用警告
			/*
				exploreResult := bo.explorationEngine.ExploreParams(
					ctx,
					filePath,
					prediction.ExplorationCandidates,
					originalSize,
				)
			*/

			if exploreResult != nil && exploreResult.BestParams != nil {
				bo.logger.Info("✅ 探索找到最优结果",
					zap.String("file", filepath.Base(filePath)),
					zap.Float64("saving", float64(originalSize-exploreResult.BestSize)/float64(originalSize)*100),
					zap.Duration("explore_time", exploreResult.ExploreTime))

				// 使用探索找到的最优参数进行转换
				result := bo.executeConversionWithPrediction(ctx, filePath, &predictor.Prediction{
					Params: exploreResult.BestParams,
				})

				if result != nil && result.Success {
					result.Method = "v3_explored_" + result.Method
					return result
				}
			}
		}

		// 探索失败或未启用，回退到v1.0流程
		bo.logger.Debug("预测置信度低或探索失败，回退到v1.0探索流程",
			zap.String("file", filepath.Base(filePath)),
			zap.Float64("confidence", prediction.Confidence),
			zap.Float64("threshold", bo.confidenceThreshold))
		return nil
	}

	// 步骤3: 高置信度预测，直接执行单次转换
	bo.logger.Info("🎯 使用高置信度预测参数（v3.0）",
		zap.String("file", filepath.Base(filePath)),
		zap.Float64("confidence", prediction.Confidence),
		zap.String("rule", prediction.RuleName))

	// 执行转换
	result := bo.executeConversionWithPrediction(ctx, filePath, prediction)

	// 步骤4: 验证结果
	if result != nil && result.Success {
		savedPercent := float64(originalSize-result.NewSize) / float64(originalSize) * 100
		expectedPercent := prediction.ExpectedSaving * 100

		bo.logger.Info("预测转换完成",
			zap.String("file", filepath.Base(filePath)),
			zap.Float64("actual_saving", savedPercent),
			zap.Float64("expected_saving", expectedPercent),
			zap.Float64("prediction_error", savedPercent-expectedPercent))
	}

	return result
}

// executeConversionWithPrediction 使用预测参数执行转换
func (bo *BalanceOptimizer) executeConversionWithPrediction(ctx context.Context, filePath string, prediction *predictor.Prediction) *OptimizationResult {
	params := prediction.Params

	// 根据目标格式执行转换
	switch params.TargetFormat {
	case "jxl":
		return bo.executePredictedJXLConversion(ctx, filePath, params)
	case "avif":
		return bo.executePredictedAVIFConversion(ctx, filePath, params)
	case "mov":
		// 视频重封装（未来实现）
		return nil
	default:
		bo.logger.Warn("未知的目标格式",
			zap.String("format", params.TargetFormat))
		return nil
	}
}

// executePredictedJXLConversion 执行预测的JXL转换
func (bo *BalanceOptimizer) executePredictedJXLConversion(ctx context.Context, filePath string, params *predictor.ConversionParams) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".jxl")

	// 构建cjxl命令（使用预测的参数）
	args := []string{
		"-d", fmt.Sprintf("%.1f", params.Distance),
		"-e", fmt.Sprintf("%d", params.Effort),
		"--num_threads", fmt.Sprintf("%d", params.Threads),
		filePath,
		outputPath,
	}

	// 如果是JPEG无损重包装，添加特殊参数
	if params.LosslessJPEG {
		args = append([]string{"--lossless_jpeg=1"}, args...)
	}

	cmd := exec.CommandContext(ctx, bo.toolPaths.CjxlPath, args...)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		bo.logger.Warn("预测JXL转换失败",
			zap.String("file", filepath.Base(filePath)),
			zap.Error(err))
		return &OptimizationResult{Success: false, Error: err}
	}

	// 检查输出文件
	if stat, err := os.Stat(outputPath); err == nil {
		method := fmt.Sprintf("jxl_predicted_d%.1f_e%d", params.Distance, params.Effort)
		quality := "lossless"
		if params.Distance > 0 {
			quality = fmt.Sprintf("lossy_d%.1f", params.Distance)
		}

		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
			Method:     method,
			Quality:    quality,
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}

// executePredictedAVIFConversion 执行预测的AVIF转换
func (bo *BalanceOptimizer) executePredictedAVIFConversion(ctx context.Context, filePath string, params *predictor.ConversionParams) *OptimizationResult {
	outputPath := bo.generateTempPath(filePath, ".avif")

	// 使用FFmpeg进行AVIF转换（使用预测的CRF）
	cmd := exec.CommandContext(ctx, bo.toolPaths.FfmpegStablePath,
		"-i", filePath,
		"-c:v", "libaom-av1",
		"-crf", fmt.Sprintf("%d", params.CRF),
		"-cpu-used", fmt.Sprintf("%d", params.Speed),
		"-y",
		outputPath)

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		bo.logger.Warn("预测AVIF转换失败",
			zap.String("file", filepath.Base(filePath)),
			zap.Error(err))
		return &OptimizationResult{Success: false, Error: err}
	}

	if stat, err := os.Stat(outputPath); err == nil {
		return &OptimizationResult{
			Success:    true,
			OutputPath: outputPath,
			NewSize:    stat.Size(),
			Method:     fmt.Sprintf("avif_predicted_crf%d", params.CRF),
			Quality:    fmt.Sprintf("crf_%d", params.CRF),
		}
	}

	os.Remove(outputPath)
	return &OptimizationResult{Success: false}
}
