package config

import "time"

// Config represents the complete Pixly v4.0 configuration
type Config struct {
	Project      ProjectConfig      `yaml:"project" mapstructure:"project"`
	Concurrency  ConcurrencyConfig  `yaml:"concurrency" mapstructure:"concurrency"`
	Conversion   ConversionConfig   `yaml:"conversion" mapstructure:"conversion"`
	Output       OutputConfig       `yaml:"output" mapstructure:"output"`
	Security     SecurityConfig     `yaml:"security" mapstructure:"security"`
	ProblemFiles ProblemFilesConfig `yaml:"problem_files" mapstructure:"problem_files"`
	Resume       ResumeConfig       `yaml:"resume" mapstructure:"resume"`
	UI           UIConfig           `yaml:"ui" mapstructure:"ui"`
	Logging      LoggingConfig      `yaml:"logging" mapstructure:"logging"`
	Tools        ToolsConfig        `yaml:"tools" mapstructure:"tools"`
	Knowledge    KnowledgeConfig    `yaml:"knowledge_base" mapstructure:"knowledge_base"`
	Advanced     AdvancedConfig     `yaml:"advanced" mapstructure:"advanced"`
	Language     LanguageConfig     `yaml:"language" mapstructure:"language"`
	Update       UpdateConfig       `yaml:"update" mapstructure:"update"`
}

// ProjectConfig contains project metadata
type ProjectConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Version string `yaml:"version" mapstructure:"version"`
	Author  string `yaml:"author" mapstructure:"author"`
}

// ConcurrencyConfig controls parallel processing
type ConcurrencyConfig struct {
	AutoAdjust        bool `yaml:"auto_adjust" mapstructure:"auto_adjust"`
	ConversionWorkers int  `yaml:"conversion_workers" mapstructure:"conversion_workers"`
	ScanWorkers       int  `yaml:"scan_workers" mapstructure:"scan_workers"`
	MemoryLimitMB     int  `yaml:"memory_limit_mb" mapstructure:"memory_limit_mb"`
	EnableMonitoring  bool `yaml:"enable_monitoring" mapstructure:"enable_monitoring"`
}

// ConversionConfig controls conversion behavior
type ConversionConfig struct {
	DefaultMode       string                  `yaml:"default_mode" mapstructure:"default_mode"`
	Predictor         PredictorConfig         `yaml:"predictor" mapstructure:"predictor"`
	Formats           FormatsConfig           `yaml:"formats" mapstructure:"formats"`
	QualityThresholds QualityThresholdsConfig `yaml:"quality_thresholds" mapstructure:"quality_thresholds"`
	SupportedFormats  SupportedFormatsConfig  `yaml:"supported_extensions" mapstructure:"supported_extensions"`
	ExcludedFormats   []string                `yaml:"excluded_extensions" mapstructure:"excluded_extensions"`
}

// PredictorConfig controls the prediction engine
type PredictorConfig struct {
	EnableKnowledgeBase   bool    `yaml:"enable_knowledge_base" mapstructure:"enable_knowledge_base"`
	ConfidenceThreshold   float64 `yaml:"confidence_threshold" mapstructure:"confidence_threshold"`
	EnableExploration     bool    `yaml:"enable_exploration" mapstructure:"enable_exploration"`
	ExplorationCandidates int     `yaml:"exploration_candidates" mapstructure:"exploration_candidates"`
}

// FormatsConfig contains format-specific settings
type FormatsConfig struct {
	PNG   PNGFormatConfig   `yaml:"png" mapstructure:"png"`
	JPEG  JPEGFormatConfig  `yaml:"jpeg" mapstructure:"jpeg"`
	GIF   GIFFormatConfig   `yaml:"gif" mapstructure:"gif"`
	WebP  WebPFormatConfig  `yaml:"webp" mapstructure:"webp"`
	Video VideoFormatConfig `yaml:"video" mapstructure:"video"`
}

