package checkpoint

import (
	"fmt"
	"log"
	"time"
)

// ExampleBasicUsage demonstrates basic checkpoint usage
func ExampleBasicUsage() {
	// 创建管理器
	manager, err := NewManager("pixly_sessions.db", 10)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()
	
	// 创建新会话
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	err = manager.CreateSession(sessionID, "/path/to/images", "/path/to/output", "auto", false)
	if err != nil {
		log.Fatal(err)
	}
	
	// 设置总文件数
	manager.SetTotalFiles(100)
	
	// 处理文件
	for i := 0; i < 100; i++ {
		filePath := fmt.Sprintf("/path/to/images/img_%d.jpg", i)
		
		// 记录开始
		manager.RecordFileStart(filePath)
		
		// 模拟处理...
		// convertFile(filePath)
		
		// 记录完成
		manager.RecordFileComplete(
			filePath,
			fmt.Sprintf("/path/to/output/img_%d.jxl", i),
			1024000,  // 原始大小
			512000,   // 新大小
			"jxl",
			"jpg",
			"jxl",
		)
	}
	
	// 完成会话
	manager.CompleteSession()
}

// ExampleResumeSession demonstrates resuming a crashed session
func ExampleResumeSession() {
	manager, err := NewManager("pixly_sessions.db", 10)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()
	
	// 加载会话
	err = manager.LoadSession("session_123456789")
	if err != nil {
		log.Fatal(err)
	}
	
	// 获取已处理的文件
	processed, err := manager.GetProcessedFiles("session_123456789")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("已处理 %d 个文件\n", len(processed))
	
	// 继续处理剩余文件...
}

// ExampleSessionManagement demonstrates session management
func ExampleSessionManagement() {
	sm, err := NewSessionManager("pixly_sessions.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sm.Close()
	
	// 查找未完成的会话
	incomplete, err := sm.ListIncompleteSessions()
	if err != nil {
		log.Fatal(err)
	}
	
	for _, session := range incomplete {
		summary := sm.GetSessionSummary(session)
		fmt.Println(summary)
		fmt.Println("---")
	}
	
	// 清理7天前的完成会话
	deleted, err := sm.CleanupOldSessions(7 * 24 * time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("已清理 %d 个旧会话\n", deleted)
}
