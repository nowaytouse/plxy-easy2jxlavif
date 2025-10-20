package batchdecision

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// displayCountdown 显示倒计时界面 - README要求的5秒倒计时
func (bdm *BatchDecisionManager) displayCountdown(decisionType DecisionType, defaultChoice UserDecisionChoice) {
	fmt.Printf("\n\033[43m\033[30m 批量决策必需 \033[0m\n")

	switch decisionType {
	case DecisionTypeCorruptedFiles:
		fmt.Printf("\n\033[31m检测到 %d 个损坏文件，请选择批量处理方式：\033[0m\n", len(bdm.pendingCorrupted))
		fmt.Println("  [1] 尝试修复")
		fmt.Println("  [2] 全部删除")
		fmt.Println("  [3] 终止任务")
		fmt.Printf("  [4] \033[33m忽略（默认）\033[0m\n")

	case DecisionTypeLowQualityFiles:
		fmt.Printf("\n\033[33m检测到 %d 个极低品质文件，请选择批量处理方式：\033[0m\n", len(bdm.pendingLowQuality))
		fmt.Printf("  [1] \033[33m跳过忽略（默认）\033[0m\n")
		fmt.Println("  [2] 全部删除")
		fmt.Println("  [3] 强制转换")
		fmt.Println("  [4] 使用表情包模式处理")
	}

	fmt.Printf("\n请输入选项 (1-4) 或直接回车使用默认选项：")
	fmt.Printf("\033[36m将在 %d 秒后自动选择默认选项...\033[0m\n", int(bdm.countdownDuration.Seconds()))

	// 实时倒计时显示
	go func() {
		for i := int(bdm.countdownDuration.Seconds()); i > 0; i-- {
			time.Sleep(1 * time.Second)
			if i > 1 {
				fmt.Printf("\r\033[36m剩余 %d 秒...\033[0m", i-1)
			}
		}
		fmt.Printf("\r\033[33m⏰ 超时，使用默认选择\033[0m\n")
	}()
}

// listenForUserInput 监听用户输入
func (bdm *BatchDecisionManager) listenForUserInput(inputChannel chan<- string) {
	defer close(inputChannel)

	// 使用bufio.Scanner从标准输入读取
	scanner := bufio.NewScanner(os.Stdin)

	// 设置读取超时
	done := make(chan bool, 1)
	go func() {
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			select {
			case inputChannel <- input:
				// 成功发送输入
			case <-time.After(100 * time.Millisecond):
				// 超时，放弃发送
			}
		}
		done <- true
	}()

	// 等待输入或超时
	select {
	case <-done:
		// 输入完成
	case <-time.After(6 * time.Second): // 略长于倒计时时间
		// 超时，停止监听
	}
}

// parseUserInput 解析用户输入
func (bdm *BatchDecisionManager) parseUserInput(input string, decisionType DecisionType) UserDecisionChoice {
	input = strings.ToLower(strings.TrimSpace(input))

	switch decisionType {
	case DecisionTypeCorruptedFiles:
		return bdm.parseCorruptedFilesInput(input)
	case DecisionTypeLowQualityFiles:
		return bdm.parseLowQualityFilesInput(input)
	default:
		return 0 // 无效选择
	}
}

// parseCorruptedFilesInput 解析损坏文件处理输入
func (bdm *BatchDecisionManager) parseCorruptedFilesInput(input string) UserDecisionChoice {
	switch input {
	case "1", "repair", "r":
		return CorruptedChoiceRepair
	case "2", "delete", "d":
		return CorruptedChoiceDeleteAll
	case "3", "terminate", "t":
		return CorruptedChoiceTerminate
	case "4", "ignore", "i", "":
		return CorruptedChoiceIgnore
	default:
		bdm.logger.Warn("无效的用户输入", zap.String("input", input))
		return 0 // 无效选择
	}
}

// parseLowQualityFilesInput 解析低品质文件处理输入
func (bdm *BatchDecisionManager) parseLowQualityFilesInput(input string) UserDecisionChoice {
	switch input {
	case "1", "skip", "s", "":
		return LowQualityChoiceSkip
	case "2", "delete", "d":
		return LowQualityChoiceDelete
	case "3", "force", "f":
		return LowQualityChoiceForce
	case "4", "emoji", "e":
		return LowQualityChoiceEmoji
	default:
		bdm.logger.Warn("无效的用户输入", zap.String("input", input))
		return 0 // 无效选择
	}
}

