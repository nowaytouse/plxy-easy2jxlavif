package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// InteractionGuard 交互保护器 - README要求的防卡死免问询交互机制
type InteractionGuard struct {
	logger       *zap.Logger
	debugMode    bool
	userTimeout  time.Duration  // 用户场景超时时间
	debugTimeout time.Duration  // 调试场景超时时间
	maxRetries   int            // 最大重试次数
	retryCounter map[string]int // 重试计数器
	reader       *bufio.Reader
}

// GuardConfig 保护配置
type GuardConfig struct {
	UserTimeout     time.Duration // 用户场景超时（默认60秒）
	DebugTimeout    time.Duration // 调试场景超时（默认30秒）
	MaxRetries      int           // 最大重试次数（默认3次）
	EnableDebugExit bool          // 调试模式下是否强制退出
}

// InputResult 输入结果
type InputResult struct {
	Value     string
	TimedOut  bool
	Error     error
	Retries   int
	IsDefault bool // 是否使用了默认值
}

// CountdownOption 倒计时选项
type CountdownOption struct {
	Duration      time.Duration
	DefaultValue  string
	Message       string
	ShowCountdown bool
}

// NewInteractionGuard 创建交互保护器
func NewInteractionGuard(logger *zap.Logger, config *GuardConfig) *InteractionGuard {
	// 设置默认配置
	if config == nil {
		config = &GuardConfig{
			UserTimeout:     60 * time.Second, // README要求：用户60秒
			DebugTimeout:    30 * time.Second, // README要求：调试30秒
			MaxRetries:      3,                // README要求：3次重试后强制退出
			EnableDebugExit: true,
		}
	}

	// 检测调试模式
	debugMode := os.Getenv("DEBUG_MODE") == "true" || os.Getenv("PIXLY_DEBUG") == "true"

	return &InteractionGuard{
		logger:       logger,
		debugMode:    debugMode,
		userTimeout:  config.UserTimeout,
		debugTimeout: config.DebugTimeout,
		maxRetries:   config.MaxRetries,
		retryCounter: make(map[string]int),
		reader:       bufio.NewReader(os.Stdin),
	}
}

// SafeInput 安全输入 - 带超时和重试保护的输入方法
func (ig *InteractionGuard) SafeInput(prompt string, operationName string) *InputResult {
	// 检查重试次数
	if ig.retryCounter[operationName] >= ig.maxRetries {
		ig.logger.Error("操作重试次数超限，强制退出",
			zap.String("operation", operationName),
			zap.Int("retries", ig.retryCounter[operationName]),
			zap.Int("max_retries", ig.maxRetries))

		if ig.debugMode {
			fmt.Printf("\n❌ 调试模式：操作 '%s' 重试次数超限，程序强制退出\n", operationName)
			os.Exit(1)
		} else {
			return &InputResult{
				Value:     "",
				TimedOut:  true,
				Error:     fmt.Errorf("操作 '%s' 重试次数超限", operationName),
				Retries:   ig.retryCounter[operationName],
				IsDefault: false,
			}
		}
	}

	// 确定超时时间
	timeout := ig.userTimeout
	if ig.debugMode {
		timeout = ig.debugTimeout
	}

	ig.logger.Debug("开始安全输入",
		zap.String("operation", operationName),
		zap.Duration("timeout", timeout),
		zap.Bool("debug_mode", ig.debugMode),
		zap.Int("current_retries", ig.retryCounter[operationName]))

	return ig.inputWithTimeout(prompt, timeout, operationName)
}

// SafeInputWithCountdown 带倒计时的安全输入
func (ig *InteractionGuard) SafeInputWithCountdown(prompt string, operationName string, countdown *CountdownOption) *InputResult {
	if countdown == nil {
		return ig.SafeInput(prompt, operationName)
	}

	// 检查重试次数
	if ig.retryCounter[operationName] >= ig.maxRetries {
		ig.logger.Error("倒计时操作重试次数超限，使用默认值",
			zap.String("operation", operationName),
			zap.String("default_value", countdown.DefaultValue))

		return &InputResult{
			Value:     countdown.DefaultValue,
			TimedOut:  true,
			Error:     nil,
			Retries:   ig.retryCounter[operationName],
			IsDefault: true,
		}
	}

	ig.logger.Debug("开始倒计时输入",
		zap.String("operation", operationName),
		zap.Duration("countdown", countdown.Duration),
		zap.String("default_value", countdown.DefaultValue))

	// 显示倒计时提示
	if countdown.ShowCountdown && countdown.Message != "" {
		fmt.Printf("\n%s\n", countdown.Message)
	}

	return ig.inputWithCountdown(prompt, countdown, operationName)
}

