package statemanager

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// StateManager 核心状态管理器 - README要求的断点续传机制
//
// 核心功能：
//   - 基于bbolt数据库实时记录文件处理状态
//   - 记录每个文件的处理状态（哈希、大小、修改时间）
//   - 程序意外中断后可精确恢复，避免从零开始
//   - 支持批量状态查询和更新
//   - 提供状态统计和进度跟踪
//
// 设计原则：
//   - 持久化存储：使用bbolt确保状态信息持久保存
//   - 事务安全：所有状态更新都在事务中进行
//   - 高性能：支持批量操作和索引查询
//   - 一致性：确保状态与实际文件状态同步
//   - 可恢复性：程序意外中断后能精确恢复处理进度
type StateManager struct {
	logger       *zap.Logger
	db           *bbolt.DB
	dbPath       string
	buckets      map[string]string // 数据桶名称映射
	syncInterval time.Duration     // 同步间隔
	autoSync     bool              // 是否自动同步
	stats        *StateStats       // 状态统计
	sessionID    string            // 会话ID
}

// FileState 文件状态记录
type FileState struct {
	FilePath       string            `json:"file_path"`       // 文件路径
	FileHash       string            `json:"file_hash"`       // 文件SHA256哈希
	FileSize       int64             `json:"file_size"`       // 文件大小
	ModTime        time.Time         `json:"mod_time"`        // 修改时间
	Status         ProcessingStatus  `json:"status"`          // 处理状态
	ProcessingMode string            `json:"processing_mode"` // 处理模式
	StartTime      time.Time         `json:"start_time"`      // 开始处理时间
	EndTime        time.Time         `json:"end_time"`        // 结束处理时间
	Duration       time.Duration     `json:"duration"`        // 处理耗时
	ErrorMessage   string            `json:"error_message"`   // 错误信息
	Attempts       int               `json:"attempts"`        // 尝试次数
	TargetPath     string            `json:"target_path"`     // 目标文件路径
	OriginalSize   int64             `json:"original_size"`   // 原始大小
	ProcessedSize  int64             `json:"processed_size"`  // 处理后大小
	QualityLevel   string            `json:"quality_level"`   // 品质等级
	Metadata       map[string]string `json:"metadata"`        // 附加元数据
	SessionID      string            `json:"session_id"`      // 会话ID
	LastUpdate     time.Time         `json:"last_update"`     // 最后更新时间
}

// SessionState 会话状态
type SessionState struct {
	SessionID      string            `json:"session_id"`      // 会话ID
	StartTime      time.Time         `json:"start_time"`      // 开始时间
	EndTime        time.Time         `json:"end_time"`        // 结束时间
	TargetDir      string            `json:"target_dir"`      // 目标目录
	ProcessingMode string            `json:"processing_mode"` // 处理模式
	TotalFiles     int               `json:"total_files"`     // 总文件数
	ProcessedFiles int               `json:"processed_files"` // 已处理文件数
	SuccessFiles   int               `json:"success_files"`   // 成功文件数
	FailedFiles    int               `json:"failed_files"`    // 失败文件数
	SkippedFiles   int               `json:"skipped_files"`   // 跳过文件数
	Status         SessionStatus     `json:"status"`          // 会话状态
	Configuration  map[string]string `json:"configuration"`   // 配置信息
	LastUpdate     time.Time         `json:"last_update"`     // 最后更新时间
}

// StateStats 状态统计
type StateStats struct {
	TotalFiles         int                      `json:"total_files"`
	ProcessingFiles    int                      `json:"processing_files"`
	CompletedFiles     int                      `json:"completed_files"`
	FailedFiles        int                      `json:"failed_files"`
	SkippedFiles       int                      `json:"skipped_files"`
	StatusCount        map[ProcessingStatus]int `json:"status_count"`
	SessionCount       int                      `json:"session_count"`
	ActiveSessions     int                      `json:"active_sessions"`
	CompletedSessions  int                      `json:"completed_sessions"`
	AverageProcessTime time.Duration            `json:"average_process_time"`
	TotalSavedSpace    int64                    `json:"total_saved_space"`
	LastUpdate         time.Time                `json:"last_update"`
}

// RecoveryInfo 恢复信息
type RecoveryInfo struct {
	SessionsFound     []SessionState `json:"sessions_found"`
	IncompleteFiles   []FileState    `json:"incomplete_files"`
	ProcessingFiles   []FileState    `json:"processing_files"`
	LastSession       *SessionState  `json:"last_session"`
	CanResume         bool           `json:"can_resume"`
	RecommendedAction string         `json:"recommended_action"`
	RecoveryOptions   []string       `json:"recovery_options"`
}

