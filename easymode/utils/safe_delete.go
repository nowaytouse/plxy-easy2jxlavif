package utils

import (
	"fmt"
	"os"
)

// SafeDelete 安全删除原始文件，仅在确认目标文件存在且有效的前提下才删除原始文件
//
// 参数:
//   - originalPath: 原始文件路径
//   - targetPath: 目标文件路径
//   - logger: 日志记录函数
//
// 返回值:
//   - error: 如果删除失败返回错误，否则返回nil
func SafeDelete(originalPath, targetPath string, logger func(format string, v ...interface{})) error {
	// 验证目标文件是否存在
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("目标文件不存在: %s", targetPath)
	}

	// 验证目标文件大小是否合理（不为0）
	targetStat, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("无法获取目标文件信息: %v", err)
	}

	if targetStat.Size() == 0 {
		return fmt.Errorf("目标文件大小为0")
	}

	// 安全删除原始文件
	if err := os.Remove(originalPath); err != nil {
		return fmt.Errorf("删除原始文件失败: %v", err)
	}

	logger("🗑️  已安全删除原始文件: %s", originalPath)
	return nil
}