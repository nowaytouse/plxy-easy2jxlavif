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

	// 测试所有格式（从TESTPACK）
	testFiles := map[string][]string{
		"PNG": {
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/黑白起稿.png",
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/psc.png",
		},
		"JPEG": {
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/10.jpg",
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕测试大量转换和嵌套文件夹 自动模式 应当仅使用副本 📁 测フォ_Folder Name  复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/未命名相簿/1573952589827.jpg",
		},
		"GIF_动态": {
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕 Avif动图和表情包测试使用_MuseDash 三人日常 2.0 📁 测フォ_Folder Name 复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/不要过来.gif",
			"/Users/nyamiiko/Documents/git/PIXLY最初版本backup/TESTPACK PASSIFYOUCAN!/🆕 Avif动图和表情包测试使用_MuseDash 三人日常 2.0 📁 测フォ_Folder Name 复制时必须保留文件夹名称 以便于测试你的文件夹识别功能!!_副本/吃瓜.gif",
		},
		"GIF_静态": {
			// 需要找静态GIF（如果有）
		},
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎯 Pixly v3.0 Week 5-6 - 全格式预测器测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	pred := predictor.NewPredictor(logger, "ffprobe")

	// 统计
	totalTests := 0
	successCount := 0
	formatStats := make(map[string]int)
	targetStats := make(map[string]int)

	for format, files := range testFiles {
		if len(files) == 0 {
			continue
		}

		fmt.Printf("═══ %s ═══\n\n", format)

		for _, filePath := range files {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				continue
			}

			totalTests++
			fileName := filepath.Base(filePath)
			fmt.Printf("[%s] %s\n", format, fileName)

			// 获取特征
			features, err := pred.GetFeatures(filePath)
			if err != nil {
				fmt.Printf("  ❌ 特征提取失败: %v\n\n", err)
				continue
			}

			// 预测
			prediction, err := pred.PredictOptimalParams(filePath)
			if err != nil {
				fmt.Printf("  ❌ 预测失败: %v\n\n", err)
				continue
			}

			successCount++
			formatStats[format]++
			targetStats[prediction.Params.TargetFormat]++

			// 显示特征
			fmt.Printf("  📊 特征: %dx%d | %.2f MB | 动图:%v",
				features.Width, features.Height,
				float64(features.FileSize)/(1024*1024),
				features.IsAnimated)
			if features.IsAnimated {
				fmt.Printf(" (帧数:%d)", features.FrameCount)
			}
			fmt.Println()

			// 显示预测
			fmt.Printf("  🎯 预测: %s", prediction.Params.TargetFormat)
			if prediction.Params.TargetFormat == "jxl" {
				if prediction.Params.LosslessJPEG {
					fmt.Printf(" (lossless_jpeg=1)")
				} else {
					fmt.Printf(" (distance=%.1f)", prediction.Params.Distance)
				}
			} else if prediction.Params.TargetFormat == "avif" {
				fmt.Printf(" (CRF=%d)", prediction.Params.CRF)
			} else if prediction.Params.TargetFormat == "mov" {
				fmt.Printf(" (重封装)")
			}
			fmt.Printf(" | 置信度:%.0f%% | %s\n", prediction.Confidence*100, prediction.RuleName)

			fmt.Println()
		}
	}

	// 总结
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 测试总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  总测试: %d\n", totalTests)
	fmt.Printf("  成功: %d\n", successCount)
	fmt.Printf("  成功率: %.1f%%\n\n", float64(successCount)/float64(totalTests)*100)

	fmt.Println("  格式分布:")
	for format, count := range formatStats {
		fmt.Printf("     %s: %d\n", format, count)
	}

	fmt.Println("\n  目标格式:")
	for target, count := range targetStats {
		fmt.Printf("     %s: %d\n", target, count)
	}

	fmt.Println()

	// 验证黄金规则
	fmt.Println("  🎯 黄金规则验证:")
	pngToJXL := formatStats["PNG"] > 0 && targetStats["jxl"] >= formatStats["PNG"]
	jpegToJXL := formatStats["JPEG"] > 0 && targetStats["jxl"] >= formatStats["JPEG"]
	gifAnimatedToAVIF := formatStats["GIF_动态"] > 0 && targetStats["avif"] >= formatStats["GIF_动态"]

	if pngToJXL {
		fmt.Println("     ✅ PNG → JXL")
	}
	if jpegToJXL {
		fmt.Println("     ✅ JPEG → JXL")
	}
	if gifAnimatedToAVIF {
		fmt.Println("     ✅ GIF动图 → AVIF")
	}

	if successCount == totalTests {
		fmt.Println("\n✅ 全格式预测器测试通过！")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

