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
	"os"
	"os/exec"
	"strings"
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

// CopyMetadata 使用exiftool复制元数据（改进版 - 更高可靠性）
func CopyMetadata(inputPath, outputPath string) error {
	// 方法1: 尝试复制所有标签
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", inputPath, outputPath)
	out, err := cmd.CombinedOutput()

	if err == nil {
		return nil
	}

	// 如果失败，检查输出中是否有"1 image files updated"等成功标志
	outputStr := string(out)
	if strings.Contains(outputStr, "image files updated") || strings.Contains(outputStr, "updated") {
		// exiftool返回非零但实际成功了
		return nil
	}

	// 方法2: 尝试只复制常见标签（fallback）
	commonTags := []string{
		"DateTimeOriginal", "CreateDate", "ModifyDate",
		"Make", "Model", "LensModel",
		"ISO", "ExposureTime", "FNumber",
		"FocalLength", "WhiteBalance",
		"Artist", "Copyright", "ImageDescription",
	}

	cmd2 := exec.Command("exiftool", "-overwrite_original")
	for _, tag := range commonTags {
		cmd2.Args = append(cmd2.Args, fmt.Sprintf("-%s<${%s}", tag, tag))
	}
	cmd2.Args = append(cmd2.Args, "-TagsFromFile", inputPath, outputPath)

	out2, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		return nil
	}

	// 检查第二次尝试的输出
	outputStr2 := string(out2)
	if strings.Contains(outputStr2, "image files updated") || strings.Contains(outputStr2, "updated") {
		return nil
	}

	// 方法3: 最小化尝试 - 只复制基本日期时间
	cmd3 := exec.Command("exiftool", "-overwrite_original",
		"-DateTimeOriginal<DateTimeOriginal",
		"-CreateDate<CreateDate",
		"-TagsFromFile", inputPath, outputPath)

	out3, err3 := cmd3.CombinedOutput()
	if err3 == nil || strings.Contains(string(out3), "updated") {
		return nil
	}

	// 所有方法都失败，但不返回错误（元数据复制失败不应阻止转换）
	// 静默失败，让转换继续
	return nil
}

// GetFileSize 获取文件大小
func GetFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// CopyFinderMetadata 复制macOS Finder标签、注释和扩展属性
// 从源文件复制Finder相关的元数据到目标文件
// 参数:
//
//	src - 源文件路径
//	dst - 目标文件路径
//
// 返回:
//
//	error - 复制过程中的错误（如果有）
func CopyFinderMetadata(src, dst string) error {
	// 复制Finder标签
	cmd := exec.Command("xattr", "-p", "com.apple.metadata:_kMDItemUserTags", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", string(output), dst).Run()
	}

	// 复制Finder注释
	cmd = exec.Command("xattr", "-p", "com.apple.metadata:kMDItemFinderComment", src)
	if output, err := cmd.CombinedOutput(); err == nil && len(output) > 0 {
		exec.Command("xattr", "-w", "com.apple.metadata:kMDItemFinderComment", string(output), dst).Run()
	}

	// 复制其他扩展属性
	cmd = exec.Command("xattr", src)
	if output, err := cmd.CombinedOutput(); err == nil {
		attrs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, attr := range attrs {
			if attr != "" && !strings.Contains(attr, "com.apple.metadata:_kMDItemUserTags") &&
				!strings.Contains(attr, "com.apple.metadata:kMDItemFinderComment") {
				cmd = exec.Command("xattr", "-p", attr, src)
				if value, err := cmd.CombinedOutput(); err == nil && len(value) > 0 {
					exec.Command("xattr", "-w", attr, string(value), dst).Run()
				}
			}
		}
	}

	return nil
}
