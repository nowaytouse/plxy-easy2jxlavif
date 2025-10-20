package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"pixly/pkg/core/types"
	"pixly/pkg/engine/quality"

	"go.uber.org/zap"
)

// UnifiedScanArchitecture 统一扫描架构 - README要求的一次性统一扫描→缓存分析结果→智能处理
//
// 核心改进：
//   - 避免旧版边处理边分析的重复工作
//   - 采用高并发扫描（CPU核心数 x 2）
//   - 智能缓存机制，避免重复分析
//   - 分阶段处理：扫描→分析→缓存→智能处理
//   - 大幅提升效率，减少I/O操作
//
// 架构流程：
//
//	阶段1: 文件发现 - 高并发目录遍历
//	阶段2: 形态分析 - 并行文件类型识别
//	阶段3: 品质评估 - 智能品质判断
//	阶段4: 结果缓存 - 持久化分析结果
//	阶段5: 智能处理 - 基于缓存的快速决策
type UnifiedScanArchitecture struct {
	logger               *zap.Logger
	fileScanner          *Scanner                  // 基础文件扫描器
	morphologyClassifier *FileMorphologyClassifier // 文件形态分类器
	qualityEngine        *quality.QualityEngine    // 品质判断引擎
	cacheManager         *ScanCacheManager         // 扫描缓存管理器

	// 并发控制
	maxWorkers         int           // 最大工作线程数
	fileWorkerPool     chan struct{} // 文件处理工作池
	analysisWorkerPool chan struct{} // 分析工作池

	// 性能统计
	stats       *ScanStatistics // 扫描统计信息
	enableStats bool            // 启用统计
}

// ScanCacheManager 扫描缓存管理器 - 智能缓存分析结果
type ScanCacheManager struct {
	logger         *zap.Logger
	cacheDir       string
	memoryCache    map[string]*CachedScanResult // 内存缓存
	persistentMode bool                         // 持久化模式
	maxMemoryItems int                          // 最大内存缓存项数
	mutex          sync.RWMutex                 // 读写锁
}

// CachedScanResult 缓存的扫描结果
type CachedScanResult struct {
	FilePath          string                     `json:"file_path"`
	FileHash          string                     `json:"file_hash"`          // 文件哈希
	LastModified      time.Time                  `json:"last_modified"`      // 最后修改时间
	FileSize          int64                      `json:"file_size"`          // 文件大小
	MediaInfo         *types.MediaInfo           `json:"media_info"`         // 媒体信息
	MorphologyResult  *MorphologyResult          `json:"morphology_result"`  // 形态分析结果
	QualityAssessment *quality.QualityAssessment `json:"quality_assessment"` // 品质评估结果
	CacheTime         time.Time                  `json:"cache_time"`         // 缓存时间
	AccessCount       int                        `json:"access_count"`       // 访问次数
	IsValid           bool                       `json:"is_valid"`           // 是否有效
}

// ScanStatistics 扫描统计信息
type ScanStatistics struct {
	StartTime               time.Time                  `json:"start_time"`
	EndTime                 time.Time                  `json:"end_time"`
	TotalDuration           time.Duration              `json:"total_duration"`
	TotalFiles              int                        `json:"total_files"`
	MediaFiles              int                        `json:"media_files"`
	SkippedFiles            int                        `json:"skipped_files"`
	CacheHits               int                        `json:"cache_hits"`
	CacheMisses             int                        `json:"cache_misses"`
	WorkerUtilization       float64                    `json:"worker_utilization"`
	FileTypeDistribution    map[string]int             `json:"file_type_distribution"`
	QualityDistribution     map[types.QualityLevel]int `json:"quality_distribution"`
	ProcessingTimeBreakdown map[string]time.Duration   `json:"processing_time_breakdown"`
	Errors                  []string                   `json:"errors"`
}

// UnifiedScanResult 统一扫描结果
type UnifiedScanResult struct {
	Summary         *ScanSummary               `json:"summary"`
	MediaFiles      []*types.MediaInfo         `json:"media_files"`
	Statistics      *ScanStatistics            `json:"statistics"`
	CacheStatus     *CacheStatus               `json:"cache_status"`
	Recommendations *ProcessingRecommendations `json:"recommendations"`
}

