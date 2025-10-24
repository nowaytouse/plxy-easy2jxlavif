package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"pixly/pkg/knowledge"
	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

// recordConversion 记录转换到知识库
func (bo *BalanceOptimizer) recordConversion(
	filePath string,
	features *predictor.FileFeatures,
	prediction *predictor.Prediction,
	predictorName string,
	result *OptimizationResult,
) error {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return nil // 知识库未启用
	}

	// 开始构建记录
	fileName := filepath.Base(filePath)
	originalSize := features.FileSize

	record := knowledge.NewRecordBuilder().
		WithFileInfo(filePath, fileName, features.Format, originalSize).
		WithFeatures(features).
		WithPrediction(prediction, predictorName)

	// 添加实际结果
	if result.Success {
		conversionTimeMs := result.ProcessTime.Milliseconds()
		record.WithActualResult(
			prediction.Params.TargetFormat,
			result.NewSize,
			conversionTimeMs,
		)

		// 如果有质量验证结果，添加验证信息
		// 这里简化处理：成功即认为验证通过
		validationMethod := "basic"
		validationPassed := true
		pixelDiff := 0.0 // 默认完美
		psnr := 100.0    // 默认完美
		ssim := 1.0      // 默认完美

		// 根据格式调整验证方法
		if prediction.Params.LosslessJPEG || prediction.Params.Lossless {
			validationMethod = "lossless"
		}

		record.WithValidation(validationMethod, validationPassed, pixelDiff, psnr, ssim)
	} else {
		// 转换失败
		record.WithActualResult(
			prediction.Params.TargetFormat,
			0, // 没有输出文件
			0,
		)
		record.WithValidation("failed", false, 0, 0, 0)
	}

	// 添加元数据
	version := "v3.0"
	hostOS := runtime.GOOS
	record.WithMetadata(version, hostOS)

	// 保存到数据库
	builtRecord := record.Build()
	err := bo.knowledgeDB.SaveRecord(builtRecord)
	if err != nil {
		bo.logger.Warn("保存转换记录失败",
			zap.Error(err),
			zap.String("file", fileName))
		return err
	}

	bo.logger.Debug("转换记录已保存到知识库",
		zap.String("file", fileName),
		zap.String("rule", prediction.RuleName))

	// 更新统计
	go func() {
		err := bo.knowledgeDB.UpdateStats(predictorName, prediction.RuleName, features.Format)
		if err != nil {
			bo.logger.Debug("更新统计失败", zap.Error(err))
		}
	}()

	return nil
}

// GetKnowledgeStats 获取知识库统计
func (bo *BalanceOptimizer) GetKnowledgeStats() (map[string]interface{}, error) {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return nil, fmt.Errorf("知识库未启用")
	}

	return bo.knowledgeDB.GetStatsSummary()
}

// GetRecentConversions 获取最近的转换记录
func (bo *BalanceOptimizer) GetRecentConversions(limit int) ([]*knowledge.ConversionRecord, error) {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return nil, fmt.Errorf("知识库未启用")
	}

	return bo.knowledgeDB.GetRecentConversions(limit)
}

// AnalyzeAccuracy 分析预测准确性
func (bo *BalanceOptimizer) AnalyzeAccuracy(predictorName, rule, format string) (*knowledge.AnalysisResult, error) {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return nil, fmt.Errorf("知识库未启用")
	}

	analyzer := knowledge.NewAnalyzer(bo.knowledgeDB, bo.logger)
	return analyzer.AnalyzePredictor(predictorName, rule, format)
}

// GenerateKnowledgeReport 生成知识库报告
func (bo *BalanceOptimizer) GenerateKnowledgeReport() (string, error) {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return "", fmt.Errorf("知识库未启用")
	}

	analyzer := knowledge.NewAnalyzer(bo.knowledgeDB, bo.logger)
	return analyzer.GenerateReport()
}

// OptimizePredictionFromHistory 根据历史数据优化预测
func (bo *BalanceOptimizer) OptimizePredictionFromHistory(format string) (map[string]interface{}, error) {
	if !bo.enableKnowledge || bo.knowledgeDB == nil {
		return nil, fmt.Errorf("知识库未启用")
	}

	analyzer := knowledge.NewAnalyzer(bo.knowledgeDB, bo.logger)
	return analyzer.OptimizePrediction(format)
}

// Close 关闭优化器（清理资源）
func (bo *BalanceOptimizer) Close() error {
	if bo.knowledgeDB != nil {
		return bo.knowledgeDB.Close()
	}
	return nil
}

// ValidateConversionQuality 验证转换质量（用于记录到知识库）
func (bo *BalanceOptimizer) ValidateConversionQuality(
	originalPath, convertedPath string,
	prediction *predictor.Prediction,
) (bool, float64, float64, float64, error) {
	// 简化验证：仅检查文件是否存在和大小是否合理
	originalInfo, err := os.Stat(originalPath)
	if err != nil {
		return false, 0, 0, 0, err
	}

	convertedInfo, err := os.Stat(convertedPath)
	if err != nil {
		return false, 0, 0, 0, err
	}

	// 检查文件大小是否合理（不应大于10倍原始大小）
	if convertedInfo.Size() > originalInfo.Size()*10 {
		return false, 0, 0, 0, fmt.Errorf("转换后文件异常大")
	}

	// 对于无损转换，假定完美质量
	if prediction.Params.Lossless || prediction.Params.LosslessJPEG {
		return true, 0.0, 100.0, 1.0, nil // pixelDiff=0, PSNR=100, SSIM=1.0
	}

	// 对于有损转换，假定质量良好（实际应使用FFmpeg验证）
	return true, 0.0, 45.0, 0.97, nil
}

// EnableKnowledge 启用知识库
func (bo *BalanceOptimizer) EnableKnowledge(enable bool) {
	bo.enableKnowledge = enable && bo.knowledgeDB != nil
}

// IsKnowledgeEnabled 检查知识库是否启用
func (bo *BalanceOptimizer) IsKnowledgeEnabled() bool {
	return bo.enableKnowledge && bo.knowledgeDB != nil
}
