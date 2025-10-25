package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager manages Pixly configuration
type Manager struct {
	config     *Config
	configPath string
	loader     *Loader
	validator  *Validator
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		loader: NewLoader(),
	}
}

// Load loads configuration from default locations with validation
func (m *Manager) Load() error {
	// 加载配置
	config, err := m.loader.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	m.config = config

	// 验证配置
	m.validator = NewValidator(m.config)
	if err := m.validator.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	return nil
}

// LoadFromFile loads configuration from a specific file
func (m *Manager) LoadFromFile(configPath string) error {
	// 加载配置
	config, err := m.loader.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	m.config = config
	m.configPath = configPath

	// 验证配置
	m.validator = NewValidator(m.config)
	if err := m.validator.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	if m.config == nil {
		return DefaultConfig()
	}
	return m.config
}

// SaveToFile saves the current configuration to a file
func (m *Manager) SaveToFile(configPath string) error {
	if m.config == nil {
		return fmt.Errorf("没有配置可保存")
	}

	// 创建目录
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	m.configPath = configPath
	return nil
}

// SaveDefault saves the default configuration to ~/.pixly/config.yaml
func (m *Manager) SaveDefault() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	configPath := filepath.Join(homeDir, ".pixly", "config.yaml")

	// 如果文件已存在，不覆盖
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("配置文件已存在: %s", configPath)
	}

	// 创建默认配置
	m.config = DefaultConfig()

	return m.SaveToFile(configPath)
}

// CreateDefaultIfNotExists creates default config if it doesn't exist
func (m *Manager) CreateDefaultIfNotExists() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	configPath := filepath.Join(homeDir, ".pixly", "config.yaml")

	// 如果配置文件已存在，直接返回路径
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// 创建默认配置
	m.config = DefaultConfig()

	if err := m.SaveToFile(configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// GetConfigPath returns the path to the loaded configuration file
func (m *Manager) GetConfigPath() string {
	if m.configPath != "" {
		return m.configPath
	}

	// 尝试检测配置文件位置
	if homeDir, err := os.UserHomeDir(); err == nil {
		pixlyConfig := filepath.Join(homeDir, ".pixly", "config.yaml")
		if _, err := os.Stat(pixlyConfig); err == nil {
			return pixlyConfig
		}
	}

	// 检查当前目录
	localConfig := ".pixly.yaml"
	if _, err := os.Stat(localConfig); err == nil {
		return localConfig
	}

	return "未找到配置文件"
}

// MergeFromCommandLine merges command line flags into configuration
func (m *Manager) MergeFromCommandLine(overrides map[string]interface{}) {
	if m.config == nil {
		m.config = DefaultConfig()
	}

	// 合并常用的命令行参数
	if val, ok := overrides["workers"]; ok {
		if workers, ok := val.(int); ok {
			m.config.Concurrency.ConversionWorkers = workers
		}
	}

	if val, ok := overrides["mode"]; ok {
		if mode, ok := val.(string); ok {
			m.config.Conversion.DefaultMode = mode
		}
	}

	if val, ok := overrides["ui-mode"]; ok {
		if uiMode, ok := val.(string); ok {
			m.config.UI.Mode = uiMode
		}
	}

	if val, ok := overrides["log-level"]; ok {
		if level, ok := val.(string); ok {
			m.config.Logging.Level = level
		}
	}

	if val, ok := overrides["enable-monitoring"]; ok {
		if enable, ok := val.(bool); ok {
			m.config.Concurrency.EnableMonitoring = enable
		}
	}

	if val, ok := overrides["keep-original"]; ok {
		if keep, ok := val.(bool); ok {
			m.config.Output.KeepOriginal = keep
		}
	}

	// 可以根据需要添加更多参数
}

// Reload reloads configuration from file
func (m *Manager) Reload() error {
	if m.configPath == "" {
		return m.Load()
	}
	return m.LoadFromFile(m.configPath)
}

// Reset resets configuration to default values
func (m *Manager) Reset() {
	m.config = DefaultConfig()
	m.configPath = ""
}

// DisplaySummary prints a summary of the current configuration
func (m *Manager) DisplaySummary() {
	if m.config == nil {
		fmt.Println("⚠️  未加载配置")
		return
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║   📋 Pixly 配置摘要                                          ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("📁 配置文件: %s\n", m.GetConfigPath())
	fmt.Println()

	fmt.Println("⚙️  核心设置:")
	fmt.Printf("  • 转换模式: %s\n", m.config.Conversion.DefaultMode)
	fmt.Printf("  • 并发Worker: %d\n", m.config.Concurrency.ConversionWorkers)
	fmt.Printf("  • 性能监控: %v\n", m.config.Concurrency.EnableMonitoring)
	fmt.Printf("  • 知识库: %v\n", m.config.Knowledge.Enable)
	fmt.Println()

	fmt.Println("🎨 格式设置:")
	fmt.Printf("  • PNG  → %s (effort: %d)\n", m.config.Conversion.Formats.PNG.Target, m.config.Conversion.Formats.PNG.Effort)
	fmt.Printf("  • JPEG → %s (effort: %d)\n", m.config.Conversion.Formats.JPEG.Target, m.config.Conversion.Formats.JPEG.Effort)
	fmt.Printf("  • GIF动 → %s (CRF: %d)\n", m.config.Conversion.Formats.GIF.AnimatedTarget, m.config.Conversion.Formats.GIF.AnimatedCRF)
	fmt.Printf("  • GIF静 → %s\n", m.config.Conversion.Formats.GIF.StaticTarget)
	fmt.Println()

	fmt.Println("🖥️  用户界面:")
	fmt.Printf("  • UI模式: %s\n", m.config.UI.Mode)
	fmt.Printf("  • 主题: %s\n", m.config.UI.Theme)
	fmt.Printf("  • Emoji: %v\n", m.config.UI.EnableEmoji)
	fmt.Printf("  • 动画: %v\n", m.config.UI.EnableAnimations)
	fmt.Println()

	fmt.Println("🔒 安全设置:")
	fmt.Printf("  • 路径检查: %v\n", m.config.Security.EnablePathCheck)
	fmt.Printf("  • 磁盘检查: %v\n", m.config.Security.CheckDiskSpace)
	fmt.Printf("  • 最小剩余空间: %d MB\n", m.config.Security.MinFreeSpaceMB)
	fmt.Println()

	fmt.Println("📊 输出设置:")
	fmt.Printf("  • 保留原文件: %v\n", m.config.Output.KeepOriginal)
	fmt.Printf("  • 生成报告: %v\n", m.config.Output.GenerateReport)
	fmt.Printf("  • 性能报告: %v\n", m.config.Output.GeneratePerformanceReport)
	fmt.Println()
}
