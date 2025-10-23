// merge_xmp - XMP元数据合并工具
//
// 功能说明：
// - 将XMP侧边文件合并到对应的媒体文件中
// - 支持多种媒体格式（图像、视频等）
// - 自动检测XMP文件（.xmp和sidecar.xmp格式）
// - 使用exiftool进行元数据合并
// - 提供详细的处理日志和错误报告
//
// 作者：AI Assistant
// 版本：2.1.0
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pixly/utils"
)

// 程序常量定义
const (
	toolName = "merge_xmp"    // 工具名称
	version  = "2.1.0"        // 程序版本号
	author   = "AI Assistant" // 作者信息
)

// 全局变量定义
var (
	logger *log.Logger // 全局日志记录器，同时输出到控制台和文件
)

// init 函数在main函数之前执行，用于初始化日志记录器
func init() {
	// 设置日志记录器，带大小轮转，同时输出到控制台和文件
	rl, lf, err := utils.NewRotatingLogger("merge_xmp.log", 50*1024*1024)
	if err != nil {
		log.Fatalf("无法初始化轮转日志: %v", err)
	}
	logger = rl
	_ = lf
}

// main 函数是程序的入口点
func main() {
	logger.Printf("🔗 XMP元数据合并工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 解析命令行参数
	dir := flag.String("dir", "", "📁 要处理的目录")
	flag.Parse()

	if *dir == "" {
		logger.Fatal("❌ 错误: 必须指定目录路径。使用方法: merge_xmp -dir <路径>")
	}

	// 检查exiftool依赖
	logger.Println("🔍 检查系统依赖...")
	if _, err := exec.LookPath("exiftool"); err != nil {
		logger.Fatalf("❌ 错误: 依赖工具 'exiftool' 未找到。请安装后继续运行。")
	}
	logger.Printf("✅ exiftool 已就绪")

	// 扫描目录中的文件
	logger.Printf("📁 扫描目录: %s", *dir)
	var files []string
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		logger.Fatalf("❌ 错误: 扫描目录失败 %q: %v", *dir, err)
	}

	logger.Printf("📊 发现 %d 个文件", len(files))

	// 处理每个文件
	processedCount := 0
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file), ".xmp") {
			if processXMPFile(file) {
				processedCount++
			}
		}
	}

	logger.Printf("✅ 合并完成，总计处理 %d 个XMP文件", processedCount)
}

// processXMPFile 处理单个XMP文件并将其合并到对应的媒体文件
func processXMPFile(xmpPath string) bool {
	// 验证文件路径安全性
	if !isValidFilePath(xmpPath) {
		logger.Printf("⚠️  跳过不安全的文件路径: %s", xmpPath)
		return false
	}

	// 查找媒体文件路径
	mediaPath := strings.TrimSuffix(xmpPath, ".xmp")
	if _, err := os.Stat(mediaPath); os.IsNotExist(err) {
		logger.Printf("⚠️  媒体文件不存在: %s", mediaPath)
		return false
	}

	// 验证媒体文件扩展名
	if !isMediaFile(filepath.Ext(mediaPath)) {
		logger.Printf("⚠️  媒体文件扩展名无效: %s", filepath.Base(mediaPath))
		return false
	}

	// 验证XMP文件内容
	if !isValidXMPFile(xmpPath) {
		logger.Printf("⚠️  XMP文件格式无效: %s", filepath.Base(xmpPath))
		return false
	}

	logger.Printf("🔍 发现媒体文件 '%s' 和XMP侧边文件 '%s'", filepath.Base(mediaPath), filepath.Base(xmpPath))

	// 合并XMP元数据
	mergeCmd := exec.Command("exiftool", "-tagsfromfile", xmpPath, "-all:all", "-overwrite_original", mediaPath)
	if output, err := mergeCmd.CombinedOutput(); err != nil {
		logger.Printf("❌ 合并XMP失败 %s: %v. 输出: %s", filepath.Base(mediaPath), err, string(output))
		return false
	}

	// 验证合并结果
	if !verifyMerge(mediaPath, xmpPath) {
		logger.Printf("⚠️  XMP合并验证失败: %s", filepath.Base(mediaPath))
		return false
	}

	logger.Printf("✅ 成功合并XMP到 %s", filepath.Base(mediaPath))
	return true
}

