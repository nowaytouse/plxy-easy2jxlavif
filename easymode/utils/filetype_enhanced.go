// utils/filetype_enhanced.go - 增强文件类型检测模块
//
// 功能说明：
// - 提供增强的文件类型检测功能
// - 解决标准库无法识别某些格式的问题
// - 支持动画文件检测和MIME类型识别
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

// EnhancedFileType 增强的文件类型信息结构体
// 包含文件扩展名、MIME类型、有效性、媒体类型和动画状态等信息
type EnhancedFileType struct {
	Extension  string // 文件扩展名
	MIME       string // MIME类型
	IsValid    bool   // 是否为有效文件
	IsImage    bool   // 是否为图像文件
	IsVideo    bool   // 是否为视频文件
	IsAnimated bool   // 是否为动画文件
}

// DetectFileType 增强的文件类型检测函数
// 结合filetype库和自定义检测逻辑，解决标准库无法识别某些格式的问题
// 参数:
//
//	filePath - 要检测的文件路径
//
// 返回:
//
//	EnhancedFileType - 增强的文件类型信息
//	error - 检测过程中的错误（如果有）
func DetectFileType(filePath string) (EnhancedFileType, error) {
	// 首先尝试filetype库检测
	file, err := os.Open(filePath)
	if err != nil {
		return EnhancedFileType{}, fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	// 读取文件头（前512字节用于类型检测）
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return EnhancedFileType{}, fmt.Errorf("无法读取文件头: %v", err)
	}

	// 使用filetype库进行文件头检测
	kind, err := filetype.Match(header[:n])
	if err != nil {
		return EnhancedFileType{}, fmt.Errorf("filetype检测失败: %v", err)
	}

	// 获取文件扩展名并标准化
	ext := strings.ToLower(filepath.Ext(filePath))
	ext = strings.TrimPrefix(ext, ".")

	// 初始化结果结构体
	result := EnhancedFileType{
		Extension: ext,
		MIME:      kind.MIME.Value,
		IsValid:   true,
	}

	// 如果filetype无法识别，使用扩展名和文件头进行增强检测
	if kind == types.Unknown {
		result = detectByExtensionAndHeader(filePath, ext, header[:n])
	} else {
		// 设置基本类型信息
		result.IsImage = isImageType(kind)
		result.IsVideo = isVideoType(kind)
		result.IsAnimated = isAnimatedType(kind, ext)
	}

	return result, nil
}

// detectByExtensionAndHeader 基于扩展名和文件头的增强检测
// 当filetype库无法识别时，使用扩展名和文件头进行自定义检测
// 参数:
//
//	filePath - 文件路径
//	ext - 文件扩展名
//	header - 文件头数据
//
// 返回:
//
//	EnhancedFileType - 检测结果
func detectByExtensionAndHeader(filePath, ext string, header []byte) EnhancedFileType {
	// 初始化结果结构体
	result := EnhancedFileType{
		Extension: ext,
		IsValid:   true,
	}

	// AVIF/HEIF格式检测
	if ext == "avif" || ext == "heic" || ext == "heif" {
		result.MIME = "image/avif"
		result.IsImage = true
		result.IsAnimated = detectAVIFAnimation(filePath)
		return result
	}

	// WebP格式检测
	if ext == "webp" {
		result.MIME = "image/webp"
		result.IsImage = true
		result.IsAnimated = detectWebPAnimation(header)
		return result
	}

	// APNG 检测
	if ext == "apng" {
		result.MIME = "image/apng"
		result.IsImage = true
		result.IsAnimated = detectAPNGAnimation(header)
		return result
	}

	// 其他格式的fallback检测
	switch ext {
	case "jpg", "jpeg":
		result.MIME = "image/jpeg"
		result.IsImage = true
	case "png":
		result.MIME = "image/png"
		result.IsImage = true
	case "gif":
		result.MIME = "image/gif"
		result.IsImage = true
		result.IsAnimated = detectGIFAnimation(header)
	case "bmp":
		result.MIME = "image/bmp"
		result.IsImage = true
	case "tiff", "tif":
		result.MIME = "image/tiff"
		result.IsImage = true
	case "ico":
		result.MIME = "image/x-icon"
		result.IsImage = true
	case "cur":
		result.MIME = "image/x-cursor"
		result.IsImage = true
	case "mp4":
		result.MIME = "video/mp4"
		result.IsVideo = true
	case "mov":
		result.MIME = "video/quicktime"
		result.IsVideo = true
	case "avi":
		result.MIME = "video/x-msvideo"
		result.IsVideo = true
	case "mkv":
		result.MIME = "video/x-matroska"
		result.IsVideo = true
	case "webm":
		result.MIME = "video/webm"
		result.IsVideo = true
	default:
		result.IsValid = false
	}

	return result
}

