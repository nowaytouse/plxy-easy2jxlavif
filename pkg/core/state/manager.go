package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"pixly/pkg/core/config"
	"pixly/pkg/core/types"

	"go.etcd.io/bbolt"
)

const (
	// Bucket names
	MediaFilesBucket = "media_files"
	ResultsBucket    = "results"
	MetadataBucket   = "metadata"
	StatsBucket      = "stats"

	// Keys
	SessionKey       = "current_session"
	ProcessingDirKey = "processing_dir"
	LastUpdateKey    = "last_update"
)

// StateManager 状态管理器
type StateManager struct {
	db       *bbolt.DB
	dbPath   string
	session  string
	readonly bool
}

// NewStateManager 创建新的状态管理器
func NewStateManager(readonly bool) (*StateManager, error) {
	dbPath, err := config.GetStateDBPath()
	if err != nil {
		return nil, fmt.Errorf("获取数据库路径失败: %w", err)
	}

	// 如果只读模式，检查文件是否存在
	if readonly {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("状态数据库不存在: %s", dbPath)
		}
	}

	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{
		Timeout:  time.Second * 5,
		ReadOnly: readonly,
	})
	if err != nil {
		return nil, fmt.Errorf("打开状态数据库失败: %w", err)
	}

	sm := &StateManager{
		db:       db,
		dbPath:   dbPath,
		session:  fmt.Sprintf("session_%d", time.Now().Unix()),
		readonly: readonly,
	}

	// 初始化buckets（只在写模式下）
	if !readonly {
		if err := sm.initBuckets(); err != nil {
			db.Close()
			return nil, fmt.Errorf("初始化数据库buckets失败: %w", err)
		}
	}

	return sm, nil
}

// Close 关闭状态管理器
func (sm *StateManager) Close() error {
	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

// initBuckets 初始化数据库buckets
func (sm *StateManager) initBuckets() error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{
			MediaFilesBucket,
			ResultsBucket,
			MetadataBucket,
			StatsBucket,
		}

		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("创建bucket %s 失败: %w", bucket, err)
			}
		}

		return nil
	})
}

// SaveSession 保存会话信息
func (sm *StateManager) SaveSession(processingDir string) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存会话")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MetadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket 不存在")
		}

		metadata := map[string]interface{}{
			"session_id":     sm.session,
			"processing_dir": processingDir,
			"created_at":     time.Now(),
			"last_update":    time.Now(),
		}

		data, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("序列化会话数据失败: %w", err)
		}

		if err := bucket.Put([]byte(SessionKey), data); err != nil {
			return fmt.Errorf("保存会话数据失败: %w", err)
		}

		return bucket.Put([]byte(ProcessingDirKey), []byte(processingDir))
	})
}

// LoadSession 加载会话信息
func (sm *StateManager) LoadSession() (string, error) {
	var processingDir string

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MetadataBucket))
		if bucket == nil {
			return fmt.Errorf("metadata bucket 不存在")
		}

		data := bucket.Get([]byte(ProcessingDirKey))
		if data != nil {
			processingDir = string(data)
		}

		return nil
	})

	return processingDir, err
}

// SaveMediaFiles 保存媒体文件信息
func (sm *StateManager) SaveMediaFiles(files []*types.MediaInfo) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MediaFilesBucket))
		if bucket == nil {
			return fmt.Errorf("media_files bucket 不存在")
		}

		// 清空现有数据
		if err := tx.DeleteBucket([]byte(MediaFilesBucket)); err != nil {
			return fmt.Errorf("清空media_files bucket失败: %w", err)
		}

		bucket, err := tx.CreateBucket([]byte(MediaFilesBucket))
		if err != nil {
			return fmt.Errorf("重新创建media_files bucket失败: %w", err)
		}

		// 保存所有文件，包含新的元数据字段
		for _, file := range files {
			data, err := json.Marshal(file)
			if err != nil {
				return fmt.Errorf("序列化媒体文件失败: %w", err)
			}

			key := []byte(file.Path)
			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("保存媒体文件失败: %w", err)
			}
		}

		return sm.updateLastModified(tx)
	})
}

