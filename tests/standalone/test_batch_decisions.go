package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"pixly/pkg/core/types"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("🧪 Pixly 批量决策功能测试程序")
	fmt.Println("================================")

	// 初始化日志
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 创建模拟的扫描结果
	scanResults := createMockScanResults()

	// 创建读取器
	reader := bufio.NewReader(os.Stdin)

	// 测试批量决策功能
	fmt.Println("🚀 开始测试批量决策功能...")
	fmt.Println()

	err := performBatchDecisions(reader, scanResults, logger)
	if err != nil {
		color.Red("❌ 批量决策测试失败: %v", err)
		return
	}

	color.Green("✅ 批量决策测试完成！")
}

// createMockScanResults 创建模拟的扫描结果
func createMockScanResults() []*types.MediaInfo {
	return []*types.MediaInfo{
		{
			Path:         "/test/good_image.jpg",
			Size:         2048000,
			Quality:      types.QualityHigh,
			Status:       types.StatusPending,
			QualityScore: 8.5,
		},
		{
			Path:         "/test/corrupted_image.jpg",
			Size:         1024,
			Quality:      types.QualityUnknown,
			Status:       types.StatusPending,
			IsCorrupted:  true,
			ErrorMessage: "文件头损坏，无法读取",
		},
		{
			Path:         "/test/low_quality.jpg",
			Size:         512000,
			Quality:      types.QualityVeryLow,
			Status:       types.StatusPending,
			QualityScore: 2.1,
		},
		{
			Path:         "/test/another_corrupted.png",
			Size:         0,
			Quality:      types.QualityUnknown,
			Status:       types.StatusPending,
			IsCorrupted:  true,
			ErrorMessage: "文件大小为0",
		},
	}
}