// isImageType 检查是否为图像类型
func isImageType(kind types.Type) bool {
	return strings.HasPrefix(kind.MIME.Type, "image")
}

// isVideoType 检查是否为视频类型
func isVideoType(kind types.Type) bool {
	return strings.HasPrefix(kind.MIME.Type, "video")
}

// isAnimatedType 检查是否为动画类型
func isAnimatedType(kind types.Type, ext string) bool {
	switch ext {
	case "gif", "webp", "avif", "heic", "heif", "apng":
		return true
	}
	return false
}

// detectAVIFAnimation 检测AVIF是否为动画
func detectAVIFAnimation(filePath string) bool {
	// 使用exiftool检查AVIF动画
	// 这里可以添加更复杂的检测逻辑
	return false // 默认返回false，可以根据需要增强
}

// detectWebPAnimation 检测WebP是否为动画
func detectWebPAnimation(header []byte) bool {
	// WebP动画检测：检查VP8X chunk
	if len(header) >= 12 {
		// 检查WebP文件头
		if bytes.HasPrefix(header, []byte("RIFF")) && bytes.Contains(header[8:12], []byte("WEBP")) {
			// 查找VP8X chunk
			for i := 12; i < len(header)-8; i++ {
				if bytes.Equal(header[i:i+4], []byte("VP8X")) {
					// 检查动画标志位
					if i+8 < len(header) {
						flags := header[i+8]
						return (flags & 0x02) != 0 // 动画标志位
					}
				}
			}
		}
	}
	return false
}

// detectAPNGAnimation 检测APNG是否为动画
func detectAPNGAnimation(header []byte) bool {
	// APNG检测：查找acTL chunk
	if len(header) >= 8 {
		for i := 8; i < len(header)-8; i++ {
			if bytes.Equal(header[i:i+4], []byte("acTL")) {
				return true
			}
		}
	}
	return false
}

// detectGIFAnimation 检测GIF是否为动画
func detectGIFAnimation(header []byte) bool {
	// GIF动画检测：查找多个图像描述符
	if len(header) >= 6 {
		// 检查GIF文件头
		if bytes.HasPrefix(header, []byte("GIF87a")) || bytes.HasPrefix(header, []byte("GIF89a")) {
			// 查找图像描述符 (0x2C)
			imageCount := 0
			for i := 6; i < len(header); i++ {
				if header[i] == 0x2C {
					imageCount++
					if imageCount > 1 {
						return true
					}
				}
			}
		}
	}
	return false
}

// IsSupportedImageFormat 检查是否为支持的图像格式
func IsSupportedImageFormat(filePath string) bool {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return false
	}
	return fileType.IsImage && fileType.IsValid
}

// IsSupportedVideoFormat 检查是否为支持的视频格式
func IsSupportedVideoFormat(filePath string) bool {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return false
	}
	return fileType.IsVideo && fileType.IsValid
}

// IsAnimatedImage 检查是否为动画图像
func IsAnimatedImage(filePath string) bool {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return false
	}
	return fileType.IsAnimated
}

// IsLivePhoto detects if the file is part of an Apple Live Photo (HEIC + MOV pair)
func IsLivePhoto(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".heic" && ext != ".heif" {
		return false
	}
	base := strings.TrimSuffix(filePath, ext)
	movPath := base + ".mov"
	_, err := os.Stat(movPath)
	return err == nil
}

// GetFileTypeInfo 获取文件类型信息（用于调试）
func GetFileTypeInfo(filePath string) string {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return fmt.Sprintf("检测失败: %v", err)
	}

	info := fmt.Sprintf("扩展名: %s, MIME: %s, 有效: %t",
		fileType.Extension, fileType.MIME, fileType.IsValid)

	if fileType.IsImage {
		info += ", 图像: 是"
		if fileType.IsAnimated {
			info += " (动画)"
		}
	}

	if fileType.IsVideo {
		info += ", 视频: 是"
	}

	return info
}
