package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// Locale represents a language locale
type Locale string

const (
	ZhCN Locale = "zh-CN" // 简体中文
	EnUS Locale = "en-US" // English
)

// Message keys
const (
	// 通用
	MsgWelcome          = "msg.welcome"
	MsgStarting         = "msg.starting"
	MsgCompleted        = "msg.completed"
	MsgFailed           = "msg.failed"
	MsgProgress         = "msg.progress"
	MsgSuccess          = "msg.success"
	MsgError            = "msg.error"
	MsgWarning          = "msg.warning"
	
	// 文件操作
	MsgScanningFiles    = "msg.scanning_files"
	MsgProcessingFile   = "msg.processing_file"
	MsgFileCompleted    = "msg.file_completed"
	MsgFileFailed       = "msg.file_failed"
	MsgFileSkipped      = "msg.file_skipped"
	
	// 转换
	MsgConversionStart  = "msg.conversion_start"
	MsgConversionDone   = "msg.conversion_done"
	MsgSpaceSaved       = "msg.space_saved"
	MsgQualityAnalysis  = "msg.quality_analysis"
	
	// 会话
	MsgSessionCreate    = "msg.session_create"
	MsgSessionLoad      = "msg.session_load"
	MsgSessionComplete  = "msg.session_complete"
	MsgSessionResume    = "msg.session_resume"
	
	// 错误
	ErrFileNotFound     = "err.file_not_found"
	ErrPermissionDenied = "err.permission_denied"
	ErrDiskFull         = "err.disk_full"
	ErrInvalidFormat    = "err.invalid_format"
	ErrConversionFailed = "err.conversion_failed"
	
	// 统计
	StatTotalFiles      = "stat.total_files"
	StatProcessed       = "stat.processed"
	StatSucceeded       = "stat.succeeded"
	StatFailed          = "stat.failed"
	StatSkipped         = "stat.skipped"
	StatSpaceSaved      = "stat.space_saved"
	StatDuration        = "stat.duration"
)

// Translator manages translations
type Translator struct {
	locale   Locale
	messages map[Locale]map[string]string
	mu       sync.RWMutex
}

// NewTranslator creates a new translator
func NewTranslator(locale Locale) *Translator {
	t := &Translator{
		locale:   locale,
		messages: make(map[Locale]map[string]string),
	}
	
	// 加载默认语言包
	t.LoadMessages(ZhCN, getZhCNMessages())
	t.LoadMessages(EnUS, getEnUSMessages())
	
	return t
}

// LoadMessages loads messages for a locale
func (t *Translator) LoadMessages(locale Locale, messages map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages[locale] = messages
}

// SetLocale sets the current locale
func (t *Translator) SetLocale(locale Locale) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locale = locale
}

// GetLocale returns the current locale
func (t *Translator) GetLocale() Locale {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.locale
}

// T translates a message key
func (t *Translator) T(key string, args ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// 获取当前语言的消息
	msgs, ok := t.messages[t.locale]
	if !ok {
		// 回退到中文
		msgs = t.messages[ZhCN]
	}
	
	message, ok := msgs[key]
	if !ok {
		// 如果key不存在，返回key本身
		return key
	}
	
	// 替换占位符
	if len(args) > 0 {
		message = replacePlaceholders(message, args...)
	}
	
	return message
}

// TF translates a message key with formatting
func (t *Translator) TF(key string, args ...interface{}) string {
	return t.T(key, args...)
}

// replacePlaceholders replaces {0}, {1}, etc. with args
func replacePlaceholders(message string, args ...interface{}) string {
	result := message
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		value := fmt.Sprintf("%v", arg)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// Global translator instance
var defaultTranslator *Translator
var once sync.Once

// Init initializes the default translator
func Init(locale Locale) {
	once.Do(func() {
		defaultTranslator = NewTranslator(locale)
	})
}

// T translates using the default translator
func T(key string, args ...interface{}) string {
	if defaultTranslator == nil {
		Init(ZhCN)
	}
	return defaultTranslator.T(key, args...)
}

// SetLocale sets the locale for the default translator
func SetLocale(locale Locale) {
	if defaultTranslator == nil {
		Init(locale)
	} else {
		defaultTranslator.SetLocale(locale)
	}
}

// GetLocale returns the current locale
func GetLocale() Locale {
	if defaultTranslator == nil {
		Init(ZhCN)
	}
	return defaultTranslator.GetLocale()
}

// GetTranslator returns the default translator
func GetTranslator() *Translator {
	if defaultTranslator == nil {
		Init(ZhCN)
	}
	return defaultTranslator
}
