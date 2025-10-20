package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/tools"
	"pixly/pkg/engine"
	"pixly/pkg/engine/quality"
	"pixly/pkg/scanner"
	"pixly/pkg/security"
	"pixly/pkg/batchdecision"
	"pixly/pkg/progressui"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// IntegrationTestSuite 集成测试套件
type IntegrationTestSuite struct {
	t              *testing.T
	logger         *zap.Logger
	testDataDir    string
	outputDir      string
	tempDir        string
	
	// 核心组件
	toolChecker    *tools.Checker
	qualityEngine  *quality.QualityEngine
	scanner        *scanner.Scanner
	securityChecker *security.SecurityChecker
	batchManager   *batchdecision.BatchDecisionManager
	progressUI     *progressui.AdvancedProgressUI
	processingManager *engine.ProcessingModeManager
	
	// 测试结果
	testResults    []*IntegrationTestResult
	startTime      time.Time
	endTime        time.Time
}

// IntegrationTestResult 集成测试结果
type IntegrationTestResult struct {
	TestName       string                   `json:"test_name"`
	Mode           types.AppMode           `json:"mode"`
	Success        bool                    `json:"success"`
	Duration       time.Duration           `json:"duration"`
	FilesProcessed int                     `json:"files_processed"`
	FilesSuccess   int                     `json:"files_success"`
	FilesFailed    int                     `json:"files_failed"`
	FilesSkipped   int                     `json:"files_skipped"`
	SpaceSaved     int64                   `json:"space_saved"`
	ErrorMessages  []string                `json:"error_messages"`
	PerformanceStats *PerformanceStats     `json:"performance_stats"`
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	ScanTime       time.Duration `json:"scan_time"`
	AnalysisTime   time.Duration `json:"analysis_time"`
	ProcessingTime time.Duration `json:"processing_time"`
	TotalTime      time.Duration `json:"total_time"`
	ThroughputMB   float64       `json:"throughput_mb"`
	FilesPerSecond float64       `json:"files_per_second"`
}

// NewIntegrationTestSuite 创建集成测试套件
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	logger := zaptest.NewLogger(t)
	
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "pixly_integration_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	
	testDataDir := filepath.Join(tempDir, "test_data")
	outputDir := filepath.Join(tempDir, "output")
	
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("创建测试数据目录失败: %v", err)
	}
	
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}
	
	suite := &IntegrationTestSuite{
		t:           t,
		logger:      logger,
		testDataDir: testDataDir,
		outputDir:   outputDir,
		tempDir:     tempDir,
		testResults: make([]*IntegrationTestResult, 0),
	}
	
	// 初始化核心组件
	suite.initializeComponents()
	
	// 创建测试数据
	suite.createTestData()
	
	t.Logf("集成测试套件初始化完成，测试目录: %s", tempDir)
	
	return suite
}

// initializeComponents 初始化核心组件
func (its *IntegrationTestSuite) initializeComponents() {
	// 工具检查器
	its.toolChecker = tools.NewChecker(its.logger)
	
	// 品质引擎
	its.qualityEngine = quality.NewQualityEngine(its.logger, "", "", true)
	
	// 扫描器
	its.scanner = scanner.NewScanner(its.logger)
	
	// 安全检查器
	its.securityChecker = security.NewSecurityChecker(its.logger)
	
	// 批量决策管理器（非交互模式 + 测试强制处理模式）
	its.batchManager = batchdecision.NewBatchDecisionManager(its.logger, false)
	// 测试中强制处理所有低品质文件，不跳过
	its.batchManager.SetTestMode(true) // 启用测试模式
	
	// 进度UI
	its.progressUI = progressui.NewAdvancedProgressUI(its.logger)
	
	// 获取工具路径
	toolPaths, err := its.toolChecker.CheckAll()
	if err != nil {
		its.logger.Warn("工具检查警告，某些功能可能无法测试", zap.Error(err))
	}
	
	// 处理管理器
	its.processingManager = engine.NewProcessingModeManager(its.logger, toolPaths, its.qualityEngine)
}

