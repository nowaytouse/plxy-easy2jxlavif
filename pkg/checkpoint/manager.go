package checkpoint

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// Manager manages checkpoint operations
type Manager struct {
	store          *BoltStore
	currentSession *SessionInfo
	stats          *Statistics
	mu             sync.RWMutex
	saveCounter    int
	saveInterval   int // 每处理N个文件保存一次
}

// NewManager creates a new checkpoint manager
func NewManager(dbPath string, saveInterval int) (*Manager, error) {
	store, err := NewBoltStore(dbPath)
	if err != nil {
		return nil, err
	}
	
	return &Manager{
		store:        store,
		saveInterval: saveInterval,
	}, nil
}

// Close closes the manager
func (m *Manager) Close() error {
	return m.store.Close()
}

// CreateSession creates a new session
func (m *Manager) CreateSession(sessionID, targetDir, outputDir, mode string, inPlace bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session := &SessionInfo{
		SessionID:  sessionID,
		TargetDir:  targetDir,
		OutputDir:  outputDir,
		Mode:       mode,
		InPlace:    inPlace,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		Status:     SessionRunning,
	}
	
	if err := m.store.SaveSession(session); err != nil {
		return err
	}
	
	m.currentSession = session
	m.stats = &Statistics{
		SessionID: sessionID,
		UpdatedAt: time.Now(),
	}
	
	return nil
}

// LoadSession loads an existing session
func (m *Manager) LoadSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, err := m.store.GetSession(sessionID)
	if err != nil {
		return err
	}
	
	stats, err := m.store.GetStatistics(sessionID)
	if err != nil {
		// 如果没有统计信息，创建新的
		stats = &Statistics{
			SessionID: sessionID,
			UpdatedAt: time.Now(),
		}
	}
	
	m.currentSession = session
	m.stats = stats
	
	// 更新会话状态为运行中
	m.currentSession.Status = SessionRunning
	m.currentSession.LastUpdate = time.Now()
	
	return m.store.SaveSession(m.currentSession)
}

// RecordFileStart records file processing start
func (m *Manager) RecordFileStart(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	record := &FileRecord{
		FilePath:     filePath,
		RelativePath: m.getRelativePath(filePath),
		Status:       StatusProcessing,
		StartTime:    time.Now(),
	}
	
	return m.store.SaveFileRecord(m.currentSession.SessionID, record)
}

// RecordFileComplete records file processing completion
func (m *Manager) RecordFileComplete(filePath, outputPath string, originalSize, newSize int64, method, format, targetFormat string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	record := &FileRecord{
		FilePath:      filePath,
		RelativePath:  m.getRelativePath(filePath),
		Status:        StatusCompleted,
		StartTime:     time.Now(), // 这里应该从之前的记录获取
		EndTime:       time.Now(),
		OutputPath:    outputPath,
		OriginalSize:  originalSize,
		NewSize:       newSize,
		SpaceSaved:    originalSize - newSize,
		Method:        method,
		Format:        format,
		TargetFormat:  targetFormat,
	}
	
	record.Duration = record.EndTime.Sub(record.StartTime)
	
	if err := m.store.SaveFileRecord(m.currentSession.SessionID, record); err != nil {
		return err
	}
	
	// 更新统计信息
	m.updateStats(record)
	
	// 更新会话信息
	m.currentSession.Processed++
	m.currentSession.Completed++
	m.currentSession.TotalBytesBefore += originalSize
	m.currentSession.TotalBytesAfter += newSize
	m.currentSession.LastUpdate = time.Now()
	
	// 每N个文件保存一次
	m.saveCounter++
	if m.saveCounter >= m.saveInterval {
		if err := m.Save(); err != nil {
			return err
		}
		m.saveCounter = 0
	}
	
	return nil
}

