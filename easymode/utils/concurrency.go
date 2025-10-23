// utils/concurrency.go - 并发控制模块
//
// 功能说明：
// - 提供智能线程调整功能
// - 根据系统资源动态调整并发数
// - 防止系统资源过载
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"github.com/shirou/gopsutil/mem"
)

// SmartThreadAdjustment 根据系统内存使用率动态调整工作线程数量
// 智能调整并发线程数，防止系统资源过载，提升处理稳定性
// 当内存占用超过80%时，将工作线程数减半，最低保证为1
// 参数:
//
//	currentWorkers - 当前工作线程数
//
// 返回:
//
//	int - 调整后的工作线程数
func SmartThreadAdjustment(currentWorkers int) int {
	// 如果当前线程数已经是最小值，直接返回
	if currentWorkers <= 1 {
		return 1
	}

	// 获取系统内存使用情况
	v, err := mem.VirtualMemory()
	if err != nil {
		// 获取内存信息失败时，保持当前线程数
		return currentWorkers
	}

	// 当内存使用率超过80%时，减少线程数
	if v.UsedPercent > 80 {
		// 将线程数减半
		nw := currentWorkers / 2
		// 确保线程数不少于1
		if nw < 1 {
			nw = 1
		}
		return nw
	}

	// 内存使用率正常，保持当前线程数
	return currentWorkers
}