// 枚举定义
type ProcessingStatus int
type SessionStatus int

const (
	// 处理状态
	StatusPending    ProcessingStatus = iota // 等待处理
	StatusScanning                           // 扫描中
	StatusAnalyzing                          // 分析中
	StatusProcessing                         // 处理中
	StatusCompleted                          // 已完成
	StatusFailed                             // 失败
	StatusSkipped                            // 跳过
	StatusCancelled                          // 取消
)

const (
	// 会话状态
	SessionActive    SessionStatus = iota // 活动中
	SessionCompleted                      // 已完成
	SessionCancelled                      // 已取消
	SessionFailed                         // 失败
	SessionPaused                         // 暂停
)

// 数据桶常量
const (
	BucketFileStates = "file_states"
	BucketSessions   = "sessions"
	BucketMetadata   = "metadata"
	BucketStats      = "statistics"
	BucketRecovery   = "recovery"
)

// NewStateManager 创建状态管理器
func NewStateManager(logger *zap.Logger, dbPath string) (*StateManager, error) {
	sm := &StateManager{
		logger:       logger,
		dbPath:       dbPath,
		syncInterval: 5 * time.Second, // 默认5秒同步一次
		autoSync:     true,
		stats: &StateStats{
			StatusCount: make(map[ProcessingStatus]int),
		},
		sessionID: generateSessionID(),
	}

	// 初始化数据桶映射
	sm.buckets = map[string]string{
		"files":    BucketFileStates,
		"sessions": BucketSessions,
		"metadata": BucketMetadata,
		"stats":    BucketStats,
		"recovery": BucketRecovery,
	}

	// 确保数据库目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库
	var err error
	sm.db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("打开状态数据库失败: %w", err)
	}

	// 初始化数据桶
	if err := sm.initializeBuckets(); err != nil {
		sm.db.Close()
		return nil, fmt.Errorf("初始化数据桶失败: %w", err)
	}

	// 启动自动同步
	if sm.autoSync {
		go sm.autoSyncRoutine()
	}

	sm.logger.Info("状态管理器初始化完成",
		zap.String("db_path", dbPath),
		zap.String("session_id", sm.sessionID))

	return sm, nil
}

// initializeBuckets 初始化数据桶
func (sm *StateManager) initializeBuckets() error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		for _, bucketName := range sm.buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
				return fmt.Errorf("创建数据桶 %s 失败: %w", bucketName, err)
			}
		}
		return nil
	})
}

// SaveFileState 保存文件状态 - README核心功能
func (sm *StateManager) SaveFileState(ctx context.Context, state *FileState) error {
	// 自动填充基本信息
	if state.SessionID == "" {
		state.SessionID = sm.sessionID
	}
	state.LastUpdate = time.Now()

	// 计算文件哈希和大小（如果未提供）
	if state.FileHash == "" || state.FileSize == 0 {
		if err := sm.updateFileInfo(state); err != nil {
			sm.logger.Warn("更新文件信息失败",
				zap.String("file", state.FilePath),
				zap.Error(err))
		}
	}

	// 序列化状态
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化文件状态失败: %w", err)
	}

	// 保存到数据库
	err = sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketFileStates))
		if bucket == nil {
			return fmt.Errorf("文件状态桶不存在")
		}

		key := []byte(state.FilePath)
		return bucket.Put(key, data)
	})

	if err != nil {
		return fmt.Errorf("保存文件状态失败: %w", err)
	}

	// 更新统计信息
	sm.updateStats(state)

	sm.logger.Debug("文件状态已保存",
		zap.String("file", filepath.Base(state.FilePath)),
		zap.String("status", state.Status.String()),
		zap.String("session", state.SessionID))

	return nil
}

// GetFileState 获取文件状态
func (sm *StateManager) GetFileState(filePath string) (*FileState, error) {
	var state FileState
	var found bool

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketFileStates))
		if bucket == nil {
			return fmt.Errorf("文件状态桶不存在")
		}

		data := bucket.Get([]byte(filePath))
		if data == nil {
			return nil // 文件状态不存在
		}

		found = true
		return json.Unmarshal(data, &state)
	})

	if err != nil {
		return nil, fmt.Errorf("获取文件状态失败: %w", err)
	}

	if !found {
		return nil, nil // 文件状态不存在
	}

	return &state, nil
}

