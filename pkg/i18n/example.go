package i18n

import (
	"fmt"
)

// ExampleBasicUsage demonstrates basic i18n usage
func ExampleBasicUsage() {
	// 初始化（自动检测系统语言）
	Init(ZhCN)
	
	// 简单翻译
	fmt.Println(T(MsgWelcome))
	// 输出: 欢迎使用Pixly图像转换工具
	
	// 带参数的翻译
	fmt.Println(T(MsgProgress, "75%"))
	// 输出: 进度: 75%
	
	// 切换语言
	SetLocale(EnUS)
	fmt.Println(T(MsgWelcome))
	// 输出: Welcome to Pixly Image Converter
}

// ExampleManagerUsage demonstrates manager usage
func ExampleManagerUsage() {
	// 创建管理器
	mgr := NewManager()
	
	// 自动检测语言
	locale := mgr.AutoDetectAndSet()
	fmt.Printf("Detected locale: %s\n", locale)
	
	// 翻译消息
	fmt.Println(mgr.T(MsgScanningFiles))
	fmt.Println(mgr.T(MsgProcessingFile, "image.jpg"))
	
	// 切换语言
	mgr.SetLocale(EnUS)
	fmt.Println(mgr.T(MsgScanningFiles))
	// 输出: Scanning files...
}

// ExampleCLIIntegration demonstrates CLI integration
func ExampleCLIIntegration() {
	// 从命令行参数设置语言
	var langFlag string = "en-US" // 从flag获取
	
	locale := ParseLocaleString(langFlag)
	Init(locale)
	
	// 显示帮助
	fmt.Println(T("cli.help"))
	fmt.Println(T("cli.input"))
	fmt.Println(T("cli.output"))
	
	// 显示进度
	for i := 1; i <= 5; i++ {
		filename := fmt.Sprintf("file%d.jpg", i)
		fmt.Println(T(MsgProcessingFile, filename))
	}
}

// ExampleWithConfiguration demonstrates config integration
func ExampleWithConfiguration() {
	// 假设从配置文件读取
	type Config struct {
		Language string
	}
	
	config := Config{Language: "zh-CN"}
	
	// 设置语言
	locale := ParseLocaleString(config.Language)
	if !IsValidLocale(locale) {
		locale = ZhCN // 回退到默认
	}
	
	Init(locale)
	
	// 使用翻译
	fmt.Println(T("config.loading"))
	// 模拟加载...
	fmt.Println(T("config.loaded"))
}

// ExampleMultipleMessages demonstrates multiple message types
func ExampleMultipleMessages() {
	Init(ZhCN)
	
	// 文件操作消息
	fmt.Println(T(MsgFileCompleted, "photo.jpg"))
	fmt.Println(T(MsgFileFailed, "corrupt.jpg"))
	fmt.Println(T(MsgFileSkipped, "existing.jpg"))
	
	// 统计消息
	fmt.Println(T(StatTotalFiles, 100))
	fmt.Println(T(StatSucceeded, 95))
	fmt.Println(T(StatFailed, 3))
	fmt.Println(T(StatSkipped, 2))
	
	// 错误消息
	fmt.Println(T(ErrFileNotFound, "/path/to/file.jpg"))
	fmt.Println(T(ErrPermissionDenied, "/protected/file.jpg"))
}

// ExampleQualityAnalysis demonstrates quality-related messages
func ExampleQualityAnalysis() {
	Init(ZhCN)
	
	fmt.Println(T("quality.analyzing"))
	
	// 质量等级
	levels := []string{
		"quality.level.extreme",
		"quality.level.high",
		"quality.level.medium",
		"quality.level.low",
		"quality.level.very_low",
	}
	
	for _, level := range levels {
		fmt.Printf("- %s\n", T(level))
	}
	
	// 内容类型
	types := []string{
		"quality.content.photo",
		"quality.content.graphic",
		"quality.content.screenshot",
		"quality.content.mixed",
	}
	
	for _, t := range types {
		fmt.Printf("- %s\n", T(t))
	}
}

// ExampleCheckpoint demonstrates checkpoint-related messages
func ExampleCheckpoint() {
	Init(EnUS)
	
	// 断点续传流程
	fmt.Println(T("checkpoint.found"))
	fmt.Println(T("checkpoint.resume_ask"))
	
	// 用户选择恢复
	fmt.Println(T("checkpoint.loading"))
	fmt.Println(T("checkpoint.loaded"))
	
	// 处理过程中保存
	fmt.Println(T("checkpoint.saving"))
	fmt.Println(T("checkpoint.saved"))
}
