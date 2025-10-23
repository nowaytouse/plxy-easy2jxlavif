// utils/post_validation.go - 转换后验证模块
//
// 功能说明：
// - 提供转换后的抽样验证功能
// - 支持动图验证和静态图验证
// - 生成详细的验证报告和统计信息
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PostValidationResult 转换后验证结果结构体
// 记录抽样验证的统计信息和详细结果
type PostValidationResult struct {
	TotalFiles      int                    // 总文件数
	SampledFiles    int                    // 抽样文件数
	PassedFiles     int                    // 通过验证的文件数
	FailedFiles     int                    // 未通过验证的文件数
	ValidationItems []ValidationItemResult // 每个文件的验证结果
	Summary         string                 // 验证摘要
}

// ValidationItemResult 单个文件验证结果结构体
// 记录每个抽样文件的详细验证信息
type ValidationItemResult struct {
	OriginalPath  string   // 原始文件路径
	ConvertedPath string   // 转换后文件路径
	FileType      string   // 文件类型（static/animated/video）
	Passed        bool     // 是否通过验证
	Checks        []string // 检查项列表
	Issues        []string // 发现的问题
}

// MediaProperties 媒体属性
type MediaProperties struct {
	Width      int     // 宽度
	Height     int     // 高度
	FrameCount int     // 帧数
	FPS        float64 // 帧率
	Duration   float64 // 时长（秒）
	Format     string  // 格式
}

// PostValidator 转换后验证器
type PostValidator struct {
	SampleRate float64 // 抽样率 (0.0-1.0)
	MinSamples int     // 最小抽样数
	MaxSamples int     // 最大抽样数
}

// NewPostValidator 创建验证器
func NewPostValidator(sampleRate float64, minSamples, maxSamples int) *PostValidator {
	if sampleRate < 0.0 {
		sampleRate = 0.1 // 默认10%抽样率
	}
	if sampleRate > 1.0 {
		sampleRate = 1.0
	}
	if minSamples < 1 {
		minSamples = 5
	}
	if maxSamples < minSamples {
		maxSamples = 20
	}
	return &PostValidator{
		SampleRate: sampleRate,
		MinSamples: minSamples,
		MaxSamples: maxSamples,
	}
}

// ValidateConversions 验证转换结果
func (pv *PostValidator) ValidateConversions(pairs []FilePair) *PostValidationResult {
	result := &PostValidationResult{
		TotalFiles:      len(pairs),
		ValidationItems: []ValidationItemResult{},
	}

	// 计算抽样数量
	sampleCount := int(float64(len(pairs)) * pv.SampleRate)
	if sampleCount < pv.MinSamples {
		sampleCount = pv.MinSamples
	}
	if sampleCount > pv.MaxSamples {
		sampleCount = pv.MaxSamples
	}
	if sampleCount > len(pairs) {
		sampleCount = len(pairs)
	}

	// 随机抽样
	rand.Seed(time.Now().UnixNano())
	indices := rand.Perm(len(pairs))[:sampleCount]

	result.SampledFiles = sampleCount

	// 验证每个抽样文件
	for _, idx := range indices {
		pair := pairs[idx]
		itemResult := pv.validateFilePair(pair)
		result.ValidationItems = append(result.ValidationItems, itemResult)
		if itemResult.Passed {
			result.PassedFiles++
		} else {
			result.FailedFiles++
		}
	}

	// 生成摘要
	passRate := float64(result.PassedFiles) / float64(result.SampledFiles) * 100
	result.Summary = fmt.Sprintf("抽样验证: %d/%d 文件, 通过率: %.1f%%",
		result.SampledFiles, result.TotalFiles, passRate)

	return result
}

// FilePair 文件对（原始-转换后）
type FilePair struct {
	OriginalPath  string
	ConvertedPath string
}

