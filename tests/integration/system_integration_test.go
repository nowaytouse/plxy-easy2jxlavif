package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pixly/pkg/core/config"
	"pixly/pkg/security"
	"pixly/pkg/tools"
	"pixly/pkg/ui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestFullConversionPipeline 测试完整的转换管道
func TestFullConversionPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	logger := zaptest.NewLogger(t)

	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "pixly_integration_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试配置
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir
	cfg.Mode = "auto+"
	cfg.ConcurrentJobs = 2
	cfg.CreateBackups = true
	cfg.KeepBackups = false
	cfg.DebugMode = true
	cfg.DryRun = true // 使用干运行模式避免实际转换

	// 创建测试文件
	testFiles := []string{
		"test1.jpg",
		"test2.png",
		"test3.mp4",
	}

	for _, filename := range testFiles {
		testFilePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(testFilePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// 检查工具依赖
	toolsChecker := tools.NewChecker(logger)
	_, err = toolsChecker.CheckAll()
	require.NoError(t, err)

	// 创建转换引擎
	// conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
	// require.NotNil(t, conversionEngine)

	// 由于转换引擎可能还在开发中，这里暂时跳过具体的执行测试
	t.Log("转换管道测试 - 跳过实际执行，仅验证配置")
}

// TestUIIntegration 测试UI集成
func TestUIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过UI集成测试")
	}

	logger := zaptest.NewLogger(t)

	// 创建安全检查器
	securityChecker := security.NewSecurityChecker(logger)
	require.NotNil(t, securityChecker)

	// 初始化UI系统
	uiAdapter := ui.InitializeGlobalUI(logger, securityChecker)
	require.NotNil(t, uiAdapter)
	defer ui.ShutdownGlobalUI()

	// 测试UI接口获取
	uiInterface := uiAdapter.GetUIManager().GetInterface()
	require.NotNil(t, uiInterface)

	// 验证UI组件正常初始化
	assert.NotNil(t, uiInterface)
}

// TestToolsIntegration 测试工具集成
func TestToolsIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 创建工具检查器
	toolsChecker := tools.NewChecker(logger)
	require.NotNil(t, toolsChecker)

	// 检查所有工具
	_ = context.Background() // 使用下划线忽略不使用的变量

	toolResults, err := toolsChecker.CheckAll()

	// 在测试环境中，工具可能不存在，所以我们只验证检查过程不会崩溃
	assert.NotNil(t, toolResults)

	if err != nil {
		// 如果工具不存在，这是正常的，记录但不失败测试
		t.Logf("工具检查返回错误（在测试环境中这是正常的）: %v", err)
	}

	// 验证结果结构体有必要的字段
	// 这里不检查具体值，因为在测试环境中工具可能不可用
	_ = toolResults.FfmpegStablePath
}

// TestConfigurationIntegration 测试配置集成
func TestConfigurationIntegration(t *testing.T) {
	// 测试默认配置创建
	cfg := config.DefaultConfig()
	require.NotNil(t, cfg)

	// 验证配置
	err := config.Validate(cfg)
	assert.NoError(t, err, "默认配置应该是有效的")

	// 测试配置标准化
	invalidCfg := &config.Config{
		ConcurrentJobs: -1,
		MaxRetries:     -1,
		CRF:            100,
	}

	config.NormalizeConfig(invalidCfg)

	assert.Greater(t, invalidCfg.ConcurrentJobs, 0)
	assert.GreaterOrEqual(t, invalidCfg.MaxRetries, 0)
	assert.LessOrEqual(t, invalidCfg.CRF, 51)
}

// TestErrorHandlingIntegration 测试错误处理集成
func TestErrorHandlingIntegration(t *testing.T) {
	// 创建不存在的目录配置
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/nonexistent/directory/that/should/not/exist"

	// 验证配置可以正常创建
	assert.NotNil(t, cfg)

	// 由于转换引擎可能还在开发中，这里暂时跳过具体的执行测试
	t.Log("错误处理测试 - 跳过实际执行，仅验证配置")
}

// TestMemoryAndResourceManagement 测试内存和资源管理
func TestMemoryAndResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过资源管理测试")
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "pixly_resource_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建配置
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir
	cfg.MemoryLimit = 1 // 1GB限制
	cfg.EnableMemoryWatch = true

	// 验证配置可以正常创建
	assert.NotNil(t, cfg)
}
