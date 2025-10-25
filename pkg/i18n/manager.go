package i18n

import (
	"os"
	"strings"
)

// LocaleDetector detects system locale
type LocaleDetector struct{}

// NewLocaleDetector creates a new locale detector
func NewLocaleDetector() *LocaleDetector {
	return &LocaleDetector{}
}

// DetectLocale detects the system locale
func (ld *LocaleDetector) DetectLocale() Locale {
	// 1. 检查环境变量 PIXLY_LANG
	if lang := os.Getenv("PIXLY_LANG"); lang != "" {
		return parseLocale(lang)
	}
	
	// 2. 检查环境变量 LANG
	if lang := os.Getenv("LANG"); lang != "" {
		return parseLocale(lang)
	}
	
	// 3. 检查环境变量 LC_ALL
	if lang := os.Getenv("LC_ALL"); lang != "" {
		return parseLocale(lang)
	}
	
	// 4. 默认中文
	return ZhCN
}

// parseLocale parses locale string
func parseLocale(lang string) Locale {
	lang = strings.ToLower(lang)
	
	// 中文判断
	if strings.Contains(lang, "zh") || 
	   strings.Contains(lang, "chinese") ||
	   strings.Contains(lang, "cn") {
		return ZhCN
	}
	
	// 英文判断
	if strings.Contains(lang, "en") ||
	   strings.Contains(lang, "english") ||
	   strings.Contains(lang, "us") ||
	   strings.Contains(lang, "gb") {
		return EnUS
	}
	
	// 默认中文
	return ZhCN
}

// Manager manages i18n operations
type Manager struct {
	translator *Translator
	detector   *LocaleDetector
}

// NewManager creates a new i18n manager
func NewManager() *Manager {
	detector := NewLocaleDetector()
	locale := detector.DetectLocale()
	translator := NewTranslator(locale)
	
	return &Manager{
		translator: translator,
		detector:   detector,
	}
}

// SetLocale sets the locale
func (m *Manager) SetLocale(locale Locale) {
	m.translator.SetLocale(locale)
}

// GetLocale returns current locale
func (m *Manager) GetLocale() Locale {
	return m.translator.GetLocale()
}

// T translates a key
func (m *Manager) T(key string, args ...interface{}) string {
	return m.translator.T(key, args...)
}

// GetTranslator returns the translator
func (m *Manager) GetTranslator() *Translator {
	return m.translator
}

// GetDetector returns the locale detector
func (m *Manager) GetDetector() *LocaleDetector {
	return m.detector
}

// AutoDetectAndSet auto-detects and sets locale
func (m *Manager) AutoDetectAndSet() Locale {
	locale := m.detector.DetectLocale()
	m.SetLocale(locale)
	return locale
}

// ParseLocaleString parses a locale string
func ParseLocaleString(s string) Locale {
	return parseLocale(s)
}

// GetAvailableLocales returns all available locales
func GetAvailableLocales() []Locale {
	return []Locale{ZhCN, EnUS}
}

// IsValidLocale checks if a locale is valid
func IsValidLocale(locale Locale) bool {
	available := GetAvailableLocales()
	for _, l := range available {
		if l == locale {
			return true
		}
	}
	return false
}
