package fixtures

import (
	"os"
	"path/filepath"
	"testing"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestFixtures 测试夹具结构
type TestFixtures struct {
	TempDir     string
	TestFiles   []string
	Logger      *testing.T
	CleanupFunc func()
}

// CreateTestFixtures 创建测试夹具
func CreateTestFixtures(t *testing.T) *TestFixtures {
	tempDir, err := os.MkdirTemp("", "pixly_test_fixtures_")
	if err != nil {
		t.Fatal(err)
	}

	// 创建测试文件
	testFiles := []string{
		"sample1.jpg",
		"sample2.png",
		"sample3.mp4",
		"sample4.mov",
		"sample5.webp",
		"sample6.heic",
		"corrupted.jpg", // 模拟损坏文件
		"tiny.png",      // 模拟极小文件
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		var content []byte

		if filename == "corrupted.jpg" {
			// 创建损坏的文件内容
			content = []byte("not a valid image file")
		} else if filename == "tiny.png" {
			// 创建极小的文件
			content = []byte("tiny")
		} else {
			// 创建正常的测试内容
			content = []byte("test image content for " + filename)
		}

		err := os.WriteFile(filePath, content, 0644)
		if err != nil {
			os.RemoveAll(tempDir)
			t.Fatal(err)
		}
	}

	return &TestFixtures{
		TempDir:   tempDir,
		TestFiles: testFiles,
		Logger:    t,
		CleanupFunc: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// Cleanup 清理测试夹具
func (tf *TestFixtures) Cleanup() {
	if tf.CleanupFunc != nil {
		tf.CleanupFunc()
	}
}

// GetTestConfig 获取测试配置
func (tf *TestFixtures) GetTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.TargetDir = tf.TempDir
	cfg.DebugMode = true
	cfg.DryRun = true
	cfg.ConcurrentJobs = 2
	cfg.MaxRetries = 1
	cfg.CreateBackups = false
	return cfg
}

// GetTestToolResults 获取测试工具结果
func (tf *TestFixtures) GetTestToolResults() types.ToolCheckResults {
	return types.ToolCheckResults{
		FfmpegStablePath: "/usr/local/bin/ffmpeg",
		// 其他工具路径可以根据需要添加
	}
}

// CreateSubDirectory 创建子目录测试结构
func (tf *TestFixtures) CreateSubDirectory(name string) string {
	subDir := filepath.Join(tf.TempDir, name)
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		tf.Logger.Fatal(err)
	}

	// 在子目录中创建一些文件
	testFile := filepath.Join(subDir, "subdir_test.jpg")
	err = os.WriteFile(testFile, []byte("subdir test content"), 0644)
	if err != nil {
		tf.Logger.Fatal(err)
	}

	return subDir
}

// CreateLargeFile 创建大文件用于测试
func (tf *TestFixtures) CreateLargeFile(filename string, sizeMB int) string {
	filePath := filepath.Join(tf.TempDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		tf.Logger.Fatal(err)
	}
	defer file.Close()

	// 写入指定大小的数据
	data := make([]byte, 1024*1024) // 1MB chunk
	for i := 0; i < sizeMB; i++ {
		_, err := file.Write(data)
		if err != nil {
			tf.Logger.Fatal(err)
		}
	}

	return filePath
}

// CreateEmptyDirectory 创建空目录
func (tf *TestFixtures) CreateEmptyDirectory(name string) string {
	emptyDir := filepath.Join(tf.TempDir, name)
	err := os.MkdirAll(emptyDir, 0755)
	if err != nil {
		tf.Logger.Fatal(err)
	}
	return emptyDir
}

// GetTestLogger 获取测试日志器
func GetTestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t)
}
