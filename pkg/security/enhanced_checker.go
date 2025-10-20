package security

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// EnhancedSecurityChecker 增强版安全检查器 - README要求的升级版本
type EnhancedSecurityChecker struct {
	*SecurityChecker
	whitelistManager *WhitelistManager
	permissionEngine *PermissionEngine
	auditLogger      *AuditLogger
	pathCache        map[string]*CachedSecurityResult
	config           *EnhancedSecurityConfig
}

// WhitelistManager 白名单管理器 - README核心功能
type WhitelistManager struct {
	logger             *zap.Logger
	staticWhitelist    []string
	dynamicWhitelist   []string
	temporaryWhitelist map[string]time.Time
	userWhitelist      []string
	projectWhitelist   []string
	whitelistHistory   []WhitelistChange
	enabled            bool
}

// PermissionEngine 增强权限引擎 - README要求的权限预检
type PermissionEngine struct {
	logger          *zap.Logger
	currentUser     *user.User
	preCheckEnabled bool
	permissionCache map[string]*EnhancedPermissionResult
	cacheExpiry     time.Duration
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logger       *zap.Logger
	enabled      bool
	recentEvents []AuditEvent
	maxEvents    int
}

// 数据结构
type EnhancedSecurityConfig struct {
	EnableWhitelistManager bool          `json:"enable_whitelist_manager"`
	EnablePermissionEngine bool          `json:"enable_permission_engine"`
	EnableAuditLogging     bool          `json:"enable_audit_logging"`
	EnablePathCache        bool          `json:"enable_path_cache"`
	CacheMaxAge            time.Duration `json:"cache_max_age"`
	WhitelistPersistFile   string        `json:"whitelist_persist_file"`
}

