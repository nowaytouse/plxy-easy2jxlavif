package scenarios_test

import (
	"context"
	"testing"
	"time"

	"pixly/pkg/core/config"
	"pixly/pkg/engine"
	"pixly/pkg/security"
	"pixly/pkg/tools"
	"pixly/tests/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScenario_AutoPlusMode 测试场景：自动模式+
func TestScenario_AutoPlusMode(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.Mode = "auto+"
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_QualityMode 测试场景：质量模式
func TestScenario_QualityMode(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.Mode = "quality"
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_StickerMode 测试场景：表情包模式
func TestScenario_StickerMode(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.Mode = "sticker"
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_EmptyDirectory 测试场景：空目录处理
func TestScenario_EmptyDirectory(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	// 创建空目录
	emptyDir := tf.CreateEmptyDirectory("empty")

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.TargetDir = emptyDir
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	// 空目录应该正常处理，不报错
	assert.NoError(t, err)
}

// TestScenario_SubDirectories 测试场景：子目录处理
func TestScenario_SubDirectories(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	// 创建子目录
	tf.CreateSubDirectory("subdir1")
	tf.CreateSubDirectory("subdir2")

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_LargeFiles 测试场景：大文件处理
func TestScenario_LargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过大文件测试")
	}

	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	// 创建一个较大的测试文件 (10MB)
	tf.CreateLargeFile("large_test.jpg", 10)

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.MemoryLimit = 1 // 1GB内存限制
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_HighConcurrency 测试场景：高并发处理
func TestScenario_HighConcurrency(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.ConcurrentJobs = 8 // 高并发设置
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_BackupEnabled 测试场景：启用备份
func TestScenario_BackupEnabled(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.CreateBackups = true
	cfg.KeepBackups = true
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_ErrorRecovery 测试场景：错误恢复
func TestScenario_ErrorRecovery(t *testing.T) {
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	logger := fixtures.GetTestLogger(t)
	cfg := tf.GetTestConfig()
	cfg.MaxRetries = 3
	toolResults := tf.GetTestToolResults()

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	require.NotNil(t, conversionEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := conversionEngine.Execute(ctx)
	assert.NoError(t, err)
}

// TestScenario_SecurityCheck 测试场景：安全检查
func TestScenario_SecurityCheck(t *testing.T) {
	logger := fixtures.GetTestLogger(t)
	securityChecker := security.NewSecurityChecker(logger)

	// 测试安全目录
	tf := fixtures.CreateTestFixtures(t)
	defer tf.Cleanup()

	isSafe := securityChecker.IsDirectorySafe(tf.TempDir)
	assert.True(t, isSafe)

	// 测试不安全目录
	unsafeDirs := []string{"/", "/System", "/usr/bin"}
	for _, dir := range unsafeDirs {
		isSafe := securityChecker.IsDirectorySafe(dir)
		assert.False(t, isSafe, "目录应该被标记为不安全: "+dir)
	}
}

// TestScenario_ToolsCheck 测试场景：工具检查
func TestScenario_ToolsCheck(t *testing.T) {
	logger := fixtures.GetTestLogger(t)
	toolsChecker := tools.NewChecker(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	toolResults, err := toolsChecker.CheckAll()
	// 在测试环境中工具可能不存在，所以不强制要求成功
	assert.NotNil(t, toolResults)
	if err != nil {
		t.Logf("工具检查返回错误（测试环境正常）: %v", err)
	}
}

// TestScenario_ConfigurationEdgeCases 测试场景：配置边界情况
func TestScenario_ConfigurationEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		configModifier func(*config.Config)
		expectSuccess  bool
	}{
		{
			name: "极小并发数",
			configModifier: func(c *config.Config) {
				c.ConcurrentJobs = 1
			},
			expectSuccess: true,
		},
		{
			name: "极大并发数",
			configModifier: func(c *config.Config) {
				c.ConcurrentJobs = 16
			},
			expectSuccess: true,
		},
		{
			name: "禁用硬件加速",
			configModifier: func(c *config.Config) {
				c.HwAccel = false
			},
			expectSuccess: true,
		},
		{
			name: "最大重试次数",
			configModifier: func(c *config.Config) {
				c.MaxRetries = 10
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := fixtures.CreateTestFixtures(t)
			defer tf.Cleanup()

			logger := fixtures.GetTestLogger(t)
			cfg := tf.GetTestConfig()
			tt.configModifier(cfg)

			// 验证配置
			err := config.Validate(cfg)
			if tt.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				return
			}

			toolResults := tf.GetTestToolResults()
			conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
			require.NotNil(t, conversionEngine)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			err = conversionEngine.Execute(ctx)
			if tt.expectSuccess {
				assert.NoError(t, err)
			}
		})
	}
}