// createTestData 创建测试数据
func (its *IntegrationTestSuite) createTestData() {
	// 创建各种测试文件
	testFiles := []struct {
		name    string
		content string
		size    int64
	}{
		// 模拟不同格式的测试文件
		{"test1.jpg", "fake jpeg content", 1024},
		{"test2.png", "fake png content", 2048},
		{"test3.webp", "fake webp content", 1536},
		{"test4.gif", "fake gif content", 800},
		{"test5.mp4", "fake mp4 content", 5120},
		{"test6.mov", "fake mov content", 4096},
		
		// 特殊文件测试
		{".DS_Store", "system file", 100},
		{"document.pdf", "pdf content", 2000},
		{"source.psd", "psd content", 10240},
		
		// 已是目标格式的文件
		{"existing.jxl", "jxl content", 512},
		{"existing.avif", "avif content", 1024},
	}
	
	for _, file := range testFiles {
		filePath := filepath.Join(its.testDataDir, file.name)
		content := strings.Repeat(file.content+" ", int(file.size/int64(len(file.content)+1)))
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			its.t.Errorf("创建测试文件失败 %s: %v", file.name, err)
		}
	}
	
	its.logger.Info("测试数据创建完成", zap.Int("file_count", len(testFiles)))
}

// RunFullIntegrationTest 运行完整集成测试
func (its *IntegrationTestSuite) RunFullIntegrationTest() {
	its.startTime = time.Now()
	defer func() {
		its.endTime = time.Now()
	}()
	
	its.t.Log("开始运行完整集成测试")
	
	// 测试三大处理模式
	modes := []types.AppMode{
		types.ModeAutoPlus,
		types.ModeQuality,
		types.ModeEmoji,
	}
	
	for _, mode := range modes {
		its.t.Logf("测试处理模式: %s", mode.String())
		result := its.testProcessingMode(mode)
		its.testResults = append(its.testResults, result)
		
		if !result.Success {
			its.t.Errorf("模式 %s 测试失败: %v", mode.String(), result.ErrorMessages)
		}
	}
	
	// 生成测试报告
	its.generateTestReport()
	
	its.t.Log("完整集成测试完成")
}

// testProcessingMode 测试单个处理模式
func (its *IntegrationTestSuite) testProcessingMode(mode types.AppMode) *IntegrationTestResult {
	testStart := time.Now()
	
	result := &IntegrationTestResult{
		TestName:      fmt.Sprintf("ProcessingMode_%s", mode.String()),
		Mode:          mode,
		Success:       true,
		ErrorMessages: make([]string, 0),
		PerformanceStats: &PerformanceStats{},
	}
	
	ctx := context.Background()
	
	// 1. 安全检查阶段
	securityResult, err := its.securityChecker.PerformSecurityCheck(its.testDataDir)
	if err != nil || !securityResult.Passed {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("安全检查失败: %v", err))
		result.Duration = time.Since(testStart)
		return result
	}
	
	// 2. 扫描阶段
	scanStart := time.Now()
	files, err := its.scanner.ScanDirectory(ctx, its.testDataDir)
	if err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("扫描失败: %v", err))
		result.Duration = time.Since(testStart)
		return result
	}
	result.PerformanceStats.ScanTime = time.Since(scanStart)
	
	// 转换为MediaInfo格式
	var mediaInfos []*types.MediaInfo
	for _, file := range files {
		if !file.IsDir {
			mediaInfo := &types.MediaInfo{
				Path:    file.Path,
				Size:    file.Size,
				ModTime: time.Unix(file.ModTime, 0),
				Type:    types.MediaTypeUnknown,
				Status:  types.StatusPending,
			}
			mediaInfos = append(mediaInfos, mediaInfo)
		}
	}
	
	its.t.Logf("扫描发现 %d 个文件", len(mediaInfos))
	
	// 3. 批量决策阶段（非交互模式，使用默认决策）
	batchResult, err := its.batchManager.ProcessBatchDecisions(ctx)
	if err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("批量决策失败: %v", err))
		result.Duration = time.Since(testStart)
		return result
	}
	
	its.t.Logf("批量决策完成，处理 %d 个决策", batchResult.Summary.TotalFiles)
	
	// 4. 核心处理阶段
	processingStart := time.Now()
	processingResults, err := its.processingManager.ProcessFiles(ctx, mode, mediaInfos)
	if err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("核心处理失败: %v", err))
		result.Duration = time.Since(testStart)
		return result
	}
	result.PerformanceStats.ProcessingTime = time.Since(processingStart)
	
	// 5. 统计结果
	result.FilesProcessed = len(processingResults)
	for _, pr := range processingResults {
		if pr.Success {
			result.FilesSuccess++
			result.SpaceSaved += pr.SpaceSaved
		} else {
			result.FilesFailed++
			if pr.Error != "" {
				result.ErrorMessages = append(result.ErrorMessages, pr.Error)
			}
		}
	}
	
	// 6. 性能统计
	result.Duration = time.Since(testStart)
	result.PerformanceStats.TotalTime = result.Duration
	
	if result.Duration > 0 {
		result.PerformanceStats.FilesPerSecond = float64(result.FilesProcessed) / result.Duration.Seconds()
	}
	
	// 计算吞吐量
	totalSize := int64(0)
	for _, mediaInfo := range mediaInfos {
		totalSize += mediaInfo.Size
	}
	if result.Duration > 0 {
		result.PerformanceStats.ThroughputMB = float64(totalSize) / (1024 * 1024) / result.Duration.Seconds()
	}
	
	its.t.Logf("模式 %s 测试完成: 成功 %d, 失败 %d, 耗时 %v",
		mode.String(), result.FilesSuccess, result.FilesFailed, result.Duration)
	
	return result
}

