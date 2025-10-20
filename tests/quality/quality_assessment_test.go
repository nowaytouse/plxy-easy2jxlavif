package quality

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pixly/pkg/batchdecision"
	"pixly/pkg/core/types"
	"pixly/pkg/engine/quality"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestQualityAssessmentWithRealFiles 使用真实文件测试品质评估
func TestQualityAssessmentWithRealFiles(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 查找examples目录
	examplesDir := findExamplesDir(t)
	if examplesDir == "" {
		t.Skip("未找到examples目录，跳过真实文件测试")
		return
	}

	t.Logf("使用examples目录: %s", examplesDir)

	// 创建品质评估引擎
	qualityEngine := quality.NewQualityEngine(logger, "", "", true)

	// 扫描examples目录中的文件
	files, err := scanExamplesFiles(examplesDir)
	require.NoError(t, err, "扫描examples文件失败")

	t.Logf("发现 %d 个文件", len(files))

	// 分类统计
	qualityStats := make(map[types.QualityLevel]int)
	mediaTypeStats := make(map[types.MediaType]int)
	var lowQualityFiles []string

	ctx := context.Background()

	// 逐个评估文件品质
	for _, filePath := range files {
		t.Logf("评估文件: %s", filepath.Base(filePath))

		assessment, err := qualityEngine.AssessFile(ctx, filePath)
		if err != nil {
			t.Logf("评估失败 %s: %v", filepath.Base(filePath), err)
			continue
		}

		// 统计品质分布
		qualityStats[assessment.QualityLevel]++
		mediaTypeStats[assessment.MediaType]++

		// 记录低品质文件
		if assessment.QualityLevel == types.QualityLow || assessment.QualityLevel == types.QualityVeryLow {
			lowQualityFiles = append(lowQualityFiles, filePath)
		}

		t.Logf("  格式: %s, 媒体类型: %s, 品质: %s, 分数: %.2f, 置信度: %.2f",
			assessment.Format,
			assessment.MediaType.String(),
			assessment.QualityLevel.String(),
			assessment.Score,
			assessment.Confidence)
	}

	// 输出统计信息
	t.Logf("\n=== 品质评估统计 ===")
	for quality, count := range qualityStats {
		if count > 0 {
			t.Logf("%s: %d 文件", quality.String(), count)
		}
	}

	t.Logf("\n=== 媒体类型统计 ===")
	for mediaType, count := range mediaTypeStats {
		if count > 0 {
			t.Logf("%s: %d 文件", mediaType.String(), count)
		}
	}

	// 验证品质评估的合理性
	assert.True(t, len(qualityStats) > 0, "应该有品质评估结果")
	assert.True(t, len(mediaTypeStats) > 0, "应该有媒体类型识别结果")

	// 测试强制处理低品质文件
	if len(lowQualityFiles) > 0 {
		t.Logf("\n=== 测试强制处理低品质文件 ===")
		t.Logf("发现 %d 个低品质文件，将进行强制转换测试", len(lowQualityFiles))

		testForcedConversionForLowQualityFiles(t, logger, lowQualityFiles)
	} else {
		t.Logf("未发现低品质文件，所有文件品质评估良好")
	}
}

// testForcedConversionForLowQualityFiles 测试低品质文件的强制转换
func testForcedConversionForLowQualityFiles(t *testing.T, logger *zap.Logger, lowQualityFiles []string) {
	// 创建批量决策管理器并启用测试模式
	batchManager := batchdecision.NewBatchDecisionManager(logger, false)
	batchManager.SetTestMode(true) // 启用强制处理模式

	// 模拟添加低品质文件到决策管理器
	for _, filePath := range lowQualityFiles {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		lowQualityFile := &batchdecision.LowQualityFile{
			FilePath:     filePath,
			QualityScore: 0.3, // 模拟低品质分数
			FileSize:     fileInfo.Size(),
			CanConvert:   true,
		}

		err = batchManager.AddLowQualityFile(lowQualityFile)
		assert.NoError(t, err, "添加低品质文件应该成功")
	}

	// 执行批量决策处理
	ctx := context.Background()
	result, err := batchManager.ProcessBatchDecisions(ctx)

	// 在测试模式下，应该强制处理所有低品质文件
	assert.NoError(t, err, "批量决策处理应该成功")
	assert.NotNil(t, result, "应该有处理结果")

	if result != nil {
		t.Logf("批量决策结果:")
		t.Logf("  总文件数: %d", result.Summary.TotalFiles)
		t.Logf("  成功处理: %d", result.Summary.SuccessfulFiles)
		t.Logf("  失败文件: %d", result.Summary.FailedFiles)
		t.Logf("  跳过文件: %d", result.Summary.SkippedFiles)

		// 验证在测试模式下没有文件被跳过（除非出错）
		assert.Equal(t, 0, result.Summary.SkippedFiles,
			"测试模式下不应该跳过任何低品质文件")
	}
}

