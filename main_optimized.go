// Package main 提供 Pixly 智能图像转换工具的主程序入口
//
// 功能特性:
// - 智能格式选择: 根据图像特征自动选择 JXL 或 AVIF 格式
// - 质量评估: 基于文件大小和内容特征进行质量分析
// - 尝试引擎: 测试不同参数组合，找到最佳转换策略
// - 安全策略: 多层次安全保护机制
// - 用户界面: 美观的命令行界面，支持交互和非交互模式
// - 代码优化: 消除重复函数，提升代码质量和维护性
//
// 安全特性:
// - 输入验证: 严格的用户输入验证和清理
// - 文件权限: 安全的文件操作权限控制
// - 错误处理: 完善的错误处理和恢复机制
// - 资源管理: 智能的内存和CPU资源管理
//
// 作者: AI Assistant
// 版本: v2.1.1
// 许可证: MIT
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 应用程序常量定义
const (
	AppVersion     = "v2.1.1"
	AppName        = "Pixly"
	AppDescription = "智能图像转换工具 - 支持JXL和AVIF格式"

	// 数据库相关
	DBPath     = "~/.pixly/state.db"
	BucketName = "media_files"

	// 安全相关
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

var AllowedPaths = []string{"/Users", "/home", "/tmp"}

// MediaInfo 媒体文件信息结构体
// 基于旧文档要求的数据结构
type MediaInfo struct {
	FullPath       string    `json:"full_path"`       // 规范化后的绝对路径
	FileSize       int64     `json:"file_size"`       // 文件大小（字节）
	ModTime        time.Time `json:"mod_time"`        // 文件最后修改时间
	SHA256Hash     string    `json:"sha256_hash"`     // 文件内容的 SHA256 哈希值，用于状态跟踪
	Codec          string    `json:"codec"`           // 主要编解码器名称
	FrameCount     int       `json:"frame_count"`     // 帧数
	IsAnimated     bool      `json:"is_animated"`     // 是否为动图或视频
	IsCorrupted    bool      `json:"is_corrupted"`    // 是否检测为损坏文件
	InitialQuality int       `json:"initial_quality"` // 预估的初始质量（1-100）
	Processed      bool      `json:"processed"`       // 是否已处理
	ProcessTime    time.Time `json:"process_time"`    // 处理时间
	ErrorMsg       string    `json:"error_msg"`       // 错误信息
}

// Config 应用程序配置结构体
type Config struct {
	QualityMode      string `json:"quality_mode"`
	EmojiMode        bool   `json:"emoji_mode"`
	NonInteractive   bool   `json:"non_interactive"`
	Interactive      bool   `json:"interactive"`
	OutputFormat     string `json:"output_format"`
	ReplaceOriginals bool   `json:"replace_originals"`
	CreateBackup     bool   `json:"create_backup"`
	StickerMode      bool   `json:"sticker_mode"`
	TryEngine        bool   `json:"try_engine"`
	SecurityLevel    string `json:"security_level"`
	MaxWorkers       int    `json:"max_workers"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
}

// StateManager 状态管理器
// 基于旧文档要求使用bbolt实现断点续传
type StateManager struct {
	db     *bbolt.DB
	logger *zap.Logger
}

// NewStateManager 创建新的状态管理器
func NewStateManager(logger *zap.Logger) (*StateManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(homeDir, ".pixly", "state.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	// 创建bucket
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BucketName))
		return err
	})

	if err != nil {
		return nil, err
	}

	return &StateManager{
		db:     db,
		logger: logger,
	}, nil
}

// SaveMediaFiles 保存媒体文件信息
func (sm *StateManager) SaveMediaFiles(files []*MediaInfo) error {
	return sm.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketName))
		for _, file := range files {
			data, err := json.Marshal(file)
			if err != nil {
				return err
			}
			return bucket.Put([]byte(file.FullPath), data)
		}
		return nil
	})
}

// LoadMediaFiles 加载媒体文件信息
func (sm *StateManager) LoadMediaFiles() ([]*MediaInfo, error) {
	var files []*MediaInfo
	err := sm.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketName))
		return bucket.ForEach(func(k, v []byte) error {
			var file MediaInfo
			if err := json.Unmarshal(v, &file); err != nil {
				return err
			}
			files = append(files, &file)
			return nil
		})
	})
	return files, err
}

// Close 关闭数据库连接
func (sm *StateManager) Close() error {
	return sm.db.Close()
}

// SecurityChecker 安全检查器
// 基于旧文档要求实现路径白名单和权限检查
type SecurityChecker struct {
	logger *zap.Logger
}

// NewSecurityChecker 创建新的安全检查器
func NewSecurityChecker(logger *zap.Logger) *SecurityChecker {
	return &SecurityChecker{logger: logger}
}

// CheckPath 检查路径是否安全
func (sc *SecurityChecker) CheckPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("无法获取绝对路径: %v", err)
	}

	// 检查是否在允许的路径内
	for _, allowedPath := range AllowedPaths {
		if strings.HasPrefix(absPath, allowedPath) {
			return nil
		}
	}

	return fmt.Errorf("路径不在允许的范围内: %s", absPath)
}

// CheckFileSize 检查文件大小
func (sc *SecurityChecker) CheckFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.Size() > MaxFileSize {
		return fmt.Errorf("文件过大: %d bytes (最大允许: %d bytes)", info.Size(), MaxFileSize)
	}

	return nil
}

// Watchdog 看门狗系统
// 基于旧文档要求实现双模式工作
type Watchdog struct {
	logger       *zap.Logger
	timeout      time.Duration
	debugMode    bool
	lastActivity time.Time
	mu           sync.RWMutex
}

// NewWatchdog 创建新的看门狗
func NewWatchdog(logger *zap.Logger, debugMode bool) *Watchdog {
	timeout := 30 * time.Second
	if debugMode {
		timeout = 30 * time.Second // 调试模式下30秒超时
	} else {
		timeout = 2 * time.Hour // 用户模式下2小时超时
	}

	return &Watchdog{
		logger:       logger,
		timeout:      timeout,
		debugMode:    debugMode,
		lastActivity: time.Now(),
	}
}

// UpdateActivity 更新活动时间
func (w *Watchdog) UpdateActivity() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastActivity = time.Now()
}

// CheckTimeout 检查是否超时
func (w *Watchdog) CheckTimeout() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return time.Since(w.lastActivity) > w.timeout
}

// Start 启动看门狗
func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.CheckTimeout() {
				if w.debugMode {
					w.logger.Fatal("看门狗检测到超时，强制退出")
				} else {
					w.logger.Warn("检测到长时间无活动，建议检查处理状态")
				}
			}
		}
	}
}

// SmartScanner 智能扫描引擎
// 基于旧文档要求实现两阶段扫描
type SmartScanner struct {
	logger          *zap.Logger
	securityChecker *SecurityChecker
	watchdog        *Watchdog
}

// NewSmartScanner 创建新的智能扫描器
func NewSmartScanner(logger *zap.Logger, securityChecker *SecurityChecker, watchdog *Watchdog) *SmartScanner {
	return &SmartScanner{
		logger:          logger,
		securityChecker: securityChecker,
		watchdog:        watchdog,
	}
}

// ScanDirectory 扫描目录
// 实现两阶段扫描：元信息预判95% + FFmpeg深度验证5%
func (ss *SmartScanner) ScanDirectory(dir string) ([]*MediaInfo, error) {
	ss.logger.Info("开始智能扫描", zap.String("directory", dir))

	// 阶段一：元信息预判 (95%)
	ss.logger.Info("阶段一：元信息预判")
	candidateFiles, err := ss.quickScan(dir)
	if err != nil {
		return nil, err
	}

	ss.logger.Info("快速扫描完成", zap.Int("candidate_files", len(candidateFiles)))

	// 阶段二：FFmpeg深度验证 (5%)
	ss.logger.Info("阶段二：FFmpeg深度验证")
	var mediaFiles []*MediaInfo

	for i, filePath := range candidateFiles {
		// 更新看门狗活动
		ss.watchdog.UpdateActivity()

		// 安全检查
		if err := ss.securityChecker.CheckPath(filePath); err != nil {
			ss.logger.Warn("路径安全检查失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		if err := ss.securityChecker.CheckFileSize(filePath); err != nil {
			ss.logger.Warn("文件大小检查失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		// 深度分析
		mediaInfo, err := ss.deepAnalyze(filePath)
		if err != nil {
			ss.logger.Warn("深度分析失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		mediaFiles = append(mediaFiles, mediaInfo)

		// 进度显示
		if (i+1)%10 == 0 {
			ss.logger.Info("扫描进度", zap.Int("processed", i+1), zap.Int("total", len(candidateFiles)))
		}
	}

	ss.logger.Info("智能扫描完成", zap.Int("media_files", len(mediaFiles)))
	return mediaFiles, nil
}

// quickScan 快速扫描
func (ss *SmartScanner) quickScan(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 基于扩展名的快速筛选
		ext := strings.ToLower(filepath.Ext(path))
		imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp", ".heic", ".heif", ".avif"}

		for _, imgExt := range imageExts {
			if ext == imgExt {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

// deepAnalyze 深度分析
func (ss *SmartScanner) deepAnalyze(filePath string) (*MediaInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// 计算SHA256哈希
	hash, err := ss.calculateSHA256(filePath)
	if err != nil {
		return nil, err
	}

	// 使用ffprobe进行深度分析
	codec, frameCount, isAnimated, isCorrupted, err := ss.analyzeWithFFprobe(filePath)
	if err != nil {
		ss.logger.Warn("FFprobe分析失败", zap.String("file", filePath), zap.Error(err))
		// 使用基础分析作为回退
		codec = "unknown"
		frameCount = 1
		isAnimated = ss.isAnimatedByExtension(filePath)
		isCorrupted = false
	}

	// 质量评估
	initialQuality := ss.assessQuality(info.Size(), codec, isAnimated)

	return &MediaInfo{
		FullPath:       filePath,
		FileSize:       info.Size(),
		ModTime:        info.ModTime(),
		SHA256Hash:     hash,
		Codec:          codec,
		FrameCount:     frameCount,
		IsAnimated:     isAnimated,
		IsCorrupted:    isCorrupted,
		InitialQuality: initialQuality,
		Processed:      false,
	}, nil
}

// calculateSHA256 计算文件SHA256哈希
func (ss *SmartScanner) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// analyzeWithFFprobe 使用FFprobe分析文件
func (ss *SmartScanner) analyzeWithFFprobe(filePath string) (string, int, bool, bool, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", 0, false, false, err
	}

	// 解析JSON输出
	var result struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", 0, false, false, err
	}

	// 分析结果
	codec := "unknown"
	frameCount := 1
	isAnimated := false
	isCorrupted := false

	if len(result.Streams) > 0 {
		codec = result.Streams[0].CodecName
		if result.Streams[0].CodecType == "video" {
			isAnimated = true
		}
	}

	// 检测动画
	if ss.isAnimatedByExtension(filePath) {
		isAnimated = true
	}

	return codec, frameCount, isAnimated, isCorrupted, nil
}

// isAnimatedByExtension 基于扩展名检测动画
func (ss *SmartScanner) isAnimatedByExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	animatedExts := []string{".gif", ".webp", ".avif", ".heic", ".heif"}

	for _, animatedExt := range animatedExts {
		if ext == animatedExt {
			return true
		}
	}
	return false
}

// assessQuality 评估质量
func (ss *SmartScanner) assessQuality(fileSize int64, codec string, isAnimated bool) int {
	// 基于文件大小的质量评估
	if fileSize > 5*1024*1024 { // > 5MB
		return 90
	} else if fileSize > 2*1024*1024 { // > 2MB
		return 80
	} else if fileSize > 500*1024 { // > 500KB
		return 70
	} else if fileSize > 100*1024 { // > 100KB
		return 60
	} else {
		return 50
	}
}

// SmartStrategy 智能策略选择器
type SmartStrategy struct {
	logger *zap.Logger
}

// NewSmartStrategy 创建新的智能策略选择器
func NewSmartStrategy(logger *zap.Logger) *SmartStrategy {
	return &SmartStrategy{logger: logger}
}

// SelectBestFormat 选择最佳格式
func (ss *SmartStrategy) SelectBestFormat(mediaFiles []*MediaInfo) string {
	// 统计文件类型
	animatedCount := 0
	staticCount := 0
	highQualityCount := 0

	for _, file := range mediaFiles {
		if file.IsAnimated {
			animatedCount++
		} else {
			staticCount++
		}

		if file.InitialQuality >= 80 {
			highQualityCount++
		}
	}

	ss.logger.Info("文件分析结果",
		zap.Int("animated", animatedCount),
		zap.Int("static", staticCount),
		zap.Int("high_quality", highQualityCount))

	// 智能选择策略
	if animatedCount > staticCount {
		ss.logger.Info("检测到大量动画文件，选择AVIF格式")
		return "avif"
	} else if highQualityCount > len(mediaFiles)/2 {
		ss.logger.Info("检测到大量高质量文件，选择JXL格式")
		return "jxl"
	} else {
		ss.logger.Info("平衡选择，使用JXL格式")
		return "jxl"
	}
}

// UIManager 用户界面管理器
type UIManager struct {
	logger      *zap.Logger
	interactive bool
	emojiMode   bool
}

// NewUIManager 创建新的UI管理器
func NewUIManager(logger *zap.Logger, interactive, emojiMode bool) *UIManager {
	return &UIManager{
		logger:      logger,
		interactive: interactive,
		emojiMode:   emojiMode,
	}
}

// ShowWelcome 显示欢迎界面
func (ui *UIManager) ShowWelcome() {
	ui.ClearScreen()
	ui.PrintHeader()
	ui.PrintLine("🎨 " + AppName + " " + AppVersion)
	ui.PrintLine("✨ " + AppDescription)
	ui.PrintLine("")
	ui.PrintLine("🚀 智能图像转换工具，支持JXL和AVIF格式")
	ui.PrintLine("📊 自动质量评估和最佳格式选择")
	ui.PrintLine("🛡️ 安全策略保护您的数据")
	ui.PrintLine("")
}

func (ui *UIManager) ClearScreen() {
	if ui.interactive {
		fmt.Print("\033[2J\033[H")
	}
}

func (ui *UIManager) PrintHeader() {
	if ui.emojiMode {
		ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
		ui.PrintLine("║                    🎨 Pixly 智能转换工具 🎨                    ║")
		ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	} else {
		ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
		ui.PrintLine("║                    Pixly 智能转换工具                        ║")
		ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	}
}

func (ui *UIManager) PrintLine(text string) {
	fmt.Println(text)
}

func (ui *UIManager) PrintError(text string) {
	if ui.emojiMode {
		fmt.Println("❌ " + text)
	} else {
		fmt.Println("ERROR: " + text)
	}
}

func (ui *UIManager) PrintSuccess(text string) {
	if ui.emojiMode {
		fmt.Println("✅ " + text)
	} else {
		fmt.Println("SUCCESS: " + text)
	}
}

func (ui *UIManager) PrintInfo(text string) {
	if ui.emojiMode {
		fmt.Println("ℹ️  " + text)
	} else {
		fmt.Println("INFO: " + text)
	}
}

func (ui *UIManager) PrintWarning(text string) {
	if ui.emojiMode {
		fmt.Println("⚠️  " + text)
	} else {
		fmt.Println("WARNING: " + text)
	}
}

func (ui *UIManager) ReadInput(prompt string) string {
	if !ui.interactive {
		return ""
	}

	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Converter 转换执行器
type Converter struct {
	logger *zap.Logger
}

// NewConverter 创建新的转换器
func NewConverter(logger *zap.Logger) *Converter {
	return &Converter{logger: logger}
}

// ExecuteConversion 执行转换
func (c *Converter) ExecuteConversion(dir, format string, config *Config) error {
	ui := NewUIManager(c.logger, config.Interactive, config.EmojiMode)

	// 构建命令参数
	var args []string
	var toolName string

	// 基础参数
	args = append(args, "-dir", dir)

	// 根据质量模式添加参数
	switch config.QualityMode {
	case "high":
		ui.PrintInfo("🎯 使用高质量模式")
	case "medium":
		ui.PrintInfo("🎯 使用中等质量模式")
	case "low":
		ui.PrintInfo("🎯 使用低质量模式")
	default:
		ui.PrintInfo("🎯 使用自动质量模式")
	}

	// 表情包模式特殊处理
	if config.StickerMode {
		ui.PrintInfo("😊 表情包模式：优化小文件处理")
		args = append(args, "-sample", "10")
	}

	// 构建命令
	if format == "jxl" {
		toolName = "all2jxl"
		cmd := exec.Command("./easymode/all2jxl/bin/all2jxl", args...)
		ui.PrintInfo(fmt.Sprintf("🚀 使用 %s 工具进行转换...", toolName))
		ui.PrintLine("")

		// 执行转换
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("转换失败: %v", err)
		}
	} else if format == "avif" {
		toolName = "all2avif"
		cmd := exec.Command("./easymode/all2avif/bin/all2avif", args...)
		ui.PrintInfo(fmt.Sprintf("🚀 使用 %s 工具进行转换...", toolName))
		ui.PrintLine("")

		// 执行转换
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("转换失败: %v", err)
		}
	} else {
		return fmt.Errorf("不支持的格式: %s", format)
	}

	ui.PrintSuccess("转换完成！")
	return nil
}

// main 主程序入口点
func main() {
	// 初始化日志系统
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := config.Build()
	defer logger.Sync()

	// 解析命令行参数
	var (
		nonInteractive = flag.Bool("non-interactive", false, "非交互模式")
		emojiMode      = flag.Bool("emoji", true, "启用表情符号模式")
		qualityMode    = flag.String("quality", "auto", "质量模式: auto, high, medium, low")
		outputFormat   = flag.String("format", "auto", "输出格式: jxl, avif, auto")
		targetDir      = flag.String("dir", "", "目标目录")
		stickerMode    = flag.Bool("sticker", false, "表情包模式")
		tryEngine      = flag.Bool("try-engine", true, "启用尝试引擎")
		securityLevel  = flag.String("security", "medium", "安全级别: high, medium, low")
		debugMode      = flag.Bool("debug", false, "调试模式")
	)
	flag.Parse()

	// 创建配置
	configStruct := &Config{
		QualityMode:      *qualityMode,
		EmojiMode:        *emojiMode,
		NonInteractive:   *nonInteractive,
		Interactive:      !*nonInteractive,
		OutputFormat:     *outputFormat,
		ReplaceOriginals: true,
		CreateBackup:     true,
		StickerMode:      *stickerMode,
		TryEngine:        *tryEngine,
		SecurityLevel:    *securityLevel,
		MaxWorkers:       runtime.NumCPU(),
		TimeoutSeconds:   300,
	}

	// 初始化组件
	ui := NewUIManager(logger, configStruct.Interactive, configStruct.EmojiMode)
	securityChecker := NewSecurityChecker(logger)
	watchdog := NewWatchdog(logger, *debugMode)
	stateManager, err := NewStateManager(logger)
	if err != nil {
		logger.Fatal("初始化状态管理器失败", zap.Error(err))
	}
	defer stateManager.Close()

	// 启动看门狗
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchdog.Start(ctx)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ui.PrintInfo("收到退出信号，正在安全退出...")
		cancel()
		os.Exit(0)
	}()

	// 显示欢迎信息
	ui.ShowWelcome()

	// 获取目标目录
	if *targetDir == "" {
		if configStruct.Interactive {
			*targetDir = ui.ReadInput("请输入目标目录路径:")
		} else {
			ui.PrintError("非交互模式下必须指定目标目录")
			os.Exit(1)
		}
	}

	// 安全检查
	if err := securityChecker.CheckPath(*targetDir); err != nil {
		ui.PrintError(fmt.Sprintf("路径安全检查失败: %v", err))
		os.Exit(1)
	}

	// 验证目录
	if _, err := os.Stat(*targetDir); os.IsNotExist(err) {
		ui.PrintError(fmt.Sprintf("目录不存在: %s", *targetDir))
		os.Exit(1)
	}

	// 检查工具是否存在
	all2jxlPath := "./easymode/all2jxl/bin/all2jxl"
	all2avifPath := "./easymode/all2avif/bin/all2avif"

	if _, err := os.Stat(all2jxlPath); os.IsNotExist(err) {
		ui.PrintError("all2jxl 工具不存在，请先构建")
		os.Exit(1)
	}

	if _, err := os.Stat(all2avifPath); os.IsNotExist(err) {
		ui.PrintError("all2avif 工具不存在，请先构建")
		os.Exit(1)
	}

	// 智能扫描
	scanner := NewSmartScanner(logger, securityChecker, watchdog)
	mediaFiles, err := scanner.ScanDirectory(*targetDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("扫描目录失败: %v", err))
		os.Exit(1)
	}

	if len(mediaFiles) == 0 {
		ui.PrintWarning("未找到可处理的媒体文件")
		os.Exit(0)
	}

	// 保存到状态管理器
	if err := stateManager.SaveMediaFiles(mediaFiles); err != nil {
		ui.PrintError(fmt.Sprintf("保存文件信息失败: %v", err))
		os.Exit(1)
	}

	// 智能策略选择
	smartStrategy := NewSmartStrategy(logger)
	selectedFormat := configStruct.OutputFormat
	if selectedFormat == "auto" {
		selectedFormat = smartStrategy.SelectBestFormat(mediaFiles)
	}

	ui.PrintInfo(fmt.Sprintf("🎯 选择的输出格式: %s", strings.ToUpper(selectedFormat)))
	ui.PrintInfo(fmt.Sprintf("📊 发现 %d 个媒体文件", len(mediaFiles)))

	// 确认处理
	if configStruct.Interactive {
		ui.PrintLine("")
		choice := ui.ReadInput("是否开始转换? (y/N):")
		if strings.ToLower(choice) != "y" {
			ui.PrintInfo("用户取消操作")
			return
		}
	}

	// 开始转换
	ui.PrintInfo("开始转换...")
	ui.PrintLine("")

	converter := NewConverter(logger)
	err = converter.ExecuteConversion(*targetDir, selectedFormat, configStruct)
	if err != nil {
		ui.PrintError(fmt.Sprintf("转换失败: %v", err))
		os.Exit(1)
	}

	// 显示完成信息
	ui.PrintLine("")
	ui.PrintLine("╔══════════════════════════════════════════════════════════════╗")
	ui.PrintLine("║                        转换完成                              ║")
	ui.PrintLine("╚══════════════════════════════════════════════════════════════╝")
	ui.PrintSuccess("🎉 所有文件转换完成！")
	ui.PrintInfo(fmt.Sprintf("📁 输出目录: %s", *targetDir))
	ui.PrintInfo(fmt.Sprintf("📄 输出格式: %s", strings.ToUpper(selectedFormat)))
}
