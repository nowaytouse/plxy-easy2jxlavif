package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 测试结果结构
type ConversionTestResult struct {
	SourceFile       string
	SourceExt        string
	TargetExt        string
	SourceSize       float64 // MB
	TargetSize       float64 // MB
	CompressionRatio float64 // 压缩率 %
	EffortUsed       int     // 使用的努力值
	Success          bool
	Duration         time.Duration
	Error            error
}

func main() {
	testDir := "/Users/nameko_1/Documents/Pixly/test_pack_all/测试_新副本_20250828_055908"

	fmt.Println("🧪 ==============================================")
	fmt.Println("🧪 Pixly 媒体转换验证测试")
	fmt.Println("🧪 ==============================================")
	fmt.Printf("📂 测试目录: %s\n\n", testDir)

	// 扫描媒体文件
	mediaFiles, err := scanMediaFiles(testDir)
	if err != nil {
		fmt.Printf("❌ 扫描失败: %v\n", err)
		return
	}

	fmt.Printf("📋 发现 %d 个媒体文件\n", len(mediaFiles))

	// 按格式分类显示
	formatCount := make(map[string]int)
	for _, file := range mediaFiles {
		ext := strings.ToLower(filepath.Ext(file))
		formatCount[ext]++
	}

	fmt.Println("\n📊 格式分布:")
	for ext, count := range formatCount {
		fmt.Printf("  %s: %d 个文件\n", ext, count)
	}

	// 执行转换测试
	fmt.Println("\n🎯 开始转换验证...")
	fmt.Println(strings.Repeat("-", 80))

	results := make([]ConversionTestResult, 0)

	for i, file := range mediaFiles {
		result := testFileConversion(file, i+1, len(mediaFiles))
		results = append(results, result)
	}

	// 生成详细报告
	generateTestReport(results)
}

func scanMediaFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if isTestableFormat(ext) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isTestableFormat(ext string) bool {
	// 只测试主要的媒体格式，避免处理已转换格式
	testableFormats := map[string]bool{
		// 图片格式 - 主要测试对象
		".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true,
		".png": true, ".bmp": true, ".tiff": true,
		".gif": true, ".webp": true,
		".heif": true, ".heic": true,

		// 视频格式 - 重包装测试
		".mp4": true, ".mov": true, ".webm": true, ".avi": true,
	}

	return testableFormats[ext]
}

func testFileConversion(filePath string, current, total int) ConversionTestResult {
	result := ConversionTestResult{
		SourceFile: filePath,
		SourceExt:  strings.ToLower(filepath.Ext(filePath)),
	}

	// 计算源文件大小
	if info, err := os.Stat(filePath); err == nil {
		result.SourceSize = float64(info.Size()) / (1024 * 1024)
	}

	// 确定目标格式（按照修复后的逻辑）
	result.TargetExt = determineTargetFormat(result.SourceExt)

	// 生成临时输出文件路径
	baseName := strings.TrimSuffix(filepath.Base(filePath), result.SourceExt)
	outputDir := filepath.Dir(filePath)
	outputFile := filepath.Join(outputDir, baseName+"_test"+result.TargetExt)

	fmt.Printf("[%d/%d] 🔄 %s → %s: %s ",
		current, total, result.SourceExt, result.TargetExt, filepath.Base(filePath))

	// 执行转换
	startTime := time.Now()
	err := performTestConversion(filePath, outputFile, result.TargetExt, &result)
	result.Duration = time.Since(startTime)
	result.Error = err
	result.Success = err == nil

	// 计算压缩效果
	if result.Success {
		if info, err := os.Stat(outputFile); err == nil {
			result.TargetSize = float64(info.Size()) / (1024 * 1024)
			if result.SourceSize > 0 {
				result.CompressionRatio = (1 - result.TargetSize/result.SourceSize) * 100
			}
		}

		// 清理临时文件
		os.Remove(outputFile)

		fmt.Printf("✅ (%.1f MB → %.1f MB, %+.1f%%) [%v]\n",
			result.SourceSize, result.TargetSize, result.CompressionRatio, result.Duration)
	} else {
		fmt.Printf("❌ %v [%v]\n", err, result.Duration)
	}

	return result
}

func determineTargetFormat(sourceExt string) string {
	// 按照修复后的自动模式+逻辑
	switch sourceExt {
	case ".jpg", ".jpeg", ".jpe", ".jfif", ".png", ".bmp", ".tiff", ".heif", ".heic":
		return ".jxl"
	case ".gif", ".webp":
		return ".avif"
	case ".mp4", ".mov", ".webm", ".avi":
		return ".mov"
	default:
		return ".jxl"
	}
}

