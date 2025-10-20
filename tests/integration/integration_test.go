package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixly/pkg/batchdecision"
	"pixly/pkg/core/types"
)

// TestFullIntegration 完整集成测试 - 覆盖所有处理模式
func TestFullIntegration(t *testing.T) {
	// 创建集成测试套件
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// 运行完整集成测试
	suite.RunFullIntegrationTest()

	// 验证测试结果
	results := suite.GetTestResults()
	if len(results) == 0 {
		t.Fatal("没有获得任何测试结果")
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			t.Errorf("模式 %s 测试失败", result.Mode.String())
		}
	}

	t.Logf("集成测试完成: %d/%d 个模式测试成功", successCount, len(results))

	// 要求至少一个模式测试成功
	if successCount == 0 {
		t.Fatal("所有处理模式测试都失败了")
	}
}

// TestAutoModeIntegration 自动模式+集成测试
func TestAutoModeIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	result := suite.testProcessingMode(types.ModeAutoPlus)
	if !result.Success {
		t.Errorf("自动模式+测试失败: %v", result.ErrorMessages)
	}

	t.Logf("自动模式+测试: 处理 %d 文件, 成功 %d, 失败 %d",
		result.FilesProcessed, result.FilesSuccess, result.FilesFailed)
}

// TestQualityModeIntegration 品质模式集成测试
func TestQualityModeIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	result := suite.testProcessingMode(types.ModeQuality)
	if !result.Success {
		t.Errorf("品质模式测试失败: %v", result.ErrorMessages)
	}

	t.Logf("品质模式测试: 处理 %d 文件, 成功 %d, 失败 %d",
		result.FilesProcessed, result.FilesSuccess, result.FilesFailed)
}

// TestEmojiModeIntegration 表情包模式集成测试
func TestEmojiModeIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	result := suite.testProcessingMode(types.ModeEmoji)
	if !result.Success {
		t.Errorf("表情包模式测试失败: %v", result.ErrorMessages)
	}

	t.Logf("表情包模式测试: 处理 %d 文件, 成功 %d, 失败 %d",
		result.FilesProcessed, result.FilesSuccess, result.FilesFailed)
}

// TestConcurrentProcessingIntegration 并发处理集成测试
func TestConcurrentProcessingIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	suite.TestConcurrentProcessing(t)
}

// TestErrorRecoveryIntegration 错误恢复集成测试
func TestErrorRecoveryIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	suite.TestErrorRecovery(t)
}

// TestSecurityIntegration 安全检查集成测试
func TestSecurityIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// 测试安全检查功能
	securityResult, err := suite.securityChecker.PerformSecurityCheck(suite.testDataDir)
	if err != nil {
		t.Errorf("安全检查失败: %v", err)
	}

	if !securityResult.Passed {
		t.Errorf("安全检查未通过: %d 个问题", len(securityResult.Issues))
		for _, issue := range securityResult.Issues {
			t.Logf("安全问题: %s - %s", issue.Type.String(), issue.Message)
		}
	}

	t.Logf("安全检查完成: 通过=%v", securityResult.Passed)
}

// TestScannerIntegration 扫描器集成测试
func TestScannerIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()
	files, err := suite.scanner.ScanDirectory(ctx, suite.testDataDir)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	if len(files) == 0 {
		t.Error("扫描没有发现任何文件")
	}

	// 验证扫描结果
	imageFiles := 0
	videoFiles := 0
	otherFiles := 0

	for _, file := range files {
		if file.IsDir {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			imageFiles++
		case ".mp4", ".mov":
			videoFiles++
		default:
			otherFiles++
		}
	}

	t.Logf("扫描结果: 总文件 %d, 图片 %d, 视频 %d, 其他 %d",
		len(files), imageFiles, videoFiles, otherFiles)
}

