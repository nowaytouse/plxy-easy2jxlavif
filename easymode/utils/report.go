// utils/report.go - 报告生成模块
//
// 功能说明：
// - 支持JSON和CSV格式的报告生成
// - 提供统一的文件处理日志结构
// - 支持统计数据的导出和分析
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
)

// ReportFormat 报告格式枚举
// 定义支持的报告输出格式
type ReportFormat string

const (
	ReportJSON ReportFormat = "json" // JSON格式报告
	ReportCSV  ReportFormat = "csv"  // CSV格式报告
)

// FileLog 文件处理日志结构体
// 为各工具上报的最小处理日志项，用于生成处理报告
type FileLog struct {
	FilePath   string `json:"file_path"`           // 文件路径
	FileType   string `json:"file_type"`           // 文件类型
	Success    bool   `json:"success"`             // 处理是否成功
	ErrorMsg   string `json:"error_msg,omitempty"` // 错误信息（如果失败）
	SizeBefore int64  `json:"size_before"`         // 处理前文件大小（字节）
	SizeAfter  int64  `json:"size_after"`          // 处理后文件大小（字节）
	DurationMs int64  `json:"duration_ms"`         // 处理耗时（毫秒）
}

// WriteReport 将日志写出为指定格式的报告文件
// 参数:
//
//	path - 输出文件路径
//	fmt - 报告格式（JSON或CSV）
//	logs - 文件处理日志列表
//
// 返回:
//
//	error - 写入错误（如果有）
func WriteReport(path string, fmt ReportFormat, logs []FileLog) error {
	// 检查路径是否为空
	if path == "" {
		return nil
	}

	// 创建输出文件
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// 根据格式类型生成报告
	switch fmt {
	case ReportJSON:
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(logs)
	case ReportCSV:
		w := csv.NewWriter(f)
		_ = w.Write([]string{"file_path", "file_type", "success", "error_msg", "size_before", "size_after", "duration_ms"})
		for _, l := range logs {
			rec := []string{l.FilePath, l.FileType}
			if l.Success {
				rec = append(rec, "true")
			} else {
				rec = append(rec, "false")
			}
			rec = append(rec,
				l.ErrorMsg,
				int64ToStr(l.SizeBefore),
				int64ToStr(l.SizeAfter),
				int64ToStr(l.DurationMs),
			)
			_ = w.Write(rec)
		}
		w.Flush()
		return w.Error()
	default:
		return nil
	}
}

// int64ToStr 将int64转换为字符串
// 用于CSV格式报告中的数值字段转换
func int64ToStr(v int64) string { return strconv.FormatInt(v, 10) }