// PNGFormatConfig for PNG conversion
type PNGFormatConfig struct {
	Target          string `yaml:"target" mapstructure:"target"`
	Lossless        bool   `yaml:"lossless" mapstructure:"lossless"`
	Distance        int    `yaml:"distance" mapstructure:"distance"`
	Effort          int    `yaml:"effort" mapstructure:"effort"`
	EffortLargeFile int    `yaml:"effort_large_file" mapstructure:"effort_large_file"`
	EffortSmallFile int    `yaml:"effort_small_file" mapstructure:"effort_small_file"`
	LargeFileSizeMB int    `yaml:"large_file_size_mb" mapstructure:"large_file_size_mb"`
	SmallFileSizeKB int    `yaml:"small_file_size_kb" mapstructure:"small_file_size_kb"`
}

// JPEGFormatConfig for JPEG conversion
type JPEGFormatConfig struct {
	Target       string `yaml:"target" mapstructure:"target"`
	LosslessJPEG bool   `yaml:"lossless_jpeg" mapstructure:"lossless_jpeg"`
	Effort       int    `yaml:"effort" mapstructure:"effort"`
}

// GIFFormatConfig for GIF conversion
type GIFFormatConfig struct {
	StaticTarget   string `yaml:"static_target" mapstructure:"static_target"`
	AnimatedTarget string `yaml:"animated_target" mapstructure:"animated_target"`
	StaticDistance int    `yaml:"static_distance" mapstructure:"static_distance"`
	AnimatedCRF    int    `yaml:"animated_crf" mapstructure:"animated_crf"`
	AnimatedSpeed  int    `yaml:"animated_speed" mapstructure:"animated_speed"`
}

// WebPFormatConfig for WebP conversion
type WebPFormatConfig struct {
	StaticTarget   string `yaml:"static_target" mapstructure:"static_target"`
	AnimatedTarget string `yaml:"animated_target" mapstructure:"animated_target"`
}

// VideoFormatConfig for video processing
type VideoFormatConfig struct {
	Target         string `yaml:"target" mapstructure:"target"`
	RepackageOnly  bool   `yaml:"repackage_only" mapstructure:"repackage_only"`
	EnableReencode bool   `yaml:"enable_reencode" mapstructure:"enable_reencode"`
	CRF            int    `yaml:"crf" mapstructure:"crf"`
}

// QualityThresholdsConfig defines quality classification thresholds
type QualityThresholdsConfig struct {
	Enable    bool              `yaml:"enable" mapstructure:"enable"`
	Image     QualityThresholds `yaml:"image" mapstructure:"image"`
	Photo     QualityThresholds `yaml:"photo" mapstructure:"photo"`
	Animation QualityThresholds `yaml:"animation" mapstructure:"animation"`
	Video     QualityThresholds `yaml:"video" mapstructure:"video"`
}

// QualityThresholds for a specific media type
type QualityThresholds struct {
	HighQuality   float64 `yaml:"high_quality" mapstructure:"high_quality"`
	MediumQuality float64 `yaml:"medium_quality" mapstructure:"medium_quality"`
	LowQuality    float64 `yaml:"low_quality" mapstructure:"low_quality"`
}

// SupportedFormatsConfig defines supported file extensions
type SupportedFormatsConfig struct {
	Image []string `yaml:"image" mapstructure:"image"`
	Video []string `yaml:"video" mapstructure:"video"`
}

// OutputConfig controls output behavior
type OutputConfig struct {
	KeepOriginal              bool   `yaml:"keep_original" mapstructure:"keep_original"`
	GenerateReport            bool   `yaml:"generate_report" mapstructure:"generate_report"`
	GeneratePerformanceReport bool   `yaml:"generate_performance_report" mapstructure:"generate_performance_report"`
	ReportFormat              string `yaml:"report_format" mapstructure:"report_format"`
	FilenameTemplate          string `yaml:"filename_template" mapstructure:"filename_template"`
	DirectoryTemplate         string `yaml:"directory_template" mapstructure:"directory_template"`
}

