package memory

import (
	"context"

	"go.uber.org/zap"
)

// MemoryMonitor 内存监控器
type MemoryMonitor struct {
	logger *zap.Logger
}

// NewMemoryMonitor 创建新的内存监控器
func NewMemoryMonitor(logger *zap.Logger) *MemoryMonitor {
	return &MemoryMonitor{
		logger: logger,
	}
}

// SetCallbacks 设置回调函数
func (mm *MemoryMonitor) SetCallbacks(warnCallback func(float64), criticalCallback func(float64), lowMemCallback func()) {
	// 占位实现
	mm.logger.Debug("设置内存监控回调")
}

// Start 启动监控
func (mm *MemoryMonitor) Start(ctx context.Context) {
	// 占位实现
	mm.logger.Debug("启动内存监控")
}

// Stop 停止监控
func (mm *MemoryMonitor) Stop() {
	// 占位实现
	mm.logger.Debug("停止内存监控")
}
