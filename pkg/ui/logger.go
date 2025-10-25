package ui

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewInteractiveLogger 创建交互模式专用logger（减少刷屏）
func NewInteractiveLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	// 交互模式：仅显示INFO及以上（隐藏DEBUG）
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	// 简化输出格式（避免刷屏）
	config.Encoding = "console"
	config.EncoderConfig.TimeKey = ""   // 隐藏时间戳
	config.EncoderConfig.LevelKey = ""  // 隐藏级别
	config.EncoderConfig.CallerKey = "" // 隐藏调用位置
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	return config.Build()
}

// NewDebugLogger 创建调试模式logger（显示所有日志）
func NewDebugLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return config.Build()
}

// NewNonInteractiveLogger 创建非交互模式logger
func NewNonInteractiveLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.WarnLevel) // 仅显示警告和错误
	return config.Build()
}
