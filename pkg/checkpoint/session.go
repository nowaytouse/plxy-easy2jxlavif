package checkpoint

import (
	"fmt"
	"sort"
	"time"
)

// SessionManager provides high-level session management operations
type SessionManager struct {
	manager *Manager
}

// NewSessionManager creates a new session manager
func NewSessionManager(dbPath string) (*SessionManager, error) {
	manager, err := NewManager(dbPath, 10) // 每10个文件保存一次
	if err != nil {
		return nil, err
	}
	
	return &SessionManager{
		manager: manager,
	}, nil
}

// Close closes the session manager
func (sm *SessionManager) Close() error {
	return sm.manager.Close()
}

// FindIncompleteSession finds an incomplete session that can be resumed
func (sm *SessionManager) FindIncompleteSession(targetDir string) (*SessionInfo, error) {
	sessions, err := sm.manager.ListSessions()
	if err != nil {
		return nil, err
	}
	
	for _, session := range sessions {
		if session.TargetDir == targetDir && 
		   (session.Status == SessionRunning || session.Status == SessionPaused || session.Status == SessionCrashed) {
			return session, nil
		}
	}
	
	return nil, nil
}

// ListIncompleteSessions lists all incomplete sessions
func (sm *SessionManager) ListIncompleteSessions() ([]*SessionInfo, error) {
	sessions, err := sm.manager.ListSessions()
	if err != nil {
		return nil, err
	}
	
	var incomplete []*SessionInfo
	for _, session := range sessions {
		if session.Status != SessionCompleted && session.Status != SessionCancelled {
			incomplete = append(incomplete, session)
		}
	}
	
	// 按最后更新时间排序
	sort.Slice(incomplete, func(i, j int) bool {
		return incomplete[i].LastUpdate.After(incomplete[j].LastUpdate)
	})
	
	return incomplete, nil
}

// ListCompletedSessions lists all completed sessions
func (sm *SessionManager) ListCompletedSessions() ([]*SessionInfo, error) {
	sessions, err := sm.manager.ListSessions()
	if err != nil {
		return nil, err
	}
	
	var completed []*SessionInfo
	for _, session := range sessions {
		if session.Status == SessionCompleted {
			completed = append(completed, session)
		}
	}
	
	// 按结束时间排序
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].EndTime.After(completed[j].EndTime)
	})
	
	return completed, nil
}

// CleanupOldSessions deletes completed sessions older than specified duration
func (sm *SessionManager) CleanupOldSessions(olderThan time.Duration) (int, error) {
	sessions, err := sm.manager.ListSessions()
	if err != nil {
		return 0, err
	}
	
	cutoff := time.Now().Add(-olderThan)
	deleted := 0
	
	for _, session := range sessions {
		if session.Status == SessionCompleted && session.EndTime.Before(cutoff) {
			if err := sm.manager.DeleteSession(session.SessionID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	
	return deleted, nil
}

// GetSessionProgress calculates session progress percentage
func (sm *SessionManager) GetSessionProgress(session *SessionInfo) float64 {
	if session.TotalFiles == 0 {
		return 0
	}
	return float64(session.Processed) / float64(session.TotalFiles) * 100
}

// GetSessionSummary generates a summary of session
func (sm *SessionManager) GetSessionSummary(session *SessionInfo) string {
	progress := sm.GetSessionProgress(session)
	
	var status string
	switch session.Status {
	case SessionRunning:
		status = "运行中"
	case SessionPaused:
		status = "已暂停"
	case SessionCompleted:
		status = "已完成"
	case SessionCrashed:
		status = "异常中断"
	case SessionCancelled:
		status = "已取消"
	default:
		status = "未知"
	}
	
	summary := fmt.Sprintf("会话: %s\n", session.SessionID)
	summary += fmt.Sprintf("状态: %s\n", status)
	summary += fmt.Sprintf("目录: %s\n", session.TargetDir)
	summary += fmt.Sprintf("进度: %.1f%% (%d/%d)\n", progress, session.Processed, session.TotalFiles)
	summary += fmt.Sprintf("成功: %d, 失败: %d, 跳过: %d\n", session.Completed, session.Failed, session.Skipped)
	summary += fmt.Sprintf("开始: %s\n", session.StartTime.Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("最后更新: %s\n", session.LastUpdate.Format("2006-01-02 15:04:05"))
	
	if session.Status == SessionCompleted {
		summary += fmt.Sprintf("结束: %s\n", session.EndTime.Format("2006-01-02 15:04:05"))
		summary += fmt.Sprintf("总耗时: %v\n", session.TotalDuration)
	}
	
	if session.TotalBytesBefore > 0 {
		saved := session.TotalBytesBefore - session.TotalBytesAfter
		percent := float64(saved) / float64(session.TotalBytesBefore) * 100
		summary += fmt.Sprintf("空间节省: %.2f%% (%.2f MB → %.2f MB)\n",
			percent,
			float64(session.TotalBytesBefore)/(1024*1024),
			float64(session.TotalBytesAfter)/(1024*1024))
	}
	
	return summary
}

// GetManager returns the underlying manager
func (sm *SessionManager) GetManager() *Manager {
	return sm.manager
}
