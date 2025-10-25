package i18n

import (
	"testing"
)

func TestTranslator(t *testing.T) {
	translator := NewTranslator(ZhCN)
	
	// 测试简单翻译
	msg := translator.T(MsgWelcome)
	if msg == "" {
		t.Error("翻译结果不应为空")
	}
	if msg == MsgWelcome {
		t.Error("未找到翻译")
	}
	
	// 测试带参数的翻译
	msgWithArgs := translator.T(MsgProgress, "50%")
	if !contains(msgWithArgs, "50%") {
		t.Errorf("占位符替换失败: %s", msgWithArgs)
	}
}

func TestLocaleSwitch(t *testing.T) {
	translator := NewTranslator(ZhCN)
	
	// 中文翻译
	zhMsg := translator.T(MsgWelcome)
	
	// 切换到英文
	translator.SetLocale(EnUS)
	enMsg := translator.T(MsgWelcome)
	
	// 两者应该不同
	if zhMsg == enMsg {
		t.Error("中英文翻译不应相同")
	}
}

func TestGlobalTranslator(t *testing.T) {
	Init(ZhCN)
	
	msg1 := T(MsgWelcome)
	if msg1 == "" {
		t.Error("全局翻译器返回空")
	}
	
	SetLocale(EnUS)
	msg2 := T(MsgWelcome)
	
	if msg1 == msg2 {
		t.Error("语言切换未生效")
	}
}

func TestLocaleDetector(t *testing.T) {
	detector := NewLocaleDetector()
	locale := detector.DetectLocale()
	
	if !IsValidLocale(locale) {
		t.Errorf("检测到无效语言: %s", locale)
	}
}

func TestParseLocaleString(t *testing.T) {
	tests := []struct {
		input    string
		expected Locale
	}{
		{"zh-CN", ZhCN},
		{"zh_CN", ZhCN},
		{"chinese", ZhCN},
		{"en-US", EnUS},
		{"en_US", EnUS},
		{"english", EnUS},
		{"unknown", ZhCN}, // 默认
	}
	
	for _, tt := range tests {
		result := ParseLocaleString(tt.input)
		if result != tt.expected {
			t.Errorf("ParseLocaleString(%s) = %s, 期望 %s", 
				tt.input, result, tt.expected)
		}
	}
}

func TestPlaceholderReplacement(t *testing.T) {
	tests := []struct {
		message  string
		args     []interface{}
		expected string
	}{
		{"Hello {0}", []interface{}{"World"}, "Hello World"},
		{"{0} + {1} = {2}", []interface{}{1, 2, 3}, "1 + 2 = 3"},
		{"No placeholders", []interface{}{}, "No placeholders"},
	}
	
	for _, tt := range tests {
		result := replacePlaceholders(tt.message, tt.args...)
		if result != tt.expected {
			t.Errorf("replacePlaceholders(%s, %v) = %s, 期望 %s",
				tt.message, tt.args, result, tt.expected)
		}
	}
}

func TestManager(t *testing.T) {
	mgr := NewManager()
	
	// 测试自动检测
	locale := mgr.AutoDetectAndSet()
	if !IsValidLocale(locale) {
		t.Error("自动检测返回无效语言")
	}
	
	// 测试翻译
	msg := mgr.T(MsgWelcome)
	if msg == "" {
		t.Error("Manager翻译返回空")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
