package checkpoint

import (
	"encoding/json"
	"fmt"
	"time"
	
	bolt "go.etcd.io/bbolt"
)

var (
	sessionsBucket = []byte("sessions")
	filesBucket    = []byte("files")
	statsBucket    = []byte("statistics")
)

// BoltStore manages BoltDB operations
type BoltStore struct {
	db   *bolt.DB
	path string
}

// NewBoltStore creates a new BoltDB store
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("打开BoltDB失败: %w", err)
	}
	
	// 创建buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(sessionsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(filesBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(statsBucket); err != nil {
			return err
		}
		return nil
	})
	
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化BoltDB失败: %w", err)
	}
	
	return &BoltStore{
		db:   db,
		path: path,
	}, nil
}

// Close closes the BoltDB connection
func (bs *BoltStore) Close() error {
	if bs.db != nil {
		return bs.db.Close()
	}
	return nil
}

// SaveSession saves session information
func (bs *BoltStore) SaveSession(session *SessionInfo) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化会话失败: %w", err)
	}
	
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionsBucket)
		return b.Put([]byte(session.SessionID), data)
	})
}

// GetSession retrieves session information
func (bs *BoltStore) GetSession(sessionID string) (*SessionInfo, error) {
	var session SessionInfo
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionsBucket)
		data := b.Get([]byte(sessionID))
		if data == nil {
			return fmt.Errorf("会话不存在: %s", sessionID)
		}
		return json.Unmarshal(data, &session)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &session, nil
}

// ListSessions lists all sessions
func (bs *BoltStore) ListSessions() ([]*SessionInfo, error) {
	var sessions []*SessionInfo
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionsBucket)
		return b.ForEach(func(k, v []byte) error {
			var session SessionInfo
			if err := json.Unmarshal(v, &session); err != nil {
				return err
			}
			sessions = append(sessions, &session)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return sessions, nil
}

// DeleteSession deletes a session
func (bs *BoltStore) DeleteSession(sessionID string) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(sessionsBucket)
		return b.Delete([]byte(sessionID))
	})
}

// SaveFileRecord saves a file processing record
func (bs *BoltStore) SaveFileRecord(sessionID string, record *FileRecord) error {
	key := fmt.Sprintf("%s:%s", sessionID, record.FilePath)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("序列化文件记录失败: %w", err)
	}
	
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		return b.Put([]byte(key), data)
	})
}

// GetFileRecord retrieves a file record
func (bs *BoltStore) GetFileRecord(sessionID, filePath string) (*FileRecord, error) {
	key := fmt.Sprintf("%s:%s", sessionID, filePath)
	var record FileRecord
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("文件记录不存在")
		}
		return json.Unmarshal(data, &record)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &record, nil
}

// ListFileRecords lists all file records for a session
func (bs *BoltStore) ListFileRecords(sessionID string) ([]*FileRecord, error) {
	var records []*FileRecord
	prefix := []byte(sessionID + ":")
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		c := b.Cursor()
		
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var record FileRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			records = append(records, &record)
		}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return records, nil
}

// DeleteFileRecords deletes all file records for a session
func (bs *BoltStore) DeleteFileRecords(sessionID string) error {
	prefix := []byte(sessionID + ":")
	
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		c := b.Cursor()
		
		for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveStatistics saves session statistics
func (bs *BoltStore) SaveStatistics(stats *Statistics) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化统计信息失败: %w", err)
	}
	
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(statsBucket)
		return b.Put([]byte(stats.SessionID), data)
	})
}

// GetStatistics retrieves session statistics
func (bs *BoltStore) GetStatistics(sessionID string) (*Statistics, error) {
	var stats Statistics
	
	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(statsBucket)
		data := b.Get([]byte(sessionID))
		if data == nil {
			return fmt.Errorf("统计信息不存在")
		}
		return json.Unmarshal(data, &stats)
	})
	
	if err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// DeleteStatistics deletes session statistics
func (bs *BoltStore) DeleteStatistics(sessionID string) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(statsBucket)
		return b.Delete([]byte(sessionID))
	})
}
