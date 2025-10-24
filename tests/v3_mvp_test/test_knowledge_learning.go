package main

import (
	"fmt"
	"os"
	"runtime"

	"pixly/pkg/knowledge"
	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎯 Pixly v3.0 Week 7-8 - 知识库学习循环测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 步骤1: 创建知识库
	dbPath := "/tmp/pixly_test_knowledge.db"
	os.Remove(dbPath) // 清除旧数据

	db, err := knowledge.NewDatabase(dbPath, logger)
	if err != nil {
		fmt.Printf("❌ 创建知识库失败: %v\n", err)
		return
	}
	defer db.Close()

	fmt.Println("✅ 知识库创建成功")
	fmt.Println()

	// 步骤2: 模拟一些转换记录
	fmt.Println("📝 模拟转换记录...")
	fmt.Println()

	testCases := []struct {
		format     string
		predictor  string
		rule       string
		originalMB float64
		savingPct  float64
		perfect    bool
	}{
		// PNG测试
		{"png", "PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", 1.2, 0.65, true},
		{"png", "PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", 3.5, 0.70, true},
		{"png", "PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", 0.8, 0.60, true},
		{"png", "PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", 5.2, 0.68, true},
		{"png", "PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", 2.1, 0.72, true},

		// JPEG测试
		{"jpg", "JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", 2.4, 0.25, true},
		{"jpg", "JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", 1.8, 0.28, true},
		{"jpg", "JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", 3.6, 0.22, true},
		{"jpg", "JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", 0.5, 0.30, true},

		// GIF动图测试
		{"gif", "GIFPredictor", "GIF_ANIMATED_AVIF", 1.5, 0.75, false},
		{"gif", "GIFPredictor", "GIF_ANIMATED_AVIF", 2.8, 0.80, false},
		{"gif", "GIFPredictor", "GIF_ANIMATED_AVIF", 0.6, 0.70, false},
	}

	for i, tc := range testCases {
		originalSize := int64(tc.originalMB * 1024 * 1024)
		actualSize := int64(float64(originalSize) * (1 - tc.savingPct))

		record := knowledge.NewRecordBuilder().
			WithFileInfo(
				fmt.Sprintf("/test/file%d.%s", i, tc.format),
				fmt.Sprintf("file%d.%s", i, tc.format),
				tc.format,
				originalSize,
			).
			WithFeatures(&predictor.FileFeatures{
				Format:   tc.format,
				FileSize: originalSize,
				Width:    1920,
				Height:   1080,
			}).
			WithPrediction(&predictor.Prediction{
				Params: &predictor.ConversionParams{
					TargetFormat: "jxl",
					Lossless:     true,
					Distance:     0,
					Effort:       7,
				},
				RuleName:          tc.rule,
				Confidence:        0.95,
				ExpectedSaving:    tc.savingPct * 0.9, // 预测稍保守
				ExpectedSizeBytes: int64(float64(originalSize) * (1 - tc.savingPct*0.9)),
			}, tc.predictor).
			WithActualResult("jxl", actualSize, 150).
			WithValidation("lossless", tc.perfect, 0, 100, 1.0).
			WithMetadata("v3.0-test", runtime.GOOS).
			Build()

		err = db.SaveRecord(record)
		if err != nil {
			fmt.Printf("❌ 保存记录失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ [%s] %s → 节省%.1f%%\n", tc.format, tc.rule, tc.savingPct*100)
	}

	fmt.Println()

	// 步骤3: 更新统计
	fmt.Println("📊 更新统计...")
	fmt.Println()

	for _, tc := range []struct{ predictor, rule, format string }{
		{"PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", "png"},
		{"JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", "jpg"},
		{"GIFPredictor", "GIF_ANIMATED_AVIF", "gif"},
	} {
		err = db.UpdateStats(tc.predictor, tc.rule, tc.format)
		if err != nil {
			fmt.Printf("❌ 更新统计失败: %v\n", err)
		}
	}

	fmt.Println("✅ 统计更新完成")
	fmt.Println()

	// 步骤4: 查询统计摘要
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📈 知识库统计摘要")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	summary, err := db.GetStatsSummary()
	if err != nil {
		fmt.Printf("❌ 获取统计摘要失败: %v\n", err)
		return
	}

	fmt.Printf("  总转换次数: %v\n", summary["total_conversions"])
	fmt.Printf("  平均空间节省: %.1f%%\n", summary["avg_saving_percent"])
	fmt.Printf("  质量通过率: %.1f%%\n", summary["quality_pass_rate"])
	fmt.Printf("  平均预测误差: %.1f%%\n", summary["avg_prediction_error"])
	fmt.Println()

	// 步骤5: 分析预测准确性
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 预测准确性分析")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	analyzer := knowledge.NewAnalyzer(db, logger)

	for _, tc := range []struct{ predictor, rule, format string }{
		{"PNGPredictor", "PNG_ALWAYS_JXL_LOSSLESS", "png"},
		{"JPEGPredictor", "JPEG_ALWAYS_JXL_LOSSLESS", "jpg"},
		{"GIFPredictor", "GIF_ANIMATED_AVIF", "gif"},
	} {
		result, err := analyzer.AnalyzePredictor(tc.predictor, tc.rule, tc.format)
		if err != nil {
			fmt.Printf("  ❌ 分析失败 [%s]: %v\n", tc.format, err)
			continue
		}

		fmt.Printf("  [%s] %s\n", tc.format, tc.rule)
		fmt.Printf("    样本数: %d\n", result.TotalSamples)
		fmt.Printf("    成功率: %.1f%%\n", result.SuccessRate*100)
		fmt.Printf("    平均预测误差: %.1f%%\n", result.AvgPredictionError*100)
		fmt.Printf("    平均实际节省: %.1f%%\n", result.AvgActualSaving*100)
		fmt.Printf("    完美质量率: %.1f%%\n", result.PerfectQualityRate*100)

		if len(result.Recommendations) > 0 {
			fmt.Println("    建议:")
			for _, rec := range result.Recommendations {
				fmt.Printf("      %s\n", rec)
			}
		}
		fmt.Println()
	}

	// 步骤6: 生成完整报告
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 完整分析报告")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	report, err := analyzer.GenerateReport()
	if err != nil {
		fmt.Printf("❌ 生成报告失败: %v\n", err)
		return
	}

	fmt.Println(report)

	// 步骤7: 查询API测试
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔎 查询API测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	queryAPI := knowledge.NewQueryAPI(db)

	// 获取PNG的最佳转换
	bestPNG, err := queryAPI.GetBestConversions("png", 3)
	if err == nil && len(bestPNG) > 0 {
		fmt.Println("  📊 PNG最佳转换 (Top 3):")
		for i, record := range bestPNG {
			fmt.Printf("    %d. %s → 节省%.1f%%\n",
				i+1, record.FileName, record.ActualSavingPercent*100)
		}
		fmt.Println()
	}

	// 获取聚合统计
	for _, format := range []string{"png", "jpg", "gif"} {
		stats, err := queryAPI.GetAggregateStats(format)
		if err == nil {
			fmt.Printf("  [%s] 聚合统计:\n", format)
			fmt.Printf("    记录数: %d\n", stats.TotalRecords)
			fmt.Printf("    平均节省: %.1f%%\n", stats.AvgSavingPercent*100)
			fmt.Printf("    质量通过率: %.1f%%\n", stats.QualityPassRate)
			fmt.Println()
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 知识库学习循环测试完成！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Printf("数据库位置: %s\n", dbPath)
	dbSize := getFileSize(dbPath)
	fmt.Printf("数据库大小: %.2f KB\n", float64(dbSize)/1024)
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
