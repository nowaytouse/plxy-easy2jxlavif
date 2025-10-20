package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	testDir := "/Users/nameko_1/Documents/Pixly/test_pack_all/测试_新副本_20250828_055908"
	
	fmt.Println("🧪 完整媒体文件转换测试")
	fmt.Printf("📂 测试目录: %s\n\n", testDir)
	
	// 扫描所有媒体文件
	mediaFiles := scanMediaFiles(testDir)
	fmt.Printf("📋 发现 %d 个媒体文件\n\n", len(mediaFiles))
	
	// 显示文件列表
	for i, file := range mediaFiles {
		ext := strings.ToLower(filepath.Ext(file))
		size := getFileSizeMB(file)
		fmt.Printf("%d. %s (%s, %.1f MB)\n", i+1, filepath.Base(file), ext, size)
	}
	
	fmt.Println("\n🎯 开始转换测试...")
	
	successCount := 0
	failCount := 0
	
	// 测试每个文件的转换
	for i, file := range mediaFiles {
		ext := strings.ToLower(filepath.Ext(file))
		baseName := strings.TrimSuffix(file, filepath.Ext(file))
		
		// 确定目标格式
		var targetExt string
		switch ext {
		case ".jpg", ".jpeg", ".jpe", ".jfif", ".png", ".bmp", ".tiff":
			targetExt = ".jxl"
		case ".gif", ".webp":
			targetExt = ".avif"
		case ".heif", ".heic":
			targetExt = ".jxl"
		case ".mp4", ".mov", ".webm":
			targetExt = ".mp4" // 重包装
		default:
			fmt.Printf("%d. ⏭️  跳过 %s (不支持的格式)\n", i+1, filepath.Base(file))
			continue
		}
		
		outputFile := baseName + "_test" + targetExt
		
		fmt.Printf("%d. 🔄 %s → %s: ", i+1, ext, targetExt)
		
		startTime := time.Now()
		err := convertFile(file, outputFile, targetExt)
		duration := time.Since(startTime)
		
		if err != nil {
			fmt.Printf("❌ 失败 (%v) [%v]\n", err, duration)
			failCount++
		} else {
			// 检查输出文件
			if _, err := os.Stat(outputFile); err == nil {
				outputSize := getFileSizeMB(outputFile)
				sourceSize := getFileSizeMB(file)
				ratio := (1 - outputSize/sourceSize) * 100
				fmt.Printf("✅ 成功 (%.1f MB → %.1f MB, 压缩: %.1f%%) [%v]\n", 
					sourceSize, outputSize, ratio, duration)
				successCount++
			} else {
				fmt.Printf("❌ 输出文件不存在 [%v]\n", duration)
				failCount++
			}
		}
	}
	
	// 最终统计
	total := successCount + failCount
	fmt.Printf("\n📊 测试完成:\n")
	fmt.Printf("✅ 成功: %d/%d (%.1f%%)\n", successCount, total, float64(successCount)/float64(total)*100)
	fmt.Printf("❌ 失败: %d/%d (%.1f%%)\n", failCount, total, float64(failCount)/float64(total)*100)
}

func scanMediaFiles(dir string) []string {
	var files []string
	
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		
		ext := strings.ToLower(filepath.Ext(path))
		if isMediaExt(ext) {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files
}

func isMediaExt(ext string) bool {
	mediaExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true,
		".png": true, ".gif": true, ".webp": true, ".bmp": true,
		".heif": true, ".heic": true, ".tiff": true, ".avif": true,
		".mp4": true, ".mov": true, ".webm": true,
	}
	return mediaExts[ext]
}

func convertFile(sourcePath, targetPath, targetExt string) error {
	sourceExt := strings.ToLower(filepath.Ext(sourcePath))
	
	switch targetExt {
	case ".jxl":
		isJpeg := sourceExt == ".jpg" || sourceExt == ".jpeg" || sourceExt == ".jpe" || sourceExt == ".jfif"
		
		var cmd *exec.Cmd
		if isJpeg {
			cmd = exec.Command("cjxl", sourcePath, targetPath, "--lossless_jpeg=1", "-e", "7")
		} else {
			cmd = exec.Command("cjxl", sourcePath, targetPath, "--lossless_jpeg=0", "-q", "85", "-e", "7")
		}
		return cmd.Run()
		
	case ".avif":
		cmd := exec.Command("ffmpeg", "-i", sourcePath, "-c:v", "libaom-av1", "-crf", "32", "-y", targetPath)
		return cmd.Run()
		
	case ".mp4":
		cmd := exec.Command("ffmpeg", "-i", sourcePath, "-c", "copy", "-y", targetPath)
		return cmd.Run()
		
	default:
		return fmt.Errorf("不支持的格式: %s", targetExt)
	}
}

func getFileSizeMB(path string) float64 {
	if info, err := os.Stat(path); err == nil {
		return float64(info.Size()) / (1024 * 1024)
	}
	return 0
}