// SecurityConfig controls security settings
type SecurityConfig struct {
	EnablePathCheck      bool     `yaml:"enable_path_check" mapstructure:"enable_path_check"`
	ForbiddenDirectories []string `yaml:"forbidden_directories" mapstructure:"forbidden_directories"`
	AllowedDirectories   []string `yaml:"allowed_directories" mapstructure:"allowed_directories"`
	CheckDiskSpace       bool     `yaml:"check_disk_space" mapstructure:"check_disk_space"`
	MinFreeSpaceMB       int64    `yaml:"min_free_space_mb" mapstructure:"min_free_space_mb"`
	MaxFileSizeMB        int64    `yaml:"max_file_size_mb" mapstructure:"max_file_size_mb"`
	EnableBackup         bool     `yaml:"enable_backup" mapstructure:"enable_backup"`
}

// ProblemFilesConfig controls how to handle problem files
type ProblemFilesConfig struct {
	CorruptedStrategy             string   `yaml:"corrupted_strategy" mapstructure:"corrupted_strategy"`
	CodecIncompatibleStrategy     string   `yaml:"codec_incompatible_strategy" mapstructure:"codec_incompatible_strategy"`
	ContainerIncompatibleStrategy string   `yaml:"container_incompatible_strategy" mapstructure:"container_incompatible_strategy"`
	TrashStrategy                 string   `yaml:"trash_strategy" mapstructure:"trash_strategy"`
	TrashExtensions               []string `yaml:"trash_extensions" mapstructure:"trash_extensions"`
	TrashKeywords                 []string `yaml:"trash_keywords" mapstructure:"trash_keywords"`
}

// ResumeConfig controls checkpoint/resume behavior
type ResumeConfig struct {
	Enable            bool `yaml:"enable" mapstructure:"enable"`
	SaveInterval      int  `yaml:"save_interval" mapstructure:"save_interval"`
	AutoResumeOnCrash bool `yaml:"auto_resume_on_crash" mapstructure:"auto_resume_on_crash"`
	PromptUser        bool `yaml:"prompt_user" mapstructure:"prompt_user"`
}

// UIConfig controls user interface
type UIConfig struct {
	Mode               string             `yaml:"mode" mapstructure:"mode"`
	Theme              string             `yaml:"theme" mapstructure:"theme"`
	EnableEmoji        bool               `yaml:"enable_emoji" mapstructure:"enable_emoji"`
	EnableASCIIArt     bool               `yaml:"enable_ascii_art" mapstructure:"enable_ascii_art"`
	EnableAnimations   bool               `yaml:"enable_animations" mapstructure:"enable_animations"`
	AnimationIntensity string             `yaml:"animation_intensity" mapstructure:"animation_intensity"`
	Colors             UIColorsConfig     `yaml:"colors" mapstructure:"colors"`
	Progress           ProgressUIConfig   `yaml:"progress" mapstructure:"progress"`
	MonitorPanel       MonitorPanelConfig `yaml:"monitor_panel" mapstructure:"monitor_panel"`
}

// UIColorsConfig defines UI color scheme
type UIColorsConfig struct {
	Primary   string `yaml:"primary" mapstructure:"primary"`
	Secondary string `yaml:"secondary" mapstructure:"secondary"`
	Success   string `yaml:"success" mapstructure:"success"`
	Warning   string `yaml:"warning" mapstructure:"warning"`
	Error     string `yaml:"error" mapstructure:"error"`
	Info      string `yaml:"info" mapstructure:"info"`
}

// ProgressUIConfig for progress display
type ProgressUIConfig struct {
	RefreshIntervalMS int  `yaml:"refresh_interval_ms" mapstructure:"refresh_interval_ms"`
	AntiFlicker       bool `yaml:"anti_flicker" mapstructure:"anti_flicker"`
	ShowFileIcons     bool `yaml:"show_file_icons" mapstructure:"show_file_icons"`
	ShowETA           bool `yaml:"show_eta" mapstructure:"show_eta"`
}

// MonitorPanelConfig for monitoring panel
type MonitorPanelConfig struct {
	Enable           bool   `yaml:"enable" mapstructure:"enable"`
	Position         string `yaml:"position" mapstructure:"position"`
	RefreshIntervalS int    `yaml:"refresh_interval_s" mapstructure:"refresh_interval_s"`
	ShowCharts       bool   `yaml:"show_charts" mapstructure:"show_charts"`
}