// NeedsProcessing 检查文件是否需要处理 - README核心功能
func (sm *StateManager) NeedsProcessing(filePath string) (bool, error) {
	// 获取当前文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 获取存储的文件状态
	state, err := sm.GetFileState(filePath)
	if err != nil {
		return false, err
	}

	// 如果没有状态记录，需要处理
	if state == nil {
		return true, nil
	}

	// 检查文件是否已成功处理
	if state.Status == StatusCompleted {
		// 检查文件是否被修改
		if fileInfo.Size() != state.FileSize ||
			!fileInfo.ModTime().Equal(state.ModTime) {
			sm.logger.Debug("文件已修改，需要重新处理",
				zap.String("file", filepath.Base(filePath)))
			return true, nil
		}

		// 验证文件哈希（可选）
		if state.FileHash != "" {
			currentHash, err := sm.calculateFileHash(filePath)
			if err == nil && currentHash != state.FileHash {
				sm.logger.Debug("文件哈希变化，需要重新处理",
					zap.String("file", filepath.Base(filePath)))
				return true, nil
			}
		}

		// 文件未变化且已完成处理
		return false, nil
	}

	// 其他状态都需要处理
	return true, nil
}

// StartSession 开始新会话
func (sm *StateManager) StartSession(targetDir, processingMode string) error {
	session := &SessionState{
		SessionID:      sm.sessionID,
		StartTime:      time.Now(),
		TargetDir:      targetDir,
		ProcessingMode: processingMode,
		Status:         SessionActive,
		Configuration:  make(map[string]string),
		LastUpdate:     time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化会话状态失败: %w", err)
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketSessions))
		if bucket == nil {
			return fmt.Errorf("会话桶不存在")
		}

		key := []byte(sm.sessionID)
		return bucket.Put(key, data)
	})
}

// CompleteSession 完成会话
func (sm *StateManager) CompleteSession() error {
	return sm.updateSessionStatus(SessionCompleted)
}

// GetRecoveryInfo 获取恢复信息 - README核心功能
func (sm *StateManager) GetRecoveryInfo() (*RecoveryInfo, error) {
	recovery := &RecoveryInfo{
		SessionsFound:   make([]SessionState, 0),
		IncompleteFiles: make([]FileState, 0),
		ProcessingFiles: make([]FileState, 0),
		RecoveryOptions: make([]string, 0),
	}

	// 查找所有会话
	err := sm.db.View(func(tx *bbolt.Tx) error {
		// 查找会话
		sessionBucket := tx.Bucket([]byte(BucketSessions))
		if sessionBucket != nil {
			sessionBucket.ForEach(func(k, v []byte) error {
				var session SessionState
				if json.Unmarshal(v, &session) == nil {
					recovery.SessionsFound = append(recovery.SessionsFound, session)

					// 找到最新会话
					if recovery.LastSession == nil ||
						session.StartTime.After(recovery.LastSession.StartTime) {
						recovery.LastSession = &session
					}
				}
				return nil
			})
		}

		// 查找未完成文件
		fileBucket := tx.Bucket([]byte(BucketFileStates))
		if fileBucket != nil {
			fileBucket.ForEach(func(k, v []byte) error {
				var state FileState
				if json.Unmarshal(v, &state) == nil {
					switch state.Status {
					case StatusPending, StatusScanning, StatusAnalyzing:
						recovery.IncompleteFiles = append(recovery.IncompleteFiles, state)
					case StatusProcessing:
						recovery.ProcessingFiles = append(recovery.ProcessingFiles, state)
					}
				}
				return nil
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("获取恢复信息失败: %w", err)
	}

	// 分析恢复选项
	sm.analyzeRecoveryOptions(recovery)

	sm.logger.Info("恢复信息分析完成",
		zap.Int("sessions", len(recovery.SessionsFound)),
		zap.Int("incomplete_files", len(recovery.IncompleteFiles)),
		zap.Int("processing_files", len(recovery.ProcessingFiles)),
		zap.Bool("can_resume", recovery.CanResume))

	return recovery, nil
}

// BatchUpdateStates 批量更新状态
func (sm *StateManager) BatchUpdateStates(states []*FileState) error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketFileStates))
		if bucket == nil {
			return fmt.Errorf("文件状态桶不存在")
		}

		for _, state := range states {
			if state.SessionID == "" {
				state.SessionID = sm.sessionID
			}
			state.LastUpdate = time.Now()

			data, err := json.Marshal(state)
			if err != nil {
				return fmt.Errorf("序列化文件状态失败: %w", err)
			}

			key := []byte(state.FilePath)
			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("保存文件状态失败: %w", err)
			}

			// 更新统计信息
			sm.updateStats(state)
		}

		return nil
	})
}

