// utils/filesystem_metadata.go - 文件系统元数据保留模块
// 保留macOS Finder可见的元数据：创建时间、修改时间、扩展属性等

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// FileSystemMetadata 文件系统元数据
type FileSystemMetadata struct {
	CreationTime     time.Time         // 创建时间
	ModificationTime time.Time         // 修改时间
	AccessTime       time.Time         // 访问时间
	ExtendedAttrs    map[string][]byte // macOS扩展属性（xattr）
}

// CaptureFileSystemMetadata 捕获文件系统元数据
func CaptureFileSystemMetadata(filePath string) (*FileSystemMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	metadata := &FileSystemMetadata{
		ModificationTime: info.ModTime(),
		ExtendedAttrs:    make(map[string][]byte),
	}

	// 获取创建时间（macOS特有）
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		// macOS上Birthtimespec是创建时间
		metadata.CreationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
		metadata.AccessTime = time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
	}

	// 捕获macOS扩展属性（xattr）
	attrs, err := listExtendedAttributes(filePath)
	if err == nil {
		for _, attr := range attrs {
			value, err := getExtendedAttribute(filePath, attr)
			if err == nil {
				metadata.ExtendedAttrs[attr] = value
			}
		}
	}

	return metadata, nil
}

// ApplyFileSystemMetadata 应用文件系统元数据到目标文件
func ApplyFileSystemMetadata(targetPath string, metadata *FileSystemMetadata) error {
	if metadata == nil {
		return fmt.Errorf("元数据为空")
	}

	// 1. 恢复文件修改时间和访问时间
	// 注意：os.Chtimes只能设置访问时间和修改时间，不能设置创建时间
	if err := os.Chtimes(targetPath, metadata.AccessTime, metadata.ModificationTime); err != nil {
		return fmt.Errorf("设置文件时间失败: %v", err)
	}

	// 2. 恢复macOS扩展属性（xattr）
	for attrName, attrValue := range metadata.ExtendedAttrs {
		if err := setExtendedAttribute(targetPath, attrName, attrValue); err != nil {
			// 扩展属性设置失败不应阻止整个流程，仅记录警告
			fmt.Printf("⚠️  设置扩展属性失败 %s: %v\n", attrName, err)
		}
	}

	// 3. 尝试恢复创建时间（macOS特有，需要特殊处理）
	if err := setCreationTime(targetPath, metadata.CreationTime); err != nil {
		// 创建时间设置失败不应阻止整个流程
		fmt.Printf("⚠️  设置创建时间失败: %v\n", err)
	}

	return nil
}

// listExtendedAttributes 列出文件的所有扩展属性
func listExtendedAttributes(filePath string) ([]string, error) {
	cmd := exec.Command("xattr", "-l", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// 解析xattr -l的输出
	var attrs []string
	lines := string(output)
	if lines == "" {
		return attrs, nil
	}

	// xattr -l输出格式：
	// com.apple.metadata:kMDItemFinderComment:
	//     00000000  46 69 6E 64 65 72 E6 B5  8B E8 AF 95 E6 B3 A8 E9  |Finder.......|
	// 我们需要提取属性名

	// 使用xattr不带参数列出所有属性名更简单
	cmd = exec.Command("xattr", filePath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines = string(output)
	for _, line := range splitLines(lines) {
		if line != "" {
			attrs = append(attrs, line)
		}
	}

	return attrs, nil
}

// getExtendedAttribute 获取指定扩展属性的值
func getExtendedAttribute(filePath, attrName string) ([]byte, error) {
	cmd := exec.Command("xattr", "-p", attrName, filePath)
	return cmd.CombinedOutput()
}

// setExtendedAttribute 设置扩展属性
func setExtendedAttribute(filePath, attrName string, attrValue []byte) error {
	cmd := exec.Command("xattr", "-w", attrName, string(attrValue), filePath)
	return cmd.Run()
}

// setCreationTime 设置文件创建时间（macOS）
func setCreationTime(filePath string, creationTime time.Time) error {
	// macOS上设置创建时间需要使用SetFile命令（来自Xcode Command Line Tools）
	// 或者使用touch -t

	// 方法1: 使用SetFile（如果可用）
	if _, err := exec.LookPath("SetFile"); err == nil {
		// SetFile -d "MM/DD/YYYY HH:MM:SS" file
		timeStr := creationTime.Format("01/02/2006 15:04:05")
		cmd := exec.Command("SetFile", "-d", timeStr, filePath)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// 方法2: 使用touch（不够精确，但广泛可用）
	// touch -t [[CC]YY]MMDDhhmm[.ss] file
	timeStr := creationTime.Format("200601021504.05")
	cmd := exec.Command("touch", "-t", timeStr, filePath)
	return cmd.Run()
}

// CopyAllMetadata 复制所有元数据（文件内部+文件系统）
func CopyAllMetadata(src, dst string) error {
	// 1. 捕获源文件的文件系统元数据
	fsMetadata, err := CaptureFileSystemMetadata(src)
	if err != nil {
		return fmt.Errorf("捕获文件系统元数据失败: %v", err)
	}

	// 2. 复制文件内部元数据（EXIF/XMP）
	cmd := exec.Command("exiftool", "-overwrite_original", "-TagsFromFile", src, "-all:all", dst)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exiftool复制失败: %v\n输出: %s", err, string(output))
	}

	// 3. 应用文件系统元数据
	if err := ApplyFileSystemMetadata(dst, fsMetadata); err != nil {
		return fmt.Errorf("应用文件系统元数据失败: %v", err)
	}

	return nil
}

// PreserveTimestampsOnly 仅保留文件时间戳（快速版本）
func PreserveTimestampsOnly(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 获取创建时间（macOS）
	var creationTime time.Time
	if stat, ok := srcInfo.Sys().(*syscall.Stat_t); ok {
		creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}

	// 设置修改时间和访问时间
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return err
	}

	// 设置创建时间
	if !creationTime.IsZero() {
		setCreationTime(dst, creationTime)
	}

	return nil
}

// splitLines 分割行
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
