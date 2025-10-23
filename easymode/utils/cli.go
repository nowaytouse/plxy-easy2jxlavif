// utils/cli.go - 命令行接口模块
//
// 功能说明：
// - 提供统一的命令行参数处理
// - 定义标准退出码
// - 处理参数兼容性和标准化
//
// 作者: AI Assistant
// 版本: v2.2.0
// 更新: 2025-10-24

package utils

import (
	"fmt"
)

// 统一退出码定义
// 为所有工具提供一致的退出状态码
const (
	ExitOK            = 0 // 成功 - 程序正常完成
	ExitInvalidArgs   = 1 // 参数/配置错误 - 命令行参数无效
	ExitRuntimeFailed = 2 // 运行时失败 - 程序执行过程中发生错误
	ExitPartialFailed = 3 // 部分文件失败 - 部分文件处理失败但程序继续执行
)

// NormalizeInputFlags 统一处理输入目录参数
// 处理 -input 与 -dir 参数的兼容性，确保向后兼容
// 参数:
//
//	inputFlag - 新的 -input 参数值
//	dirFlag - 旧的 -dir 参数值
//
// 返回:
//
//	string - 最终使用的输入目录路径
//	bool - 是否使用了过时的 -dir 标志
func NormalizeInputFlags(inputFlag string, dirFlag string) (string, bool) {
	// 优先使用新的 -input 参数
	if inputFlag != "" {
		return inputFlag, false
	}

	// 兼容旧的 -dir 参数，但给出提示
	if dirFlag != "" {
		// 兼容提示：提醒用户使用新参数
		fmt.Println("⚠️  提示：-dir 已弃用，请改用 -input。当前版本仍兼容 -dir。")
		return dirFlag, true
	}

	// 两个参数都为空
	return "", false
}
