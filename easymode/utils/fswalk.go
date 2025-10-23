// utils/fswalk.go - 文件系统遍历模块
//
// 功能说明：
// - 提供高效的文件系统遍历功能
// - 支持扩展名过滤和目录忽略
// - 使用godirwalk库提升遍历性能
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"
)

// WalkMedia 扫描目录，返回满足扩展名过滤条件的文件列表
// 高效遍历文件系统，支持扩展名过滤和目录忽略
// 参数:
//
//	root - 根目录路径
//	exts - 扩展名过滤映射，空映射表示不过滤
//	ignoreDir - 要忽略的目录路径
//
// 返回:
//
//	[]string - 符合条件的文件路径列表
//	error - 遍历过程中的错误（如果有）
func WalkMedia(root string, exts map[string]bool, ignoreDir string) ([]string, error) {
	// 预分配文件列表容量，提升性能
	files := make([]string, 0, 1024)

	// 使用godirwalk进行高效遍历
	err := godirwalk.Walk(root, &godirwalk.Options{
		Unsorted: true, // 不排序，提升遍历速度
		Callback: func(p string, de *godirwalk.Dirent) error {
			// 处理目录
			if de.IsDir() {
				// 跳过指定的忽略目录
				if ignoreDir != "" && filepath.Clean(p) == filepath.Clean(ignoreDir) {
					return filepath.SkipDir
				}
				return nil
			}

			// 处理文件：检查扩展名过滤条件
			ext := strings.ToLower(filepath.Ext(p))
			if len(exts) == 0 || exts[ext] {
				files = append(files, p)
			}
			return nil
		},
		// 错误处理：跳过有问题的节点，继续遍历
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			return godirwalk.SkipNode
		},
	})
	return files, err
}

// EnsureDir 确保目录存在
// 创建目录及其所有父目录，如果目录已存在则不报错
// 参数:
//
//	dir - 目录路径
//
// 返回:
//
//	error - 创建过程中的错误（如果有）
func EnsureDir(dir string) error {
	// 空路径直接返回
	if dir == "" {
		return nil
	}
	// 创建目录，权限为0755（rwxr-xr-x）
	return os.MkdirAll(dir, 0755)
}
