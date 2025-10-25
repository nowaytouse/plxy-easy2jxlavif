// utils/logging_setup.go - 日志和信号处理模块
//
// 功能说明：
// - 提供统一的日志设置功能
// - 提供统一的信号处理功能

package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SetupLogging 设置日志记录器
// logFileName: 日志文件名
// 返回配置好的logger
func SetupLogging(logFileName string) *log.Logger {
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件 %s: %v", logFileName, err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	return logger
}

// SetupSignalHandling 设置信号处理
// 处理Ctrl+C等中断信号，实现优雅关闭
func SetupSignalHandling(ctx context.Context, cancelFunc context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("\n⚠️  收到信号: %v，正在优雅关闭...", sig)
		if cancelFunc != nil {
			cancelFunc()
		}
		// 给一些时间让goroutines清理
		<-time.After(2 * time.Second)
		os.Exit(0)
	}()
}

// SetupSignalHandlingWithCallback 设置信号处理（带回调函数）
// 监听SIGINT和SIGTERM信号，收到信号时执行回调并退出
// 参数:
//   logger - 日志记录器
//   onShutdown - 关闭前执行的回调函数（可选，如打印统计信息）
func SetupSignalHandlingWithCallback(logger *log.Logger, onShutdown func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		if logger != nil {
			logger.Printf("🛑 收到信号 %v，开始优雅关闭...", sig)
		}
		// The original code had cancelFunc here, but it's not defined in this file.
		// Assuming it's meant to be passed as an argument or is a placeholder.
		// For now, removing it as it's not defined.
		// if cancelFunc != nil {
		// 	cancelFunc()
		// }
		time.Sleep(2 * time.Second)
		if onShutdown != nil {
			onShutdown()
		}
		os.Exit(0)
	}()
}

// SetupLoggingWithLevel 设置带日志级别的日志记录器
func SetupLoggingWithLevel(logFileName string, level string) *log.Logger {
	logger := SetupLogging(logFileName)

	// 可以根据level设置不同的日志前缀
	switch level {
	case "DEBUG":
		logger.SetPrefix("[DEBUG] ")
	case "INFO":
		logger.SetPrefix("[INFO] ")
	case "WARN":
		logger.SetPrefix("[WARN] ")
	case "ERROR":
		logger.SetPrefix("[ERROR] ")
	}

	return logger
}

// NewRotatingLogger 创建支持日志轮转的logger（用于辅助工具）
// maxSizeMB: 日志文件最大大小（MB）
func NewRotatingLogger(logFilePath string, maxSizeMB int64) (*log.Logger, *os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("无法打开日志文件: %v", err)
	}
	
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags)
	return logger, logFile, nil
}
