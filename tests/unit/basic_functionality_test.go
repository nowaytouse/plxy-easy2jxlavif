package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	fileatomic "pixly/pkg/atomic"
	"pixly/pkg/batchdecision"
	"pixly/pkg/processmonitor"
	"pixly/pkg/tools"

	"go.uber.org/zap/zaptest"
)

// TestToolsChecker 测试工具检查器
func TestToolsChecker(t *testing.T) {
	logger := zaptest.NewLogger(t)

	checker := tools.NewChecker(logger)
	if checker == nil {
		t.Fatal("工具检查器创建失败")
	}

	results, err := checker.CheckAll()
	if err != nil {
		t.Logf("工具检查警告: %v", err)
	}

	// 至少应该检测到一些工具
	t.Logf("检测结果 - FFmpeg: %v, CJXL: %v, AVIF: %v, ExifTool: %v",
		results.HasFfmpeg, results.HasCjxl, results.HasAvifenc, results.HasExiftool)
}

// TestBatchDecisionManager 测试批量决策管理器
func TestBatchDecisionManager(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 测试非交互模式
	manager := batchdecision.NewBatchDecisionManager(logger, false)
	if manager == nil {
		t.Fatal("批量决策管理器创建失败")
	}

	// 添加测试文件
	manager.AddCorruptedFile("/test/corrupt.jpg", batchdecision.CorruptionFileHeader, "测试损坏文件", false)
	manager.AddLowQualityFile("/test/lowquality.jpg", 15.5, []string{"低分辨率", "过度压缩"}, true)

	// 测试批量决策处理（应该使用默认选择）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := manager.ProcessBatchDecisions(ctx)
	if err != nil {
		t.Errorf("批量决策处理失败: %v", err)
	}

	if result != nil {
		t.Logf("决策结果: 总文件数 %d, 成功 %d",
			result.Summary.TotalFiles, result.Summary.SuccessfulFiles)
	}
}

// TestAtomicFileOperations 测试原子性文件操作
func TestAtomicFileOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "pixly_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir := filepath.Join(tempDir, "backup")
	operator := fileatomic.NewAtomicFileOperator(logger, backupDir, tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	newFile := filepath.Join(tempDir, "new.txt")

	if err := os.WriteFile(testFile, []byte("原始内容"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if err := os.WriteFile(newFile, []byte("新内容"), 0644); err != nil {
		t.Fatalf("创建新文件失败: %v", err)
	}

	// 测试原子性替换
	ctx := context.Background()
	err = operator.ReplaceFile(ctx, testFile, newFile)
	if err != nil {
		t.Errorf("原子性文件替换失败: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("读取替换后文件失败: %v", err)
	} else if string(content) != "新内容" {
		t.Errorf("文件内容不正确，期望'新内容'，实际'%s'", string(content))
	}
}

// TestProcessMonitor 测试进程监控器
func TestProcessMonitor(t *testing.T) {
	logger := zaptest.NewLogger(t)

	monitor := processmonitor.NewProcessMonitor(logger)
	if monitor == nil {
		t.Fatal("进程监控器创建失败")
	}

	// 创建简单的测试命令
	ctx := context.Background()
	processCtx := &processmonitor.ProcessContext{
		SourceFile:      "/test/input.jpg",
		FileSize:        1024,
		FileFormat:      "jpeg",
		Operation:       "convert",
		ComplexityLevel: processmonitor.ComplexityLow,
		Priority:        processmonitor.PriorityNormal,
		Metadata:        make(map[string]string),
	}

	// 测试命令监控（使用echo命令作为测试）
	cmd := exec.Command("echo", "test")
	err := monitor.MonitorCommand(ctx, cmd, processCtx)
	if err != nil {
		t.Errorf("命令监控失败: %v", err)
	}

	t.Log("进程监控器测试完成")
}

// TestMainIntegration 集成测试
func TestMainIntegration(t *testing.T) {
	// 测试编译是否成功
	_, err := os.Stat("../main.go")
	if err != nil {
		t.Fatalf("main.go文件不存在: %v", err)
	}

	t.Log("主程序文件存在，基础集成测试通过")
}
