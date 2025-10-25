package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles configuration loading from multiple sources
type Loader struct {
	viper *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		viper: viper.New(),
	}
}

// Load loads configuration with the following priority (highest to lowest):
// 1. Command line flags (handled by cobra, will be merged separately)
// 2. Environment variables (PIXLY_*)
// 3. ~/.pixly/config.yaml
// 4. ./.pixly.yaml (current directory)
// 5. Default values
func (l *Loader) Load() (*Config, error) {
	// 设置配置文件信息
	l.viper.SetConfigName("config")
	l.viper.SetConfigType("yaml")

	// 添加配置文件搜索路径
	// 1. 用户home目录下的.pixly/config.yaml
	if homeDir, err := os.UserHomeDir(); err == nil {
		pixlyDir := filepath.Join(homeDir, ".pixly")
		l.viper.AddConfigPath(pixlyDir)
	}

	// 2. 当前工作目录下的.pixly.yaml
	l.viper.AddConfigPath(".")

	// 3. 项目根目录的config/config.yaml（开发时使用）
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		l.viper.AddConfigPath(filepath.Join(exeDir, "config"))
	}

	// 设置环境变量前缀
	l.viper.SetEnvPrefix("PIXLY")
	l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	l.viper.AutomaticEnv()

	// 尝试读取配置文件（如果不存在则使用默认值）
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// 配置文件存在但读取失败
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认配置
		// 这是正常情况，不返回错误
	}

	// 创建配置对象并设置默认值
	config := DefaultConfig()

	// 将viper配置解析到config结构体
	if err := l.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 扩展路径中的波浪号（~）
	config = l.expandPaths(config)

	return config, nil
}

// LoadFromFile loads configuration from a specific file
func (l *Loader) LoadFromFile(configPath string) (*Config, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 设置配置文件路径
	l.viper.SetConfigFile(configPath)

	// 读取配置文件
	if err := l.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 创建配置对象
	config := DefaultConfig()

	// 解析配置
	if err := l.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 扩展路径
	config = l.expandPaths(config)

	return config, nil
}

// expandPaths expands ~ in file paths to home directory
func (l *Loader) expandPaths(config *Config) *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config
	}

	// 扩展日志路径
	if strings.HasPrefix(config.Logging.FilePath, "~") {
		config.Logging.FilePath = strings.Replace(config.Logging.FilePath, "~", homeDir, 1)
	}

	// 扩展知识库路径
	if strings.HasPrefix(config.Knowledge.DBPath, "~") {
		config.Knowledge.DBPath = strings.Replace(config.Knowledge.DBPath, "~", homeDir, 1)
	}

	return config
}

// GetViper returns the underlying viper instance for advanced usage
func (l *Loader) GetViper() *viper.Viper {
	return l.viper
}
