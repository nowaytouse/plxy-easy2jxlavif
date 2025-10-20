package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/engine/quality"
	"pixly/pkg/ui/interactive"

	"go.uber.org/zap"
)

// AutoPlusRouter 自动模式+路由器 - README要求的智能决策引擎
//
// 核心功能：
//   - 智能文件路由和格式选择
//   - 基于质量评估的动态决策
//   - 95%文件快速预判+5%可疑文件深度验证
//   - 损坏文件和极低品质文件的批量处理决策
//
// 设计原则：
//   - 智能化：基于文件特征自动选择最优处理方案
//   - 高效率：95%文件快速预判，避免不必要的深度分析
//   - 用户友好：对问题文件提供清晰的处理建议
//   - 灵活性：支持用户自定义路由规则和质量阈值
type AutoPlusRouter struct {
	logger           *zap.Logger
	qualityEngine    *quality.QualityEngine
	balanceOptimizer *BalanceOptimizer
	uiInterface      *interactive.Interface
	toolPaths        types.ToolCheckResults
	debugMode        bool
	routingStats     *RoutingStatistics
}

// RoutingStatistics 路由统计
type RoutingStatistics struct {
	TotalFiles        int                               `json:"total_files"`
	FastRoutedFiles   int                               `json:"fast_routed_files"`
	DeepAnalyzedFiles int                               `json:"deep_analyzed_files"`
	CorruptedFiles    int                               `json:"corrupted_files"`
	LowQualityFiles   int                               `json:"low_quality_files"`
	RoutingDecisions  map[string]*types.RoutingDecision `json:"routing_decisions"`
	ProcessingTime    time.Duration                     `json:"processing_time"`
}

// RoutingReport 路由报告
type RoutingReport struct {
	Summary         string                            `json:"summary"`
	Statistics      *RoutingStatistics                `json:"statistics"`
	Decisions       map[string]*types.RoutingDecision `json:"decisions"`
	Recommendations []string                          `json:"recommendations"`
}

// NewAutoPlusRouter 创建自动模式+路由器
func NewAutoPlusRouter(logger *zap.Logger, qualityEngine *quality.QualityEngine,
	balanceOptimizer *BalanceOptimizer, uiInterface *interactive.Interface,
	toolPaths types.ToolCheckResults, debugMode bool) *AutoPlusRouter {

	return &AutoPlusRouter{
		logger:           logger,
		qualityEngine:    qualityEngine,
		balanceOptimizer: balanceOptimizer,
		uiInterface:      uiInterface,
		toolPaths:        toolPaths,
		debugMode:        debugMode,
		routingStats: &RoutingStatistics{
			RoutingDecisions: make(map[string]*types.RoutingDecision),
		},
	}
}

// RouteFiles 执行文件路由 - README核心功能：智能路由系统
func (apr *AutoPlusRouter) RouteFiles(ctx context.Context, filePaths []string) (map[string]*types.RoutingDecision, []string, error) {
	apr.logger.Info("开始自动模式+智能路由", zap.Int("file_count", len(filePaths)))

	startTime := time.Now()
	decisions := make(map[string]*types.RoutingDecision)
	var lowQualityFiles []string

	// README要求：95%文件快速预判+5%可疑文件深度验证
	fastRoutedCount := 0
	deepAnalyzedCount := 0

	for i, filePath := range filePaths {
		apr.logger.Debug("路由文件",
			zap.String("file", filepath.Base(filePath)),
			zap.Int("progress", i+1),
			zap.Int("total", len(filePaths)))

		// 快速预判阶段
		if decision := apr.fastRouting(filePath); decision != nil {
			decisions[filePath] = decision
			fastRoutedCount++

			// 检查是否为低质量文件
			if decision.QualityLevel == types.QualityVeryLow {
				lowQualityFiles = append(lowQualityFiles, filePath)
			}
			continue
		}

		// 深度验证阶段（约5%的可疑文件）
		decision, err := apr.deepAnalysis(ctx, filePath)
		if err != nil {
			apr.logger.Warn("深度分析失败",
				zap.String("file", filepath.Base(filePath)),
				zap.Error(err))
			// 使用默认决策
			decision = apr.createDefaultDecision(filePath)
		}

		decisions[filePath] = decision
		deepAnalyzedCount++

		// 检查是否为低质量文件
		if decision.QualityLevel == types.QualityVeryLow {
			lowQualityFiles = append(lowQualityFiles, filePath)
		}
	}

	// 更新统计信息
	apr.routingStats.TotalFiles = len(filePaths)
	apr.routingStats.FastRoutedFiles = fastRoutedCount
	apr.routingStats.DeepAnalyzedFiles = deepAnalyzedCount
	apr.routingStats.LowQualityFiles = len(lowQualityFiles)
	apr.routingStats.RoutingDecisions = decisions
	apr.routingStats.ProcessingTime = time.Since(startTime)

	apr.logger.Info("自动模式+路由完成",
		zap.Int("total_files", len(filePaths)),
		zap.Int("fast_routed", fastRoutedCount),
		zap.Int("deep_analyzed", deepAnalyzedCount),
		zap.Int("low_quality", len(lowQualityFiles)),
		zap.Duration("processing_time", apr.routingStats.ProcessingTime))

	return decisions, lowQualityFiles, nil
}