// inputWithTimeout 带超时的输入实现
func (ig *InteractionGuard) inputWithTimeout(prompt string, timeout time.Duration, operationName string) *InputResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		input string
		err   error
	}

	resultCh := make(chan result, 1)

	// 启动输入协程
	go func() {
		fmt.Print(prompt)
		input, err := ig.reader.ReadString('\n')
		resultCh <- result{input: strings.TrimSpace(input), err: err}
	}()

	// 等待输入或超时
	select {
	case res := <-resultCh:
		// 成功获取输入，重置重试计数器
		if res.err == nil {
			ig.retryCounter[operationName] = 0
		} else {
			ig.retryCounter[operationName]++
		}

		return &InputResult{
			Value:     res.input,
			TimedOut:  false,
			Error:     res.err,
			Retries:   ig.retryCounter[operationName],
			IsDefault: false,
		}

	case <-ctx.Done():
		// 超时处理
		ig.retryCounter[operationName]++

		ig.logger.Warn("用户输入超时",
			zap.String("operation", operationName),
			zap.Duration("timeout", timeout),
			zap.Bool("debug_mode", ig.debugMode),
			zap.Int("retries", ig.retryCounter[operationName]))

		if ig.debugMode {
			fmt.Printf("\n❌ 调试模式：输入超时（%v），程序强制退出\n", timeout)
			os.Exit(1)
			return nil // 永远不会执行到这里，但需要满足编译器要求
		} else {
			fmt.Printf("\n⏰ 用户输入超时（%v），将使用默认处理\n", timeout)
			return &InputResult{
				Value:     "",
				TimedOut:  true,
				Error:     context.DeadlineExceeded,
				Retries:   ig.retryCounter[operationName],
				IsDefault: true,
			}
		}
	}
}

// inputWithCountdown 带倒计时的输入实现
func (ig *InteractionGuard) inputWithCountdown(prompt string, countdown *CountdownOption, operationName string) *InputResult {
	ctx, cancel := context.WithTimeout(context.Background(), countdown.Duration)
	defer cancel()

	type result struct {
		input string
		err   error
	}

	resultCh := make(chan result, 1)

	// 启动输入协程
	go func() {
		fmt.Print(prompt)
		input, err := ig.reader.ReadString('\n')
		resultCh <- result{input: strings.TrimSpace(input), err: err}
	}()

	// 倒计时显示协程
	if countdown.ShowCountdown {
		go ig.showCountdown(countdown.Duration)
	}

	// 等待输入或倒计时结束
	select {
	case res := <-resultCh:
		// 成功获取输入
		if res.err == nil {
			ig.retryCounter[operationName] = 0
		} else {
			ig.retryCounter[operationName]++
		}

		return &InputResult{
			Value:     res.input,
			TimedOut:  false,
			Error:     res.err,
			Retries:   ig.retryCounter[operationName],
			IsDefault: false,
		}

	case <-ctx.Done():
		// 倒计时结束，使用默认值
		ig.logger.Info("倒计时结束，使用默认选择",
			zap.String("operation", operationName),
			zap.String("default_value", countdown.DefaultValue))

		fmt.Printf("\n⏰ 倒计时结束，自动选择：%s\n", countdown.DefaultValue)

		return &InputResult{
			Value:     countdown.DefaultValue,
			TimedOut:  true,
			Error:     nil,
			Retries:   ig.retryCounter[operationName],
			IsDefault: true,
		}
	}
}

// showCountdown 显示倒计时
func (ig *InteractionGuard) showCountdown(duration time.Duration) {
	seconds := int(duration.Seconds())
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for remaining := seconds; remaining > 0; remaining-- {
		select {
		case <-ticker.C:
			fmt.Printf("\r⏰ 倒计时：%d 秒", remaining)
		}
	}
	fmt.Print("\r")
}