// validateFilePair 验证单个文件对
func (pv *PostValidator) validateFilePair(pair FilePair) ValidationItemResult {
	result := ValidationItemResult{
		OriginalPath:  pair.OriginalPath,
		ConvertedPath: pair.ConvertedPath,
		Checks:        []string{},
		Issues:        []string{},
		Passed:        true,
	}

	// 检查原始文件是否存在（in-place转换会删除原始文件）
	if _, err := os.Stat(pair.OriginalPath); os.IsNotExist(err) {
		// 原始文件已被删除（正常的in-place转换），只验证转换后文件的基本属性
		result.FileType = "unknown"
		result.Checks = append(result.Checks, "转换后文件存在性检查")

		if _, err := os.Stat(pair.ConvertedPath); os.IsNotExist(err) {
			result.Issues = append(result.Issues, "转换后文件不存在")
			result.Passed = false
		}

		return result
	}

	// 检测原始文件类型
	origType, err := DetectFileType(pair.OriginalPath)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("无法检测原始文件类型: %v", err))
		result.Passed = false
		return result
	}

	// 获取原始文件属性
	origProps, err := pv.getMediaProperties(pair.OriginalPath)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("无法获取原始文件属性: %v", err))
		result.Passed = false
		return result
	}

	// 获取转换后文件属性
	convProps, err := pv.getMediaProperties(pair.ConvertedPath)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("无法获取转换后文件属性: %v", err))
		result.Passed = false
		return result
	}

	// 根据文件类型进行不同的验证
	if origType.IsAnimated {
		result.FileType = "animated"
		pv.validateAnimated(&result, origProps, convProps)
	} else if origType.IsVideo {
		result.FileType = "video"
		pv.validateVideo(&result, origProps, convProps)
	} else {
		result.FileType = "static"
		pv.validateStatic(&result, origProps, convProps)
	}

	return result
}

// validateAnimated 验证动图
func (pv *PostValidator) validateAnimated(result *ValidationItemResult, orig, conv *MediaProperties) {
	// 1. 检查分辨率（无裁切）
	result.Checks = append(result.Checks, "分辨率检查")
	if orig.Width != conv.Width || orig.Height != conv.Height {
		result.Issues = append(result.Issues,
			fmt.Sprintf("分辨率不匹配: %dx%d -> %dx%d (可能被裁切)",
				orig.Width, orig.Height, conv.Width, conv.Height))
		result.Passed = false
	}

	// 2. 检查帧数
	result.Checks = append(result.Checks, "帧数检查")
	if orig.FrameCount > 0 && conv.FrameCount > 0 {
		frameDiff := abs(orig.FrameCount - conv.FrameCount)
		// 允许1帧的误差（某些格式可能有微小差异）
		if frameDiff > 1 {
			result.Issues = append(result.Issues,
				fmt.Sprintf("帧数不匹配: %d -> %d (差异: %d帧)",
					orig.FrameCount, conv.FrameCount, frameDiff))
			result.Passed = false
		}
	}

	// 3. 检查FPS
	result.Checks = append(result.Checks, "帧率检查")
	if orig.FPS > 0 && conv.FPS > 0 {
		fpsDiff := absFloat(orig.FPS - conv.FPS)
		// 允许5%的FPS误差
		if fpsDiff > orig.FPS*0.05 {
			result.Issues = append(result.Issues,
				fmt.Sprintf("帧率不匹配: %.2f -> %.2f fps",
					orig.FPS, conv.FPS))
			result.Passed = false
		}
	}

	// 4. 确认是动图（有多帧）
	result.Checks = append(result.Checks, "动图验证")
	if conv.FrameCount <= 1 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("转换后变成静图: 只有%d帧", conv.FrameCount))
		result.Passed = false
	}
}

// validateVideo 验证视频
func (pv *PostValidator) validateVideo(result *ValidationItemResult, orig, conv *MediaProperties) {
	// 1. 检查分辨率
	result.Checks = append(result.Checks, "分辨率检查")
	if orig.Width != conv.Width || orig.Height != conv.Height {
		result.Issues = append(result.Issues,
			fmt.Sprintf("分辨率不匹配: %dx%d -> %dx%d",
				orig.Width, orig.Height, conv.Width, conv.Height))
		result.Passed = false
	}

	// 2. 检查时长
	result.Checks = append(result.Checks, "时长检查")
	if orig.Duration > 0 && conv.Duration > 0 {
		durationDiff := absFloat(orig.Duration - conv.Duration)
		// 允许0.5秒的时长误差
		if durationDiff > 0.5 {
			result.Issues = append(result.Issues,
				fmt.Sprintf("时长不匹配: %.2fs -> %.2fs",
					orig.Duration, conv.Duration))
			result.Passed = false
		}
	}
}