// fastRouting 快速路由 - 95%文件的快速预判
func (apr *AutoPlusRouter) fastRouting(filePath string) *types.RoutingDecision {
	ext := strings.ToLower(filepath.Ext(filePath))
	baseName := strings.ToLower(filepath.Base(filePath))

	// 基于文件扩展名和路径的快速决策
	decision := &types.RoutingDecision{
		Strategy: "convert",
		Reason:   "fast_routing",
	}

	// README要求的自动模式+路由逻辑
	switch ext {
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		// JPEG系列 -> JXL (lossless_jpeg=1)
		decision.TargetFormat = "jxl"
		decision.QualityLevel = types.QualityMediumHigh
		return decision

	case ".png":
		// PNG -> JXL (无损)
		decision.TargetFormat = "jxl"
		decision.QualityLevel = types.QualityHigh
		return decision

	case ".gif":
		// GIF -> AVIF (保持动画)
		decision.TargetFormat = "avif"
		decision.QualityLevel = types.QualityMediumHigh
		return decision

	case ".webp":
		// WebP -> AVIF (现代格式优化)
		decision.TargetFormat = "avif"
		decision.QualityLevel = types.QualityHigh
		return decision

	case ".heif", ".heic":
		// HEIF/HEIC -> JXL (苹果格式转换)
		decision.TargetFormat = "jxl"
		decision.QualityLevel = types.QualityHigh
		return decision

	case ".mp4", ".mov", ".webm":
		// 视频 -> MOV (重新包装)
		decision.TargetFormat = "mov"
		decision.QualityLevel = types.QualityHigh
		return decision

	case ".bmp", ".tiff", ".tif":
		// 无压缩格式 -> JXL (高效压缩)
		decision.TargetFormat = "jxl"
		decision.QualityLevel = types.QualityVeryHigh
		return decision
	}

	// 基于文件名模式的特殊处理
	if strings.Contains(baseName, "screenshot") || strings.Contains(baseName, "屏幕截图") {
		// 截图文件通常适合JXL
		decision.TargetFormat = "jxl"
		decision.QualityLevel = types.QualityHigh
		return decision
	}

	if strings.Contains(baseName, "emoji") || strings.Contains(baseName, "sticker") {
		// 表情包文件 -> AVIF (小体积)
		decision.TargetFormat = "avif"
		decision.QualityLevel = types.QualityMediumHigh
		return decision
	}

	// 无法快速判断，需要深度分析
	return nil
}