type WhitelistChange struct {
	Timestamp time.Time  `json:"timestamp"`
	Action    string     `json:"action"`
	Path      string     `json:"path"`
	Reason    string     `json:"reason"`
	Source    string     `json:"source"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type EnhancedPermissionResult struct {
	BasicPermission *PermissionResult `json:"basic_permission"`
	OwnerInfo       *OwnerInfo        `json:"owner_info"`
	ThreatScore     int               `json:"threat_score"`
	Recommendations []string          `json:"recommendations"`
	AccessTime      time.Time         `json:"access_time"`
}

type OwnerInfo struct {
	Username      string `json:"username"`
	IsAdmin       bool   `json:"is_admin"`
	IsCurrentUser bool   `json:"is_current_user"`
}

type AuditEvent struct {
	Timestamp     time.Time         `json:"timestamp"`
	EventType     string            `json:"event_type"`
	Path          string            `json:"path"`
	Action        string            `json:"action"`
	Result        string            `json:"result"`
	SecurityLevel string            `json:"security_level"`
	ThreatScore   int               `json:"threat_score"`
	Metadata      map[string]string `json:"metadata"`
}

type CachedSecurityResult struct {
	Result    *SecurityCheckResult
	Timestamp time.Time
	TTL       time.Duration
}

// NewEnhancedSecurityChecker 创建增强版安全检查器
func NewEnhancedSecurityChecker(logger *zap.Logger, config *EnhancedSecurityConfig) (*EnhancedSecurityChecker, error) {
	baseChecker := NewSecurityChecker(logger)

	if config == nil {
		config = &EnhancedSecurityConfig{
			EnableWhitelistManager: true,
			EnablePermissionEngine: true,
			EnableAuditLogging:     true,
			EnablePathCache:        true,
			CacheMaxAge:            10 * time.Minute,
		}
	}

	enhanced := &EnhancedSecurityChecker{
		SecurityChecker: baseChecker,
		config:          config,
		pathCache:       make(map[string]*CachedSecurityResult),
	}

	if err := enhanced.initializeComponents(); err != nil {
		return nil, fmt.Errorf("初始化增强组件失败: %w", err)
	}

	logger.Info("增强版安全检查器初始化完成")
	return enhanced, nil
}

// initializeComponents 初始化组件
func (esc *EnhancedSecurityChecker) initializeComponents() error {
	var err error

	if esc.config.EnableWhitelistManager {
		esc.whitelistManager = NewWhitelistManager(esc.logger)
		esc.whitelistManager.LoadDefaultWhitelist()
	}

	if esc.config.EnablePermissionEngine {
		esc.permissionEngine, err = NewPermissionEngine(esc.logger)
		if err != nil {
			return fmt.Errorf("初始化权限引擎失败: %w", err)
		}
	}

	if esc.config.EnableAuditLogging {
		esc.auditLogger = NewAuditLogger(esc.logger)
	}

	return nil
}

// EnhancedSecurityCheck 增强版安全检查 - README核心功能
func (esc *EnhancedSecurityChecker) EnhancedSecurityCheck(ctx context.Context, targetPath string) (*SecurityCheckResult, error) {
	startTime := time.Now()

	// 检查缓存
	if esc.config.EnablePathCache {
		if cached := esc.getCachedResult(targetPath); cached != nil {
			esc.logger.Debug("使用缓存结果", zap.String("path", filepath.Base(targetPath)))
			return cached.Result, nil
		}
	}

	esc.logger.Info("开始增强版安全检查", zap.String("target_path", targetPath))

	// 基础安全检查
	baseResult, err := esc.SecurityChecker.PerformSecurityCheck(targetPath)
	if err != nil {
		return nil, fmt.Errorf("基础安全检查失败: %w", err)
	}

	enhancedResult := *baseResult

	// 1. 白名单检查 - README核心功能
	if esc.whitelistManager != nil && esc.whitelistManager.enabled {
		if esc.whitelistManager.CheckWhitelist(targetPath) {
			enhancedResult.PathCheck.IsAllowed = true
			enhancedResult.PathCheck.SecurityLevel = SecurityLevelAllowed
			enhancedResult.PathCheck.Reason = "路径在白名单中"
			enhancedResult.Passed = true

			esc.logger.Debug("路径在白名单中，允许访问",
				zap.String("path", filepath.Base(targetPath)))
		}
	}

	// 2. 增强权限检查 - README核心功能
	if esc.permissionEngine != nil && esc.permissionEngine.preCheckEnabled {
		permResult, err := esc.permissionEngine.EnhancedPermissionCheck(targetPath)
		if err == nil {
			esc.enhancePermissionResult(&enhancedResult, permResult)
		}
	}

	// 3. 威胁分数计算
	threatScore := esc.calculateThreatScore(targetPath, &enhancedResult)

	// 4. 记录审计事件
	if esc.auditLogger != nil && esc.auditLogger.enabled {
		esc.auditLogger.LogEvent(AuditEvent{
			Timestamp:     time.Now(),
			EventType:     "enhanced_security_check",
			Path:          targetPath,
			Action:        "security_check",
			Result:        fmt.Sprintf("passed=%t", enhancedResult.Passed),
			SecurityLevel: fmt.Sprintf("%d", enhancedResult.PathCheck.SecurityLevel),
			ThreatScore:   threatScore,
			Metadata: map[string]string{
				"duration": time.Since(startTime).String(),
			},
		})
	}

	// 5. 缓存结果
	if esc.config.EnablePathCache {
		esc.cacheResult(targetPath, &enhancedResult)
	}

	esc.logger.Info("增强版安全检查完成",
		zap.String("target_path", filepath.Base(targetPath)),
		zap.Bool("passed", enhancedResult.Passed),
		zap.Int("threat_score", threatScore))

	return &enhancedResult, nil
}

// AddToWhitelist 添加到白名单 - README要求的白名单管理
func (esc *EnhancedSecurityChecker) AddToWhitelist(path string, reason string, source string, expiresAt *time.Time) error {
	if esc.whitelistManager == nil {
		return fmt.Errorf("白名单管理器未初始化")
	}

	return esc.whitelistManager.AddPath(path, reason, source, expiresAt)
}

// RemoveFromWhitelist 从白名单移除
func (esc *EnhancedSecurityChecker) RemoveFromWhitelist(path string, reason string) error {
	if esc.whitelistManager == nil {
		return fmt.Errorf("白名单管理器未初始化")
	}

	return esc.whitelistManager.RemovePath(path, reason)
}

// 组件实现
func NewWhitelistManager(logger *zap.Logger) *WhitelistManager {
	return &WhitelistManager{
		logger:             logger,
		staticWhitelist:    make([]string, 0),
		dynamicWhitelist:   make([]string, 0),
		temporaryWhitelist: make(map[string]time.Time),
		userWhitelist:      make([]string, 0),
		projectWhitelist:   make([]string, 0),
		whitelistHistory:   make([]WhitelistChange, 0),
		enabled:            true,
	}
}

func NewPermissionEngine(logger *zap.Logger) (*PermissionEngine, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("获取当前用户失败: %w", err)
	}

	return &PermissionEngine{
		logger:          logger,
		currentUser:     currentUser,
		preCheckEnabled: true,
		permissionCache: make(map[string]*EnhancedPermissionResult),
		cacheExpiry:     5 * time.Minute,
	}, nil
}

func NewAuditLogger(logger *zap.Logger) *AuditLogger {
	return &AuditLogger{
		logger:       logger,
		enabled:      true,
		recentEvents: make([]AuditEvent, 0),
		maxEvents:    1000,
	}
}

func (wm *WhitelistManager) LoadDefaultWhitelist() error {
	// 加载默认白名单路径
	defaultPaths := []string{
		"/Users/*/Documents/*",
		"/Users/*/Desktop/*",
		"/Users/*/Downloads/*",
		"/Users/*/Pictures/*",
		"/tmp/*",
		"/Users/Shared/*",
	}

	wm.staticWhitelist = append(wm.staticWhitelist, defaultPaths...)
	wm.logger.Info("默认白名单加载完成", zap.Int("count", len(defaultPaths)))
	return nil
}

func (wm *WhitelistManager) CheckWhitelist(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 检查所有白名单
	allLists := [][]string{
		wm.staticWhitelist,
		wm.dynamicWhitelist,
		wm.userWhitelist,
		wm.projectWhitelist,
	}

	for _, list := range allLists {
		for _, whitelistedPath := range list {
			if wm.matchPath(absPath, whitelistedPath) {
				return true
			}
		}
	}

	// 检查临时白名单
	for whitelistedPath, expiresAt := range wm.temporaryWhitelist {
		if time.Now().Before(expiresAt) && wm.matchPath(absPath, whitelistedPath) {
			return true
		}
	}

	return false
}

func (wm *WhitelistManager) matchPath(path, pattern string) bool {
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
		}
	}
	return strings.HasPrefix(path, pattern)
}

func (wm *WhitelistManager) AddPath(path, reason, source string, expiresAt *time.Time) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("路径无效: %w", err)
	}

	// 记录变更
	change := WhitelistChange{
		Timestamp: time.Now(),
		Action:    "add",
		Path:      absPath,
		Reason:    reason,
		Source:    source,
		ExpiresAt: expiresAt,
	}

	// 根据来源添加到相应列表
	switch source {
	case "user":
		wm.userWhitelist = append(wm.userWhitelist, absPath)
	case "project":
		wm.projectWhitelist = append(wm.projectWhitelist, absPath)
	case "dynamic":
		wm.dynamicWhitelist = append(wm.dynamicWhitelist, absPath)
	case "temporary":
		if expiresAt != nil {
			wm.temporaryWhitelist[absPath] = *expiresAt
		}
	default:
		wm.staticWhitelist = append(wm.staticWhitelist, absPath)
	}

	wm.whitelistHistory = append(wm.whitelistHistory, change)
	wm.logger.Info("路径已添加到白名单", zap.String("path", absPath), zap.String("source", source))
	return nil
}

func (wm *WhitelistManager) RemovePath(path, reason string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("路径无效: %w", err)
	}

	// 从所有列表中移除
	wm.removeFromSlice(&wm.staticWhitelist, absPath)
	wm.removeFromSlice(&wm.dynamicWhitelist, absPath)
	wm.removeFromSlice(&wm.userWhitelist, absPath)
	wm.removeFromSlice(&wm.projectWhitelist, absPath)
	delete(wm.temporaryWhitelist, absPath)

	// 记录变更
	change := WhitelistChange{
		Timestamp: time.Now(),
		Action:    "remove",
		Path:      absPath,
		Reason:    reason,
	}
	wm.whitelistHistory = append(wm.whitelistHistory, change)

	wm.logger.Info("路径已从白名单移除", zap.String("path", absPath))
	return nil
}

func (wm *WhitelistManager) removeFromSlice(slice *[]string, item string) {
	for i, v := range *slice {
		if v == item {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
}

func (pe *PermissionEngine) EnhancedPermissionCheck(path string) (*EnhancedPermissionResult, error) {
	// 检查缓存
	if cached, exists := pe.permissionCache[path]; exists {
		if time.Since(cached.AccessTime) < pe.cacheExpiry {
			return cached, nil
		}
		delete(pe.permissionCache, path)
	}

	// 获取文件信息
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 基础权限检查
	mode := info.Mode()
	basicPerm := &PermissionResult{
		CanRead:    mode&0400 != 0,
		CanWrite:   mode&0200 != 0,
		CanExecute: mode&0100 != 0,
		Issues:     make([]string, 0),
	}

	// 获取所有者信息
	ownerInfo := &OwnerInfo{
		Username:      "unknown",
		IsCurrentUser: true, // 简化实现
		IsAdmin:       false,
	}

	// 计算威胁分数
	threatScore := 0
	if basicPerm.CanExecute {
		threatScore += 20
	}
	if !ownerInfo.IsCurrentUser {
		threatScore += 10
	}

	// 生成建议
	var recommendations []string
	if threatScore > 30 {
		recommendations = append(recommendations, "高风险文件，建议仔细检查")
	}
	if !basicPerm.CanWrite {
		recommendations = append(recommendations, "文件只读，处理前请确认权限")
	}

	result := &EnhancedPermissionResult{
		BasicPermission: basicPerm,
		OwnerInfo:       ownerInfo,
		ThreatScore:     threatScore,
		Recommendations: recommendations,
		AccessTime:      time.Now(),
	}

	// 缓存结果
	pe.permissionCache[path] = result

	return result, nil
}

func (al *AuditLogger) LogEvent(event AuditEvent) {
	if !al.enabled {
		return
	}

	al.recentEvents = append(al.recentEvents, event)

	// 限制事件数量
	if len(al.recentEvents) > al.maxEvents {
		al.recentEvents = al.recentEvents[1:]
	}

	// 记录到日志
	al.logger.Info("安全审计事件",
		zap.String("event_type", event.EventType),
		zap.String("path", filepath.Base(event.Path)),
		zap.String("result", event.Result))
}

// 辅助方法
func (esc *EnhancedSecurityChecker) calculateThreatScore(path string, result *SecurityCheckResult) int {
	score := 0

	// 基于路径的威胁分析
	suspiciousPatterns := []string{"../", "..\\", "/proc/", "/sys/"}
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(path, pattern) {
			score += 15
		}
	}

	// 基于检查结果
	score += len(result.Issues) * 10
	score += len(result.Warnings) * 5

	if result.PathCheck.SecurityLevel == SecurityLevelCritical {
		score += 50
	}

	return score
}

func (esc *EnhancedSecurityChecker) enhancePermissionResult(result *SecurityCheckResult, permResult *EnhancedPermissionResult) {
	if permResult.ThreatScore > 30 {
		result.Warnings = append(result.Warnings, SecurityWarning{
			Type:    WarningTypeNearSpaceLimit,
			Message: "权限检查发现潜在风险",
			Path:    result.PathCheck.AbsolutePath,
		})
	}

	// 更新权限检查结果
	result.PermissionCheck = *permResult.BasicPermission
}

func (esc *EnhancedSecurityChecker) getCachedResult(path string) *CachedSecurityResult {
	if cached, exists := esc.pathCache[path]; exists {
		if time.Since(cached.Timestamp) < cached.TTL {
			return cached
		}
		delete(esc.pathCache, path)
	}
	return nil
}

func (esc *EnhancedSecurityChecker) cacheResult(path string, result *SecurityCheckResult) {
	esc.pathCache[path] = &CachedSecurityResult{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       esc.config.CacheMaxAge,
	}

	// 限制缓存大小
	if len(esc.pathCache) > 1000 {
		esc.evictOldCache()
	}
}

func (esc *EnhancedSecurityChecker) evictOldCache() {
	// 简单的LRU清理
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range esc.pathCache {
		if oldestKey == "" || cached.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.Timestamp
		}
	}

	if oldestKey != "" {
		delete(esc.pathCache, oldestKey)
	}
}

// GetWhitelistStats 获取白名单统计信息
func (esc *EnhancedSecurityChecker) GetWhitelistStats() map[string]interface{} {
	if esc.whitelistManager == nil {
		return nil
	}

	return map[string]interface{}{
		"static_count":    len(esc.whitelistManager.staticWhitelist),
		"dynamic_count":   len(esc.whitelistManager.dynamicWhitelist),
		"temporary_count": len(esc.whitelistManager.temporaryWhitelist),
		"user_count":      len(esc.whitelistManager.userWhitelist),
		"project_count":   len(esc.whitelistManager.projectWhitelist),
		"history_count":   len(esc.whitelistManager.whitelistHistory),
		"enabled":         esc.whitelistManager.enabled,
	}
}

// GetAuditEvents 获取审计事件
func (esc *EnhancedSecurityChecker) GetAuditEvents(limit int) []AuditEvent {
	if esc.auditLogger == nil {
		return nil
	}

	events := esc.auditLogger.recentEvents
	if limit > 0 && len(events) > limit {
		return events[len(events)-limit:]
	}

	return events
}
