// utils/imaging.go - 图像处理模块
//
// 功能说明：
// - 提供图像格式转换功能
// - 支持HEIC/HEIF到PNG/TIFF的转换
// - 处理各种图像格式的中间态转换
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToPNGOrTIFF 将输入图像转为中间态PNG/TIFF格式
// 主要用于AVIF/HEIC/HEIF等格式的预处理，生成标准格式的中间文件
// 参数:
//
//	inputPath - 输入图像文件路径
//	tempOutPath - 临时输出文件基础路径
//	preferTIFF - 是否优先使用TIFF格式
//
// 返回:
//
//	string - 生成的临时文件路径（由调用方负责清理）
//	error - 转换过程中的错误（如果有）
func ToPNGOrTIFF(inputPath, tempOutPath string, preferTIFF bool) (string, error) {
	// 如果优先使用TIFF格式，尝试转换
	if preferTIFF {
		tiff := tempOutPath + ".tiff"
		cmd := exec.Command("magick", inputPath, tiff)
		if out, err := cmd.CombinedOutput(); err == nil {
			return tiff, nil
		} else {
			// TIFF转换失败，忽略错误继续尝试PNG
			_ = out
		}
	}

	// 准备PNG输出路径
	png := tempOutPath + ".png"

	// 获取输入文件扩展名
	inputExt := strings.ToLower(filepath.Ext(inputPath))
	
	// 对于HEIC/HEIF文件，必须使用完整图像而非缩略图
	if inputExt == ".heic" || inputExt == ".heif" {
		// 优先使用macOS的sips工具（最可靠，支持完整分辨率）
		cmd := exec.Command("sips", "-s", "format", "png", inputPath, "--out", png)
		if out, err := cmd.CombinedOutput(); err == nil {
			return png, nil
		} else {
			// sips工具失败，尝试其他工具
			_ = out
		}
		
		// 备用方案1: heif-convert (如果安装了libheif)
		cmd = exec.Command("heif-convert", inputPath, png)
		if out, err := cmd.CombinedOutput(); err == nil {
			return png, nil
		} else {
			_ = out
		}
		
		// 备用方案2: 尝试ImageMagick（可能会有问题）
		cmd = exec.Command("magick", inputPath, "-auto-orient", png)
		if out, err := cmd.CombinedOutput(); err == nil {
			return png, nil
		} else {
			_ = out
		}
		
		// 备用方案3: ffmpeg（可能只提取缩略图）
		cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", inputPath, "-frames:v", "1", "-pix_fmt", "rgb24", "-y", png)
		if out, err := cmd.CombinedOutput(); err == nil {
			return png, nil
		} else {
			_ = out
		}
		
		// 所有工具都失败，返回错误
		return "", fmt.Errorf("HEIC/HEIF转PNG失败(所有工具都无法处理): %s", inputPath)
	}
	
	// 对于AVIF文件，优先使用ffmpeg（magick可能生成有问题的PNG）
	if inputExt == ".avif" {
		cmd := exec.Command("ffmpeg", "-hwaccel", "none", "-i", inputPath, "-frames:v", "1", "-pix_fmt", "rgb24", "-y", png)
		if outAv, errAv := cmd.CombinedOutput(); errAv == nil {
			return png, nil
		} else {
			// ffmpeg失败，继续尝试其他方法
			_ = outAv
		}
	}

	// 对于可能的动画文件(GIF/WebP等)，仅提取第一帧
	// 使用-coalesce和[0]确保正确提取第一帧
	cmd := exec.Command("magick", inputPath+"[0]", "-coalesce", "-auto-orient", png)
	if out, err := cmd.CombinedOutput(); err == nil {
		return png, nil
	} else {
		_ = out
		// 尝试不带coalesce参数
		cmd = exec.Command("magick", inputPath+"[0]", png)
		if out2, err2 := cmd.CombinedOutput(); err2 == nil {
			_ = out2
			return png, nil
		} else {
			_ = out2
			// 尝试不带帧选择器
			cmd = exec.Command("magick", inputPath, png)
			if out3, err3 := cmd.CombinedOutput(); err3 == nil {
				_ = out3
				return png, nil
			} else {
				_ = out3
				// 最后尝试ffmpeg单帧提取
				cmd = exec.Command("ffmpeg", "-hwaccel", "none", "-i", inputPath, "-frames:v", "1", "-pix_fmt", "rgb24", "-y", png)
				if out4, err4 := cmd.CombinedOutput(); err4 == nil {
					_ = out4
					return png, nil
				}
			}
		}
	}
	// 所有转换方法都失败
	return "", fmt.Errorf("failed to generate PNG/TIFF intermediate for %s", inputPath)
}