// TestBatchDecisionIntegration 批量决策集成测试
func TestBatchDecisionIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	// 添加一些测试文件到批量决策管理器
	suite.batchManager.AddCorruptedFile(
		filepath.Join(suite.testDataDir, "corrupted.jpg"),
		batchdecision.CorruptionFileHeader,
		"测试损坏文件",
		false)

	// 创建低品质文件结构体
	lowQualityFile := &batchdecision.LowQualityFile{
		FilePath:        filepath.Join(suite.testDataDir, "low_quality.jpg"),
		QualityScore:    15.5,
		QualityIssues:   []string{"低分辨率", "过度压缩"},
		FileSize:        1024,
		DetectedAt:      time.Now(),
		CanConvert:      true,
		RecommendedMode: batchdecision.ProcessingModeAuto,
	}
	suite.batchManager.AddLowQualityFile(lowQualityFile)

	// 处理批量决策
	result, err := suite.batchManager.ProcessBatchDecisions(ctx)
	if err != nil {
		t.Errorf("批量决策处理失败: %v", err)
	}

	if result != nil {
		t.Logf("批量决策完成: 总文件 %d, 成功 %d",
			result.Summary.TotalFiles, result.Summary.SuccessfulFiles)
	}
}

// TestToolCheckerIntegration 工具检查器集成测试
func TestToolCheckerIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	results, err := suite.toolChecker.CheckAll()
	if err != nil {
		t.Logf("工具检查警告: %v", err)
		// 工具检查失败不算测试失败，因为可能没有安装相关工具
	}

	t.Logf("工具检查结果: FFmpeg=%v, CJXL=%v, AVIF=%v, ExifTool=%v",
		results.HasFfmpeg, results.HasCjxl, results.HasAvifenc, results.HasExiftool)

	// 至少应该有一个工具可用
	hasAnyTool := results.HasFfmpeg || results.HasCjxl || results.HasAvifenc || results.HasExiftool
	if !hasAnyTool {
		t.Log("警告: 没有检测到任何可用的处理工具，某些功能可能无法正常工作")
	}
}

// TestProgressUIIntegration 进度UI集成测试
func TestProgressUIIntegration(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// 测试进度UI的各个阶段
	totalFiles := int64(10)

	// 开始扫描阶段
	suite.progressUI.StartScanningPhase(totalFiles)

	// 模拟扫描进度
	for i := int64(1); i <= totalFiles; i++ {
		suite.progressUI.UpdateScanProgress(i)
		time.Sleep(10 * time.Millisecond) // 模拟扫描耗时
	}

	// 开始分析阶段
	suite.progressUI.StartAnalysisPhase(totalFiles)

	// 模拟分析进度
	qualityStats := make(map[types.QualityLevel]int64)
	for i := int64(1); i <= totalFiles; i++ {
		qualityStats[types.QualityHigh] = i
		suite.progressUI.UpdateAnalysisProgress(i, qualityStats)
		time.Sleep(5 * time.Millisecond)
	}

	// 开始处理阶段
	suite.progressUI.StartProcessingPhase(totalFiles)

	// 模拟处理进度
	for i := int64(1); i <= totalFiles; i++ {
		suite.progressUI.UpdateProcessingProgress(i, i, 0, 0, 1.5)
		time.Sleep(20 * time.Millisecond) // 模拟处理耗时
	}

	// 完成处理
	suite.progressUI.CompleteProcessing()

	// 获取统计信息
	stats := suite.progressUI.GetStats()
	if stats.TotalFiles != totalFiles {
		t.Errorf("预期总文件数 %d, 实际 %d", totalFiles, stats.TotalFiles)
	}

	t.Logf("进度UI测试完成: 总文件 %d, 处理时间 %v",
		stats.TotalFiles, stats.ElapsedTime)

	// 生成统计报告
	report := suite.progressUI.GenerateStatisticsReport()
	if len(report) == 0 {
		t.Error("统计报告为空")
	}

	t.Log("生成的统计报告:")
	t.Log(report)
}
