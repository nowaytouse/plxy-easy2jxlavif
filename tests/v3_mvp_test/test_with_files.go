package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

func main() {
	// 创建logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 使用之前glob搜索找到的PNG文件（来自TESTPACK）
	testFiles := []string{
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/黑白起稿.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/蘑菇老师答疑 整理.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/肌配色の組み合わせ.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/绘画.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/空气透视规律.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/烟囱修正提示.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/五分之一分段画法.jpg.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/两边小屋步骤1.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/上色7.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/上色3.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/上色2.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/上色1.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/psc.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/49908524_p0.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/4c72ae5eaa1f3ad0e2fab48c4283c57f.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/3-1.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/20191225192302.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/2-1.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/050.png",
		"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/048.png",
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎯 Pixly v3.0 MVP - PNG智能预测器测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📦 测试文件数: %d\n", len(testFiles))
	fmt.Println()

	// 创建预测器
	pred := predictor.NewPredictor(logger, "ffprobe")

	// 统计数据
	totalPredictTime := time.Duration(0)
	totalFeatureTime := time.Duration(0)
	successCount := 0
	totalExpectedSaving := 0.0
	totalActualSize := int64(0)
	totalExpectedSize := int64(0)

	// 测试每个文件
	for i, filePath := range testFiles {
		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] ⏭️  跳过（文件不存在）: %s\n\n", i+1, len(testFiles), filepath.Base(filePath))
			continue
		}

		fmt.Printf("[%d/%d] %s\n", i+1, len(testFiles), filepath.Base(filePath))

		featureStart := time.Now()
		features, err := pred.GetFeatures(filePath)
		if err != nil {
			fmt.Printf("  ❌ 特征提取失败: %v\n\n", err)
			continue
		}
		featureTime := time.Since(featureStart)
		totalFeatureTime += featureTime

		// 预测
		prediction, err := pred.PredictOptimalParams(filePath)
		if err != nil {
			fmt.Printf("  ❌ 预测失败: %v\n\n", err)
			continue
		}

		successCount++
		totalPredictTime += prediction.PredictionTime
		totalExpectedSaving += prediction.ExpectedSaving
		totalActualSize += features.FileSize
		totalExpectedSize += prediction.ExpectedSizeBytes

		// 显示结果
		fmt.Printf("  ✅ 预测成功 (耗时: %v)\n", featureTime+prediction.PredictionTime)
		fmt.Printf("     尺寸: %dx%d | 大小: %.2f MB | Alpha: %v\n",
			features.Width, features.Height,
			float64(features.FileSize)/(1024*1024),
			features.HasAlpha)
		fmt.Printf("     PixFmt: %s | Bytes/Pixel: %.4f\n",
			features.PixFmt, features.BytesPerPixel)
		fmt.Printf("  🎯 预测: JXL distance=%.1f effort=%d | 置信度: %.0f%%\n",
			prediction.Params.Distance,
			prediction.Params.Effort,
			prediction.Confidence*100)
		fmt.Printf("     预期节省: %.1f%% (%.2f MB → %.2f MB)\n",
			prediction.ExpectedSaving*100,
			float64(features.FileSize)/(1024*1024),
			float64(prediction.ExpectedSizeBytes)/(1024*1024))
		fmt.Println()
	}

	// 总结
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 测试总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  测试文件: %d\n", len(testFiles))
	fmt.Printf("  成功预测: %d\n", successCount)
	fmt.Printf("  成功率: %.2f%%\n", float64(successCount)/float64(len(testFiles))*100)

	if successCount > 0 {
		avgFeatureTime := totalFeatureTime / time.Duration(successCount)
		avgPredictTime := totalPredictTime / time.Duration(successCount)
		avgTotalTime := (totalFeatureTime + totalPredictTime) / time.Duration(successCount)
		avgExpectedSaving := totalExpectedSaving / float64(successCount) * 100

		totalSaving := float64(totalActualSize-totalExpectedSize) / float64(totalActualSize) * 100

		fmt.Printf("\n  ⚡ 性能指标:\n")
		fmt.Printf("     平均特征提取: %v\n", avgFeatureTime)
		fmt.Printf("     平均预测耗时: %v\n", avgPredictTime)
		fmt.Printf("     平均总耗时: %v\n", avgTotalTime)

		fmt.Printf("\n  💾 空间预测:\n")
		fmt.Printf("     平均预期节省: %.1f%%\n", avgExpectedSaving)
		fmt.Printf("     总体预期节省: %.1f%% (%.2f MB → %.2f MB)\n",
			totalSaving,
			float64(totalActualSize)/(1024*1024),
			float64(totalExpectedSize)/(1024*1024))

		fmt.Printf("\n  🎯 v3.0 MVP验证:\n")
		if avgTotalTime < 100*time.Millisecond {
			fmt.Printf("     ✅ 预测速度: %v < 100ms (目标达成)\n", avgTotalTime)
		} else {
			fmt.Printf("     ⚠️  预测速度: %v (目标: <100ms)\n", avgTotalTime)
		}

		if avgExpectedSaving > 80 {
			fmt.Printf("     ✅ 空间节省: %.1f%% > 80%% (目标达成)\n", avgExpectedSaving)
		} else {
			fmt.Printf("     ⚠️  空间节省: %.1f%% (目标: >80%%)\n", avgExpectedSaving)
		}

		if successCount == len(testFiles) {
			fmt.Println("     ✅ 成功率: 100% (目标达成)")
		}
	}

	fmt.Println()

	if successCount == len(testFiles) {
		fmt.Println("✅ PNG预测器MVP测试通过！")
		fmt.Println()
		fmt.Println("下一步：实际转换测试（验证预测准确性）")
	} else {
		fmt.Printf("⚠️  部分文件预测失败: %d/%d\n", len(testFiles)-successCount, len(testFiles))
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
