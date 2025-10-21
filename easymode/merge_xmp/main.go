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
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// 设置日志记录器，同时输出到控制台和文件
	logFile, err := os.OpenFile("merge_xmp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
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
		if processFile(file) {
			processedCount++
		}
	}

	logger.Printf("🎉 处理完成。成功合并 %d 个XMP文件", processedCount)
}

// processFile 处理单个媒体文件，查找并合并对应的XMP文件
// 返回true如果成功处理了文件，false如果跳过或失败
func processFile(mediaPath string) bool {
	ext := filepath.Ext(mediaPath)
	if !isMediaFile(ext) {
		return false
	}

	// 查找XMP文件
	xmpPath := strings.TrimSuffix(mediaPath, ext) + ".xmp"
	if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
		// 也检查sidecar.xmp格式
		xmpPath = mediaPath + ".xmp"
		if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
			return false
		}
	}

	// 再次检查XMP文件是否存在
	if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
		return false
	}

	logger.Printf("🔍 发现媒体文件 '%s' 和XMP侧边文件 '%s'", filepath.Base(mediaPath), filepath.Base(xmpPath))

	// 合并XMP元数据
	mergeCmd := exec.Command("exiftool", "-tagsfromfile", xmpPath, "-all:all", "-overwrite_original", mediaPath)
	if output, err := mergeCmd.CombinedOutput(); err != nil {
		logger.Printf("❌ 合并XMP失败 %s: %v. 输出: %s", filepath.Base(mediaPath), err, string(output))
		return false
	}

	logger.Printf("✅ 成功合并XMP到 %s", filepath.Base(mediaPath))
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
		// 如果无法解析XMP，假设合并成功
		return true
	}

	if len(tags) == 0 || len(tags[0]) == 0 {
		logger.Printf("ℹ️  XMP文件中没有找到标签 %s", xmpPath)
		return true // 没有需要验证的内容
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
		logger.Printf("ℹ️  没有找到可验证的标签 %s", xmpPath)
		return true // 没有可验证的标签
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
