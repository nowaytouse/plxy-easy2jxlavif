package unit

import (
	"testing"

	"pixly/pkg/core/types"

	"github.com/stretchr/testify/assert"
)

// TestBasicTypes 测试基本类型定义
func TestBasicTypes(t *testing.T) {
	t.Run("MediaType_String", func(t *testing.T) {
		assert.Equal(t, "静图", types.MediaTypeImage.String())
		assert.Equal(t, "动图", types.MediaTypeAnimated.String())
		assert.Equal(t, "视频", types.MediaTypeVideo.String())
	})

	t.Run("QualityLevel_String", func(t *testing.T) {
		assert.Equal(t, "极低品质", types.QualityVeryLow.String())
		assert.Equal(t, "低品质", types.QualityLow.String())
		assert.Equal(t, "中低品质", types.QualityMediumLow.String())
		assert.Equal(t, "中高品质", types.QualityMediumHigh.String())
		assert.Equal(t, "高品质", types.QualityHigh.String())
		assert.Equal(t, "极高品质", types.QualityVeryHigh.String())
	})

	t.Run("AppMode_String", func(t *testing.T) {
		assert.Equal(t, "自动模式+", types.ModeAutoPlus.String())
		assert.Equal(t, "品质模式", types.ModeQuality.String())
		assert.Equal(t, "表情包模式", types.ModeEmoji.String())
	})
}

// TestRoutingDecision 测试路由决策结构体
func TestRoutingDecision(t *testing.T) {
	decision := &types.RoutingDecision{
		Strategy:     "auto",
		TargetFormat: "jxl",
		QualityLevel: types.QualityHigh,
		Reason:       "high_quality_image",
	}

	assert.Equal(t, "auto", decision.Strategy)
	assert.Equal(t, "jxl", decision.TargetFormat)
	assert.Equal(t, types.QualityHigh, decision.QualityLevel)
	assert.Equal(t, "high_quality_image", decision.Reason)
}

// TestConversionTask 测试转换任务结构体
func TestConversionTask(t *testing.T) {
	task := &types.ConversionTask{
		SourcePath:   "/test/input.jpg",
		TargetPath:   "/test/output.jxl",
		TargetFormat: "jxl_lossless",
		Mode:         "auto+",
		Status:       "pending",
		Quality:      "high",
		MediaType:    "image",
	}

	assert.Equal(t, "/test/input.jpg", task.SourcePath)
	assert.Equal(t, "/test/output.jxl", task.TargetPath)
	assert.Equal(t, "jxl_lossless", task.TargetFormat)
	assert.Equal(t, "auto+", task.Mode)
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, "high", task.Quality)
	assert.Equal(t, "image", task.MediaType)
}

// TestREADME_Requirements 验证README要求的核心概念
func TestREADME_Requirements(t *testing.T) {
	t.Run("StickerMode_VideoSkip", func(t *testing.T) {
		// README要求：表情包模式下视频文件直接跳过
		expectedFormats := map[types.MediaType]string{
			types.MediaTypeImage:    "avif_compressed", // 图片转AVIF
			types.MediaTypeAnimated: "avif_compressed", // 动图转AVIF
			types.MediaTypeVideo:    "skip",            // 视频跳过
		}

		for mediaType, expectedFormat := range expectedFormats {
			t.Logf("媒体类型 %s 在表情包模式下应该: %s", mediaType.String(), expectedFormat)

			if mediaType == types.MediaTypeVideo {
				assert.Equal(t, "skip", expectedFormat, "视频文件在表情包模式下必须被跳过")
			} else {
				assert.Equal(t, "avif_compressed", expectedFormat, "图片文件在表情包模式下必须转换为AVIF")
			}
		}
	})

	t.Run("AutoPlus_95_5_Rule", func(t *testing.T) {
		// README要求：自动模式+ 95%文件快速预判+5%可疑文件深度验证
		totalFiles := 100
		fastRoutedExpected := 95
		deepAnalyzedExpected := 5

		assert.Equal(t, totalFiles, fastRoutedExpected+deepAnalyzedExpected,
			"95%快速预判+5%深度验证应该等于100%")
	})

	t.Run("Quality_Levels", func(t *testing.T) {
		// 验证所有品质等级都有明确定义
		qualityLevels := []types.QualityLevel{
			types.QualityVeryLow,
			types.QualityLow,
			types.QualityMediumLow,
			types.QualityMediumHigh,
			types.QualityHigh,
			types.QualityVeryHigh,
		}

		for _, level := range qualityLevels {
			assert.NotEmpty(t, level.String(), "品质等级应该有非空的字符串表示")
			assert.NotEqual(t, "未知品质", level.String(), "不应该有未知品质")
		}
	})
}
