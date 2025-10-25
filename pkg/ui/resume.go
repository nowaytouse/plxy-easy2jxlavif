package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
)

// ResumePoint 断点信息
type ResumePoint struct {
	InputDir       string    `json:"input_dir"`
	OutputDir      string    `json:"output_dir"`
	InPlace        bool      `json:"in_place"`
	TotalFiles     int       `json:"total_files"`
	ProcessedFiles []string  `json:"processed_files"`
	ProcessedCount int       `json:"processed_count"`
	SuccessCount   int       `json:"success_count"`
	FailCount      int       `json:"fail_count"`
	SkipCount      int       `json:"skip_count"`
	LastFile       string    `json:"last_file"`
	Timestamp      time.Time `json:"timestamp"`
}

// ResumeManager 断点续传管理器
type ResumeManager struct {
	resumeFile string
}

// NewResumeManager 创建断点管理器
func NewResumeManager() *ResumeManager {
	homeDir, _ := os.UserHomeDir()
	resumeFile := filepath.Join(homeDir, ".pixly", "resume.json")

	// 确保目录存在
	os.MkdirAll(filepath.Dir(resumeFile), 0755)

	return &ResumeManager{
		resumeFile: resumeFile,
	}
}

// SaveResumePoint 保存断点
func (rm *ResumeManager) SaveResumePoint(point *ResumePoint) error {
	point.Timestamp = time.Now()

	data, err := json.MarshalIndent(point, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rm.resumeFile, data, 0644)
}

// LoadResumePoint 加载断点
func (rm *ResumeManager) LoadResumePoint() (*ResumePoint, error) {
	data, err := os.ReadFile(rm.resumeFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 没有断点
		}
		return nil, err
	}

	var point ResumePoint
	if err := json.Unmarshal(data, &point); err != nil {
		return nil, err
	}

	return &point, nil
}

// ClearResumePoint 清除断点
func (rm *ResumeManager) ClearResumePoint() error {
	if err := os.Remove(rm.resumeFile); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需清除
		}
		return err
	}
	return nil
}

// HasResumePoint 检查是否有断点
func (rm *ResumeManager) HasResumePoint() bool {
	_, err := os.Stat(rm.resumeFile)
	return err == nil
}

// ShowResumePrompt 显示断点续传提示
func (rm *ResumeManager) ShowResumePrompt(point *ResumePoint) (bool, error) {
	pterm.Println()

	// 显示断点信息框
	infoBox := pterm.DefaultBox.
		WithTitle("🔄 发现断点记录").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightYellow))

	timeSince := time.Since(point.Timestamp)
	timeStr := formatDuration(timeSince)

	message := fmt.Sprintf(`上次转换未完成

目录: %s
进度: %d/%d (%.1f%%)
成功: ✅ %d
失败: ❌ %d
跳过: ⏭️  %d
时间: %s前

是否继续上次的转换？`,
		point.InputDir,
		point.ProcessedCount,
		point.TotalFiles,
		float64(point.ProcessedCount)/float64(point.TotalFiles)*100,
		point.SuccessCount,
		point.FailCount,
		point.SkipCount,
		timeStr)

	infoBox.Println(message)
	pterm.Println()

	// 询问用户
	options := []string{
		"✅ 继续上次转换（断点续传）",
		"🆕 开始新的转换",
		"❌ 返回主菜单",
	}

	selected, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText("请选择").
		Show()

	pterm.Println()

	switch selected {
	case options[0]: // 继续
		pterm.Success.Println("✅ 将从上次中断处继续转换")
		return true, nil
	case options[1]: // 开始新的
		pterm.Info.Println("🆕 将清除断点，开始新的转换")
		rm.ClearResumePoint()
		return false, nil
	default: // 返回
		pterm.Info.Println("❌ 已取消操作")
		return false, fmt.Errorf("用户取消")
	}
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f秒", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f分钟", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1f小时", d.Hours())
	}
	return fmt.Sprintf("%.1f天", d.Hours()/24)
}

// IsProcessed 检查文件是否已处理
func (point *ResumePoint) IsProcessed(filePath string) bool {
	for _, processed := range point.ProcessedFiles {
		if processed == filePath {
			return true
		}
	}
	return false
}
