// utils/metadata.go - 元数据处理模块
//
// 功能说明：
// - 提供元数据复制功能
// - 支持超时控制和错误处理
// - 使用exiftool进行元数据操作
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CopyMetadataWithTimeout 使用exiftool在超时内复制元数据
// 从源文件复制所有元数据到目标文件，支持超时控制
// 参数:
//
//	ctx - 上下文，用于取消操作
//	src - 源文件路径
//	dst - 目标文件路径
//	timeoutSec - 超时时间（秒），如果<=0则使用默认值3秒
//
// 返回:
//
//	error - 复制过程中的错误（如果有）
func CopyMetadataWithTimeout(ctx context.Context, src, dst string, timeoutSec int) error {
	// 设置默认超时时间
	if timeoutSec <= 0 {
		timeoutSec = 3
	}

	// 创建带超时的上下文
	c, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// 执行exiftool命令复制元数据
	// -overwrite_original: 直接修改目标文件
	// -TagsFromFile: 从源文件复制标签到目标文件
	cmd := exec.CommandContext(c, "exiftool", "-overwrite_original", "-TagsFromFile", src, dst)

	// 执行命令并检查结果
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exiftool failed: %v, out=%s", err, string(out))
	}
	return nil
}
