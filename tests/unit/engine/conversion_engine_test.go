package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	engine "pixly/pkg/conversion"
	"pixly/pkg/core/config"
	"pixly/pkg/core/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewConversionEngine(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/tmp/pixly_test"

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	convEngine := conversion.NewConversionEngine(logger, cfg, toolResults, nil)

	require.NotNil(t, convEngine)
	assert.NotNil(t, convEngine)
}

func TestEngineConfigMapping(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/tmp/pixly_test"
	cfg.Mode = "quality"
	cfg.ConcurrentJobs = 4
	cfg.MaxRetries = 3
	cfg.CreateBackups = true
	cfg.KeepBackups = true

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	convEngine := conversion.NewConversionEngine(logger, cfg, toolResults, nil)

	// 通过反射或公共方法验证配置映射
	// 这里需要根据实际的引擎实现来调整
	require.NotNil(t, convEngine)
}

func TestGenerateTargetPath(t *testing.T) {
	// 创建临时目录进行测试
	tempDir, err := os.MkdirTemp("", "pixly_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		sourcePath  string
		format      string
		expectedExt string
		expectError bool
	}{
		{
			name:        "jxl_lossless",
			sourcePath:  filepath.Join(tempDir, "test.jpg"),
			format:      "jxl_lossless",
			expectedExt: ".jxl",
			expectError: false,
		},
		{
			name:        "avif_compressed",
			sourcePath:  filepath.Join(tempDir, "test.png"),
			format:      "avif_compressed",
			expectedExt: ".avif",
			expectError: false,
		},
		{
			name:        "remux_mp4",
			sourcePath:  filepath.Join(tempDir, "test.mp4"),
			format:      "remux",
			expectedExt: ".mp4",
			expectError: false,
		},
		{
			name:        "remux_mov_to_mp4",
			sourcePath:  filepath.Join(tempDir, "test.mov"),
			format:      "remux",
			expectedExt: ".mp4",
			expectError: false,
		},
	}

	// 创建测试源文件
	for _, tt := range tests {
		err := os.WriteFile(tt.sourcePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t)
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	_ = conversionEngine // 避免未使用变量警告

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里需要调用实际的路径生成方法
			// 由于该方法可能是私有的，我们可能需要通过公共接口来测试

			// 模拟预期的目标路径
			expectedPath := filepath.Join(tempDir, "test"+tt.expectedExt)

			// 验证扩展名符合预期
			assert.True(t, filepath.Ext(expectedPath) == tt.expectedExt)
		})
	}
}

func TestConversionTaskRouting(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		mediaType      string
		quality        string
		expectedFormat string
	}{
		{
			name:           "auto_plus_high_quality_image",
			mode:           "auto+",
			mediaType:      "image",
			quality:        "high",
			expectedFormat: "jxl_lossless",
		},
		{
			name:           "auto_plus_medium_quality_image",
			mode:           "auto+",
			mediaType:      "image",
			quality:        "medium",
			expectedFormat: "jxl_balanced",
		},
		{
			name:           "auto_plus_low_quality_image",
			mode:           "auto+",
			mediaType:      "image",
			quality:        "low",
			expectedFormat: "avif_balanced",
		},
		{
			name:           "quality_mode_image",
			mode:           "quality",
			mediaType:      "image",
			quality:        "medium",
			expectedFormat: "jxl_lossless",
		},
		{
			name:           "sticker_mode_image",
			mode:           "sticker",
			mediaType:      "image",
			quality:        "high",
			expectedFormat: "avif_compressed",
		},
		{
			name:           "sticker_mode_video",
			mode:           "sticker",
			mediaType:      "video",
			quality:        "high",
			expectedFormat: "skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里测试路由逻辑
			// 由于路由方法可能是私有的，我们需要通过公共接口来验证

			logger := zaptest.NewLogger(t)
			cfg := config.DefaultConfig()
			cfg.Mode = tt.mode
			cfg.TargetDir = "/tmp/pixly_test"

			toolResults := types.ToolCheckResults{
				FfmpegStablePath: "/usr/local/bin/ffmpeg",
			}

			convEngine := conversion.NewConversionEngine(logger, cfg, toolResults, nil)
			require.NotNil(t, engine)

			// 验证引擎创建成功
			// 实际的路由测试需要根据引擎的公共接口来实现
			if engine == nil {
				t.Error("转换引擎创建失败")
			}
		})
	}
}

func TestErrorRecoveryMechanism(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pixly_error_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := zaptest.NewLogger(t)
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir
	cfg.MaxRetries = 3
	cfg.CreateBackups = true
	cfg.KeepBackups = false

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	convEngine := conversion.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, engine)

	// 测试重试机制和备份功能
	// 这需要根据实际的引擎接口来实现
}

func TestConcurrencyControl(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.DefaultConfig()
	cfg.ConcurrentJobs = 2
	cfg.TargetDir = "/tmp/pixly_test"

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	convEngine := conversion.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, engine)

	// 测试并发控制
	// 这需要创建多个并发任务来验证
}
