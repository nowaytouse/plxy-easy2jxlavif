package config_test

import (
	"testing"

	"pixly/pkg/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// 验证关键字段
	assert.Equal(t, "auto+", cfg.Mode)
	assert.True(t, cfg.CreateBackups)
	assert.True(t, cfg.HwAccel)
	assert.Equal(t, 2, cfg.MaxRetries)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 28, cfg.CRF)

	// 验证并发设置
	assert.Greater(t, cfg.ConcurrentJobs, 0)
	assert.LessOrEqual(t, cfg.ConcurrentJobs, 32)

	// 验证内存设置
	assert.Equal(t, uint64(8), cfg.MemoryLimit)

	// 验证质量设置
	assert.True(t, cfg.EnableQualityAssessment)
	assert.Greater(t, cfg.HighQualityThreshold, 0.0)
	assert.Greater(t, cfg.LowQualityThreshold, 0.0)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config",
			config: &config.Config{
				ConcurrentJobs: 4,
				MaxWorkers:     6,
				MemoryLimit:    8,
				MaxRetries:     3,
				CRF:            28,
				JXLEffort:      7,
				AVIFSpeed:      6,
			},
			expectError: false,
		},
		{
			name: "invalid_concurrent_jobs_too_low",
			config: &config.Config{
				ConcurrentJobs: 0,
				MaxWorkers:     6,
				MemoryLimit:    8,
				MaxRetries:     3,
				CRF:            28,
				JXLEffort:      7,
				AVIFSpeed:      6,
			},
			expectError: true,
			errorMsg:    "无效的并发任务数",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.Validate(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigNormalization(t *testing.T) {
	// 创建一个有问题的配置
	cfg := &config.Config{
		ConcurrentJobs: -1,  // 无效值
		MaxRetries:     -5,  // 无效值
		CRF:            100, // 无效值
		MemoryLimit:    0,   // 无效值
		Mode:           "",  // 空值
		LogLevel:       "",  // 空值
	}

	// 标准化配置
	config.NormalizeConfig(cfg)

	// 验证修复后的值
	assert.Greater(t, cfg.ConcurrentJobs, 0)
	assert.LessOrEqual(t, cfg.ConcurrentJobs, 32)
	assert.GreaterOrEqual(t, cfg.MaxRetries, 0)
	assert.LessOrEqual(t, cfg.MaxRetries, 10)
	assert.GreaterOrEqual(t, cfg.CRF, 0)
	assert.LessOrEqual(t, cfg.CRF, 51)
	assert.Greater(t, cfg.MemoryLimit, uint64(0))
	assert.NotEmpty(t, cfg.Mode)
	assert.NotEmpty(t, cfg.LogLevel)
}
