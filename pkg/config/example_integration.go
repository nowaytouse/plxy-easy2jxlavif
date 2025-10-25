package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ExampleUsage demonstrates how to integrate the config system with cobra/main
// This is an example file showing the integration pattern
func ExampleUsage() {
	// 示例：在主程序中使用配置系统
	
	// 1. 创建配置管理器
	configManager := NewManager()
	
	// 2. 尝试加载配置（自动处理迁移）
	if err := loadConfigWithMigration(configManager); err != nil {
		fmt.Printf("❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	
	// 3. 获取配置
	config := configManager.GetConfig()
	
	// 4. 使用配置
	fmt.Printf("使用配置: %s 模式\n", config.Conversion.DefaultMode)
	fmt.Printf("Worker数量: %d\n", config.Concurrency.ConversionWorkers)
	
	// 5. 显示配置摘要
	configManager.DisplaySummary()
}

// loadConfigWithMigration loads config and handles migration if needed
func loadConfigWithMigration(manager *Manager) error {
	// 尝试加载配置
	if err := manager.Load(); err != nil {
		// 如果加载失败，检查是否需要迁移
		migrator := NewMigrator()
		
		// 检测旧配置
		shouldMigrate, oldConfigPath := migrator.PromptMigration()
		
		if shouldMigrate {
			homeDir, _ := os.UserHomeDir()
			newConfigPath := homeDir + "/.pixly/config.yaml"
			
			if err := migrator.MigrateAndSave(oldConfigPath, newConfigPath); err != nil {
				return fmt.Errorf("迁移配置失败: %w", err)
			}
			
			// 重新加载配置
			return manager.LoadFromFile(newConfigPath)
		}
		
		// 如果不迁移，创建默认配置
		if _, err := manager.CreateDefaultIfNotExists(); err != nil {
			return fmt.Errorf("创建默认配置失败: %w", err)
		}
		
		return manager.Load()
	}
	
	return nil
}

// AddConfigFlags adds configuration flags to cobra command
func AddConfigFlags(cmd *cobra.Command, overrides *map[string]interface{}) {
	// 添加常用配置参数到命令行
	cmd.Flags().IntP("workers", "w", 0, "转换worker数量（0=自动）")
	cmd.Flags().StringP("mode", "m", "", "转换模式：auto+, auto, smart, batch")
	cmd.Flags().StringP("config", "c", "", "配置文件路径")
	cmd.Flags().String("ui-mode", "", "UI模式：interactive, non-interactive, silent")
	cmd.Flags().String("log-level", "", "日志级别：debug, info, warn, error")
	cmd.Flags().Bool("enable-monitoring", true, "启用性能监控")
	cmd.Flags().Bool("keep-original", false, "保留原文件")
	cmd.Flags().Int("png-effort", 0, "PNG转换effort（1-9）")
	cmd.Flags().Int("jpeg-effort", 0, "JPEG转换effort（1-9）")
	cmd.Flags().Bool("no-animations", false, "禁用UI动画")
	cmd.Flags().String("theme", "", "UI主题：dark, light, auto")
	
	// 解析标志并存储到overrides
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if *overrides == nil {
			*overrides = make(map[string]interface{})
		}
		
		if workers, _ := cmd.Flags().GetInt("workers"); workers > 0 {
			(*overrides)["workers"] = workers
		}
		
		if mode, _ := cmd.Flags().GetString("mode"); mode != "" {
			(*overrides)["mode"] = mode
		}
		
		if uiMode, _ := cmd.Flags().GetString("ui-mode"); uiMode != "" {
			(*overrides)["ui-mode"] = uiMode
		}
		
		if logLevel, _ := cmd.Flags().GetString("log-level"); logLevel != "" {
			(*overrides)["log-level"] = logLevel
		}
		
		if enableMonitoring, _ := cmd.Flags().GetBool("enable-monitoring"); cmd.Flags().Changed("enable-monitoring") {
			(*overrides)["enable-monitoring"] = enableMonitoring
		}
		
		if keepOriginal, _ := cmd.Flags().GetBool("keep-original"); cmd.Flags().Changed("keep-original") {
			(*overrides)["keep-original"] = keepOriginal
		}
	}
}

// ExampleCobraIntegration shows how to use config in a cobra command
func ExampleCobraIntegration() *cobra.Command {
	var configPath string
	var overrides = make(map[string]interface{})
	
	cmd := &cobra.Command{
		Use:   "convert [path]",
		Short: "转换媒体文件",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. 创建配置管理器
			manager := NewManager()
			
			// 2. 加载配置
			if configPath != "" {
				if err := manager.LoadFromFile(configPath); err != nil {
					fmt.Printf("❌ 加载配置失败: %v\n", err)
					return
				}
			} else {
				if err := loadConfigWithMigration(manager); err != nil {
					fmt.Printf("❌ 配置初始化失败: %v\n", err)
					return
				}
			}
			
			// 3. 合并命令行参数
			manager.MergeFromCommandLine(overrides)
			
			// 4. 获取最终配置
			config := manager.GetConfig()
			
			// 5. 使用配置执行转换
			fmt.Printf("🚀 开始转换，使用配置: %s\n", manager.GetConfigPath())
			fmt.Printf("   模式: %s\n", config.Conversion.DefaultMode)
			fmt.Printf("   Worker: %d\n", config.Concurrency.ConversionWorkers)
			
			// 这里继续执行实际的转换逻辑...
		},
	}
	
	// 添加配置标志
	AddConfigFlags(cmd, &overrides)
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "配置文件路径")
	
	return cmd
}

// ExampleConfigCommand creates a config management subcommand
func ExampleConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置管理",
	}
	
	// config show - 显示当前配置
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		Run: func(cmd *cobra.Command, args []string) {
			manager := NewManager()
			if err := manager.Load(); err != nil {
				fmt.Printf("❌ 加载配置失败: %v\n", err)
				return
			}
			manager.DisplaySummary()
		},
	}
	
	// config init - 初始化配置
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "初始化默认配置",
		Run: func(cmd *cobra.Command, args []string) {
			manager := NewManager()
			configPath, err := manager.CreateDefaultIfNotExists()
			if err != nil {
				fmt.Printf("❌ 创建配置失败: %v\n", err)
				return
			}
			fmt.Printf("✅ 配置已创建: %s\n", configPath)
			fmt.Println("\n💡 您可以编辑此文件来自定义Pixly的行为")
		},
	}
	
	// config validate - 验证配置
	validateCmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "验证配置文件",
		Run: func(cmd *cobra.Command, args []string) {
			manager := NewManager()
			
			if len(args) > 0 {
				if err := manager.LoadFromFile(args[0]); err != nil {
					fmt.Printf("❌ 验证失败: %v\n", err)
					return
				}
			} else {
				if err := manager.Load(); err != nil {
					fmt.Printf("❌ 验证失败: %v\n", err)
					return
				}
			}
			
			fmt.Println("✅ 配置验证通过！")
		},
	}
	
	// config migrate - 迁移旧配置
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "从v3.1.1迁移配置",
		Run: func(cmd *cobra.Command, args []string) {
			migrator := NewMigrator()
			
			migrated, err := migrator.AutoMigrate()
			if err != nil {
				fmt.Printf("❌ 迁移失败: %v\n", err)
				return
			}
			
			if migrated {
				fmt.Println("✅ 配置迁移成功！")
			} else {
				fmt.Println("ℹ️  未检测到需要迁移的旧配置")
			}
		},
	}
	
	cmd.AddCommand(showCmd, initCmd, validateCmd, migrateCmd)
	return cmd
}

