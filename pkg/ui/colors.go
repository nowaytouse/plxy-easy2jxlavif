package ui

import (
	"github.com/pterm/pterm"
)

// ColorScheme 颜色方案
type ColorScheme struct {
	Primary    pterm.Color // 主色
	Secondary  pterm.Color // 辅色
	Success    pterm.Color // 成功
	Warning    pterm.Color // 警告
	Error      pterm.Color // 错误
	Info       pterm.Color // 信息
	Accent     pterm.Color // 强调色
	Muted      pterm.Color // 柔和色
	Background pterm.Color // 背景色（仅用于参考）
	Foreground pterm.Color // 前景色（仅用于参考）
}

// GetColorScheme 获取颜色方案（黑暗/亮色兼容）
func GetColorScheme(theme string) *ColorScheme {
	switch theme {
	case "dark":
		return &ColorScheme{
			Primary:    pterm.FgLightCyan,    // 亮青色（黑暗模式下突出）
			Secondary:  pterm.FgLightMagenta, // 亮洋红（强对比）
			Success:    pterm.FgLightGreen,   // 亮绿色
			Warning:    pterm.FgLightYellow,  // 亮黄色
			Error:      pterm.FgLightRed,     // 亮红色
			Info:       pterm.FgLightBlue,    // 亮蓝色
			Accent:     pterm.FgLightMagenta, // 强调色
			Muted:      pterm.FgGray,         // 灰色
			Background: pterm.BgBlack,
			Foreground: pterm.FgWhite,
		}
	case "light":
		return &ColorScheme{
			Primary:    pterm.FgCyan,    // 深青色（亮色模式下可见）
			Secondary:  pterm.FgMagenta, // 深洋红
			Success:    pterm.FgGreen,   // 深绿色
			Warning:    pterm.FgYellow,  // 深黄色
			Error:      pterm.FgRed,     // 深红色
			Info:       pterm.FgBlue,    // 深蓝色
			Accent:     pterm.FgMagenta, // 强调色
			Muted:      pterm.FgGray,    // 灰色
			Background: pterm.BgWhite,
			Foreground: pterm.FgBlack,
		}
	default: // "auto" - 使用对比度高的颜色，两种模式都适用
		return &ColorScheme{
			// 使用高对比度颜色，在黑暗和亮色模式下都清晰
			Primary:    pterm.FgLightCyan,    // 亮青色（通用）
			Secondary:  pterm.FgLightMagenta, // 亮洋红（通用）
			Success:    pterm.FgLightGreen,   // 亮绿色（通用）
			Warning:    pterm.FgYellow,       // 黄色（通用）
			Error:      pterm.FgRed,          // 红色（通用）
			Info:       pterm.FgLightBlue,    // 亮蓝色（通用）
			Accent:     pterm.FgLightMagenta, // 强调色
			Muted:      pterm.FgGray,         // 灰色
			Background: pterm.BgDefault,
			Foreground: pterm.FgDefault,
		}
	}
}

// GradientText 创建渐变文本
func GradientText(text string, startColor, endColor pterm.Color) string {
	// 简化版：仅支持单一颜色（pterm限制）
	// 实际渐变效果通过多行不同颜色实现
	return pterm.NewStyle(startColor).Sprint(text)
}

// CreateGradientColors 创建渐变颜色数组
func CreateGradientColors(start, end pterm.Color, steps int) []pterm.Color {
	// 预定义渐变序列
	gradients := map[string][]pterm.Color{
		"cyan_to_magenta": {
			pterm.FgLightCyan,
			pterm.FgCyan,
			pterm.FgLightBlue,
			pterm.FgBlue,
			pterm.FgLightMagenta,
			pterm.FgMagenta,
		},
		"green_to_blue": {
			pterm.FgLightGreen,
			pterm.FgGreen,
			pterm.FgLightCyan,
			pterm.FgCyan,
			pterm.FgLightBlue,
			pterm.FgBlue,
		},
		"rainbow": {
			pterm.FgRed,
			pterm.FgYellow,
			pterm.FgGreen,
			pterm.FgCyan,
			pterm.FgBlue,
			pterm.FgMagenta,
		},
	}

	// 默认使用cyan_to_magenta
	return gradients["cyan_to_magenta"]
}

// MaterialStyle 材质样式（通过字符密度模拟）
type MaterialStyle int

const (
	MaterialFlat  MaterialStyle = iota // 平面
	MaterialGlass                      // 玻璃质感
	MaterialMetal                      // 金属质感
	MaterialNeon                       // 霓虹灯效果
)

// ApplyMaterialEffect 应用材质效果
func ApplyMaterialEffect(text string, material MaterialStyle, scheme *ColorScheme) string {
	switch material {
	case MaterialGlass:
		// 玻璃效果：使用浅色+半透明感
		return pterm.NewStyle(scheme.Primary, pterm.BgDefault).Sprint(text)

	case MaterialMetal:
		// 金属效果：使用灰色系+高亮
		return pterm.NewStyle(pterm.FgLightWhite, pterm.BgGray).Sprint(text)

	case MaterialNeon:
		// 霓虹效果：亮色+粗体
		return pterm.NewStyle(scheme.Accent, pterm.Bold).Sprint(text)

	default: // Flat
		return pterm.NewStyle(scheme.Primary).Sprint(text)
	}
}
