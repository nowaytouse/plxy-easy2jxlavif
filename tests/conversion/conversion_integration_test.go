package conversion

import (
	"context"
	"os"
	"testing"

	"pixly/pkg/core/types"
	"pixly/pkg/engine"
	"pixly/pkg/engine/quality"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// TestConversionIntegration 测试真实的转换集成功能
func TestConversionIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 检查工具可用性
	toolPaths := types.ToolCheckResults{
		HasCjxl:    true, // 假设有cjxl
		HasAvifenc: true, // 假设有avifenc
		HasFfmpeg:  true, // 假设有ffmpeg
	}

	// 创建质量引擎
	qualityEngine := quality.NewQualityEngine(logger, "", "", true)

	// 创建处理模式管理器
	processingManager := engine.NewProcessingModeManager(logger, toolPaths, qualityEngine)

	// 测试文件信息
	testFile := &types.MediaInfo{
		Path: "/Users/nameko_1/Documents/Pixly/Go_Source_code_Updata/examples/file_example_JPG_2500kB.jpg",
		Type: types.MediaTypeImage,
		Size: 2500 * 1024, // 2.5MB
	}

	// 检查文件是否存在
	if _, err := os.Stat(testFile.Path); os.IsNotExist(err) {
		t.Skip("测试文件不存在，跳过真实转换测试")
		return
	}

	ctx := context.Background()

	t.Run("自动模式+转换测试", func(t *testing.T) {
		autoMode := processingManager.GetMode(types.ModeAutoPlus)

		// 检查是否应该跳过
		shouldSkip, reason := autoMode.ShouldSkipFile(testFile)
		if shouldSkip {
			t.Logf("文件被跳过: %s", reason)
			return
		}

		// 执行转换
		result, err := autoMode.ProcessFile(ctx, testFile)

		// 这里不要求一定成功，因为可能没有工具
		if err != nil {
			t.Logf("转换失败（可能是工具不可用）: %v", err)
		} else {
			t.Logf("转换结果: 成功=%v, 原始大小=%d, 新大小=%d",
				result.Success, result.OriginalSize, result.NewSize)
		}

		assert.NotNil(t, result, "应该返回转换结果")
	})

	t.Run("品质模式转换测试", func(t *testing.T) {
		qualityMode := processingManager.GetMode(types.ModeQuality)

		// 检查是否应该跳过
		shouldSkip, reason := qualityMode.ShouldSkipFile(testFile)
		if shouldSkip {
			t.Logf("文件被跳过: %s", reason)
			return
		}

		// 执行转换
		result, err := qualityMode.ProcessFile(ctx, testFile)

		// 这里不要求一定成功，因为可能没有工具
		if err != nil {
			t.Logf("转换失败（可能是工具不可用）: %v", err)
		} else {
			t.Logf("转换结果: 成功=%v, 原始大小=%d, 新大小=%d",
				result.Success, result.OriginalSize, result.NewSize)
		}

		assert.NotNil(t, result, "应该返回转换结果")
	})

	t.Run("表情包模式转换测试", func(t *testing.T) {
		emojiMode := processingManager.GetMode(types.ModeEmoji)

		// 检查是否应该跳过
		shouldSkip, reason := emojiMode.ShouldSkipFile(testFile)
		if shouldSkip {
			t.Logf("文件被跳过: %s", reason)
			return
		}

		// 执行转换
		result, err := emojiMode.ProcessFile(ctx, testFile)

		// 这里不要求一定成功，因为可能没有工具
		if err != nil {
			t.Logf("转换失败（可能是工具不可用）: %v", err)
		} else {
			t.Logf("转换结果: 成功=%v, 原始大小=%d, 新大小=%d",
				result.Success, result.OriginalSize, result.NewSize)
		}

		assert.NotNil(t, result, "应该返回转换结果")
	})
}

// TestProcessingStrategy 测试处理策略生成
func TestProcessingStrategy(t *testing.T) {
	logger := zaptest.NewLogger(t)

	toolPaths := types.ToolCheckResults{
		HasCjxl:    true,
		HasAvifenc: true,
		HasFfmpeg:  true,
	}

	qualityEngine := quality.NewQualityEngine(logger, "", "", true)
	processingManager := engine.NewProcessingModeManager(logger, toolPaths, qualityEngine)

	testFile := &types.MediaInfo{
		Path: "test.jpg",
		Type: types.MediaTypeImage,
		Size: 1024 * 1024,
	}

	modes := []types.AppMode{
		types.ModeAutoPlus,
		types.ModeQuality,
		types.ModeEmoji,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			processingMode := processingManager.GetMode(mode)
			strategy, err := processingMode.GetStrategy(testFile)

			assert.NoError(t, err, "获取策略不应该失败")
			assert.NotNil(t, strategy, "应该返回策略")
			assert.Equal(t, mode, strategy.Mode, "策略模式应该匹配")

			t.Logf("模式: %s, 目标格式: %s, 质量: %s, 置信度: %.2f",
				strategy.Mode.String(), strategy.TargetFormat, strategy.Quality, strategy.Confidence)
		})
	}
}
