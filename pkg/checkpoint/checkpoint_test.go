package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBoltStore(t *testing.T) {
	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	store, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("创建BoltStore失败: %v", err)
	}
	defer store.Close()
	
	// 测试保存和获取会话
	session := &SessionInfo{
		SessionID:  "test_session",
		TargetDir:  "/tmp/test",
		OutputDir:  "/tmp/output",
		Mode:       "auto",
		InPlace:    false,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		Status:     SessionRunning,
		TotalFiles: 100,
	}
	
	if err := store.SaveSession(session); err != nil {
		t.Errorf("保存会话失败: %v", err)
	}
	
	retrieved, err := store.GetSession("test_session")
	if err != nil {
		t.Errorf("获取会话失败: %v", err)
	}
	
	if retrieved.SessionID != session.SessionID {
		t.Errorf("会话ID不匹配: got %s, want %s", 
			retrieved.SessionID, session.SessionID)
	}
}

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	manager, err := NewManager(dbPath, 10)
	if err != nil {
		t.Fatalf("创建Manager失败: %v", err)
	}
	defer manager.Close()
	
	// 测试创建会话
	err = manager.CreateSession("test_001", "/tmp/test", "/tmp/output", "auto", false)
	if err != nil {
		t.Errorf("创建会话失败: %v", err)
	}
	
	// 测试设置总文件数
	err = manager.SetTotalFiles(50)
	if err != nil {
		t.Errorf("设置总文件数失败: %v", err)
	}
	
	// 测试记录文件完成
	err = manager.RecordFileComplete(
		"/tmp/test/file1.jpg",
		"/tmp/output/file1.jxl",
		1024000,
		512000,
		"jxl",
		"jpg",
		"jxl",
	)
	if err != nil {
		t.Errorf("记录文件完成失败: %v", err)
	}
	
	// 测试获取当前会话
	session := manager.GetCurrentSession()
	if session == nil {
		t.Error("获取当前会话返回nil")
	}
	
	if session.Completed != 1 {
		t.Errorf("完成计数错误: got %d, want 1", session.Completed)
	}
}

func TestSessionManager(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	sm, err := NewSessionManager(dbPath)
	if err != nil {
		t.Fatalf("创建SessionManager失败: %v", err)
	}
	defer sm.Close()
	
	// 创建测试会话
	manager := sm.GetManager()
	manager.CreateSession("session1", "/tmp/dir1", "/tmp/out1", "auto", false)
	manager.CompleteSession()
	
	manager.CreateSession("session2", "/tmp/dir2", "/tmp/out2", "auto", false)
	// session2保持运行中
	
	// 测试列出未完成会话
	incomplete, err := sm.ListIncompleteSessions()
	if err != nil {
		t.Errorf("列出未完成会话失败: %v", err)
	}
	
	if len(incomplete) != 1 {
		t.Errorf("未完成会话数量错误: got %d, want 1", len(incomplete))
	}
	
	// 测试列出完成会话
	completed, err := sm.ListCompletedSessions()
	if err != nil {
		t.Errorf("列出完成会话失败: %v", err)
	}
	
	if len(completed) != 1 {
		t.Errorf("完成会话数量错误: got %d, want 1", len(completed))
	}
}

func TestFileRecord(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	store, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("创建BoltStore失败: %v", err)
	}
	defer store.Close()
	
	// 保存文件记录
	record := &FileRecord{
		FilePath:     "/tmp/test.jpg",
		RelativePath: "test.jpg",
		Status:       StatusCompleted,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		OriginalSize: 1024,
		NewSize:      512,
		SpaceSaved:   512,
		Method:       "jxl",
		Format:       "jpg",
		TargetFormat: "jxl",
	}
	
	err = store.SaveFileRecord("test_session", record)
	if err != nil {
		t.Errorf("保存文件记录失败: %v", err)
	}
	
	// 获取文件记录
	retrieved, err := store.GetFileRecord("test_session", "/tmp/test.jpg")
	if err != nil {
		t.Errorf("获取文件记录失败: %v", err)
	}
	
	if retrieved.Status != StatusCompleted {
		t.Errorf("状态不匹配: got %s, want %s", 
			retrieved.Status, StatusCompleted)
	}
	
	// 列出所有文件记录
	records, err := store.ListFileRecords("test_session")
	if err != nil {
		t.Errorf("列出文件记录失败: %v", err)
	}
	
	if len(records) != 1 {
		t.Errorf("文件记录数量错误: got %d, want 1", len(records))
	}
}

func TestCleanupOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	sm, err := NewSessionManager(dbPath)
	if err != nil {
		t.Fatalf("创建SessionManager失败: %v", err)
	}
	defer sm.Close()
	
	manager := sm.GetManager()
	
	// 创建一个旧会话（模拟）
	manager.CreateSession("old_session", "/tmp/old", "/tmp/out", "auto", false)
	session := manager.GetCurrentSession()
	session.Status = SessionCompleted
	session.EndTime = time.Now().Add(-8 * 24 * time.Hour) // 8天前
	manager.Save()
	
	// 创建一个新会话
	manager.CreateSession("new_session", "/tmp/new", "/tmp/out", "auto", false)
	manager.CompleteSession()
	
	// 清理7天前的会话
	deleted, err := sm.CleanupOldSessions(7 * 24 * time.Hour)
	if err != nil {
		t.Errorf("清理旧会话失败: %v", err)
	}
	
	if deleted != 1 {
		t.Errorf("删除会话数量错误: got %d, want 1", deleted)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
