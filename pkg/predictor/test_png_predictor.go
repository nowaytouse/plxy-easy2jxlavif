//go:build ignore
// +build ignore

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

	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("用法: go run test_png_predictor.go <PNG文件或目录>")
		fmt.Println("示例: go run test_png_predictor.go /path/to/images/")
		os.Exit(1)
	}

	targetPath := os.Args[1]

	// 创建预测器
	pred := predictor.NewPredictor(logger, "ffprobe")

	// 检查是文件还是目录
	stat, err := os.Stat(targetPath)
	if err != nil {
		logger.Fatal("无法访问路径", zap.String("path", targetPath), zap.Error(err))
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎯 Pixly v3.0 MVP - PNG智能预测器测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	var pngFiles []string

	if stat.IsDir() {
		// 扫描目录中的PNG文件
		err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				ext := filepath.Ext(path)
				if ext == ".png" || ext == ".PNG" {
					pngFiles = append(pngFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			logger.Fatal("扫描目录失败", zap.Error(err))
		}
	} else {
		// 单个文件
		if ext := filepath.Ext(targetPath); ext == ".png" || ext == ".PNG" {
			pngFiles = append(pngFiles, targetPath)
		} else {
			logger.Fatal("不是PNG文件", zap.String("ext", ext))
		}
	}

	if len(pngFiles) == 0 {
		fmt.Println("❌ 未找到PNG文件")
		os.Exit(1)
	}

	fmt.Printf("📊 找到 %d 个PNG文件\n\n", len(pngFiles))

	// 统计数据
	totalPredictTime := time.Duration(0)
	successCount := 0
	totalExpectedSaving := 0.0

	// 测试每个文件
	for i, filePath := range pngFiles {
		fmt.Printf("[%d/%d] 测试: %s\n", i+1, len(pngFiles), filepath.Base(filePath))

		// 预测
		prediction, err := pred.PredictOptimalParams(filePath)
		if err != nil {
			fmt.Printf("  ❌ 预测失败: %v\n\n", err)
			continue
		}

		successCount++
		totalPredictTime += prediction.PredictionTime
		totalExpectedSaving += prediction.ExpectedSaving

		// 获取特征（用于显示）
		features, _ := pred.GetFeatures(filePath)

		// 显示结果
		fmt.Printf("  ✅ 预测成功\n")
		fmt.Printf("     格式: %s (%s)\n", features.Format, features.PixFmt)
		fmt.Printf("     尺寸: %dx%d\n", features.Width, features.Height)
		fmt.Printf("     大小: %.2f MB\n", float64(features.FileSize)/(1024*1024))
		fmt.Printf("     透明: %v\n", features.HasAlpha)
		fmt.Printf("     字节/像素: %.4f\n", features.BytesPerPixel)
		fmt.Printf("  🎯 预测参数:\n")
		fmt.Printf("     目标格式: %s\n", prediction.Params.TargetFormat)
		fmt.Printf("     Distance: %.1f (无损)\n", prediction.Params.Distance)
		fmt.Printf("     Effort: %d\n", prediction.Params.Effort)
		fmt.Printf("     置信度: %.2f%%\n", prediction.Confidence*100)
		fmt.Printf("     预期节省: %.2f%%\n", prediction.ExpectedSaving*100)
		fmt.Printf("     预期大小: %.2f MB → %.2f MB\n",
			float64(features.FileSize)/(1024*1024),
			float64(prediction.ExpectedSizeBytes)/(1024*1024))
		fmt.Printf("     预测耗时: %v\n", prediction.PredictionTime)
		fmt.Printf("     规则: %s\n", prediction.RuleName)
		fmt.Println()
	}

	// 总结
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 测试总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  总文件数: %d\n", len(pngFiles))
	fmt.Printf("  成功预测: %d\n", successCount)
	fmt.Printf("  成功率: %.2f%%\n", float64(successCount)/float64(len(pngFiles))*100)

	if successCount > 0 {
		avgPredictTime := totalPredictTime / time.Duration(successCount)
		avgExpectedSaving := totalExpectedSaving / float64(successCount) * 100

		fmt.Printf("  平均预测耗时: %v\n", avgPredictTime)
		fmt.Printf("  平均预期节省: %.2f%%\n", avgExpectedSaving)
	}

	fmt.Println()

	if successCount == len(pngFiles) {
		fmt.Println("✅ 所有PNG文件预测成功！")
		fmt.Println()
		fmt.Println("🎯 v3.0预测器工作正常，准备进行实际转换测试")
	} else {
		fmt.Printf("⚠️  部分文件预测失败: %d/%d\n", len(pngFiles)-successCount, len(pngFiles))
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