// getDefaultChoice 获取默认选择 - README要求的默认选择
func (bdm *BatchDecisionManager) getDefaultChoice(decisionType DecisionType) UserDecisionChoice {
	switch decisionType {
	case DecisionTypeCorruptedFiles:
		return CorruptedChoiceIgnore // README要求：默认选择"忽略"
	case DecisionTypeLowQualityFiles:
		// 测试模式下强制转换，正常模式下跳过
		if bdm.testMode {
			return LowQualityChoiceForce // 测试模式：强制转换所有低品质文件
		}
		return LowQualityChoiceSkip // README要求：默认选择"跳过忽略"
	default:
		return CorruptedChoiceIgnore
	}
}

// generateDecisionID 生成决策ID
func (bdm *BatchDecisionManager) generateDecisionID() string {
	return fmt.Sprintf("decision_%d", time.Now().UnixNano())
}

// combineResults 合并多个决策结果
func (bdm *BatchDecisionManager) combineResults(results []*BatchDecisionResult) *BatchDecisionResult {
	if len(results) == 0 {
		return &BatchDecisionResult{}
	}
	if len(results) == 1 {
		return results[0]
	}

	// 合并所有结果
	combinedResult := &BatchDecisionResult{
		ProcessedFiles: make([]ProcessedFileResult, 0),
		Summary:        &DecisionSummary{},
		ExecutionDetails: &ExecutionDetails{
			StartTime:      results[0].ExecutionDetails.StartTime,
			DecisionMethod: "batch_combined",
		},
	}

	totalFiles := 0
	successfulFiles := 0
	failedFiles := 0

	for _, result := range results {
		combinedResult.ProcessedFiles = append(combinedResult.ProcessedFiles, result.ProcessedFiles...)
		totalFiles += result.Summary.TotalFiles
		successfulFiles += result.Summary.SuccessfulFiles
		failedFiles += result.Summary.FailedFiles
	}

	combinedResult.Summary.TotalFiles = totalFiles
	combinedResult.Summary.SuccessfulFiles = successfulFiles
	combinedResult.Summary.FailedFiles = failedFiles

	if totalFiles > 0 {
		combinedResult.Summary.SuccessRate = float64(successfulFiles) / float64(totalFiles)
	}

	combinedResult.ExecutionDetails.EndTime = time.Now()
	combinedResult.ExecutionDetails.TotalDuration = combinedResult.ExecutionDetails.EndTime.Sub(combinedResult.ExecutionDetails.StartTime)

	return combinedResult
}

// updateStats 更新统计信息
func (bdm *BatchDecisionManager) updateStats(result *BatchDecisionResult) {
	bdm.stats.TotalDecisions++
	bdm.stats.TotalProcessingTime += result.ExecutionDetails.TotalDuration

	// 更新决策选择统计
	if result.DecisionRecord != nil {
		bdm.stats.DecisionChoiceStats[result.DecisionRecord.UserChoice]++

		// 更新默认选择使用统计
		if result.DecisionRecord.IsDefaultChoice {
			bdm.stats.DefaultChoiceUsage++
		} else {
			bdm.stats.InteractiveUsage++
		}
	}

	// 计算平均决策时间
	if bdm.stats.TotalDecisions > 0 {
		bdm.stats.AverageDecisionTime = bdm.stats.TotalProcessingTime / time.Duration(bdm.stats.TotalDecisions)
	}
}

// clearProcessedFiles 清理已处理的文件
func (bdm *BatchDecisionManager) clearProcessedFiles() {
	bdm.pendingCorrupted = make([]*CorruptedFile, 0)
	bdm.pendingLowQuality = make([]*LowQualityFile, 0)

	bdm.logger.Debug("已清理处理队列",
		zap.Int("cleared_corrupted", len(bdm.pendingCorrupted)),
		zap.Int("cleared_low_quality", len(bdm.pendingLowQuality)))
}