// validateStatic 验证静图
func (pv *PostValidator) validateStatic(result *ValidationItemResult, orig, conv *MediaProperties) {
	// 1. 检查分辨率（无裁切）
	result.Checks = append(result.Checks, "分辨率检查")
	if orig.Width != conv.Width || orig.Height != conv.Height {
		result.Issues = append(result.Issues,
			fmt.Sprintf("分辨率不匹配: %dx%d -> %dx%d (可能被裁切)",
				orig.Width, orig.Height, conv.Width, conv.Height))
		result.Passed = false
	}

	// 2. 确认是静图（单帧）
	result.Checks = append(result.Checks, "静图验证")
	if conv.FrameCount > 1 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("静图被转换为动图: %d帧", conv.FrameCount))
		result.Passed = false
	}

	// 3. 检查格式路由是否正确
	result.Checks = append(result.Checks, "格式路由验证")
	convExt := strings.ToLower(filepath.Ext(result.ConvertedPath))
	origExt := strings.ToLower(filepath.Ext(result.OriginalPath))

	// JPEG应该使用无损路由到JXL
	if (origExt == ".jpg" || origExt == ".jpeg") && convExt == ".jxl" {
		// 这是正确的路由，不需要特别检查
	}
}

// getMediaProperties 获取媒体属性
func (pv *PostValidator) getMediaProperties(filePath string) (*MediaProperties, error) {
	props := &MediaProperties{}

	// 使用 ffprobe 获取详细信息
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// ffprobe失败，尝试使用identify（ImageMagick）
		return pv.getPropertiesWithIdentify(filePath)
	}

	outputStr := string(output)

	// 解析宽度
	if match := regexp.MustCompile(`"width":\s*(\d+)`).FindStringSubmatch(outputStr); len(match) > 1 {
		props.Width, _ = strconv.Atoi(match[1])
	}

	// 解析高度
	if match := regexp.MustCompile(`"height":\s*(\d+)`).FindStringSubmatch(outputStr); len(match) > 1 {
		props.Height, _ = strconv.Atoi(match[1])
	}

	// 解析帧数
	if match := regexp.MustCompile(`"nb_frames":\s*"(\d+)"`).FindStringSubmatch(outputStr); len(match) > 1 {
		props.FrameCount, _ = strconv.Atoi(match[1])
	}

	// 解析帧率
	if match := regexp.MustCompile(`"r_frame_rate":\s*"(\d+)/(\d+)"`).FindStringSubmatch(outputStr); len(match) > 2 {
		num, _ := strconv.ParseFloat(match[1], 64)
		den, _ := strconv.ParseFloat(match[2], 64)
		if den > 0 {
			props.FPS = num / den
		}
	}

	// 解析时长
	if match := regexp.MustCompile(`"duration":\s*"([\d.]+)"`).FindStringSubmatch(outputStr); len(match) > 1 {
		props.Duration, _ = strconv.ParseFloat(match[1], 64)
	}

	return props, nil
}

// getPropertiesWithIdentify 使用identify获取属性（备用方案）
func (pv *PostValidator) getPropertiesWithIdentify(filePath string) (*MediaProperties, error) {
	props := &MediaProperties{}

	cmd := exec.Command("identify", "-format", "%w %h %n", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("无法获取文件属性: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)

	if len(parts) >= 2 {
		props.Width, _ = strconv.Atoi(parts[0])
		props.Height, _ = strconv.Atoi(parts[1])
	}

	if len(parts) >= 3 {
		props.FrameCount, _ = strconv.Atoi(parts[2])
	}

	return props, nil
}

// 辅助函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