// ValidateInput 输入验证 - 防止死循环的输入验证
func (ig *InteractionGuard) ValidateInput(input string, validOptions []string, operationName string) bool {
	if len(validOptions) == 0 {
		return true // 无验证要求
	}

	for _, option := range validOptions {
		if strings.EqualFold(input, option) {
			return true
		}
	}

	// 验证失败，增加重试计数
	ig.retryCounter[operationName]++

	ig.logger.Warn("用户输入验证失败",
		zap.String("operation", operationName),
		zap.String("input", input),
		zap.Strings("valid_options", validOptions),
		zap.Int("retries", ig.retryCounter[operationName]))

	return false
}

// ResetRetries 重置重试计数器
func (ig *InteractionGuard) ResetRetries(operationName string) {
	delete(ig.retryCounter, operationName)
	ig.logger.Debug("重置重试计数器", zap.String("operation", operationName))
}

// GetRetryCount 获取重试次数
func (ig *InteractionGuard) GetRetryCount(operationName string) int {
	return ig.retryCounter[operationName]
}

// IsDebugMode 检查是否为调试模式
func (ig *InteractionGuard) IsDebugMode() bool {
	return ig.debugMode
}

// ForceExit 强制退出 - 用于处理无法恢复的情况
func (ig *InteractionGuard) ForceExit(reason string, operationName string) {
	ig.logger.Error("强制退出程序",
		zap.String("reason", reason),
		zap.String("operation", operationName),
		zap.Bool("debug_mode", ig.debugMode))

	fmt.Printf("\n💥 程序无法继续执行：%s\n", reason)
	fmt.Println("📊 这通常是由于多次输入错误或系统异常导致的")

	if ig.debugMode {
		fmt.Println("🔧 调试模式：程序强制退出")
	} else {
		fmt.Println("👋 感谢使用 Pixly，程序即将退出")
	}

	os.Exit(1)
}

// GetTimeoutInfo 获取超时信息
func (ig *InteractionGuard) GetTimeoutInfo() (userTimeout, debugTimeout time.Duration, maxRetries int) {
	return ig.userTimeout, ig.debugTimeout, ig.maxRetries
}

// SetTimeouts 设置超时时间
func (ig *InteractionGuard) SetTimeouts(userTimeout, debugTimeout time.Duration) {
	ig.userTimeout = userTimeout
	ig.debugTimeout = debugTimeout

	ig.logger.Info("更新超时设置",
		zap.Duration("user_timeout", userTimeout),
		zap.Duration("debug_timeout", debugTimeout))
}

// SafeChoice 安全选择 - 带验证的多选一输入
func (ig *InteractionGuard) SafeChoice(prompt string, options []string, defaultChoice string, operationName string) *InputResult {
	for {
		result := ig.SafeInput(prompt, operationName)

		// 处理超时或错误
		if result.TimedOut || result.Error != nil {
			if defaultChoice != "" {
				result.Value = defaultChoice
				result.IsDefault = true
				return result
			}
			return result
		}

		// 验证输入
		if result.Value == "" && defaultChoice != "" {
			result.Value = defaultChoice
			result.IsDefault = true
			return result
		}

		if ig.ValidateInput(result.Value, options, operationName) {
			return result
		}

		// 验证失败，检查是否超过重试次数
		if ig.retryCounter[operationName] >= ig.maxRetries {
			ig.logger.Error("选择验证重试次数超限",
				zap.String("operation", operationName),
				zap.Int("retries", ig.retryCounter[operationName]))

			if defaultChoice != "" {
				fmt.Printf("⚠️ 重试次数超限，使用默认选择：%s\n", defaultChoice)
				return &InputResult{
					Value:     defaultChoice,
					TimedOut:  false,
					Error:     nil,
					Retries:   ig.retryCounter[operationName],
					IsDefault: true,
				}
			} else {
				ig.ForceExit("选择验证重试次数超限", operationName)
				return nil // 永远不会到达这里，但需要满足编译器要求
			}
		}

		fmt.Printf("❌ 无效选择，请输入：%s\n", strings.Join(options, ", "))
	}

	// 这行代码理论上永远不会被执行到，但需要满足编译器要求
	return nil
}
