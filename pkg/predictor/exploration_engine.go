package predictor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ExplorationEngine 智能探索引擎
// v3.0核心创新：2-3次智能探索，而非5-10次暴力尝试
type ExplorationEngine struct {
	logger     *zap.Logger
	cjxlPath   string
	ffmpegPath string
	tempDir    string
}

// NewExplorationEngine 创建探索引擎
func NewExplorationEngine(logger *zap.Logger, cjxlPath, ffmpegPath, tempDir string) *ExplorationEngine {
	return &ExplorationEngine{
		logger:     logger,
		cjxlPath:   cjxlPath,
		ffmpegPath: ffmpegPath,
		tempDir:    tempDir,
	}
}

// ExploreParams 探索参数组合
// 并行测试2-3个候选，选择最优结果
func (ee *ExplorationEngine) ExploreParams(
	ctx context.Context,
	filePath string,
	candidates []ConversionParams,
	originalSize int64,
) *ExplorationResult {

	startTime := time.Now()

	ee.logger.Info("🔍 开始智能探索",
		zap.String("file", filepath.Base(filePath)),
		zap.Int("candidates", len(candidates)))

	// 限制候选数量（v3.0原则：2-3个而非5-10个）
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}

	// 并行测试所有候选
	results := ee.parallelTest(ctx, filePath, candidates)

	// 选择最优结果
	bestParams, bestSize := ee.selectBest(results, originalSize)

	exploreTime := time.Since(startTime)

	ee.logger.Info("🎯 探索完成",
		zap.String("file", filepath.Base(filePath)),
		zap.Int("tested", len(results)),
		zap.Float64("best_saving", float64(originalSize-bestSize)/float64(originalSize)*100),
		zap.Duration("time", exploreTime))

	return &ExplorationResult{
		BestParams:   bestParams,
		BestSize:     bestSize,
		TestedParams: candidates,
		TestResults:  results,
		ExploreTime:  exploreTime,
	}
}

// parallelTest 并行测试所有候选参数
func (ee *ExplorationEngine) parallelTest(
	ctx context.Context,
	filePath string,
	candidates []ConversionParams,
) map[string]int64 {

	results := make(map[string]int64)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, candidate := range candidates {
		wg.Add(1)

		go func(idx int, params ConversionParams) {
			defer wg.Done()

			// 执行转换
			outputPath, size, err := ee.tryConversion(ctx, filePath, params, idx)

			if err == nil && size > 0 {
				mu.Lock()
				key := ee.paramsKey(params)
				results[key] = size
				mu.Unlock()

				ee.logger.Debug("探索候选完成",
					zap.Int("candidate", idx),
					zap.String("key", key),
					zap.Int64("size", size))
			}

			// 清理临时文件
			if outputPath != "" {
				os.Remove(outputPath)
			}
		}(i, candidate)
	}

	wg.Wait()
	return results
}

// tryConversion 尝试转换
func (ee *ExplorationEngine) tryConversion(
	ctx context.Context,
	filePath string,
	params ConversionParams,
	idx int,
) (string, int64, error) {

	// 生成临时输出路径
	ext := ee.getExtension(params.TargetFormat)
	baseName := filepath.Base(filePath)
	outputPath := filepath.Join(ee.tempDir, fmt.Sprintf("%s_explore_%d%s", baseName, idx, ext))

	var cmd *exec.Cmd

	switch params.TargetFormat {
	case "jxl":
		cmd = ee.buildJXLCommand(ctx, filePath, outputPath, params)
	case "avif":
		cmd = ee.buildAVIFCommand(ctx, filePath, outputPath, params)
	default:
		return "", 0, fmt.Errorf("不支持的格式: %s", params.TargetFormat)
	}

	if err := cmd.Run(); err != nil {
		return outputPath, 0, err
	}

	// 获取文件大小
	stat, err := os.Stat(outputPath)
	if err != nil {
		return outputPath, 0, err
	}

	return outputPath, stat.Size(), nil
}

// buildJXLCommand 构建JXL转换命令
func (ee *ExplorationEngine) buildJXLCommand(
	ctx context.Context,
	inputPath, outputPath string,
	params ConversionParams,
) *exec.Cmd {

	args := []string{
		"-d", fmt.Sprintf("%.1f", params.Distance),
		"-e", fmt.Sprintf("%d", params.Effort),
		inputPath,
		outputPath,
	}

	// 如果是JPEG无损重包装
	if params.LosslessJPEG {
		args = append([]string{"--lossless_jpeg=1"}, args...)
	}

	return exec.CommandContext(ctx, ee.cjxlPath, args...)
}

// buildAVIFCommand 构建AVIF转换命令
func (ee *ExplorationEngine) buildAVIFCommand(
	ctx context.Context,
	inputPath, outputPath string,
	params ConversionParams,
) *exec.Cmd {

	return exec.CommandContext(ctx, ee.ffmpegPath,
		"-i", inputPath,
		"-c:v", "libaom-av1",
		"-crf", fmt.Sprintf("%d", params.CRF),
		"-cpu-used", fmt.Sprintf("%d", params.Speed),
		"-y",
		outputPath)
}

// selectBest 选择最优结果
// 策略：选择文件最小且满足最小节省要求的结果
func (ee *ExplorationEngine) selectBest(results map[string]int64, originalSize int64) (*ConversionParams, int64) {
	const minSizeReduction = 1024  // 最小1KB节省
	const minReductionRatio = 0.05 // 最小5%节省率

	var bestKey string
	var bestSize int64 = originalSize
	bestSaving := 0.0

	for key, size := range results {
		// 检查是否满足最小节省要求
		sizeReduction := originalSize - size
		if sizeReduction < minSizeReduction {
			continue
		}

		reductionRatio := float64(sizeReduction) / float64(originalSize)
		if reductionRatio < minReductionRatio {
			continue
		}

		// 选择节省最多的
		if size < bestSize {
			bestSize = size
			bestKey = key
			bestSaving = reductionRatio * 100
		}
	}

	if bestKey == "" {
		ee.logger.Warn("探索未找到有效结果",
			zap.Int64("original_size", originalSize))
		return nil, 0
	}

	ee.logger.Info("选择最优探索结果",
		zap.String("key", bestKey),
		zap.Float64("saving", bestSaving))

	// 解析最优参数
	bestParams := ee.parseParamsFromKey(bestKey)
	return bestParams, bestSize
}

// paramsKey 生成参数的唯一键
func (ee *ExplorationEngine) paramsKey(params ConversionParams) string {
	if params.TargetFormat == "jxl" {
		if params.LosslessJPEG {
			return fmt.Sprintf("jxl_lossless_jpeg")
		}
		return fmt.Sprintf("jxl_d%.1f_e%d", params.Distance, params.Effort)
	}
	if params.TargetFormat == "avif" {
		return fmt.Sprintf("avif_crf%d", params.CRF)
	}
	return "unknown"
}

// parseParamsFromKey 从键解析参数（简化实现）
func (ee *ExplorationEngine) parseParamsFromKey(key string) *ConversionParams {
	// 简化实现：返回nil，实际使用时应该从results map中查找原始params
	// 这里的实现可以在集成时优化
	return nil
}

// getExtension 获取目标格式的扩展名
func (ee *ExplorationEngine) getExtension(format string) string {
	switch format {
	case "jxl":
		return ".jxl"
	case "avif":
		return ".avif"
	case "webp":
		return ".webp"
	default:
		return ".jxl"
	}
}