// TestQualityThresholdAdjustment 测试品质阈值调整的效果
func TestQualityThresholdAdjustment(t *testing.T) {
	logger := zaptest.NewLogger(t)
	qualityEngine := quality.NewQualityEngine(logger, "", "", true)

	// 创建测试文件（不同大小的JPEG文件）
	testCases := []struct {
		name     string
		content  string
		size     int64
		expected types.QualityLevel
	}{
		{"large.jpg", "fake jpeg large content", 3 * 1024 * 1024, types.QualityHigh},         // 3MB 应该被评为高品质
		{"medium.jpg", "fake jpeg medium content", 1 * 1024 * 1024, types.QualityMediumHigh}, // 1MB 应该被评为中高品质
		{"small.jpg", "fake jpeg small content", 300 * 1024, types.QualityMediumLow},         // 300KB 应该被评为中低品质
		{"tiny.jpg", "fake jpeg tiny content", 50 * 1024, types.QualityLow},                  // 50KB 应该被评为低品质
	}

	tempDir, err := os.MkdirTemp("", "quality_test_*")
	require.NoError(t, err, "创建临时目录失败")
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建测试文件
			filePath := filepath.Join(tempDir, tc.name)
			content := generateContent(tc.content, tc.size)
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err, "创建测试文件失败")

			// 评估品质
			assessment, err := qualityEngine.AssessFile(ctx, filePath)
			require.NoError(t, err, "品质评估失败")

			t.Logf("文件: %s, 大小: %d bytes, 评估品质: %s (分数: %.2f)",
				tc.name, tc.size, assessment.QualityLevel.String(), assessment.Score)

			// 验证品质等级是否符合预期（允许一定的容差）
			actualLevel := assessment.QualityLevel
			if actualLevel != tc.expected {
				// 检查是否在合理范围内（相邻等级）
				isReasonable := isAdjacentQualityLevel(actualLevel, tc.expected)
				if !isReasonable {
					t.Errorf("品质评估不符合预期: 期望 %s, 实际 %s",
						tc.expected.String(), actualLevel.String())
				} else {
					t.Logf("品质评估在合理范围内: 期望 %s, 实际 %s",
						tc.expected.String(), actualLevel.String())
				}
			}
		})
	}
}

// 辅助函数

func findExamplesDir(t *testing.T) string {
	// 从当前测试目录开始查找examples目录
	currentDir, _ := os.Getwd()

	// 可能的examples目录路径
	possiblePaths := []string{
		filepath.Join(currentDir, "..", "..", "examples"),
		filepath.Join(currentDir, "..", "examples"),
		filepath.Join(currentDir, "examples"),
		"/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/examples",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func scanExamplesFiles(examplesDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 过滤支持的文件格式
		ext := filepath.Ext(path)
		supportedExts := []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".heif", ".heic", ".mp4", ".mov", ".jxl", ".avif"}

		for _, supportedExt := range supportedExts {
			if ext == supportedExt {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

func generateContent(base string, targetSize int64) string {
	if int64(len(base)) >= targetSize {
		return base[:targetSize]
	}

	// 重复内容直到达到目标大小
	content := base
	for int64(len(content)) < targetSize {
		content += base
	}

	return content[:targetSize]
}

func isAdjacentQualityLevel(actual, expected types.QualityLevel) bool {
	// 定义品质等级的数值
	levels := map[types.QualityLevel]int{
		types.QualityVeryLow:    1,
		types.QualityLow:        2,
		types.QualityMediumLow:  3,
		types.QualityMediumHigh: 4,
		types.QualityHigh:       5,
		types.QualityVeryHigh:   6,
	}

	actualVal, actualExists := levels[actual]
	expectedVal, expectedExists := levels[expected]

	if !actualExists || !expectedExists {
		return false
	}

	// 允许相差1个等级
	diff := actualVal - expectedVal
	return diff >= -1 && diff <= 1
}
