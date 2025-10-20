package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// 简化的媒体文件检查函数
func isMediaFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 定义支持的媒体格式白名单 - 包含修复的格式
	mediaExtensions := map[string]bool{
		// 静图格式
		".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true, // JPEG系列 - 修复：添加了.jpe和.jfif
		".png": true, ".gif": true, ".webp": true, ".bmp": true,
		".tiff": true, ".tif": true, ".ico": true, ".svg": true,
		".avif": true, ".jxl": true, ".heif": true, ".heic": true,

		// 动图格式
		".apng": true, ".mng": true,

		// 视频格式
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
		".flv": true, ".wmv": true, ".m4v": true, ".3gp": true,
	}

	return mediaExtensions[ext]
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	testDir := "/Users/nameko_1/Documents/Pixly/test_pack_all/测试_新副本_20250828_055908"

	fmt.Println("🔍 开始检测测试目录中的媒体文件...")
	fmt.Printf("📂 测试目录: %s\n\n", testDir)

	// 扫描目录
	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 检查是否为媒体文件
		if isMediaFile(path) {
			ext := strings.ToLower(filepath.Ext(path))
			size := float64(info.Size()) / (1024 * 1024) // MB

			// 特别标注 jpe 和 jfif 格式
			if ext == ".jpe" || ext == ".jfif" {
				fmt.Printf("✅ [特殊格式] %s (%.1f MB) - %s\n",
					filepath.Base(path), size, ext)
			} else {
				fmt.Printf("📄 %s (%.1f MB) - %s\n",
					filepath.Base(path), size, ext)
			}
		} else {
			// 非媒体文件
			ext := strings.ToLower(filepath.Ext(path))
			fmt.Printf("❌ [跳过] %s - %s (非媒体文件)\n",
				filepath.Base(path), ext)
		}

		return nil
	})

	if err != nil {
		logger.Error("扫描目录失败", zap.Error(err))
		return
	}

	fmt.Println("\n🎯 格式检测测试完成！")
	fmt.Println("✅ 如果看到 .jpe 和 .jfif 文件被标记为 [特殊格式]，说明修复成功")
}
