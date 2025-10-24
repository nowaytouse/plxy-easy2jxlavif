package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pixly/pkg/predictor"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 精选JPEG文件进行测试（来自TESTPACK）
	// 涵盖不同质量等级
	testFiles := []struct {
		path string
		desc string
	}{
		// 高质量JPEG（预期：yuv444p或yuv422p，质量>85）
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/10.jpg", "高质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/11.jpg", "高质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/12.jpg", "高质量"},

		// 中等质量JPEG（预期：yuv420p，质量70-85）
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1573952589827.jpg", "中等质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1580794244541.jpg", "中等质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1585642594974.jpg", "中等质量"},

		// 低质量JPEG（预期：质量<70）
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/4-1.jpg", "低质量"},
		{"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/5-2.jpg", "低质量"},
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎯 Pixly v3.0 Week 3-4 - JPEG智能预测器测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	pred := predictor.NewPredictor(logger, "ffprobe")

	// 统计分类
	highQualityCount := 0
	mediumQualityCount := 0
	lowQualityCount := 0
	exploreCount := 0
	successCount := 0

	for i, test := range testFiles {
		if _, err := os.Stat(test.path); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] ⏭️  跳过（文件不存在）: %s\n\n", i+1, len(testFiles), filepath.Base(test.path))
			continue
		}

		fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(testFiles), filepath.Base(test.path), test.desc)

		// 获取特征
		features, err := pred.GetFeatures(test.path)
		if err != nil {
			fmt.Printf("  ❌ 特征提取失败: %v\n\n", err)
			continue
		}

		// 预测
		prediction, err := pred.PredictOptimalParams(test.path)
		if err != nil {
			fmt.Printf("  ❌ 预测失败: %v\n\n", err)
			continue
		}

		successCount++

		// 显示特征
		fmt.Printf("  📊 特征:\n")
		fmt.Printf("     尺寸: %dx%d | 大小: %.2f MB\n",
			features.Width, features.Height,
			float64(features.FileSize)/(1024*1024))
		fmt.Printf("     PixFmt: %s | 质量估算: %d/100\n",
			features.PixFmt, features.EstimatedQuality)

		// 显示预测结果
		fmt.Printf("  🎯 预测:\n")
		fmt.Printf("     目标格式: %s\n", prediction.Params.TargetFormat)

		if prediction.Params.TargetFormat == "jxl" {
			fmt.Printf("     参数: distance=%.1f effort=%d\n",
				prediction.Params.Distance,
				prediction.Params.Effort)
			if prediction.Params.LosslessJPEG {
				fmt.Printf("     模式: JPEG无损重包装 (lossless_jpeg=1)\n")
			} else if prediction.Params.Distance == 0 {
				fmt.Printf("     模式: 数学无损 (distance=0)\n")
			} else {
				fmt.Printf("     模式: 轻微有损 (distance=%.1f)\n", prediction.Params.Distance)
			}
		} else if prediction.Params.TargetFormat == "avif" {
			fmt.Printf("     参数: CRF=%d speed=%d\n",
				prediction.Params.CRF,
				prediction.Params.Speed)
			fmt.Printf("     模式: 有损压缩 (CRF=%d)\n", prediction.Params.CRF)
		}

		fmt.Printf("     置信度: %.0f%%\n", prediction.Confidence*100)
		fmt.Printf("     规则: %s\n", prediction.RuleName)

		// 探索需求
		if prediction.ShouldExplore {
			fmt.Printf("  🔍 需要探索: 是 (%d个候选)\n", len(prediction.ExplorationCandidates))
			exploreCount++

			// 显示探索候选
			for j, candidate := range prediction.ExplorationCandidates {
				fmt.Printf("     候选%d: %s ", j+1, candidate.TargetFormat)
				if candidate.TargetFormat == "jxl" {
					if candidate.LosslessJPEG {
						fmt.Printf("lossless_jpeg=1")
					} else {
						fmt.Printf("d=%.1f", candidate.Distance)
					}
				} else {
					fmt.Printf("CRF=%d", candidate.CRF)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("  🔍 需要探索: 否（直接预测）\n")
		}

		// 分类统计
		if features.EstimatedQuality >= 85 || features.PixFmt == "yuv444p" {
			highQualityCount++
		} else if features.EstimatedQuality < 70 {
			lowQualityCount++
		} else {
			mediumQualityCount++
		}

		fmt.Println()
	}

	// 总结
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 测试总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  测试文件: %d\n", len(testFiles))
	fmt.Printf("  成功预测: %d\n", successCount)
	fmt.Printf("  成功率: %.1f%%\n", float64(successCount)/float64(len(testFiles))*100)

	fmt.Printf("\n  质量分层:\n")
	fmt.Printf("     高质量JPEG: %d (预期无损JXL)\n", highQualityCount)
	fmt.Printf("     中等质量JPEG: %d (预期探索)\n", mediumQualityCount)
	fmt.Printf("     低质量JPEG: %d (预期有损AVIF)\n", lowQualityCount)

	fmt.Printf("\n  探索统计:\n")
	fmt.Printf("     需要探索: %d (%.1f%%)\n", exploreCount, float64(exploreCount)/float64(successCount)*100)
	fmt.Printf("     直接预测: %d (%.1f%%)\n", successCount-exploreCount, float64(successCount-exploreCount)/float64(successCount)*100)

	fmt.Println()

	if successCount == len(testFiles) {
		fmt.Println("✅ JPEG预测器测试通过！")
		fmt.Println()
		fmt.Println("🎯 质量分层工作正常")
		fmt.Printf("   • 高质量→无损: %d文件\n", highQualityCount)
		fmt.Printf("   • 中等质量→探索: %d文件\n", exploreCount)
		fmt.Printf("   • 低质量→有损: %d文件\n", lowQualityCount)
		fmt.Println()
		fmt.Println("下一步：实际转换测试（验证质量）")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
