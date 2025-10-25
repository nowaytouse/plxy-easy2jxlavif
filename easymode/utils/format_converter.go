// utils/format_converter.go - 格式转换中间层
//
// 功能说明：
// - 提供格式转换中间层支持，解决编码器不支持某些输入格式的问题
// - 支持ImageMagick和ffmpeg作为转换工具
// - 自动检测可用工具并选择最佳方案

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ConverterType 表示转换器类型
type ConverterType int

const (
	ConverterNone ConverterType = iota
	ConverterImageMagick
	ConverterFFmpeg
)

// FormatConverter 格式转换器
type FormatConverter struct {
	availableConverters []ConverterType
	mu                  sync.RWMutex
	tempDir             string
}

var (
	globalConverter   *FormatConverter
	converterInitOnce sync.Once
)

// GetFormatConverter 获取全局格式转换器实例
func GetFormatConverter() *FormatConverter {
	converterInitOnce.Do(func() {
		globalConverter = &FormatConverter{
			tempDir: os.TempDir(),
		}
		globalConverter.detectAvailableConverters()
	})
	return globalConverter
}

// detectAvailableConverters 检测可用的转换工具
func (fc *FormatConverter) detectAvailableConverters() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.availableConverters = []ConverterType{}

	// 检测ImageMagick (convert或magick命令)
	if fc.commandExists("magick") || fc.commandExists("convert") {
		fc.availableConverters = append(fc.availableConverters, ConverterImageMagick)
	}

	// 检测ffmpeg
	if fc.commandExists("ffmpeg") {
		fc.availableConverters = append(fc.availableConverters, ConverterFFmpeg)
	}
}

// commandExists 检查命令是否存在
func (fc *FormatConverter) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetAvailableConverters 获取可用转换器列表
func (fc *FormatConverter) GetAvailableConverters() []ConverterType {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.availableConverters
}

// HasConverter 检查是否有可用的转换器
func (fc *FormatConverter) HasConverter() bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return len(fc.availableConverters) > 0
}

// NeedsConversion 检查文件是否需要格式转换（改进版 - 特别处理WEBP/WEBM/TIFF）
func NeedsConversion(filePath, targetTool string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 特殊处理：WEBP（统一使用转换层，因为cjxl对某些WEBP支持不完整）
	if ext == ".webp" {
		// 所有WEBP都通过转换层处理（PNG中间格式）
		// 原因：cjxl对有损WEBP支持不完整，avifenc也不直接支持WEBP
		return true
	}

	// 特殊处理：WEBM视频
	if ext == ".webm" {
		// WEBM是视频格式，需要转换
		return true
	}

	// 特殊处理：TIFF格式（添加转换层提高兼容性）
	if ext == ".tiff" || ext == ".tif" {
		// 大型TIFF可能有兼容性问题，统一转PNG
		return true
	}

	// cjxl 不支持的格式列表（扩展：某些WEBP cjxl也不支持）
	cjxlUnsupported := map[string]bool{
		".avif": true,
		".heif": true,
		".heic": true,
		".webp": true, // 添加WEBP（某些有损WEBP cjxl无法处理）
		".psd":  true,
		".psb":  true,
		".tiff": true, // 添加TIFF
		".tif":  true,
	}

	// avifenc 不支持的格式列表（扩展）
	avifencUnsupported := map[string]bool{
		".avif": true,
		".heif": true,
		".heic": true,
		".webp": true, // WEBP需要转换
		".bmp":  true,
		".tiff": true,
		".tif":  true,
		".jxl":  true,
		".psd":  true,
		".psb":  true,
	}

	if targetTool == "cjxl" {
		return cjxlUnsupported[ext]
	} else if targetTool == "avifenc" {
		return avifencUnsupported[ext]
	}

	return false
}

// ConvertToIntermediateFormat 将文件转换为中间格式（通常是PNG）
// 返回转换后的文件路径，如果不需要转换则返回原始路径
func (fc *FormatConverter) ConvertToIntermediateFormat(inputPath string, outputFormat string) (string, error) {
	// 检查输入文件是否存在
	if _, err := os.Stat(inputPath); err != nil {
		return "", fmt.Errorf("输入文件不存在: %w", err)
	}

	// 如果没有可用的转换器，返回错误
	if !fc.HasConverter() {
		return "", fmt.Errorf("没有可用的格式转换工具 (需要ImageMagick或ffmpeg)")
	}

	// 生成临时输出文件路径
	inputExt := filepath.Ext(inputPath)
	baseName := filepath.Base(inputPath)
	baseName = strings.TrimSuffix(baseName, inputExt)

	outputPath := filepath.Join(fc.tempDir, fmt.Sprintf("%s_converted.%s", baseName, outputFormat))

	// 尝试使用可用的转换器
	fc.mu.RLock()
	converters := fc.availableConverters
	fc.mu.RUnlock()

	var lastErr error
	for _, converter := range converters {
		var err error
		switch converter {
		case ConverterImageMagick:
			err = fc.convertWithImageMagick(inputPath, outputPath)
		case ConverterFFmpeg:
			err = fc.convertWithFFmpeg(inputPath, outputPath)
		}

		if err == nil {
			// 转换成功
			return outputPath, nil
		}
		lastErr = err
	}

	// 所有转换器都失败
	if lastErr != nil {
		return "", fmt.Errorf("格式转换失败: %w", lastErr)
	}

	return "", fmt.Errorf("无法转换文件格式")
}