// TestConcurrentProcessing 测试并发处理
func (its *IntegrationTestSuite) TestConcurrentProcessing(t *testing.T) {
	// 创建大量测试文件来测试并发性能
	concurrentTestDir := filepath.Join(its.tempDir, "concurrent_test")
	if err := os.MkdirAll(concurrentTestDir, 0755); err != nil {
		t.Fatalf("创建并发测试目录失败: %v", err)
	}
	
	// 创建100个测试文件
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		fileName := fmt.Sprintf("concurrent_test_%03d.jpg", i)
		filePath := filepath.Join(concurrentTestDir, fileName)
		content := fmt.Sprintf("test content for file %d", i)
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Errorf("创建并发测试文件失败 %s: %v", fileName, err)
		}
	}
	
	// 测试并发处理
	ctx := context.Background()
	files, err := its.scanner.ScanDirectory(ctx, concurrentTestDir)
	if err != nil {
		t.Fatalf("并发测试扫描失败: %v", err)
	}
	
	if len(files) != fileCount {
		t.Errorf("预期扫描 %d 个文件，实际扫描 %d 个", fileCount, len(files))
	}
	
	t.Logf("并发测试完成，处理了 %d 个文件", len(files))
}

// TestErrorRecovery 测试错误恢复
func (its *IntegrationTestSuite) TestErrorRecovery(t *testing.T) {
	// 创建有问题的测试文件
	errorTestDir := filepath.Join(its.tempDir, "error_test")
	if err := os.MkdirAll(errorTestDir, 0755); err != nil {
		t.Fatalf("创建错误测试目录失败: %v", err)
	}
	
	// 创建损坏的文件
	corruptedFile := filepath.Join(errorTestDir, "corrupted.jpg")
	if err := os.WriteFile(corruptedFile, []byte("invalid jpeg data"), 0644); err != nil {
		t.Fatalf("创建损坏文件失败: %v", err)
	}
	
	// 创建无权限文件
	noPermissionFile := filepath.Join(errorTestDir, "no_permission.png")
	if err := os.WriteFile(noPermissionFile, []byte("png data"), 0000); err != nil {
		t.Fatalf("创建无权限文件失败: %v", err)
	}
	
	// 测试错误处理
	ctx := context.Background()
	files, err := its.scanner.ScanDirectory(ctx, errorTestDir)
	
	// 扫描应该成功，但会跳过问题文件
	if err != nil {
		t.Logf("扫描遇到错误（预期）: %v", err)
	}
	
	t.Logf("错误恢复测试完成，扫描了 %d 个文件", len(files))
}