// RecordFileFailed records file processing failure
func (m *Manager) RecordFileFailed(filePath, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	record := &FileRecord{
		FilePath:     filePath,
		RelativePath: m.getRelativePath(filePath),
		Status:       StatusFailed,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		ErrorMessage: errorMsg,
	}
	
	if err := m.store.SaveFileRecord(m.currentSession.SessionID, record); err != nil {
		return err
	}
	
	m.currentSession.Processed++
	m.currentSession.Failed++
	m.currentSession.LastUpdate = time.Now()
	m.stats.FailedFiles++
	
	return nil
}

// RecordFileSkipped records file being skipped
func (m *Manager) RecordFileSkipped(filePath, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	record := &FileRecord{
		FilePath:     filePath,
		RelativePath: m.getRelativePath(filePath),
		Status:       StatusSkipped,
		ErrorMessage: reason,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
	}
	
	if err := m.store.SaveFileRecord(m.currentSession.SessionID, record); err != nil {
		return err
	}
	
	m.currentSession.Skipped++
	m.currentSession.LastUpdate = time.Now()
	m.stats.SkippedFiles++
	
	return nil
}

// SetTotalFiles sets the total number of files to process
func (m *Manager) SetTotalFiles(total int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	m.currentSession.TotalFiles = total
	m.stats.TotalFiles = total
	
	return m.store.SaveSession(m.currentSession)
}

// Save saves current session and statistics
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	if err := m.store.SaveSession(m.currentSession); err != nil {
		return err
	}
	
	if err := m.store.SaveStatistics(m.stats); err != nil {
		return err
	}
	
	return nil
}

// CompleteSession marks session as completed
func (m *Manager) CompleteSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}
	
	m.currentSession.Status = SessionCompleted
	m.currentSession.EndTime = time.Now()
	m.currentSession.TotalDuration = m.currentSession.EndTime.Sub(m.currentSession.StartTime)
	
	return m.Save()
}

// GetProcessedFiles returns list of processed files
func (m *Manager) GetProcessedFiles(sessionID string) ([]string, error) {
	records, err := m.store.ListFileRecords(sessionID)
	if err != nil {
		return nil, err
	}
	
	var processed []string
	for _, record := range records {
		if record.Status == StatusCompleted || record.Status == StatusFailed || record.Status == StatusSkipped {
			processed = append(processed, record.FilePath)
		}
	}
	
	return processed, nil
}

// ListSessions lists all sessions
func (m *Manager) ListSessions() ([]*SessionInfo, error) {
	return m.store.ListSessions()
}

// DeleteSession deletes a session and all its records
func (m *Manager) DeleteSession(sessionID string) error {
	if err := m.store.DeleteSession(sessionID); err != nil {
		return err
	}
	
	if err := m.store.DeleteFileRecords(sessionID); err != nil {
		return err
	}
	
	if err := m.store.DeleteStatistics(sessionID); err != nil {
		return err
	}
	
	return nil
}

// GetCurrentSession returns the current session
func (m *Manager) GetCurrentSession() *SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSession
}

// GetCurrentStats returns current statistics
func (m *Manager) GetCurrentStats() *Statistics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// updateStats updates statistics
func (m *Manager) updateStats(record *FileRecord) {
	m.stats.ProcessedFiles++
	m.stats.CompletedFiles++
	m.stats.TotalSizeBefore += record.OriginalSize
	m.stats.TotalSizeAfter += record.NewSize
	m.stats.TotalSpaceSaved += record.SpaceSaved
	
	if m.stats.TotalSizeBefore > 0 {
		m.stats.SavingPercent = float64(m.stats.TotalSpaceSaved) / float64(m.stats.TotalSizeBefore) * 100
	}
	
	m.stats.UpdatedAt = time.Now()
}

// getRelativePath gets relative path from target directory
func (m *Manager) getRelativePath(filePath string) string {
	if m.currentSession == nil {
		return filePath
	}
	
	rel, err := filepath.Rel(m.currentSession.TargetDir, filePath)
	if err != nil {
		return filePath
	}
	
	return rel
}
