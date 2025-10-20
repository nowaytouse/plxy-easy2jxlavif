package benchmark_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"
	"pixly/pkg/engine"

	"go.uber.org/zap"
)

// BenchmarkConversionEngineCreation 基准测试：转换引擎创建
func BenchmarkConversionEngineCreation(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/tmp/pixly_benchmark"

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := engine.NewConversionEngine(logger, cfg, toolResults, nil)
		_ = engine
	}
}

// BenchmarkConfigValidation 基准测试：配置验证
func BenchmarkConfigValidation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := config.Validate(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigNormalization 基准测试：配置标准化
func BenchmarkConfigNormalization(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg := &config.Config{
			ConcurrentJobs: -1,
			MaxRetries:     -1,
			CRF:            100,
			MemoryLimit:    0,
		}

		config.NormalizeConfig(cfg)
	}
}

// BenchmarkFileScanning 基准测试：文件扫描性能
func BenchmarkFileScanning(b *testing.B) {
	// 创建临时目录和测试文件
	tempDir, err := os.MkdirTemp("", "pixly_benchmark_scan_")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFiles := []string{
		"test1.jpg", "test2.png", "test3.mp4", "test4.mov",
		"test5.webp", "test6.heic", "test7.avif", "test8.jxl",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte("benchmark test content"), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir
	cfg.DryRun = true

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = conversionEngine.Execute(ctx)
		cancel()
	}
}

// BenchmarkConcurrentProcessing 基准测试：并发处理性能
func BenchmarkConcurrentProcessing(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "pixly_benchmark_concurrent_")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多个测试文件
	for i := 0; i < 20; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.jpg", i))
		err := os.WriteFile(filename, []byte("concurrent test content"), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	logger := zap.NewNop()

	// 测试不同的并发级别
	concurrencyLevels := []int{1, 2, 4, 8}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			cfg := config.DefaultConfig()
			cfg.TargetDir = tempDir
			cfg.ConcurrentJobs = concurrency
			cfg.DryRun = true

			toolResults := types.ToolCheckResults{
				FfmpegStablePath: "/usr/local/bin/ffmpeg",
			}

			conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = conversionEngine.Execute(ctx)
				cancel()
			}
		})
	}
}

// BenchmarkMemoryUsage 基准测试：内存使用情况
func BenchmarkMemoryUsage(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/tmp/pixly_benchmark"
	cfg.MemoryLimit = 2 // 2GB限制

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := engine.NewConversionEngine(logger, cfg, toolResults, nil)

		// 模拟一些工作负载
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = engine.Execute(ctx)
		cancel()
	}
}

// BenchmarkErrorHandling 基准测试：错误处理性能
func BenchmarkErrorHandling(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.TargetDir = "/nonexistent/directory" // 故意使用不存在的目录

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_ = conversionEngine.Execute(ctx) // 预期会失败
		cancel()
	}
}

// BenchmarkRetryMechanism 基准测试：重试机制性能
func BenchmarkRetryMechanism(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "pixly_benchmark_retry_")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 创建一个测试文件
	testFile := filepath.Join(tempDir, "test.jpg")
	err = os.WriteFile(testFile, []byte("retry test content"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	logger := zap.NewNop()
	cfg := config.DefaultConfig()
	cfg.TargetDir = tempDir
	cfg.MaxRetries = 3
	cfg.DryRun = true

	toolResults := types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
	}

	conversionEngine := engine.NewConversionEngine(logger, cfg, toolResults, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = conversionEngine.Execute(ctx)
		cancel()
	}
}