// ScanSummary 扫描摘要
type ScanSummary struct {
	TotalFiles      int   `json:"total_files"`
	MediaFiles      int   `json:"media_files"`
	StaticImages    int   `json:"static_images"`
	AnimatedImages  int   `json:"animated_images"`
	Videos          int   `json:"videos"`
	UnknownFiles    int   `json:"unknown_files"`
	CorruptedFiles  int   `json:"corrupted_files"`
	SkippedFiles    int   `json:"skipped_files"`
	TotalSize       int64 `json:"total_size"`
	AverageFileSize int64 `json:"average_file_size"`
}

// CacheStatus 缓存状态
type CacheStatus struct {
	MemoryCacheSize    int     `json:"memory_cache_size"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	CacheMissRate      float64 `json:"cache_miss_rate"`
	CacheEfficiency    float64 `json:"cache_efficiency"`
	InvalidatedEntries int     `json:"invalidated_entries"`
}

// ProcessingRecommendations 处理建议
type ProcessingRecommendations struct {
	RecommendedMode       types.AppMode `json:"recommended_mode"`
	EstimatedSavings      int64         `json:"estimated_savings"`
	ProcessingTime        time.Duration `json:"estimated_processing_time"`
	RiskAssessment        string        `json:"risk_assessment"`
	SpecialConsiderations []string      `json:"special_considerations"`
	OptimizationTips      []string      `json:"optimization_tips"`
}

// NewUnifiedScanArchitecture 创建统一扫描架构
func NewUnifiedScanArchitecture(
	logger *zap.Logger,
	fileScanner *Scanner,
	morphologyClassifier *FileMorphologyClassifier,
	qualityEngine *quality.QualityEngine,
	cacheDir string,
) *UnifiedScanArchitecture {

	// README要求：高并发扫描（CPU核心数 x 2）
	maxWorkers := runtime.NumCPU() * 2

	arch := &UnifiedScanArchitecture{
		logger:               logger,
		fileScanner:          fileScanner,
		morphologyClassifier: morphologyClassifier,
		qualityEngine:        qualityEngine,
		maxWorkers:           maxWorkers,
		fileWorkerPool:       make(chan struct{}, maxWorkers),
		analysisWorkerPool:   make(chan struct{}, maxWorkers/2), // 分析用一半线程
		enableStats:          true,
		stats: &ScanStatistics{
			FileTypeDistribution:    make(map[string]int),
			QualityDistribution:     make(map[types.QualityLevel]int),
			ProcessingTimeBreakdown: make(map[string]time.Duration),
			Errors:                  make([]string, 0),
		},
	}

	// 初始化缓存管理器
	arch.cacheManager = NewScanCacheManager(logger, cacheDir)

	logger.Info("统一扫描架构初始化完成",
		zap.Int("max_workers", maxWorkers),
		zap.String("cache_dir", cacheDir))

	return arch
}

// NewScanCacheManager 创建扫描缓存管理器
func NewScanCacheManager(logger *zap.Logger, cacheDir string) *ScanCacheManager {
	return &ScanCacheManager{
		logger:         logger,
		cacheDir:       cacheDir,
		memoryCache:    make(map[string]*CachedScanResult),
		persistentMode: true,
		maxMemoryItems: 10000, // 最大缓存1万项
	}
}

// ExecuteUnifiedScan 执行统一扫描 - README要求的核心新流程
func (usa *UnifiedScanArchitecture) ExecuteUnifiedScan(ctx context.Context, targetDir string) (*UnifiedScanResult, error) {
	usa.stats.StartTime = time.Now()
	usa.logger.Info("开始执行统一扫描", zap.String("target_dir", targetDir))

	// 阶段1: 文件发现 - 高并发目录遍历
	phaseStart := time.Now()
	files, err := usa.performFilesDiscovery(ctx, targetDir)
	if err != nil {
		return nil, fmt.Errorf("文件发现阶段失败: %w", err)
	}
	usa.stats.ProcessingTimeBreakdown["file_discovery"] = time.Since(phaseStart)
	usa.stats.TotalFiles = len(files)

	// 阶段2: 形态分析 - 并行文件类型识别
	phaseStart = time.Now()
	morphologyResults, err := usa.performMorphologyAnalysis(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("形态分析阶段失败: %w", err)
	}
	usa.stats.ProcessingTimeBreakdown["morphology_analysis"] = time.Since(phaseStart)

	// 阶段3: 品质评估 - 智能品质判断
	phaseStart = time.Now()
	qualityResults, err := usa.performQualityAssessment(ctx, files, morphologyResults)
	if err != nil {
		return nil, fmt.Errorf("品质评估阶段失败: %w", err)
	}
	usa.stats.ProcessingTimeBreakdown["quality_assessment"] = time.Since(phaseStart)

	// 阶段4: 结果缓存 - 持久化分析结果
	phaseStart = time.Now()
	if err := usa.cacheResults(files, morphologyResults, qualityResults); err != nil {
		usa.logger.Warn("结果缓存失败", zap.Error(err))
	}
	usa.stats.ProcessingTimeBreakdown["result_caching"] = time.Since(phaseStart)

	// 阶段5: 智能处理 - 基于缓存的快速决策
	phaseStart = time.Now()
	mediaFiles, err := usa.generateMediaInfoList(files, morphologyResults, qualityResults)
	if err != nil {
		return nil, fmt.Errorf("媒体信息生成失败: %w", err)
	}
	usa.stats.ProcessingTimeBreakdown["media_info_generation"] = time.Since(phaseStart)

	// 生成最终结果
	result := usa.generateUnifiedResult(mediaFiles)

	usa.stats.EndTime = time.Now()
	usa.stats.TotalDuration = usa.stats.EndTime.Sub(usa.stats.StartTime)

	usa.logger.Info("统一扫描完成",
		zap.Int("total_files", usa.stats.TotalFiles),
		zap.Int("media_files", usa.stats.MediaFiles),
		zap.Duration("total_duration", usa.stats.TotalDuration),
		zap.Float64("cache_hit_rate", float64(usa.stats.CacheHits)/float64(usa.stats.CacheHits+usa.stats.CacheMisses)*100))

	return result, nil
}

// performFilesDiscovery 执行文件发现 - 阶段1
func (usa *UnifiedScanArchitecture) performFilesDiscovery(ctx context.Context, targetDir string) ([]*FileInfo, error) {
	usa.logger.Debug("开始文件发现阶段")

	// 使用现有的Scanner进行文件发现
	files, err := usa.fileScanner.ScanDirectory(ctx, targetDir)
	if err != nil {
		return nil, err
	}

	// 过滤出媒体文件候选
	var mediaFiles []*FileInfo
	for _, file := range files {
		if !file.IsDir && usa.isMediaFileCandidate(file.Path) {
			mediaFiles = append(mediaFiles, file)
		}
	}

	usa.logger.Debug("文件发现阶段完成",
		zap.Int("total_files", len(files)),
		zap.Int("media_candidates", len(mediaFiles)))

	return mediaFiles, nil
}

// performMorphologyAnalysis 执行形态分析 - 阶段2
func (usa *UnifiedScanArchitecture) performMorphologyAnalysis(ctx context.Context, files []*FileInfo) (map[string]*MorphologyResult, error) {
	usa.logger.Debug("开始形态分析阶段", zap.Int("files", len(files)))

	results := make(map[string]*MorphologyResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 创建工作通道
	fileChan := make(chan *FileInfo, len(files))
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// 启动工作协程
	for i := 0; i < usa.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				// 检查缓存
				if cached := usa.cacheManager.GetCachedResult(file.Path); cached != nil && cached.IsValid {
					mu.Lock()
					results[file.Path] = cached.MorphologyResult
					usa.stats.CacheHits++
					mu.Unlock()
					continue
				}

				// 执行形态分析
				result, err := usa.morphologyClassifier.ClassifyFile(ctx, file.Path)
				if err != nil {
					usa.logger.Warn("形态分析失败",
						zap.String("file", filepath.Base(file.Path)),
						zap.Error(err))
					continue
				}

				mu.Lock()
				results[file.Path] = result
				usa.stats.CacheMisses++
				usa.stats.FileTypeDistribution[result.MediaType.String()]++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	usa.logger.Debug("形态分析阶段完成", zap.Int("results", len(results)))
	return results, nil
}

// performQualityAssessment 执行品质评估 - 阶段3
func (usa *UnifiedScanArchitecture) performQualityAssessment(ctx context.Context, files []*FileInfo, morphologyResults map[string]*MorphologyResult) (map[string]*quality.QualityAssessment, error) {
	usa.logger.Debug("开始品质评估阶段")

	results := make(map[string]*quality.QualityAssessment)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 只对媒体文件进行品质评估
	var mediaFiles []*FileInfo
	for _, file := range files {
		if morphResult, exists := morphologyResults[file.Path]; exists {
			if morphResult.MediaType != types.MediaTypeUnknown {
				mediaFiles = append(mediaFiles, file)
			}
		}
	}

	// 创建工作通道
	fileChan := make(chan *FileInfo, len(mediaFiles))
	for _, file := range mediaFiles {
		fileChan <- file
	}
	close(fileChan)

	// 启动工作协程（使用较少的线程进行品质分析）
	workerCount := usa.maxWorkers / 2
	if workerCount < 1 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				// 检查缓存
				if cached := usa.cacheManager.GetCachedResult(file.Path); cached != nil && cached.IsValid && cached.QualityAssessment != nil {
					mu.Lock()
					results[file.Path] = cached.QualityAssessment
					mu.Unlock()
					continue
				}

				// 执行品质评估
				assessment, err := usa.qualityEngine.AssessFile(ctx, file.Path)
				if err != nil {
					usa.logger.Warn("品质评估失败",
						zap.String("file", filepath.Base(file.Path)),
						zap.Error(err))
					continue
				}

				mu.Lock()
				results[file.Path] = assessment
				usa.stats.QualityDistribution[assessment.QualityLevel]++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	usa.logger.Debug("品质评估阶段完成", zap.Int("results", len(results)))
	return results, nil
}

// cacheResults 缓存分析结果 - 阶段4
func (usa *UnifiedScanArchitecture) cacheResults(files []*FileInfo, morphologyResults map[string]*MorphologyResult, qualityResults map[string]*quality.QualityAssessment) error {
	usa.logger.Debug("开始结果缓存阶段")

	for _, file := range files {
		cachedResult := &CachedScanResult{
			FilePath:     file.Path,
			LastModified: time.Unix(file.ModTime, 0),
			FileSize:     file.Size,
			CacheTime:    time.Now(),
			IsValid:      true,
		}

		// 添加形态分析结果
		if morphResult, exists := morphologyResults[file.Path]; exists {
			cachedResult.MorphologyResult = morphResult
		}

		// 添加品质评估结果
		if qualityResult, exists := qualityResults[file.Path]; exists {
			cachedResult.QualityAssessment = qualityResult
		}

		// 存储到缓存
		usa.cacheManager.SetCachedResult(file.Path, cachedResult)
	}

	usa.logger.Debug("结果缓存阶段完成")
	return nil
}

// generateMediaInfoList 生成媒体信息列表 - 阶段5
func (usa *UnifiedScanArchitecture) generateMediaInfoList(files []*FileInfo, morphologyResults map[string]*MorphologyResult, qualityResults map[string]*quality.QualityAssessment) ([]*types.MediaInfo, error) {
	var mediaFiles []*types.MediaInfo

	for _, file := range files {
		morphResult, hasMorph := morphologyResults[file.Path]
		qualityResult, hasQuality := qualityResults[file.Path]

		// 只处理识别为媒体文件的项
		if !hasMorph || morphResult.MediaType == types.MediaTypeUnknown {
			usa.stats.SkippedFiles++
			continue
		}

		// 创建MediaInfo
		mediaInfo := &types.MediaInfo{
			Path:     file.Path,
			Size:     file.Size,
			ModTime:  time.Unix(file.ModTime, 0),
			Type:     morphResult.MediaType,
			Format:   morphResult.TrueFormat,
			Status:   types.StatusPending,
			Width:    morphResult.Width,
			Height:   morphResult.Height,
			Duration: morphResult.Duration,
		}

		// 添加品质信息
		if hasQuality {
			mediaInfo.Quality = qualityResult.QualityLevel
			mediaInfo.PixelDensity = qualityResult.PixelDensity
			mediaInfo.JpegQuality = qualityResult.JpegQuality
		}

		// 设置损坏标记
		if morphResult.FrameCount == 0 && morphResult.Duration == 0 && morphResult.MediaType != types.MediaTypeImage {
			mediaInfo.IsCorrupted = true
		}

		mediaFiles = append(mediaFiles, mediaInfo)
		usa.stats.MediaFiles++
	}

	// 按路径排序
	sort.Slice(mediaFiles, func(i, j int) bool {
		return mediaFiles[i].Path < mediaFiles[j].Path
	})

	return mediaFiles, nil
}

// generateUnifiedResult 生成统一扫描结果
func (usa *UnifiedScanArchitecture) generateUnifiedResult(mediaFiles []*types.MediaInfo) *UnifiedScanResult {
	// 生成摘要
	summary := &ScanSummary{
		TotalFiles: usa.stats.TotalFiles,
		MediaFiles: len(mediaFiles),
		TotalSize:  0,
	}

	for _, file := range mediaFiles {
		summary.TotalSize += file.Size

		switch file.Type {
		case types.MediaTypeImage:
			summary.StaticImages++
		case types.MediaTypeAnimated:
			summary.AnimatedImages++
		case types.MediaTypeVideo:
			summary.Videos++
		default:
			summary.UnknownFiles++
		}

		if file.IsCorrupted {
			summary.CorruptedFiles++
		}
	}

	if len(mediaFiles) > 0 {
		summary.AverageFileSize = summary.TotalSize / int64(len(mediaFiles))
	}
	summary.SkippedFiles = usa.stats.SkippedFiles

	// 生成缓存状态
	cacheStatus := &CacheStatus{
		MemoryCacheSize: len(usa.cacheManager.memoryCache),
		CacheHitRate:    float64(usa.stats.CacheHits) / float64(usa.stats.CacheHits+usa.stats.CacheMisses) * 100,
		CacheMissRate:   float64(usa.stats.CacheMisses) / float64(usa.stats.CacheHits+usa.stats.CacheMisses) * 100,
	}

	// 生成处理建议
	recommendations := usa.generateProcessingRecommendations(mediaFiles)

	return &UnifiedScanResult{
		Summary:         summary,
		MediaFiles:      mediaFiles,
		Statistics:      usa.stats,
		CacheStatus:     cacheStatus,
		Recommendations: recommendations,
	}
}

// 辅助方法
func (usa *UnifiedScanArchitecture) isMediaFileCandidate(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	mediaExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".heif", ".heic", ".avif", ".jxl",
		".tiff", ".tif", ".bmp", ".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v",
	}

	for _, mediaExt := range mediaExtensions {
		if ext == mediaExt {
			return true
		}
	}
	return false
}

func (usa *UnifiedScanArchitecture) generateProcessingRecommendations(mediaFiles []*types.MediaInfo) *ProcessingRecommendations {
	// 简化的处理建议生成
	highQualityCount := 0
	lowQualityCount := 0
	totalSize := int64(0)

	for _, file := range mediaFiles {
		totalSize += file.Size
		switch file.Quality {
		case types.QualityHigh, types.QualityVeryHigh:
			highQualityCount++
		case types.QualityLow, types.QualityVeryLow:
			lowQualityCount++
		}
	}

	// 基于统计推荐模式
	var recommendedMode types.AppMode
	if highQualityCount > len(mediaFiles)/2 {
		recommendedMode = types.ModeQuality
	} else if lowQualityCount > len(mediaFiles)/3 {
		recommendedMode = types.ModeEmoji
	} else {
		recommendedMode = types.ModeAutoPlus
	}

	return &ProcessingRecommendations{
		RecommendedMode:       recommendedMode,
		EstimatedSavings:      totalSize / 4, // 预估25%节省
		ProcessingTime:        time.Duration(len(mediaFiles)) * time.Second,
		RiskAssessment:        "低风险",
		SpecialConsiderations: []string{},
		OptimizationTips: []string{
			"建议使用推荐的处理模式",
			"高品质文件较多，考虑品质模式",
		},
	}
}

// 缓存管理器方法
func (scm *ScanCacheManager) GetCachedResult(filePath string) *CachedScanResult {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	if result, exists := scm.memoryCache[filePath]; exists {
		result.AccessCount++
		return result
	}
	return nil
}

func (scm *ScanCacheManager) SetCachedResult(filePath string, result *CachedScanResult) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	// 检查内存缓存大小限制
	if len(scm.memoryCache) >= scm.maxMemoryItems {
		// 简单的LRU清理：移除最少访问的条目
		scm.evictLeastUsed()
	}

	scm.memoryCache[filePath] = result
}

func (scm *ScanCacheManager) evictLeastUsed() {
	minAccess := int(^uint(0) >> 1) // 最大int值
	var evictKey string

	for key, result := range scm.memoryCache {
		if result.AccessCount < minAccess {
			minAccess = result.AccessCount
			evictKey = key
		}
	}

	if evictKey != "" {
		delete(scm.memoryCache, evictKey)
	}
}