// convertWithImageMagick 使用ImageMagick进行转换
func (fc *FormatConverter) convertWithImageMagick(inputPath, outputPath string) error {
	// 优先使用magick命令（ImageMagick 7+）
	var cmd *exec.Cmd
	if fc.commandExists("magick") {
		cmd = exec.Command("magick", "convert", inputPath, outputPath)
	} else {
		cmd = exec.Command("convert", inputPath, outputPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick转换失败: %w, 输出: %s", err, string(output))
	}

	// 验证输出文件是否存在
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("转换后的文件不存在: %w", err)
	}

	return nil
}

// convertWithFFmpeg 使用ffmpeg进行转换
func (fc *FormatConverter) convertWithFFmpeg(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-y", // 覆盖输出文件
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg转换失败: %w, 输出: %s", err, string(output))
	}

	// 验证输出文件是否存在
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("转换后的文件不存在: %w", err)
	}

	return nil
}

// CleanupTempFile 清理临时文件
func (fc *FormatConverter) CleanupTempFile(filePath string) error {
	// 只清理在临时目录中的文件
	if !strings.HasPrefix(filePath, fc.tempDir) {
		return nil
	}

	return os.Remove(filePath)
}

// ConvertIfNeeded 转换文件到PNG格式（如需要）（改进版 - 完整WEBP/WEBM/TIFF支持）
func ConvertIfNeeded(filePath, targetTool string) (string, bool, error) {
	if !NeedsConversion(filePath, targetTool) {
		return filePath, false, nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	// 特殊处理：WEBP格式（区分动态和静态）
	if ext == ".webp" {
		if IsAnimatedWebP(filePath) {
			// 动态WEBP：尝试专门的转换方法
			pngPath, err := convertAnimatedWebPToPNG(filePath)
			if err == nil {
				return pngPath, true, nil
			}
			// 如果失败，跳过此文件
			return "", false, fmt.Errorf("动态WEBP不支持: %v", err)
		} else {
			// 静态WEBP：使用标准转换流程
			return convertWebPToPNG(filePath)
		}
	}

	// 特殊处理：WEBM视频
	if ext == ".webm" {
		// WEBM视频应该被视频处理工具处理，不应该在这里
		return "", false, fmt.Errorf("WEBM是视频格式，应使用视频转换工具")
	}

	// 特殊处理：TIFF格式（使用专门方法）
	if ext == ".tiff" || ext == ".tif" {
		return convertTIFFToPNG(filePath)
	}

	// 其他格式使用标准PNG转换
	pngPath := strings.TrimSuffix(filePath, ext) + "_converted.png"

	// 使用ffmpeg转换
	cmd := exec.Command("ffmpeg", "-i", filePath, "-frames:v", "1", pngPath, "-y")
	if err := cmd.Run(); err != nil {
		// 尝试ImageMagick作为fallback
		cmd = exec.Command("magick", filePath, pngPath)
		if err := cmd.Run(); err != nil {
			return "", false, fmt.Errorf("格式转换失败: %v", err)
		}
	}

	return pngPath, true, nil
}

// convertWebPToPNG 专门处理静态WEBP转PNG
func convertWebPToPNG(webpPath string) (string, bool, error) {
	pngPath := strings.TrimSuffix(webpPath, filepath.Ext(webpPath)) + "_converted.png"

	// 方法1: 使用dwebp（libwebp官方工具，最可靠）
	cmd := exec.Command("dwebp", webpPath, "-o", pngPath)
	if err := cmd.Run(); err == nil {
		return pngPath, true, nil
	}

	// 方法2: 使用ffmpeg
	cmd = exec.Command("ffmpeg", "-i", webpPath, "-frames:v", "1", pngPath, "-y")
	if err := cmd.Run(); err == nil {
		return pngPath, true, nil
	}

	// 方法3: 使用ImageMagick
	cmd = exec.Command("magick", webpPath, pngPath)
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("静态WEBP转换失败(尝试了dwebp/ffmpeg/magick): %v", err)
	}

	return pngPath, true, nil
}

// convertTIFFToPNG 专门处理TIFF转PNG
func convertTIFFToPNG(tiffPath string) (string, bool, error) {
	pngPath := strings.TrimSuffix(tiffPath, filepath.Ext(tiffPath)) + "_converted.png"

	// 方法1: 使用ImageMagick（对TIFF支持最好）
	cmd := exec.Command("magick", tiffPath, pngPath)
	if err := cmd.Run(); err == nil {
		return pngPath, true, nil
	}

	// 方法2: 使用ffmpeg
	cmd = exec.Command("ffmpeg", "-i", tiffPath, pngPath, "-y")
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("TIFF转换失败(尝试了magick/ffmpeg): %v", err)
	}

	return pngPath, true, nil
}

// convertAnimatedWebPToPNG 专门处理动态WEBP（提取第一帧）
func convertAnimatedWebPToPNG(webpPath string) (string, error) {
	pngPath := strings.TrimSuffix(webpPath, filepath.Ext(webpPath)) + "_frame0.png"

	// 尝试使用ffmpeg提取第一帧
	cmd := exec.Command("ffmpeg", "-i", webpPath, "-vframes", "1", "-f", "image2", pngPath)
	if err := cmd.Run(); err != nil {
		// 尝试使用dwebp（libwebp工具）
		cmd = exec.Command("dwebp", webpPath, "-o", pngPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("动态WEBP转换失败: %v", err)
		}
	}

	return pngPath, nil
}

// SetTempDir 设置临时文件目录
func (fc *FormatConverter) SetTempDir(dir string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// 验证目录是否存在
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("临时目录不存在: %w", err)
	}

	fc.tempDir = dir
	return nil
}

// GetConverterName 获取转换器名称
func GetConverterName(ct ConverterType) string {
	switch ct {
	case ConverterImageMagick:
		return "ImageMagick"
	case ConverterFFmpeg:
		return "ffmpeg"
	default:
		return "None"
	}
}
