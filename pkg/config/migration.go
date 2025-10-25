package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Migrator handles configuration migration from older versions
type Migrator struct {
	oldConfigPath string
	newConfigPath string
}

// NewMigrator creates a new configuration migrator
func NewMigrator() *Migrator {
	return &Migrator{}
}

// MigrateFromV3 migrates from v3.1.1 JSON config to v4.0 YAML config
func (m *Migrator) MigrateFromV3(oldConfigPath string) (*Config, error) {
	// 读取旧配置文件
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取旧配置文件失败: %w", err)
	}
	
	// 解析旧配置（简化结构）
	var oldConfig struct {
		DefaultWorkers     int    `json:"default_workers"`
		DefaultVerifyMode  string `json:"default_verify_mode"`
		LogFileName        string `json:"log_file_name"`
		ReplaceOriginals   bool   `json:"replace_originals"`
		DefaultCJXLThreads int    `json:"default_cjxl_threads"`
	}
	
	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return nil, fmt.Errorf("解析旧配置文件失败: %w", err)
	}
	
	// 创建新配置
	newConfig := DefaultConfig()
	
	// 迁移设置
	if oldConfig.DefaultWorkers > 0 {
		newConfig.Concurrency.ConversionWorkers = oldConfig.DefaultWorkers
	}
	
	newConfig.Output.KeepOriginal = !oldConfig.ReplaceOriginals
	
	if oldConfig.LogFileName != "" {
		homeDir, _ := os.UserHomeDir()
		newConfig.Logging.FilePath = filepath.Join(homeDir, ".pixly", "logs", oldConfig.LogFileName)
	}
	
	// 根据验证模式调整设置
	switch oldConfig.DefaultVerifyMode {
	case "strict":
		newConfig.Advanced.Validation.MagicByteCheck = true
		newConfig.Advanced.Validation.SizeRatioCheck = true
	case "relaxed":
		newConfig.Advanced.Validation.MagicByteCheck = false
		newConfig.Advanced.Validation.SizeRatioCheck = false
	}
	
	return newConfig, nil
}

// DetectOldConfig detects and returns paths to old configuration files
func (m *Migrator) DetectOldConfig() []string {
	var oldConfigs []string
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return oldConfigs
	}
	
	// 检查可能的旧配置位置
	possiblePaths := []string{
		filepath.Join(homeDir, ".pixly", "config.json"),
		filepath.Join(homeDir, ".easyjxlavif", "config.json"),
		"config.json",
		".pixly.json",
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			oldConfigs = append(oldConfigs, path)
		}
	}
	
	return oldConfigs
}

// PromptMigration prompts the user to migrate from old config
func (m *Migrator) PromptMigration() (bool, string) {
	oldConfigs := m.DetectOldConfig()
	
	if len(oldConfigs) == 0 {
		return false, ""
	}
	
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║   🔄 检测到旧版本配置文件                                    ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("发现以下旧配置文件：")
	
	for i, path := range oldConfigs {
		fmt.Printf("  %d. %s\n", i+1, path)
	}
	
	fmt.Println()
	fmt.Print("是否要迁移配置到 v4.0 格式？(y/n): ")
	
	var response string
	fmt.Scanln(&response)
	
	if response == "y" || response == "Y" || response == "yes" {
		return true, oldConfigs[0]
	}
	
	return false, ""
}

// MigrateAndSave performs migration and saves to new location
func (m *Migrator) MigrateAndSave(oldConfigPath string, newConfigPath string) error {
	// 迁移配置
	newConfig, err := m.MigrateFromV3(oldConfigPath)
	if err != nil {
		return fmt.Errorf("迁移失败: %w", err)
	}
	
	// 创建配置管理器并保存
	manager := NewManager()
	manager.config = newConfig
	
	if err := manager.SaveToFile(newConfigPath); err != nil {
		return fmt.Errorf("保存新配置失败: %w", err)
	}
	
	fmt.Println()
	fmt.Println("✅ 配置迁移成功！")
	fmt.Printf("   旧配置: %s\n", oldConfigPath)
	fmt.Printf("   新配置: %s\n", newConfigPath)
	fmt.Println()
	fmt.Println("💡 旧配置文件已保留，您可以手动删除")
	fmt.Println()
	
	return nil
}

// BackupOldConfig creates a backup of the old configuration
func (m *Migrator) BackupOldConfig(oldConfigPath string) error {
	backupPath := oldConfigPath + ".backup"
	
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return fmt.Errorf("读取旧配置失败: %w", err)
	}
	
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}
	
	fmt.Printf("✅ 已备份旧配置: %s\n", backupPath)
	return nil
}

// AutoMigrate attempts to automatically migrate if old config is detected
func (m *Migrator) AutoMigrate() (bool, error) {
	oldConfigs := m.DetectOldConfig()
	
	if len(oldConfigs) == 0 {
		return false, nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("获取用户目录失败: %w", err)
	}
	
	newConfigPath := filepath.Join(homeDir, ".pixly", "config.yaml")
	
	// 如果新配置已存在，不自动迁移
	if _, err := os.Stat(newConfigPath); err == nil {
		return false, nil
	}
	
	// 备份旧配置
	if err := m.BackupOldConfig(oldConfigs[0]); err != nil {
		return false, err
	}
	
	// 执行迁移
	if err := m.MigrateAndSave(oldConfigs[0], newConfigPath); err != nil {
		return false, err
	}
	
	return true, nil
}