// isValidFilePath 验证文件路径是否安全
func isValidFilePath(filePath string) bool {
	// 检查路径是否包含非法字符
	if strings.ContainsAny(filePath, "\x00") {
		return false
	}

	// 检查路径是否包含路径遍历攻击
	if strings.Contains(filePath, "..") {
		return false
	}

	// 检查路径长度
	if len(filePath) > 4096 {
		return false
	}

	return true
}

// isValidXMPFile 验证XMP文件格式是否有效
func isValidXMPFile(xmpPath string) bool {
	file, err := os.Open(xmpPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取文件头检查XMP格式
	header := make([]byte, 100)
	n, err := file.Read(header)
	if err != nil || n < 10 {
		return false
	}

	// 检查是否包含XMP标识
	content := string(header)
	if !strings.Contains(content, "xmpmeta") && !strings.Contains(content, "XMP") {
		return false
	}

	// 检查文件大小是否合理（XMP文件通常不会太大）
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	// XMP文件大小应该在1KB到10MB之间
	if stat.Size() < 1024 || stat.Size() > 10*1024*1024 {
		return false
	}

	return true
}

// isMediaFile 检查文件扩展名是否为支持的媒体格式
func isMediaFile(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".tif", ".tiff", ".gif", ".mp4", ".mov", ".heic", ".heif", ".webp", ".avif", ".jxl":
		return true
	default:
		return false
	}
}

// verifyMerge 验证XMP合并是否成功
// 通过比较XMP文件中的关键标签与媒体文件中的标签来验证
func verifyMerge(mediaPath, xmpPath string) bool {
	// 获取XMP文件中的所有标签
	xmpTagsCmd := exec.Command("exiftool", "-j", xmpPath)
	xmpTagsOutput, err := xmpTagsCmd.CombinedOutput()
	if err != nil {
		logger.Printf("❌ 获取XMP文件标签失败 %s: %v", xmpPath, err)
		return false
	}

	var tags []map[string]interface{}
	if err := json.Unmarshal(xmpTagsOutput, &tags); err != nil {
		logger.Printf("❌ 解析XMP标签失败 %s: %v", xmpPath, err)
		// 严格模式下，不允许无法解析即通过，避免被绕过
		return false
	}

	if len(tags) == 0 || len(tags[0]) == 0 {
		logger.Printf("⚠️  XMP文件中没有找到任何可用标签 %s", xmpPath)
		// 没有可验证内容则视为验证失败，避免被空XMP绕过
		return false
	}

	// 找到一个有意义的标签进行验证，避免文件系统相关的标签
	var tagToVerify string
	for tag := range tags[0] {
		if !strings.HasPrefix(tag, "File:") && tag != "SourceFile" && tag != "ExifTool:ExifToolVersion" {
			tagToVerify = tag
			break
		}
	}

	if tagToVerify == "" {
		logger.Printf("⚠️  没有找到可验证的标签 %s", xmpPath)
		// 缺少可验证标签同样不通过，防止绕过
		return false
	}

	// 检查媒体文件中是否存在该标签
	mediaTagCmd := exec.Command("exiftool", "-"+tagToVerify, mediaPath)
	mediaTagOutput, err := mediaTagCmd.CombinedOutput()
	if err != nil {
		logger.Printf("❌ 获取媒体文件标签失败 %s: %v", mediaPath, err)
		return false
	}

	if len(strings.TrimSpace(string(mediaTagOutput))) == 0 {
		logger.Printf("❌ 标签 %s 在媒体文件中未找到 %s", tagToVerify, mediaPath)
		return false
	}

	logger.Printf("✅ 验证成功: 标签 '%s' 已正确合并", tagToVerify)
	return true
}
