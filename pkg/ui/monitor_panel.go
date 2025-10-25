package ui

import (
	"fmt"
	"time"

	"pixly/pkg/monitor"

	"github.com/pterm/pterm"
)

// MonitorPanel 实时监控面板
// 使用pterm显示系统性能指标，每3秒刷新一次
type MonitorPanel struct {
	enable   bool
	position string // top, bottom, floating
	interval time.Duration

	// pterm组件
	area *pterm.AreaPrinter

	// 显示状态
	isActive bool
	lastUpdate time.Time
}

// NewMonitorPanel 创建监控面板
func NewMonitorPanel(enable bool, position string, interval time.Duration) *MonitorPanel {
	return &MonitorPanel{
		enable:   enable,
		position: position,
		interval: interval,
	}
}

// Start 启动监控面板
func (mp *MonitorPanel) Start() error {
	if !mp.enable {
		return nil
	}

	mp.area, _ = pterm.DefaultArea.Start()
	mp.isActive = true
	return nil
}

// Update 更新监控面板显示
func (mp *MonitorPanel) Update(metrics *monitor.PerformanceMetrics, currentWorkers int) {
	if !mp.enable || !mp.isActive {
		return
	}

	// 检查更新间隔
	if time.Since(mp.lastUpdate) < mp.interval {
		return
	}

	// 生成监控面板内容
	content := mp.renderPanel(metrics, currentWorkers)

	// 更新显示
	if mp.area != nil {
		mp.area.Update(content)
		mp.lastUpdate = time.Now()
	}
}

// Stop 停止监控面板
func (mp *MonitorPanel) Stop() {
	if mp.area != nil {
		mp.area.Stop()
		mp.isActive = false
	}
}

// renderPanel 渲染监控面板
func (mp *MonitorPanel) renderPanel(metrics *monitor.PerformanceMetrics, currentWorkers int) string {
	// CPU进度条
	cpuBar := renderProgressBar(metrics.CPUUsage, 20)
	cpuPercent := metrics.CPUUsage * 100

	// 内存进度条
	memBar := renderProgressBar(metrics.MemoryUsage, 20)
	memPercent := metrics.MemoryUsage * 100
	memGB := float64(metrics.MemoryUsed) / 1024 / 1024 / 1024
	memTotalGB := float64(metrics.MemoryTotal) / 1024 / 1024 / 1024

	// 磁盘进度条
	diskBar := renderProgressBar(metrics.DiskUsagePercent, 20)
	diskPercent := metrics.DiskUsagePercent * 100

	// 组装面板
	panel := fmt.Sprintf(`
┌─────────────── 🖥️  系统监控 ───────────────┐
│                                             │
│ CPU:    %s %.1f%%      │
│         核心: %d | 负载: %.2f %.2f %.2f     │
│                                             │
│ 内存:   %s %.1f%%      │
│         使用: %.1f GB / %.1f GB            │
│         Swap: %.1f%%                        │
│                                             │
│ 磁盘:   %s %.1f%%      │
│         读: %.1f MB/s | 写: %.1f MB/s       │
│                                             │
│ 进程:   协程: %4d | 线程: %4d             │
│         内存: %4d MB | GC: %d次            │
│                                             │
│ Worker: %2d / %2d (当前/最大)              │
│         队列: %4d 个文件                    │
│                                             │
│ 性能:   吞吐: %.1f 文件/秒                  │
│         速度: %.1f MB/秒                    │
│         错误: %.2f%%                        │
│                                             │
│ 运行:   %s                       │
│                                             │
└─────────────────────────────────────────────┘
`,
		cpuBar, cpuPercent,
		metrics.CPUCores, metrics.LoadAverage1, metrics.LoadAverage5, metrics.LoadAverage15,
		memBar, memPercent,
		memGB, memTotalGB,
		metrics.SwapUsage * 100,
		diskBar, diskPercent,
		metrics.DiskReadSpeed, metrics.DiskWriteSpeed,
		metrics.GoroutineCount, metrics.ThreadCount,
		int(metrics.ProcessMemory/1024/1024), metrics.GCCount,
		currentWorkers, metrics.MaxWorkers,
		metrics.QueueLength,
		metrics.Throughput,
		metrics.ProcessingRate,
		metrics.ErrorRate * 100,
		formatDuration(metrics.Uptime),
	)

	return panel
}

// renderProgressBar 渲染ASCII进度条
func renderProgressBar(value float64, width int) string {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}

	filled := int(value * float64(width))
	empty := width - filled

	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	bar += "]"

	return bar
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// RenderCompactMetrics 渲染紧凑型指标（用于日志）
func RenderCompactMetrics(metrics *monitor.PerformanceMetrics) string {
	return fmt.Sprintf(
		"CPU:%.1f%% MEM:%.1f%% DISK:%.1f%% Workers:%d 吞吐:%.1f/s",
		metrics.CPUUsage*100,
		metrics.MemoryUsage*100,
		metrics.DiskUsagePercent*100,
		metrics.CurrentWorkers,
		metrics.Throughput,
	)
}

// RenderSimplePanel 渲染简化监控面板（单行）
func RenderSimplePanel(metrics *monitor.PerformanceMetrics) string {
	cpuIcon := getResourceIcon(metrics.CPUUsage)
	memIcon := getResourceIcon(metrics.MemoryUsage)
	
	return fmt.Sprintf(
		"%s CPU %.0f%% | %s MEM %.0f%% | ⚡ %.1f文件/s | 🔧 %d workers",
		cpuIcon, metrics.CPUUsage*100,
		memIcon, metrics.MemoryUsage*100,
		metrics.Throughput,
		metrics.CurrentWorkers,
	)
}

// getResourceIcon 根据使用率获取图标
func getResourceIcon(usage float64) string {
	if usage > 0.85 {
		return "🔴" // 高负载
	} else if usage > 0.65 {
		return "🟡" // 中负载
	} else {
		return "🟢" // 低负载
	}
}


