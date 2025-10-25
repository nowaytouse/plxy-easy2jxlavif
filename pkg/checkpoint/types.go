package checkpoint

import "time"

// FileStatus represents the processing status of a file
type FileStatus string

const (
	StatusPending    FileStatus = "pending"
	StatusProcessing FileStatus = "processing"
	StatusCompleted  FileStatus = "completed"
	StatusFailed     FileStatus = "failed"
	StatusSkipped    FileStatus = "skipped"
)

// SessionStatus represents the status of a conversion session
type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionPaused    SessionStatus = "paused"
	SessionCompleted SessionStatus = "completed"
	SessionCrashed   SessionStatus = "crashed"
	SessionCancelled SessionStatus = "cancelled"
)

// SessionInfo represents a conversion session
type SessionInfo struct {
	SessionID   string        `json:"session_id"`
	TargetDir   string        `json:"target_dir"`
	OutputDir   string        `json:"output_dir"`
	Mode        string        `json:"mode"`
	InPlace     bool          `json:"in_place"`
	StartTime   time.Time     `json:"start_time"`
	LastUpdate  time.Time     `json:"last_update"`
	EndTime     time.Time     `json:"end_time,omitempty"`
	TotalFiles  int           `json:"total_files"`
	Processed   int           `json:"processed"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	Status      SessionStatus `json:"status"`
	
	// 统计信息
	TotalBytesBefore int64         `json:"total_bytes_before"`
	TotalBytesAfter  int64         `json:"total_bytes_after"`
	TotalDuration    time.Duration `json:"total_duration"`
}

// FileRecord represents a file processing record
type FileRecord struct {
	FilePath      string     `json:"file_path"`
	RelativePath  string     `json:"relative_path"`
	Status        FileStatus `json:"status"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       time.Time  `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	OutputPath    string     `json:"output_path,omitempty"`
	OriginalSize  int64      `json:"original_size"`
	NewSize       int64      `json:"new_size"`
	SpaceSaved    int64      `json:"space_saved"`
	Method        string     `json:"method"`
	Quality       string     `json:"quality,omitempty"`
	Format        string     `json:"format"`
	TargetFormat  string     `json:"target_format"`
	RetryCount    int        `json:"retry_count"`
}

// Statistics represents session statistics
type Statistics struct {
	SessionID        string    `json:"session_id"`
	TotalFiles       int       `json:"total_files"`
	ProcessedFiles   int       `json:"processed_files"`
	CompletedFiles   int       `json:"completed_files"`
	FailedFiles      int       `json:"failed_files"`
	SkippedFiles     int       `json:"skipped_files"`
	TotalSizeBefore  int64     `json:"total_size_before"`
	TotalSizeAfter   int64     `json:"total_size_after"`
	TotalSpaceSaved  int64     `json:"total_space_saved"`
	SavingPercent    float64   `json:"saving_percent"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	UpdatedAt        time.Time `json:"updated_at"`
}