// LoadMediaFiles 加载媒体文件信息
func (sm *StateManager) LoadMediaFiles() ([]*types.MediaInfo, error) {
	var files []*types.MediaInfo

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MediaFilesBucket))
		if bucket == nil {
			return nil // 没有数据，返回空切片
		}

		return bucket.ForEach(func(k, v []byte) error {
			var file types.MediaInfo
			if err := json.Unmarshal(v, &file); err != nil {
				return fmt.Errorf("反序列化媒体文件失败: %w", err)
			}
			files = append(files, &file)
			return nil
		})
	})

	return files, err
}

// UpdateMediaFileStatus 更新媒体文件状态
func (sm *StateManager) UpdateMediaFileStatus(filePath string, status types.ProcessingStatus) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法更新数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MediaFilesBucket))
		if bucket == nil {
			return fmt.Errorf("media_files bucket 不存在")
		}

		key := []byte(filePath)
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("文件不存在: %s", filePath)
		}

		var file types.MediaInfo
		if err := json.Unmarshal(data, &file); err != nil {
			return fmt.Errorf("反序列化媒体文件失败: %w", err)
		}

		file.Status = status
		file.LastProcessed = time.Now()

		newData, err := json.Marshal(&file)
		if err != nil {
			return fmt.Errorf("序列化媒体文件失败: %w", err)
		}

		if err := bucket.Put(key, newData); err != nil {
			return fmt.Errorf("更新媒体文件失败: %w", err)
		}

		return sm.updateLastModified(tx)
	})
}

// SaveResults 保存处理结果
func (sm *StateManager) SaveResults(results []*types.ProcessingResult) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ResultsBucket))
		if bucket == nil {
			return fmt.Errorf("results bucket 不存在")
		}

		for i, result := range results {
			data, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("序列化处理结果失败: %w", err)
			}

			key := []byte(fmt.Sprintf("result_%d", i))
			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("保存处理结果失败: %w", err)
			}
		}

		return sm.updateLastModified(tx)
	})
}

// LoadResults 加载处理结果
func (sm *StateManager) LoadResults() ([]*types.ProcessingResult, error) {
	var results []*types.ProcessingResult

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ResultsBucket))
		if bucket == nil {
			return nil // 没有数据，返回空切片
		}

		return bucket.ForEach(func(k, v []byte) error {
			var result types.ProcessingResult
			if err := json.Unmarshal(v, &result); err != nil {
				return fmt.Errorf("反序列化处理结果失败: %w", err)
			}
			results = append(results, &result)
			return nil
		})
	})

	return results, err
}

// SaveTasks 保存转换任务
func (sm *StateManager) SaveTasks(tasks []*types.FileTask) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("conversion_tasks"))
		if bucket == nil {
			var err error
			bucket, err = tx.CreateBucket([]byte("conversion_tasks"))
			if err != nil {
				return fmt.Errorf("创建conversion_tasks bucket失败: %w", err)
			}
		}

		for _, task := range tasks {
			data, err := json.Marshal(task)
			if err != nil {
				return fmt.Errorf("序列化转换任务失败: %w", err)
			}

			key := []byte(task.Path)
			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("保存转换任务失败: %w", err)
			}
		}

		return sm.updateLastModified(tx)
	})
}

// LoadTasks 加载转换任务
func (sm *StateManager) LoadTasks() ([]*types.FileTask, error) {
	var tasks []*types.FileTask

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("conversion_tasks"))
		if bucket == nil {
			return nil // 没有数据，返回空切片
		}

		return bucket.ForEach(func(k, v []byte) error {
			var task types.FileTask
			if err := json.Unmarshal(v, &task); err != nil {
				return fmt.Errorf("反序列化转换任务失败: %w", err)
			}
			tasks = append(tasks, &task)
			return nil
		})
	})

	return tasks, err
}

// SaveStatistics 保存统计信息
func (sm *StateManager) SaveStatistics(stats *types.Statistics) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(StatsBucket))
		if bucket == nil {
			return fmt.Errorf("stats bucket 不存在")
		}

		data, err := json.Marshal(stats)
		if err != nil {
			return fmt.Errorf("序列化统计信息失败: %w", err)
		}

		if err := bucket.Put([]byte("current_stats"), data); err != nil {
			return fmt.Errorf("保存统计信息失败: %w", err)
		}

		return sm.updateLastModified(tx)
	})
}