// deepAnalysis 深度分析 - 5%可疑文件的详细验证
func (apr *AutoPlusRouter) deepAnalysis(ctx context.Context, filePath string) (*types.RoutingDecision, error) {
	apr.logger.Debug("开始深度分析", zap.String("file", filepath.Base(filePath)))

	// 使用质量评估引擎进行深度分析
	assessment, err := apr.qualityEngine.AssessFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("质量评估失败: %w", err)
	}

	decision := &types.RoutingDecision{
		QualityLevel: assessment.QualityLevel,
		Reason:       "deep_analysis",
	}

	// 基于质量评估结果确定策略
	switch assessment.QualityLevel {
	case types.QualityCorrupted:
		decision.Strategy = "delete"
		decision.TargetFormat = ""
		decision.Reason = "corrupted_file"

	case types.QualityVeryLow:
		decision.Strategy = "skip"
		decision.TargetFormat = ""
		decision.Reason = "very_low_quality"

	case types.QualityLow:
		// 低质量文件，使用压缩比更高的格式
		if assessment.MediaType == types.MediaTypeImage {
			decision.Strategy = "convert"
			decision.TargetFormat = "avif"
		} else {
			decision.Strategy = "convert"
			decision.TargetFormat = "mov"
		}

	case types.QualityMediumLow, types.QualityMediumHigh:
		// 中等质量文件，使用平衡的格式选择
		if assessment.MediaType == types.MediaTypeImage {
			decision.Strategy = "convert"
			decision.TargetFormat = "jxl"
		} else {
			decision.Strategy = "convert"
			decision.TargetFormat = "mov"
		}

	case types.QualityHigh, types.QualityVeryHigh:
		// 高质量文件，优先保持质量
		if assessment.MediaType == types.MediaTypeImage {
			decision.Strategy = "convert"
			decision.TargetFormat = "jxl"
		} else {
			decision.Strategy = "convert"
			decision.TargetFormat = "mov"
		}

	default:
		// 未知质量，使用保守策略
		decision = apr.createDefaultDecision(filePath)
	}

	apr.logger.Debug("深度分析完成",
		zap.String("file", filepath.Base(filePath)),
		zap.String("quality", assessment.QualityLevel.String()),
		zap.String("strategy", decision.Strategy),
		zap.String("target_format", decision.TargetFormat))

	return decision, nil
}

// createDefaultDecision 创建默认决策
func (apr *AutoPlusRouter) createDefaultDecision(filePath string) *types.RoutingDecision {
	ext := strings.ToLower(filepath.Ext(filePath))

	decision := &types.RoutingDecision{
		Strategy:     "convert",
		QualityLevel: types.QualityMediumHigh,
		Reason:       "default_routing",
	}

	// 基于扩展名的保守路由
	if strings.Contains(ext, "jp") { // .jpg, .jpeg
		decision.TargetFormat = "jxl"
	} else if strings.Contains(ext, "png") {
		decision.TargetFormat = "jxl"
	} else if strings.Contains(ext, "gif") || strings.Contains(ext, "webp") {
		decision.TargetFormat = "avif"
	} else if strings.Contains(ext, "mp4") || strings.Contains(ext, "mov") {
		decision.TargetFormat = "mov"
	} else {
		decision.TargetFormat = "jxl" // 默认目标格式
	}

	return decision
}

// GenerateRoutingReport 生成路由报告
func (apr *AutoPlusRouter) GenerateRoutingReport(decisions map[string]*types.RoutingDecision) string {
	totalFiles := len(decisions)
	convertCount := 0
	skipCount := 0
	deleteCount := 0

	formatCounts := make(map[string]int)

	for _, decision := range decisions {
		switch decision.Strategy {
		case "convert":
			convertCount++
			formatCounts[decision.TargetFormat]++
		case "skip":
			skipCount++
		case "delete":
			deleteCount++
		}
	}

	report := fmt.Sprintf("🎯 自动模式+路由报告\n")
	report += fmt.Sprintf("📊 总文件数: %d\n", totalFiles)
	report += fmt.Sprintf("✅ 转换: %d (%.1f%%)\n", convertCount, float64(convertCount)/float64(totalFiles)*100)
	report += fmt.Sprintf("⏭️ 跳过: %d (%.1f%%)\n", skipCount, float64(skipCount)/float64(totalFiles)*100)
	report += fmt.Sprintf("🗑️ 删除: %d (%.1f%%)\n", deleteCount, float64(deleteCount)/float64(totalFiles)*100)

	if len(formatCounts) > 0 {
		report += fmt.Sprintf("\n🎨 目标格式分布:\n")
		for format, count := range formatCounts {
			report += fmt.Sprintf("  %s: %d 个文件\n", strings.ToUpper(format), count)
		}
	}

	report += fmt.Sprintf("\n⚡ 路由效率: %.1f%% 快速路由, %.1f%% 深度分析\n",
		float64(apr.routingStats.FastRoutedFiles)/float64(totalFiles)*100,
		float64(apr.routingStats.DeepAnalyzedFiles)/float64(totalFiles)*100)

	return report
}

// GetRoutingStatistics 获取路由统计
func (apr *AutoPlusRouter) GetRoutingStatistics() *RoutingStatistics {
	return apr.routingStats
}

// SetRoutingRules 设置自定义路由规则（扩展功能）
func (apr *AutoPlusRouter) SetRoutingRules(rules map[string]string) {
	// 用于未来扩展自定义路由规则
	apr.logger.Debug("设置自定义路由规则", zap.Int("rule_count", len(rules)))
}