func performTestConversion(sourcePath, targetPath, targetExt string, result *ConversionTestResult) error {
	sourceExt := strings.ToLower(filepath.Ext(sourcePath))

	// 创建30秒超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch targetExt {
	case ".jxl":
		// 计算动态努力值
		effort := calculateTestEffort(sourcePath)
		result.EffortUsed = effort

		isJpegFamily := sourceExt == ".jpg" || sourceExt == ".jpeg" || sourceExt == ".jpe" || sourceExt == ".jfif"

		var cmd *exec.Cmd
		if isJpegFamily {
			cmd = exec.CommandContext(ctx, "cjxl", sourcePath, targetPath,
				"--lossless_jpeg=1", "-e", fmt.Sprintf("%d", effort))
		} else {
			cmd = exec.CommandContext(ctx, "cjxl", sourcePath, targetPath,
				"--lossless_jpeg=0", "-q", "85", "-e", fmt.Sprintf("%d", effort))
		}
		return cmd.Run()

	case ".avif":
		cmd := exec.CommandContext(ctx, "ffmpeg", "-i", sourcePath,
			"-c:v", "libaom-av1", "-crf", "32", "-b:v", "0", "-y", targetPath)
		return cmd.Run()

	case ".mov":
		cmd := exec.CommandContext(ctx, "ffmpeg", "-i", sourcePath,
			"-c", "copy", "-y", targetPath)
		return cmd.Run()

	default:
		return fmt.Errorf("不支持的目标格式: %s", targetExt)
	}
}

func calculateTestEffort(filePath string) int {
	// 复制修复后的动态努力值逻辑
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 7
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	ext := strings.ToLower(filepath.Ext(filePath))

	var effort int
	if fileSizeMB > 50 {
		effort = 7
	} else if fileSizeMB > 10 {
		effort = 8
	} else if fileSizeMB > 1 {
		effort = 9
	} else {
		effort = 10
	}

	// 格式微调
	switch ext {
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		if effort < 10 {
			effort++
		}
	case ".png":
		if effort < 9 {
			effort++
		}
	}

	// 确保范围 7-10
	if effort < 7 {
		effort = 7
	} else if effort > 10 {
		effort = 10
	}

	return effort
}

func generateTestReport(results []ConversionTestResult) {
	fmt.Println("\n🧪 ==============================================")
	fmt.Println("🧪 转换验证测试报告")
	fmt.Println("🧪 ==============================================")

	totalTests := len(results)
	successCount := 0
	failureCount := 0
	var totalSourceSize, totalTargetSize float64
	var totalDuration time.Duration

	// 统计数据
	formatStats := make(map[string]map[string]int) // [sourceExt][result] = count

	for _, result := range results {
		if result.Success {
			successCount++
			totalSourceSize += result.SourceSize
			totalTargetSize += result.TargetSize
		} else {
			failureCount++
		}
		totalDuration += result.Duration

		// 格式统计
		if formatStats[result.SourceExt] == nil {
			formatStats[result.SourceExt] = make(map[string]int)
		}
		if result.Success {
			formatStats[result.SourceExt]["success"]++
		} else {
			formatStats[result.SourceExt]["fail"]++
		}
	}

	// 基本统计
	fmt.Printf("📊 基本统计:\n")
	fmt.Printf("  总测试文件: %d 个\n", totalTests)
	fmt.Printf("  转换成功: %d 个 (%.1f%%)\n", successCount, float64(successCount)/float64(totalTests)*100)
	fmt.Printf("  转换失败: %d 个 (%.1f%%)\n", failureCount, float64(failureCount)/float64(totalTests)*100)
	fmt.Printf("  总处理时间: %v (平均: %v/文件)\n", totalDuration, totalDuration/time.Duration(totalTests))

	if successCount > 0 {
		totalCompression := (1 - totalTargetSize/totalSourceSize) * 100
		fmt.Printf("  总体压缩效果: %.1f MB → %.1f MB (%.1f%%)\n",
			totalSourceSize, totalTargetSize, totalCompression)
	}

	// 格式统计
	fmt.Println("\n📈 格式转换统计:")
	for ext, stats := range formatStats {
		total := stats["success"] + stats["fail"]
		successRate := float64(stats["success"]) / float64(total) * 100
		fmt.Printf("  %s: %d/%d 成功 (%.1f%%)\n", ext, stats["success"], total, successRate)
	}

	// 失败分析
	if failureCount > 0 {
		fmt.Println("\n❌ 失败原因分析:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  %s: %v\n", filepath.Base(result.SourceFile), result.Error)
			}
		}
	}

	// 努力值使用统计
	effortStats := make(map[int]int)
	for _, result := range results {
		if result.Success && result.TargetExt == ".jxl" {
			effortStats[result.EffortUsed]++
		}
	}

	if len(effortStats) > 0 {
		fmt.Println("\n🎯 JXL努力值使用统计:")
		for effort := 7; effort <= 10; effort++ {
			if count := effortStats[effort]; count > 0 {
				fmt.Printf("  Effort %d: %d 个文件\n", effort, count)
			}
		}
	}

	// 最终评估
	fmt.Printf("\n🎉 测试完成! ")
	if float64(successCount)/float64(totalTests) >= 0.8 {
		fmt.Printf("✅ 转换系统运行良好 (成功率 %.1f%%)\n", float64(successCount)/float64(totalTests)*100)
	} else {
		fmt.Printf("⚠️  转换系统需要优化 (成功率 %.1f%%)\n", float64(successCount)/float64(totalTests)*100)
	}
}
