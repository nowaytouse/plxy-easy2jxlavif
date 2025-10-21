// deduplicate_media - 媒体文件去重工具
//
// 功能说明：
// - 扫描目录中的重复媒体文件
// - 使用SHA256哈希值进行文件内容比较
// - 标准化文件扩展名（.jpeg -> .jpg, .tiff -> .tif）
// - 将重复文件移动到垃圾箱目录
// - 提供详细的处理日志和统计信息
//
// 作者：AI Assistant
// 版本：2.1.0
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// 程序常量定义
const (
	toolName = "deduplicate_media" // 工具名称
	version  = "2.1.0"             // 程序版本号
	author   = "AI Assistant"      // 作者信息
)

// 全局变量定义
var (
	logger *log.Logger // 全局日志记录器，同时输出到控制台和文件
)

// init 函数在main函数之前执行，用于初始化日志记录器
func init() {
	// 设置日志记录器，同时输出到控制台和文件
	logFile, err := os.OpenFile("deduplicate_media.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// main 函数是程序的入口点
func main() {
	logger.Printf("🔍 媒体文件去重工具 v%s", version)
	logger.Printf("✨ 作者: %s", author)
	logger.Printf("🔧 开始初始化...")

	// 解析命令行参数
	dir := flag.String("dir", "", "📁 要扫描重复文件的目录")
	trashDir := flag.String("trash-dir", "", "🗑️  移动重复文件到的垃圾箱目录")
	flag.Parse()

	if *dir == "" || *trashDir == "" {
		logger.Fatal("❌ 错误: 必须指定 -dir 和 -trash-dir 参数")
	}

	// 创建垃圾箱目录
	if err := os.MkdirAll(*trashDir, 0755); err != nil {
		logger.Fatalf("❌ 错误: 无法创建垃圾箱目录: %v", err)
	}

	// 在垃圾箱目录中创建说明文件
	readmePath := filepath.Join(*trashDir, "_readme_about_this_folder.txt")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readmeContent := "此文件夹包含由 deduplicate_media 脚本识别为重复的文件。您可以查看它们，如果确定不需要，可以永久删除。"
		if err := ioutil.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			logger.Printf("⚠️  无法在垃圾箱目录中创建说明文件: %v", err)
		}
	}

	// 扫描文件
	logger.Printf("📁 扫描目录: %s", *dir)
	files := findFiles(*dir)
	logger.Printf("📊 发现 %d 个文件", len(files))

	// 标准化文件扩展名
	logger.Println("🔧 标准化文件扩展名...")
	standardizeExtensions(files)

	// 重新扫描文件（标准化后）
	files = findFiles(*dir)
	logger.Printf("📊 标准化后文件数: %d", len(files))

	// 查找并移动重复文件
	logger.Println("🔍 查找重复文件...")
	findAndMoveDuplicates(files, *trashDir)

	logger.Println("🎉 去重过程完成")
}

// findFiles 扫描目录中的所有文件
// 返回文件路径列表
func findFiles(dir string) []string {
	var fileList []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		logger.Printf("❌ 扫描目录失败 %q: %v", dir, err)
	}
	return fileList
}

// standardizeExtensions 标准化文件扩展名
// 将 .jpeg 转换为 .jpg，.tiff 转换为 .tif
func standardizeExtensions(files []string) {
	logger.Println("🔧 标准化文件扩展名...")
	for _, path := range files {
		oldExt := filepath.Ext(path)
		newExt := strings.ToLower(oldExt)

		switch newExt {
		case ".jpeg":
			newExt = ".jpg"
		case ".tiff":
			newExt = ".tif"
		}

		if oldExt == newExt {
			continue
		}

		newPath := strings.TrimSuffix(path, oldExt) + newExt
		if err := os.Rename(path, newPath); err != nil {
			logger.Printf("❌ 重命名失败 %s -> %s: %v", path, newPath, err)
		} else {
			logger.Printf("✅ 重命名 %s -> %s", filepath.Base(path), filepath.Base(newPath))
		}
	}
}

// findAndMoveDuplicates 查找并移动重复文件
// 使用SHA256哈希值进行文件内容比较
func findAndMoveDuplicates(files []string, trashDir string) {
	logger.Println("🔍 查找并移动重复文件...")
	hashes := make(map[string]string)
	duplicateCount := 0

	for _, path := range files {
		if !isMediaFile(filepath.Ext(path)) {
			continue
		}

		hash, err := calculateHash(path)
		if err != nil {
			logger.Printf("❌ 计算哈希失败 %s: %v", path, err)
			continue
		}

		if originalPath, ok := hashes[hash]; ok {
			// 发现潜在重复文件，进行逐字节比较
			logger.Printf("🔍 发现潜在重复文件: %s 和 %s", originalPath, path)
			areIdentical, err := compareFiles(originalPath, path)
			if err != nil {
				logger.Printf("❌ 文件比较失败: %v", err)
				continue
			}

			if areIdentical {
				logger.Printf("✅ 文件完全相同。移动 %s 到垃圾箱", filepath.Base(path))
				moveToTrash(path, trashDir)
				duplicateCount++
			} else {
				logger.Printf("⚠️  文件哈希相同但内容不同。保留两个文件")
			}
		} else {
			hashes[hash] = path
		}
	}

	logger.Printf("📊 去重完成。移动了 %d 个重复文件", duplicateCount)
}

// isMediaFile 检查文件扩展名是否为支持的媒体格式
func isMediaFile(ext string) bool {
	switch strings.ToLower(ext) {
	// 图像格式
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tif", ".tiff", ".webp", ".heic", ".heif", ".avif", ".jxl":
		return true
	// 视频格式
	case ".mp4", ".mov", ".mkv", ".avi", ".webm", ".flv", ".wmv", ".m4v", ".3gp":
		return true
	default:
		return false
	}
}

// calculateHash 计算文件的SHA256哈希值
func calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// compareFiles 逐字节比较两个文件是否完全相同
// 返回true如果文件完全相同，false如果不同
func compareFiles(path1, path2 string) (bool, error) {
	file1, err := ioutil.ReadFile(path1)
	if err != nil {
		return false, err
	}
	file2, err := ioutil.ReadFile(path2)
	if err != nil {
		return false, err
	}

	// 首先比较文件大小
	if len(file1) != len(file2) {
		return false, nil
	}

	// 逐字节比较
	for i := range file1 {
		if file1[i] != file2[i] {
			return false, nil
		}
	}

	return true, nil
}

// moveToTrash 将文件移动到垃圾箱目录
func moveToTrash(path, trashDir string) {
	destPath := filepath.Join(trashDir, filepath.Base(path))
	if err := os.Rename(path, destPath); err != nil {
		logger.Printf("❌ 移动文件失败 %s -> %s: %v", path, destPath, err)
	} else {
		logger.Printf("✅ 已移动 %s 到垃圾箱", filepath.Base(path))
	}
}