// LoadStatistics 加载统计信息
func (sm *StateManager) LoadStatistics() (*types.Statistics, error) {
	var stats *types.Statistics

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(StatsBucket))
		if bucket == nil {
			return nil // 没有数据，返回nil
		}

		data := bucket.Get([]byte("current_stats"))
		if data == nil {
			return nil // 没有数据，返回nil
		}

		stats = &types.Statistics{}
		return json.Unmarshal(data, stats)
	})

	return stats, err
}

// GetPendingFiles 获取待处理的文件
func (sm *StateManager) GetPendingFiles() ([]*types.MediaInfo, error) {
	files, err := sm.LoadMediaFiles()
	if err != nil {
		return nil, err
	}

	var pending []*types.MediaInfo
	for _, file := range files {
		if file.Status == types.StatusPending || file.Status == types.StatusScanning {
			pending = append(pending, file)
		}
	}

	return pending, nil
}

// HasIncompleteSession 检查是否有未完成的会话
func (sm *StateManager) HasIncompleteSession(processingDir string) (bool, error) {
	savedDir, err := sm.LoadSession()
	if err != nil {
		return false, err
	}

	if savedDir == "" || savedDir != processingDir {
		return false, nil
	}

	pending, err := sm.GetPendingFiles()
	if err != nil {
		return false, err
	}

	return len(pending) > 0, nil
}

// ClearSession 清空会话数据
func (sm *StateManager) ClearSession() error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法清空数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{
			MediaFilesBucket,
			ResultsBucket,
			MetadataBucket,
			StatsBucket,
		}

		for _, bucketName := range buckets {
			if err := tx.DeleteBucket([]byte(bucketName)); err != nil {
				// 忽略bucket不存在的错误
				continue
			}

			if _, err := tx.CreateBucket([]byte(bucketName)); err != nil {
				return fmt.Errorf("重新创建bucket %s 失败: %w", bucketName, err)
			}
		}

		return nil
	})
}

// updateLastModified 更新最后修改时间
func (sm *StateManager) updateLastModified(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(MetadataBucket))
	if bucket == nil {
		return fmt.Errorf("metadata bucket 不存在")
	}

	timestamp := time.Now().Format(time.RFC3339)
	return bucket.Put([]byte(LastUpdateKey), []byte(timestamp))
}

// GetDBInfo 获取数据库信息
func (sm *StateManager) GetDBInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	err := sm.db.View(func(tx *bbolt.Tx) error {
		// 获取文件大小
		if stat, err := os.Stat(sm.dbPath); err == nil {
			info["size_bytes"] = stat.Size()
			info["mod_time"] = stat.ModTime()
		}

		// 获取bucket统计
		buckets := []string{MediaFilesBucket, ResultsBucket, MetadataBucket, StatsBucket}
		for _, bucketName := range buckets {
			bucket := tx.Bucket([]byte(bucketName))
			if bucket != nil {
				stats := bucket.Stats()
				info[bucketName+"_count"] = stats.KeyN
			}
		}

		return nil
	})

	return info, err
}

// LoadState 向后兼容函数 - 创建新的状态管理器
// 这个函数用于替换原有的LoadState函数调用
func LoadState(targetDir string) (*StateManager, error) {
	// 初次尝试以只读模式打开，如果数据库不存在则创建新的
	sm, err := NewStateManager(true)
	if err != nil {
		// 如果只读模式失败，尝试创建新的状态管理器
		return NewStateManager(false)
	}
	return sm, nil
}

// SaveICCProfile 保存ICC配置文件
func (sm *StateManager) SaveICCProfile(filePath string, profile []byte) error {
	if sm.readonly {
		return fmt.Errorf("只读模式下无法保存数据")
	}

	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MediaFilesBucket))
		if bucket == nil {
			return fmt.Errorf("media_files bucket 不存在")
		}

		// 使用文件路径作为键，保存ICC配置
		key := []byte(filePath)
		if err := bucket.Put(key, profile); err != nil {
			return fmt.Errorf("保存ICC配置失败: %w", err)
		}

		return sm.updateLastModified(tx)
	})
}

// LoadICCProfile 加载ICC配置文件
func (sm *StateManager) LoadICCProfile(filePath string) ([]byte, error) {
	var profile []byte

	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(MediaFilesBucket))
		if bucket == nil {
			return fmt.Errorf("media_files bucket 不存在")
		}

		profile = bucket.Get([]byte(filePath))
		if profile == nil {
			return fmt.Errorf("ICC配置不存在")
		}

		return nil
	})

	return profile, err
}