// GetFilesByStatus 按状态查询文件
func (sm *StateManager) GetFilesByStatus(status ProcessingStatus) ([]FileState, error) {
	var files []FileState

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketFileStates))
		if bucket == nil {
			return fmt.Errorf("文件状态桶不存在")
		}

		bucket.ForEach(func(k, v []byte) error {
			var state FileState
			if json.Unmarshal(v, &state) == nil && state.Status == status {
				files = append(files, state)
			}
			return nil
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}

	return files, nil
}

// 辅助方法
func (sm *StateManager) updateFileInfo(state *FileState) error {
	fileInfo, err := os.Stat(state.FilePath)
	if err != nil {
		return err
	}

	state.FileSize = fileInfo.Size()
	state.ModTime = fileInfo.ModTime()

	// 计算文件哈希
	hash, err := sm.calculateFileHash(state.FilePath)
	if err != nil {
		return err
	}
	state.FileHash = hash

	return nil
}

func (sm *StateManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	buffer := make([]byte, 32768) // 32KB缓冲区

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err != nil {
			break
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (sm *StateManager) updateStats(state *FileState) {
	if sm.stats == nil {
		return
	}

	// 这里应该实现详细的统计更新逻辑
	// 为简化示例，只做基本统计
	sm.stats.StatusCount[state.Status]++
	sm.stats.LastUpdate = time.Now()
}

func (sm *StateManager) updateSessionStatus(status SessionStatus) error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketSessions))
		if bucket == nil {
			return fmt.Errorf("会话桶不存在")
		}

		data := bucket.Get([]byte(sm.sessionID))
		if data == nil {
			return fmt.Errorf("会话不存在")
		}

		var session SessionState
		if err := json.Unmarshal(data, &session); err != nil {
			return err
		}

		session.Status = status
		session.LastUpdate = time.Now()
		if status == SessionCompleted {
			session.EndTime = time.Now()
		}

		newData, err := json.Marshal(session)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(sm.sessionID), newData)
	})
}

func (sm *StateManager) analyzeRecoveryOptions(recovery *RecoveryInfo) {
	hasIncomplete := len(recovery.IncompleteFiles) > 0
	hasProcessing := len(recovery.ProcessingFiles) > 0
	hasActiveSessions := false

	for _, session := range recovery.SessionsFound {
		if session.Status == SessionActive {
			hasActiveSessions = true
			break
		}
	}

	if hasActiveSessions || hasIncomplete || hasProcessing {
		recovery.CanResume = true
		recovery.RecommendedAction = "resume"
		recovery.RecoveryOptions = append(recovery.RecoveryOptions, "继续上次处理")
		recovery.RecoveryOptions = append(recovery.RecoveryOptions, "重新开始")
		recovery.RecoveryOptions = append(recovery.RecoveryOptions, "清理状态后开始")
	} else {
		recovery.CanResume = false
		recovery.RecommendedAction = "start_new"
		recovery.RecoveryOptions = append(recovery.RecoveryOptions, "开始新的处理")
	}
}

func (sm *StateManager) autoSyncRoutine() {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := sm.db.Sync(); err != nil {
			sm.logger.Warn("数据库同步失败", zap.Error(err))
		}
	}
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// String 方法
func (s ProcessingStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusScanning:
		return "scanning"
	case StatusAnalyzing:
		return "analyzing"
	case StatusProcessing:
		return "processing"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (s SessionStatus) String() string {
	switch s {
	case SessionActive:
		return "active"
	case SessionCompleted:
		return "completed"
	case SessionCancelled:
		return "cancelled"
	case SessionFailed:
		return "failed"
	case SessionPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Close 关闭状态管理器
func (sm *StateManager) Close() error {
	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

// GetStats 获取状态统计
func (sm *StateManager) GetStats() *StateStats {
	return sm.stats
}

// ClearState 清理状态（慎用）
func (sm *StateManager) ClearState() error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		for _, bucketName := range sm.buckets {
			if err := tx.DeleteBucket([]byte(bucketName)); err != nil && err != bbolt.ErrBucketNotFound {
				return err
			}
			if _, err := tx.CreateBucket([]byte(bucketName)); err != nil {
				return err
			}
		}
		return nil
	})
}
