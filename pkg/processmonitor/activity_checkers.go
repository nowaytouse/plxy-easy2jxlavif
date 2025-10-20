package processmonitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// CPUActivityChecker CPU活动检查器
type CPUActivityChecker struct {
	name        string
	lastCPUTime int64
	threshold   float64 // CPU使用率阈值
}

// DiskIOActivityChecker 磁盘IO活动检查器
type DiskIOActivityChecker struct {
	name           string
	lastReadBytes  int64
	lastWriteBytes int64
	threshold      int64 // IO字节阈值
}

// MemoryActivityChecker 内存活动检查器
type MemoryActivityChecker struct {
	name         string
	lastMemUsage int64
	threshold    int64 // 内存变化阈值（字节）
}

// NewCPUActivityChecker 创建CPU活动检查器
func NewCPUActivityChecker() ActivityChecker {
	return &CPUActivityChecker{
		name:      "cpu_activity",
		threshold: 5.0, // 5% CPU使用率阈值
	}
}

// NewDiskIOActivityChecker 创建磁盘IO活动检查器
func NewDiskIOActivityChecker() ActivityChecker {
	return &DiskIOActivityChecker{
		name:      "disk_io_activity",
		threshold: 1024 * 1024, // 1MB IO阈值
	}
}

// NewMemoryActivityChecker 创建内存活动检查器
func NewMemoryActivityChecker() ActivityChecker {
	return &MemoryActivityChecker{
		name:      "memory_activity",
		threshold: 10 * 1024 * 1024, // 10MB内存变化阈值
	}
}

// CheckActivity 检查CPU活动
func (checker *CPUActivityChecker) CheckActivity(process *MonitoredProcess) bool {
	if process.Cmd == nil || process.Cmd.Process == nil {
		return false
	}

	// 读取进程CPU使用情况
	cpuTime, err := checker.readCPUTime(process.PID)
	if err != nil {
		return false
	}

	// 首次检查，记录初始值
	if checker.lastCPUTime == 0 {
		checker.lastCPUTime = cpuTime
		return true
	}

	// 计算CPU时间差
	timeDiff := cpuTime - checker.lastCPUTime
	checker.lastCPUTime = cpuTime

	// 如果CPU时间有增长，说明进程在活动
	isActive := timeDiff > 0

	if isActive {
		// 更新资源使用统计
		if process.ResourceUsage != nil {
			process.ResourceUsage.CPUPercent = float64(timeDiff)
			process.ResourceUsage.LastUpdate = time.Now()
		}
	}

	return isActive
}

// GetName 获取检查器名称
func (checker *CPUActivityChecker) GetName() string {
	return checker.name
}

// CheckActivity 检查磁盘IO活动
func (checker *DiskIOActivityChecker) CheckActivity(process *MonitoredProcess) bool {
	if process.Cmd == nil || process.Cmd.Process == nil {
		return false
	}

	// 读取进程IO统计
	readBytes, writeBytes, err := checker.readIOStats(process.PID)
	if err != nil {
		return false
	}

	// 首次检查，记录初始值
	if checker.lastReadBytes == 0 && checker.lastWriteBytes == 0 {
		checker.lastReadBytes = readBytes
		checker.lastWriteBytes = writeBytes
		return true
	}

	// 计算IO变化
	readDiff := readBytes - checker.lastReadBytes
	writeDiff := writeBytes - checker.lastWriteBytes

	checker.lastReadBytes = readBytes
	checker.lastWriteBytes = writeBytes

	// 如果读写字节数有显著增长，说明进程在活动
	isActive := (readDiff + writeDiff) > checker.threshold

	if isActive {
		// 更新资源使用统计
		if process.ResourceUsage != nil {
			process.ResourceUsage.DiskReadMB = float64(readBytes) / (1024 * 1024)
			process.ResourceUsage.DiskWriteMB = float64(writeBytes) / (1024 * 1024)
			process.ResourceUsage.LastUpdate = time.Now()
		}
	}

	return isActive
}

