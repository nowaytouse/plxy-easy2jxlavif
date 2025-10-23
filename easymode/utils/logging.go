// utils/logging.go - 日志管理模块
//
// 功能说明：
// - 提供轮转日志记录器
// - 支持日志文件大小控制
// - 同时输出到控制台和文件
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// NewRotatingLogger 创建带大小轮转的日志记录器
// 当日志文件超出指定大小时自动进行轮转，防止日志文件过大
// 参数:
//
//	logFilePath - 日志文件路径
//	maxSizeBytes - 日志文件最大大小（字节），超出后轮转
//
// 返回:
//
//	*log.Logger - 日志记录器实例
//	*os.File - 日志文件句柄
//	error - 创建过程中的错误（如果有）
func NewRotatingLogger(logFilePath string, maxSizeBytes int64) (*log.Logger, *os.File, error) {
	// 检查并执行日志轮转
	if err := rotateIfNeeded(logFilePath, maxSizeBytes); err != nil {
		return nil, nil, err
	}

	// 打开日志文件（追加模式）
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, err
	}

	// 创建同时输出到控制台和文件的日志记录器
	logger := log.New(io.MultiWriter(os.Stdout, f), "", log.LstdFlags)
	return logger, f, nil
}

// rotateIfNeeded 在日志初始化前执行简单的大小轮转
// 当日志文件超过指定大小时，执行轮转：.log -> .log.1 -> .log.2
// 参数:
//
//	path - 日志文件路径
//	max - 最大文件大小（字节）
//
// 返回:
//
//	error - 轮转过程中的错误（如果有）
func rotateIfNeeded(path string, max int64) error {
	// 如果未设置大小限制，跳过轮转
	if max <= 0 {
		return nil
	}

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		// 文件不存在，无需轮转
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// 文件大小未超过限制，无需轮转
	if info.Size() < max {
		return nil
	}

	// 执行轮转操作
	// 注意：关闭同名已打开文件由调用方负责，此处仅做文件级轮转

	// 先移除最旧的 .2 文件
	_ = os.Remove(path + ".2")

	// 将 .1 重命名为 .2
	_ = os.Rename(path+".1", path+".2")

	// 将当前 .log 重命名为 .1
	if err := os.Rename(path, path+".1"); err != nil {
		return err
	}
	// 确保目录存在
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	return nil
}