// performBatchDecisions 批量决策实现（从main.go复制）
func performBatchDecisions(reader *bufio.Reader, scanResults []*types.MediaInfo, logger *zap.Logger) error {
	// 统计问题文件
	var corruptedFiles []*types.MediaInfo
	var lowQualityFiles []*types.MediaInfo

	for _, info := range scanResults {
		if info.IsCorrupted {
			corruptedFiles = append(corruptedFiles, info)
		} else if info.Quality == types.QualityVeryLow {
			lowQualityFiles = append(lowQualityFiles, info)
		}
	}

	// 处理损坏文件
	if len(corruptedFiles) > 0 {
		color.Yellow("🚨 发现 %d 个损坏文件", len(corruptedFiles))

		// 显示损坏文件列表（最多显示5个）
		for i, file := range corruptedFiles {
			if i >= 5 {
				color.Yellow("   ... 还有 %d 个损坏文件", len(corruptedFiles)-5)
				break
			}
			color.Yellow("   - %s", file.Path)
		}

		// 用户决策
		for {
			color.White("\n请选择处理方式:")
			color.White("1. 🗑️  跳过所有损坏文件 (推荐)")
			color.White("2. 🔧 尝试修复并转换")
			color.White("3. 📋 查看损坏文件详细信息")
			color.White("\n⚡ 请选择 (1-3，默认10秒后选择1): ")

			// 10秒倒计时选择
			choice, err := getChoiceWithTimeout(reader, 10, "1")
			if err != nil {
				logger.Warn("用户输入超时，使用默认选择", zap.Error(err))
				choice = "1"
			}

			switch choice {
			case "1":
				color.Green("✅ 已选择：跳过损坏文件")
				// 标记损坏文件为跳过状态
				for _, file := range corruptedFiles {
					file.Status = types.StatusSkipped
				}
				color.Green("✅ 损坏文件处理完成")
				goto handleLowQuality
			case "2":
				color.Yellow("⚡ 已选择：尝试修复转换")
				// 标记损坏文件为待修复状态
				for _, file := range corruptedFiles {
					file.Status = types.StatusPending // 让后续处理尝试修复
				}
				color.Green("✅ 损坏文件已标记为尝试修复")
				goto handleLowQuality
			case "3":
				// 显示详细信息
				color.Cyan("\n📋 损坏文件详细信息:")
				for _, file := range corruptedFiles {
					color.White("   文件: %s", file.Path)
					color.White("   大小: %.2f MB", float64(file.Size)/(1024*1024))
					color.White("   错误: %s", file.ErrorMessage)
					color.White("")
				}
				continue // 重新显示选择菜单
			default:
				color.Yellow("无效选择，请重新输入")
				continue
			}
		}
	}

handleLowQuality:
	// 处理极低品质文件
	if len(lowQualityFiles) > 0 {
		color.Yellow("🚨 发现 %d 个极低品质文件", len(lowQualityFiles))

		// 显示低品质文件列表（最多显示5个）
		for i, file := range lowQualityFiles {
			if i >= 5 {
				color.Yellow("   ... 还有 %d 个低品质文件", len(lowQualityFiles)-5)
				break
			}
			color.Yellow("   - %s (质量分数: %.1f)", file.Path, file.QualityScore)
		}

		// 用户决策
		for {
			color.White("\n请选择处理方式:")
			color.White("1. ⚡ 强制转换所有低品质文件")
			color.White("2. 🗑️  跳过所有低品质文件")
			color.White("3. 🎨 使用表情包模式处理")
			color.White("4. 📋 查看低品质文件详细信息")
			color.White("\n⚡ 请选择 (1-4，默认10秒后选择1): ")

			// 10秒倒计时选择
			choice, err := getChoiceWithTimeout(reader, 10, "1")
			if err != nil {
				logger.Warn("用户输入超时，使用默认选择", zap.Error(err))
				choice = "1"
			}

			switch choice {
			case "1":
				color.Green("✅ 已选择：强制转换低品质文件")
				// 保持默认状态，正常处理
				color.Green("✅ 极低品质文件处理完成")
				return nil
			case "2":
				color.Green("✅ 已选择：跳过低品质文件")
				// 标记低品质文件为跳过状态
				for _, file := range lowQualityFiles {
					file.Status = types.StatusSkipped
				}
				color.Green("✅ 极低品质文件处理完成")
				return nil
			case "3":
				color.Green("✅ 已选择：表情包模式处理")
				// 标记为表情包模式专用处理
				for _, file := range lowQualityFiles {
					file.PreferredMode = types.ModeEmoji
				}
				color.Green("✅ 极低品质文件已标记为表情包模式")
				return nil
			case "4":
				// 显示详细信息
				color.Cyan("\n📋 低品质文件详细信息:")
				for _, file := range lowQualityFiles {
					color.White("   文件: %s", file.Path)
					color.White("   大小: %.2f MB", float64(file.Size)/(1024*1024))
					color.White("   质量分数: %.1f/10.0", file.QualityScore)
					color.White("   质量等级: %s", file.Quality.String())
					color.White("")
				}
				continue // 重新显示选择菜单
			default:
				color.Yellow("无效选择，请重新输入")
				continue
			}
		}
	}

	return nil
}

// getChoiceWithTimeout 带超时的用户输入获取
func getChoiceWithTimeout(reader *bufio.Reader, timeoutSeconds int, defaultChoice string) (string, error) {
	// 利用channel实现超时机制
	inputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// 在goroutine中读取用户输入
	go func() {
		input, err := reader.ReadString('\n')
		if err != nil {
			errorChan <- err
			return
		}
		inputChan <- strings.TrimSpace(input)
	}()

	// 带倒计时的超时等待
	for i := timeoutSeconds; i > 0; i-- {
		select {
		case input := <-inputChan:
			return input, nil
		case err := <-errorChan:
			return "", err
		case <-time.After(1 * time.Second):
			if i > 1 {
				// 显示倒计时
				color.Yellow("\r⚡ 请选择 (1-3，%d秒后选择%s): ", i-1, defaultChoice)
			}
		}
	}

	// 超时后返回默认选择
	color.Yellow("\n⏰ 超时，使用默认选择: %s\n", defaultChoice)
	return defaultChoice, nil
}
