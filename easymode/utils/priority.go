// utils/priority.go - 文件处理优先级模块
//
// 功能说明：
// - 定义文件处理优先级算法
// - 优化用户体验，优先处理常见格式
// - 提升整体处理速度的体感
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"path/filepath"
	"strings"
)

// GetFileProcessingPriority 返回文件处理优先级
// 数值越大优先级越高，用于优化处理顺序以提升用户体验
// 设计原则：优先处理常见且转换链稳定的格式以提升体感速度
// 参数:
//
//	filePath - 文件路径
//
// 返回:
//
//	int - 优先级数值（1-10，数值越大优先级越高）
func GetFileProcessingPriority(filePath string) int {
	// 获取文件扩展名并转换为小写
	ext := strings.ToLower(filepath.Ext(filePath))

	// 根据文件格式返回优先级
	switch ext {
	case ".jpg", ".jpeg":
		return 10 // 最高优先级：最常见的图像格式
	case ".png":
		return 7 // 高优先级：广泛使用的无损格式
	case ".gif":
		return 5 // 中等优先级：动画格式，处理时间较长
	case ".webp":
		return 4 // 中等优先级：现代格式
	case ".heic", ".heif", ".avif":
		return 3 // 较低优先级：较新的格式，处理可能较慢
	default:
		return 1 // 最低优先级：其他格式
	}
}
