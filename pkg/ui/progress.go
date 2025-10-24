package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/pterm/pterm"
)

// ProgressManager 进度管理器（防刷屏优化）
type ProgressManager struct {
	config      *Config
	currentBar  *pterm.ProgressbarPrinter
	lastUpdate  time.Time
	updateMutex sync.Mutex
	minInterval time.Duration // 最小更新间隔（防刷屏）
	frozen      bool          // 是否冻结（异常情况）
	errorCount  int           // 错误计数
}

// NewProgressManager 创建进度管理器
func NewProgressManager(config *Config) *ProgressManager {
	return &ProgressManager{
		config:      config,
		minInterval: time.Duration(config.ProgressRefreshRate) * time.Millisecond,
		frozen:      false,
		errorCount:  0,
	}
}

// Start 启动进度条
func (pm *ProgressManager) Start(title string, total int) error {
	if !pm.config.ShouldShowProgress() {
		return nil // 进度条已禁用
	}

	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	// 创建进度条（稳定配置）
	pb, err := pterm.DefaultProgressbar.
		WithTotal(total).
		WithTitle(title).
		WithShowCount(true).
		WithShowPercentage(true).
		WithRemoveWhenDone(false). // 完成后不移除（避免UI跳动）
		Start()

	if err != nil {
		pm.handleError(err)
		return err
	}

	pm.currentBar = pb
	pm.lastUpdate = time.Now()
	return nil
}

// Update 更新进度（防刷屏）
func (pm *ProgressManager) Update(increment int) error {
	if !pm.config.ShouldShowProgress() || pm.currentBar == nil {
		return nil
	}

	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	// 检查是否冻结
	if pm.frozen {
		return fmt.Errorf("进度条已冻结（检测到异常）")
	}

	// 防刷屏：检查更新间隔
	now := time.Now()
	if now.Sub(pm.lastUpdate) < pm.minInterval {
		// 太频繁，跳过本次更新（但累积）
		return nil
	}

	// 更新进度
	pm.currentBar.Add(increment)

	pm.lastUpdate = now
	return nil
}

// UpdateWithMessage 更新进度并显示消息
func (pm *ProgressManager) UpdateWithMessage(increment int, message string) error {
	if !pm.config.ShouldShowProgress() || pm.currentBar == nil {
		// 非交互模式，直接打印消息
		if pm.config.DebugMode {
			fmt.Printf("[%d] %s\n", increment, message)
		}
		return nil
	}

	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	// 防刷屏检查
	now := time.Now()
	if now.Sub(pm.lastUpdate) < pm.minInterval {
		return nil
	}

	// 更新标题（显示当前处理的文件）
	pm.currentBar.Title = message

	// 更新进度
	pm.currentBar.Add(increment)

	pm.lastUpdate = now
	return nil
}

// Stop 停止进度条
func (pm *ProgressManager) Stop() error {
	if !pm.config.ShouldShowProgress() || pm.currentBar == nil {
		return nil
	}

	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	pm.currentBar.Stop()
	pm.currentBar = nil
	return nil
}

// Freeze 冻结进度条（异常情况）
func (pm *ProgressManager) Freeze(reason string) {
	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	pm.frozen = true

	if pm.currentBar != nil {
		pm.currentBar.Stop()
		pterm.Warning.Printfln("⚠️  进度条已冻结: %s", reason)
	}
}

// handleError 处理错误（检测UI异常）
func (pm *ProgressManager) handleError(err error) {
	pm.errorCount++

	if pm.errorCount > 5 {
		// 检测到频繁错误，冻结进度条
		pm.Freeze("检测到频繁UI错误")
	}

	if pm.config.DebugMode {
		pterm.Debug.Printfln("进度条错误 (%d): %v", pm.errorCount, err)
	}
}

// SafeProgressBar 安全进度条（包装器，防止panic）
type SafeProgressBar struct {
	manager *ProgressManager
	total   int
	current int
}

// NewSafeProgressBar 创建安全进度条
func NewSafeProgressBar(manager *ProgressManager, title string, total int) (*SafeProgressBar, error) {
	err := manager.Start(title, total)
	if err != nil {
		return nil, err
	}

	return &SafeProgressBar{
		manager: manager,
		total:   total,
		current: 0,
	}, nil
}

// Increment 安全递增（自动防刷屏）
func (spb *SafeProgressBar) Increment() {
	defer func() {
		if r := recover(); r != nil {
			// 捕获panic，防止UI崩溃
			spb.manager.Freeze(fmt.Sprintf("进度条panic: %v", r))
		}
	}()

	spb.current++
	spb.manager.Update(1)
}

// SetMessage 设置当前消息
func (spb *SafeProgressBar) SetMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			spb.manager.Freeze(fmt.Sprintf("消息更新panic: %v", r))
		}
	}()

	spb.manager.UpdateWithMessage(0, msg)
}

// Finish 完成进度条
func (spb *SafeProgressBar) Finish() {
	defer func() {
		if r := recover(); r != nil {
			// 即使panic也要尝试清理
			fmt.Println("进度条清理异常（已恢复）")
		}
	}()

	spb.manager.Stop()
}