// LoggingConfig controls logging behavior
type LoggingConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`
	Output     string `yaml:"output" mapstructure:"output"`
	FilePath   string `yaml:"file_path" mapstructure:"file_path"`
	MaxSizeMB  int    `yaml:"max_size_mb" mapstructure:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days" mapstructure:"max_age_days"`
	Compress   bool   `yaml:"compress" mapstructure:"compress"`
}

// ToolsConfig controls tool paths
type ToolsConfig struct {
	AutoDetect   bool   `yaml:"auto_detect" mapstructure:"auto_detect"`
	CJXLPath     string `yaml:"cjxl_path" mapstructure:"cjxl_path"`
	DJXLPath     string `yaml:"djxl_path" mapstructure:"djxl_path"`
	AVIFEncPath  string `yaml:"avifenc_path" mapstructure:"avifenc_path"`
	AVIFDecPath  string `yaml:"avifdec_path" mapstructure:"avifdec_path"`
	FFmpegPath   string `yaml:"ffmpeg_path" mapstructure:"ffmpeg_path"`
	FFprobePath  string `yaml:"ffprobe_path" mapstructure:"ffprobe_path"`
	ExifToolPath string `yaml:"exiftool_path" mapstructure:"exiftool_path"`
}

// KnowledgeConfig controls knowledge base
type KnowledgeConfig struct {
	Enable        bool           `yaml:"enable" mapstructure:"enable"`
	DBPath        string         `yaml:"db_path" mapstructure:"db_path"`
	AutoLearn     bool           `yaml:"auto_learn" mapstructure:"auto_learn"`
	MinConfidence float64        `yaml:"min_confidence" mapstructure:"min_confidence"`
	Analysis      AnalysisConfig `yaml:"analysis" mapstructure:"analysis"`
}

// AnalysisConfig for knowledge base analysis
type AnalysisConfig struct {
	Enable          bool `yaml:"enable" mapstructure:"enable"`
	ReportInterval  int  `yaml:"report_interval" mapstructure:"report_interval"`
	ShowSuggestions bool `yaml:"show_suggestions" mapstructure:"show_suggestions"`
}

// AdvancedConfig for advanced settings
type AdvancedConfig struct {
	EnableExperimental bool             `yaml:"enable_experimental" mapstructure:"enable_experimental"`
	EnableDebug        bool             `yaml:"enable_debug" mapstructure:"enable_debug"`
	MemoryPool         MemoryPoolConfig `yaml:"memory_pool" mapstructure:"memory_pool"`
	Validation         ValidationConfig `yaml:"validation" mapstructure:"validation"`
}

// MemoryPoolConfig for memory optimization
type MemoryPoolConfig struct {
	Enable       bool `yaml:"enable" mapstructure:"enable"`
	BufferSizeMB int  `yaml:"buffer_size_mb" mapstructure:"buffer_size_mb"`
}

// ValidationConfig for post-processing validation
type ValidationConfig struct {
	EnablePixelCheck bool    `yaml:"enable_pixel_check" mapstructure:"enable_pixel_check"`
	EnableHashCheck  bool    `yaml:"enable_hash_check" mapstructure:"enable_hash_check"`
	MagicByteCheck   bool    `yaml:"magic_byte_check" mapstructure:"magic_byte_check"`
	SizeRatioCheck   bool    `yaml:"size_ratio_check" mapstructure:"size_ratio_check"`
	MaxSizeRatio     float64 `yaml:"max_size_ratio" mapstructure:"max_size_ratio"`
}

// LanguageConfig for internationalization
type LanguageConfig struct {
	Default    string `yaml:"default" mapstructure:"default"`
	AutoDetect bool   `yaml:"auto_detect" mapstructure:"auto_detect"`
}

// UpdateConfig for auto-update checking
type UpdateConfig struct {
	AutoCheck         bool      `yaml:"auto_check" mapstructure:"auto_check"`
	CheckIntervalDays int       `yaml:"check_interval_days" mapstructure:"check_interval_days"`
	NotifyOnUpdate    bool      `yaml:"notify_on_update" mapstructure:"notify_on_update"`
	LastCheckTime     time.Time `yaml:"last_check_time,omitempty" mapstructure:"last_check_time"`
}
