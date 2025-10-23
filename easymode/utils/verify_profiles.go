// utils/verify_profiles.go - 验证配置文件模块
//
// 功能说明：
// - 定义验证参数配置文件
// - 根据格式和质量提供推荐阈值
// - 为8层验证系统提供参数建议
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

// VerifyProfile 验证配置文件结构体
// 根据格式与质量返回推荐阈值，供验证器调用
// 注意：EightLayerValidator 仍是权威实现，此处仅给出阈值建议
type VerifyProfile struct {
	MinPSNRdB        float64 // 最小PSNR值（分贝），用于图像质量评估
	MaxPixelDiffPerc float64 // 最大像素差异百分比，允许的像素变化范围
}

// GetVerifyProfile 根据格式和质量获取验证配置
// 为不同格式和质量级别提供合适的验证阈值
// 参数:
//
//	format - 文件格式（如"avif", "heic", "jxl"等）
//	quality - 质量级别（1-100）
//
// 返回:
//
//	VerifyProfile - 验证配置参数
func GetVerifyProfile(format string, quality int) VerifyProfile {
	// 现代格式（AVIF、HEIC、HEIF）使用严格标准
	switch format {
	case "avif", "heic", "heif":
		return VerifyProfile{
			MinPSNRdB:        30.0, // 高PSNR要求
			MaxPixelDiffPerc: 5.0,  // 允许5%的像素差异
		}
	default:
		// 传统格式根据质量级别调整阈值
		if quality >= 85 {
			// 高质量：严格标准
			return VerifyProfile{
				MinPSNRdB:        0,   // 不限制PSNR
				MaxPixelDiffPerc: 1.0, // 只允许1%的像素差异
			}
		} else if quality >= 70 {
			// 中等质量：中等标准
			return VerifyProfile{
				MinPSNRdB:        0,   // 不限制PSNR
				MaxPixelDiffPerc: 2.0, // 允许2%的像素差异
			}
		}
		// 低质量：宽松标准
		return VerifyProfile{
			MinPSNRdB:        0,   // 不限制PSNR
			MaxPixelDiffPerc: 5.0, // 允许5%的像素差异
		}
	}
}