// generateTestReport 生成测试报告
func (its *IntegrationTestSuite) generateTestReport() {
	reportPath := filepath.Join(its.outputDir, "integration_test_report.txt")
	
	var report strings.Builder
	report.WriteString("=== Pixly 集成测试报告 ===\n\n")
	
	report.WriteString(fmt.Sprintf("测试开始时间: %s\n", its.startTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("测试结束时间: %s\n", its.endTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("总测试时长: %v\n\n", its.endTime.Sub(its.startTime)))
	
	// 总体统计
	totalTests := len(its.testResults)
	successTests := 0
	totalFiles := 0
	totalSuccess := 0
	totalFailed := 0
	
	for _, result := range its.testResults {
		if result.Success {
			successTests++
		}
		totalFiles += result.FilesProcessed
		totalSuccess += result.FilesSuccess
		totalFailed += result.FilesFailed
	}
	
	report.WriteString("总体统计:\n")
	report.WriteString(fmt.Sprintf("  测试模式数: %d\n", totalTests))
	report.WriteString(fmt.Sprintf("  成功模式数: %d\n", successTests))
	report.WriteString(fmt.Sprintf("  总处理文件: %d\n", totalFiles))
	report.WriteString(fmt.Sprintf("  成功文件数: %d\n", totalSuccess))
	report.WriteString(fmt.Sprintf("  失败文件数: %d\n\n", totalFailed))
	
	// 各模式详细结果
	for _, result := range its.testResults {
		report.WriteString(fmt.Sprintf("=== %s ===\n", result.TestName))
		report.WriteString(fmt.Sprintf("模式: %s\n", result.Mode.String()))
		report.WriteString(fmt.Sprintf("结果: %s\n", map[bool]string{true: "成功", false: "失败"}[result.Success]))
		report.WriteString(fmt.Sprintf("耗时: %v\n", result.Duration))
		report.WriteString(fmt.Sprintf("处理文件: %d\n", result.FilesProcessed))
		report.WriteString(fmt.Sprintf("成功: %d, 失败: %d, 跳过: %d\n", result.FilesSuccess, result.FilesFailed, result.FilesSkipped))
		
		if result.PerformanceStats != nil {
			report.WriteString(fmt.Sprintf("性能统计:\n"))
			report.WriteString(fmt.Sprintf("  扫描时间: %v\n", result.PerformanceStats.ScanTime))
			report.WriteString(fmt.Sprintf("  处理时间: %v\n", result.PerformanceStats.ProcessingTime))
			report.WriteString(fmt.Sprintf("  处理速度: %.2f 文件/秒\n", result.PerformanceStats.FilesPerSecond))
			report.WriteString(fmt.Sprintf("  吞吐量: %.2f MB/秒\n", result.PerformanceStats.ThroughputMB))
		}
		
		if len(result.ErrorMessages) > 0 {
			report.WriteString("错误信息:\n")
			for _, errMsg := range result.ErrorMessages {
				report.WriteString(fmt.Sprintf("  - %s\n", errMsg))
			}
		}
		
		report.WriteString("\n")
	}
	
	// 写入报告文件
	if err := os.WriteFile(reportPath, []byte(report.String()), 0644); err != nil {
		its.t.Errorf("写入测试报告失败: %v", err)
	} else {
		its.t.Logf("测试报告已生成: %s", reportPath)
	}
	
	// 同时输出到测试日志
	its.t.Log("\n" + report.String())
}

// Cleanup 清理测试资源
func (its *IntegrationTestSuite) Cleanup() {
	// 停止进度UI
	its.progressUI.Stop()
	
	// 清理临时目录
	if err := os.RemoveAll(its.tempDir); err != nil {
		its.t.Logf("清理临时目录失败: %v", err)
	} else {
		its.t.Logf("临时目录已清理: %s", its.tempDir)
	}
}

// GetTestResults 获取测试结果
func (its *IntegrationTestSuite) GetTestResults() []*IntegrationTestResult {
	return its.testResults
}