// GetName 获取检查器名称
func (checker *DiskIOActivityChecker) GetName() string {
	return checker.name
}

// CheckActivity 检查内存活动
func (checker *MemoryActivityChecker) CheckActivity(process *MonitoredProcess) bool {
	if process.Cmd == nil || process.Cmd.Process == nil {
		return false
	}

	// 读取进程内存使用情况
	memUsage, err := checker.readMemoryUsage(process.PID)
	if err != nil {
		return false
	}

	// 首次检查，记录初始值
	if checker.lastMemUsage == 0 {
		checker.lastMemUsage = memUsage
		return true
	}

	// 计算内存变化
	memDiff := abs64(memUsage - checker.lastMemUsage)
	checker.lastMemUsage = memUsage

	// 如果内存使用有显著变化，说明进程在活动
	isActive := memDiff > checker.threshold

	if isActive {
		// 更新资源使用统计
		if process.ResourceUsage != nil {
			process.ResourceUsage.MemoryMB = float64(memUsage) / (1024 * 1024)
			process.ResourceUsage.LastUpdate = time.Now()
		}
	}

	return isActive
}

// GetName 获取检查器名称
func (checker *MemoryActivityChecker) GetName() string {
	return checker.name
}

// readCPUTime 读取进程CPU时间（适用于Linux/macOS）
func (checker *CPUActivityChecker) readCPUTime(pid int) (int64, error) {
	// 尝试读取 /proc/PID/stat 文件（Linux）
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	if data, err := os.ReadFile(statFile); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 15 {
			// 字段13和14是utime和stime（用户态和内核态CPU时间）
			utime, err1 := strconv.ParseInt(fields[13], 10, 64)
			stime, err2 := strconv.ParseInt(fields[14], 10, 64)
			if err1 == nil && err2 == nil {
				return utime + stime, nil
			}
		}
	}

	// 如果读取失败，返回一个基于时间的估算值
	// 这不是真实的CPU时间，但可以作为活动指示器
	return time.Now().UnixNano() / 1000000, nil
}

// readIOStats 读取进程IO统计（适用于Linux）
func (checker *DiskIOActivityChecker) readIOStats(pid int) (readBytes, writeBytes int64, err error) {
	// 尝试读取 /proc/PID/io 文件（Linux）
	ioFile := fmt.Sprintf("/proc/%d/io", pid)
	if data, err := os.ReadFile(ioFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "read_bytes:") {
				if val, parseErr := strconv.ParseInt(strings.Fields(line)[1], 10, 64); parseErr == nil {
					readBytes = val
				}
			} else if strings.HasPrefix(line, "write_bytes:") {
				if val, parseErr := strconv.ParseInt(strings.Fields(line)[1], 10, 64); parseErr == nil {
					writeBytes = val
				}
			}
		}
		return readBytes, writeBytes, nil
	}

	// 如果读取失败，返回基于时间的估算值
	now := time.Now().UnixNano() / 1000000
	return now, now, nil
}

// readMemoryUsage 读取进程内存使用情况
func (checker *MemoryActivityChecker) readMemoryUsage(pid int) (int64, error) {
	// 尝试读取 /proc/PID/status 文件（Linux）
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	if data, err := os.ReadFile(statusFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if val, parseErr := strconv.ParseInt(fields[1], 10, 64); parseErr == nil {
						return val * 1024, nil // 转换为字节
					}
				}
			}
		}
	}

	// 尝试读取 /proc/PID/statm 文件作为备选方案
	statmFile := fmt.Sprintf("/proc/%d/statm", pid)
	if data, err := os.ReadFile(statmFile); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 2 {
			// 第二个字段是RSS（常驻内存大小，以页为单位）
			if pages, parseErr := strconv.ParseInt(fields[1], 10, 64); parseErr == nil {
				pageSize := int64(4096) // 假设页大小为4KB
				return pages * pageSize, nil
			}
		}
	}

	// 如果所有方法都失败，返回一个基于时间的估算值
	return time.Now().UnixNano() / 1000000, nil
}


