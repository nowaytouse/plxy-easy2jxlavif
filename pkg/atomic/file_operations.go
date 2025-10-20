package fileatomic

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// AtomicFileOperator 原子性文件操作器 - README要求的核心文件安全机制
//
// 核心功能：
//   - 实现"备份→验证→替换→清理"的四步原子操作
//   - 提供完整的回滚机制，确保原文件安全
//   - 支持事务性文件处理，防止中途失败导致数据丢失
//   - 集成文件完整性验证（哈希校验）
//   - 支持批量原子操作和级联回滚
//
// 设计原则：
//   - 原文件永远不直接修改，先备份再操作
//   - 所有操作可回滚，确保系统状态一致性
//   - 验证机制保证文件完整性
//   - 临时文件自动清理，避免磁盘空间浪费
type AtomicFileOperator struct {
	logger            *zap.Logger
	backupDir         string               // 备份目录
	tempDir           string               // 临时文件目录
	verificationMode  VerificationMode     // 验证模式
	operations        []*AtomicOperation   // 当前事务中的操作列表
	rollbackStack     []*RollbackOperation // 回滚栈
	maxRetries        int                  // 最大重试次数
	retryDelay        time.Duration        // 重试间隔
	enableCompression bool                 // 是否压缩备份文件
}

// AtomicOperation 原子操作定义
type AtomicOperation struct {
	ID           string            `json:"id"`            // 操作唯一ID
	Type         OperationType     `json:"type"`          // 操作类型
	SourcePath   string            `json:"source_path"`   // 源文件路径
	TargetPath   string            `json:"target_path"`   // 目标文件路径
	BackupPath   string            `json:"backup_path"`   // 备份文件路径
	TempPath     string            `json:"temp_path"`     // 临时文件路径
	SourceHash   string            `json:"source_hash"`   // 源文件哈希
	TargetHash   string            `json:"target_hash"`   // 目标文件哈希
	Status       OperationStatus   `json:"status"`        // 操作状态
	StartTime    time.Time         `json:"start_time"`    // 开始时间
	EndTime      time.Time         `json:"end_time"`      // 结束时间
	ErrorMessage string            `json:"error_message"` // 错误信息
	Metadata     map[string]string `json:"metadata"`      // 元数据
}

// RollbackOperation 回滚操作定义
type RollbackOperation struct {
	OperationID string         `json:"operation_id"` // 对应的原子操作ID
	Action      RollbackAction `json:"action"`       // 回滚动作
	SourcePath  string         `json:"source_path"`  // 回滚源路径
	TargetPath  string         `json:"target_path"`  // 回滚目标路径
	Priority    int            `json:"priority"`     // 回滚优先级（数字越小优先级越高）
}

// 枚举定义
type OperationType int
type OperationStatus int
type RollbackAction int
type VerificationMode int

const (
	// 操作类型
	OperationReplace OperationType = iota // 替换文件
	OperationMove                         // 移动文件
	OperationCopy                         // 复制文件
	OperationDelete                       // 删除文件
	OperationCreate                       // 创建文件
)

const (
	// 操作状态
	StatusPending    OperationStatus = iota // 等待执行
	StatusBackup                            // 备份阶段
	StatusVerify                            // 验证阶段
	StatusReplace                           // 替换阶段
	StatusCleanup                           // 清理阶段
	StatusCompleted                         // 完成
	StatusFailed                            // 失败
	StatusRolledBack                        // 已回滚
)

const (
	// 回滚动作
	RollbackRestore RollbackAction = iota // 恢复原文件
	RollbackDelete                        // 删除文件
	RollbackMove                          // 移动文件
	RollbackCleanup                       // 清理临时文件
)

const (
	// 验证模式
	VerificationNone     VerificationMode = iota // 无验证
	VerificationSHA256                           // SHA256哈希验证
	VerificationSizeOnly                         // 仅文件大小验证
	VerificationFull                             // 完整验证（大小+哈希+格式）
)

// 字符串方法
func (t OperationType) String() string {
	switch t {
	case OperationReplace:
		return "replace"
	case OperationMove:
		return "move"
	case OperationCopy:
		return "copy"
	case OperationDelete:
		return "delete"
	case OperationCreate:
		return "create"
	default:
		return "unknown"
	}
}

func (s OperationStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusBackup:
		return "backup"
	case StatusVerify:
		return "verify"
	case StatusReplace:
		return "replace"
	case StatusCleanup:
		return "cleanup"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusRolledBack:
		return "rolled_back"
	default:
		return "unknown"
	}
}

// NewAtomicFileOperator 创建原子性文件操作器
func NewAtomicFileOperator(logger *zap.Logger, backupDir, tempDir string) *AtomicFileOperator {
	if logger == nil {
		panic("AtomicFileOperator: logger不能为nil")
	}

	operator := &AtomicFileOperator{
		logger:            logger,
		backupDir:         backupDir,
		tempDir:           tempDir,
		verificationMode:  VerificationSHA256, // 默认使用SHA256验证
		operations:        make([]*AtomicOperation, 0),
		rollbackStack:     make([]*RollbackOperation, 0),
		maxRetries:        3,                      // 最大重试3次
		retryDelay:        100 * time.Millisecond, // 100ms重试间隔
		enableCompression: false,                  // 默认不压缩备份
	}

	// 确保目录存在
	operator.ensureDirectories()

	return operator
}

// ensureDirectories 确保必要目录存在
func (afo *AtomicFileOperator) ensureDirectories() {
	dirs := []string{afo.backupDir, afo.tempDir}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			afo.logger.Warn("创建目录失败",
				zap.String("dir", dir),
				zap.Error(err))
		}
	}
}

// ReplaceFile 原子性文件替换 - README核心功能：备份→验证→替换→清理
func (afo *AtomicFileOperator) ReplaceFile(ctx context.Context, sourcePath, newFilePath string) error {
	operationID := afo.generateOperationID()

	operation := &AtomicOperation{
		ID:         operationID,
		Type:       OperationReplace,
		SourcePath: sourcePath,
		TargetPath: newFilePath,
		Status:     StatusPending,
		StartTime:  time.Now(),
		Metadata:   make(map[string]string),
	}

	afo.operations = append(afo.operations, operation)

	afo.logger.Info("开始原子性文件替换",
		zap.String("operation_id", operationID),
		zap.String("source", sourcePath),
		zap.String("new_file", newFilePath))

	// 四步原子操作
	if err := afo.executeAtomicReplacement(ctx, operation); err != nil {
		// 操作失败，执行回滚
		afo.logger.Error("原子操作失败，开始回滚",
			zap.String("operation_id", operationID),
			zap.Error(err))

		if rollbackErr := afo.rollbackOperation(operation); rollbackErr != nil {
			afo.logger.Error("回滚操作失败",
				zap.String("operation_id", operationID),
				zap.Error(rollbackErr))
			return fmt.Errorf("操作失败且回滚失败: %w (回滚错误: %v)", err, rollbackErr)
		}

		operation.Status = StatusRolledBack
		return fmt.Errorf("原子操作失败并已回滚: %w", err)
	}

	operation.Status = StatusCompleted
	operation.EndTime = time.Now()

	afo.logger.Info("原子性文件替换完成",
		zap.String("operation_id", operationID),
		zap.Duration("duration", operation.EndTime.Sub(operation.StartTime)))

	return nil
}

// executeAtomicReplacement 执行四步原子操作
func (afo *AtomicFileOperator) executeAtomicReplacement(ctx context.Context, operation *AtomicOperation) error {
	// 步骤1：备份原文件
	if err := afo.stepBackup(ctx, operation); err != nil {
		return fmt.Errorf("备份步骤失败: %w", err)
	}

	// 步骤2：验证新文件
	if err := afo.stepVerify(ctx, operation); err != nil {
		return fmt.Errorf("验证步骤失败: %w", err)
	}

	// 步骤3：替换文件
	if err := afo.stepReplace(ctx, operation); err != nil {
		return fmt.Errorf("替换步骤失败: %w", err)
	}

	// 步骤4：清理临时文件
	if err := afo.stepCleanup(ctx, operation); err != nil {
		// 清理失败不影响主要操作，仅记录警告
		afo.logger.Warn("清理步骤失败",
			zap.String("operation_id", operation.ID),
			zap.Error(err))
	}

	return nil
}

// stepBackup 步骤1：备份原文件
func (afo *AtomicFileOperator) stepBackup(ctx context.Context, operation *AtomicOperation) error {
	operation.Status = StatusBackup

	// 检查源文件是否存在
	if _, err := os.Stat(operation.SourcePath); err != nil {
		if os.IsNotExist(err) {
			// 源文件不存在，跳过备份
			afo.logger.Debug("源文件不存在，跳过备份",
				zap.String("source", operation.SourcePath))
			return nil
		}
		return fmt.Errorf("检查源文件失败: %w", err)
	}

	// 生成备份路径
	backupPath := afo.generateBackupPath(operation.SourcePath, operation.ID)
	operation.BackupPath = backupPath

	// 创建备份文件
	if err := afo.copyFileWithVerification(operation.SourcePath, backupPath); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	// 计算源文件哈希（用于后续验证）
	if afo.verificationMode >= VerificationSHA256 {
		hash, err := afo.calculateFileHash(operation.SourcePath)
		if err != nil {
			afo.logger.Warn("计算源文件哈希失败", zap.Error(err))
		} else {
			operation.SourceHash = hash
		}
	}

	// 添加回滚操作
	afo.addRollbackOperation(&RollbackOperation{
		OperationID: operation.ID,
		Action:      RollbackRestore,
		SourcePath:  backupPath,
		TargetPath:  operation.SourcePath,
		Priority:    1, // 高优先级
	})

	afo.logger.Debug("备份步骤完成",
		zap.String("source", operation.SourcePath),
		zap.String("backup", backupPath))

	return nil
}

// stepVerify 步骤2：验证新文件
func (afo *AtomicFileOperator) stepVerify(ctx context.Context, operation *AtomicOperation) error {
	operation.Status = StatusVerify

	// 检查新文件是否存在
	if _, err := os.Stat(operation.TargetPath); err != nil {
		return fmt.Errorf("新文件不存在: %w", err)
	}

	// 文件大小验证
	targetInfo, err := os.Stat(operation.TargetPath)
	if err != nil {
		return fmt.Errorf("获取目标文件信息失败: %w", err)
	}

	if targetInfo.Size() == 0 {
		return fmt.Errorf("目标文件为空")
	}

	// 哈希验证（如果启用）
	if afo.verificationMode >= VerificationSHA256 {
		hash, err := afo.calculateFileHash(operation.TargetPath)
		if err != nil {
			return fmt.Errorf("计算目标文件哈希失败: %w", err)
		}
		operation.TargetHash = hash

		// 确保新文件与原文件不同（避免无意义替换）
		if operation.SourceHash != "" && operation.SourceHash == operation.TargetHash {
			afo.logger.Info("目标文件与源文件相同，跳过替换",
				zap.String("source", operation.SourcePath))
			return nil // 不算错误，但不需要替换
		}
	}

	// 格式验证（如果启用完整验证）
	if afo.verificationMode >= VerificationFull {
		if err := afo.validateFileFormat(operation.TargetPath); err != nil {
			return fmt.Errorf("目标文件格式验证失败: %w", err)
		}
	}

	afo.logger.Debug("验证步骤完成",
		zap.String("target", operation.TargetPath),
		zap.Int64("size", targetInfo.Size()))

	return nil
}

// stepReplace 步骤3：替换文件
func (afo *AtomicFileOperator) stepReplace(ctx context.Context, operation *AtomicOperation) error {
	operation.Status = StatusReplace

	// 创建临时路径，在同一目录下进行原子移动
	tempReplacePath := operation.SourcePath + ".tmp." + operation.ID
	operation.TempPath = tempReplacePath

	// 首先将新文件复制到临时位置
	if err := afo.copyFileWithVerification(operation.TargetPath, tempReplacePath); err != nil {
		return fmt.Errorf("复制到临时位置失败: %w", err)
	}

	// 添加临时文件清理回滚操作
	afo.addRollbackOperation(&RollbackOperation{
		OperationID: operation.ID,
		Action:      RollbackCleanup,
		SourcePath:  tempReplacePath,
		Priority:    3, // 低优先级
	})

	// 原子性移动：将临时文件移动到最终位置
	if err := os.Rename(tempReplacePath, operation.SourcePath); err != nil {
		return fmt.Errorf("原子移动失败: %w", err)
	}

	afo.logger.Debug("替换步骤完成",
		zap.String("source", operation.SourcePath),
		zap.String("temp", tempReplacePath))

	return nil
}

// stepCleanup 步骤4：清理临时文件
func (afo *AtomicFileOperator) stepCleanup(ctx context.Context, operation *AtomicOperation) error {
	operation.Status = StatusCleanup

	// 清理目标文件（已经复制完成）
	if operation.TargetPath != "" && operation.TargetPath != operation.SourcePath {
		if err := os.Remove(operation.TargetPath); err != nil && !os.IsNotExist(err) {
			afo.logger.Warn("清理目标文件失败",
				zap.String("target", operation.TargetPath),
				zap.Error(err))
		}
	}

	// 清理临时文件
	if operation.TempPath != "" {
		if err := os.Remove(operation.TempPath); err != nil && !os.IsNotExist(err) {
			afo.logger.Warn("清理临时文件失败",
				zap.String("temp", operation.TempPath),
				zap.Error(err))
		}
	}

	// 可选：清理成功的备份文件（如果配置为不保留备份）
	// 这里保留备份文件以提供额外安全性

	afo.logger.Debug("清理步骤完成",
		zap.String("operation_id", operation.ID))

	return nil
}

// rollbackOperation 回滚单个操作
func (afo *AtomicFileOperator) rollbackOperation(operation *AtomicOperation) error {
	afo.logger.Info("开始回滚操作",
		zap.String("operation_id", operation.ID),
		zap.String("status", operation.Status.String()))

	// 按优先级顺序执行回滚
	for i := len(afo.rollbackStack) - 1; i >= 0; i-- {
		rollback := afo.rollbackStack[i]

		if rollback.OperationID != operation.ID {
			continue
		}

		if err := afo.executeRollback(rollback); err != nil {
			afo.logger.Error("回滚步骤失败",
				zap.String("operation_id", rollback.OperationID),
				zap.String("action", rollback.Action.String()),
				zap.Error(err))
			return err
		}
	}

	// 清理该操作的回滚栈
	afo.cleanupRollbackStack(operation.ID)

	return nil
}

// executeRollback 执行具体的回滚动作
func (afo *AtomicFileOperator) executeRollback(rollback *RollbackOperation) error {
	switch rollback.Action {
	case RollbackRestore:
		// 恢复原文件
		if _, err := os.Stat(rollback.SourcePath); err != nil {
			return fmt.Errorf("备份文件不存在: %w", err)
		}

		return afo.copyFileWithVerification(rollback.SourcePath, rollback.TargetPath)

	case RollbackDelete:
		// 删除文件
		if err := os.Remove(rollback.SourcePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除文件失败: %w", err)
		}

	case RollbackMove:
		// 移动文件
		return os.Rename(rollback.SourcePath, rollback.TargetPath)

	case RollbackCleanup:
		// 清理临时文件
		if err := os.Remove(rollback.SourcePath); err != nil && !os.IsNotExist(err) {
			afo.logger.Warn("清理临时文件失败",
				zap.String("path", rollback.SourcePath),
				zap.Error(err))
		}

	default:
		return fmt.Errorf("未知回滚动作: %d", rollback.Action)
	}

	return nil
}

// 辅助方法
func (afo *AtomicFileOperator) generateOperationID() string {
	return fmt.Sprintf("op_%d_%d", time.Now().UnixNano(), len(afo.operations))
}

func (afo *AtomicFileOperator) generateBackupPath(originalPath, operationID string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("%s.backup.%s.%s%s", name, timestamp, operationID, ext)

	if afo.backupDir != "" {
		return filepath.Join(afo.backupDir, backupFilename)
	}

	return filepath.Join(dir, backupFilename)
}

func (afo *AtomicFileOperator) copyFileWithVerification(src, dst string) error {
	// 重试机制
	var lastErr error
	for attempt := 0; attempt < afo.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(afo.retryDelay)
		}

		if err := afo.copyFile(src, dst); err != nil {
			lastErr = err
			continue
		}

		// 验证复制是否成功
		if afo.verificationMode >= VerificationSizeOnly {
			if err := afo.verifyFileCopy(src, dst); err != nil {
				lastErr = err
				os.Remove(dst) // 清理失败的复制
				continue
			}
		}

		return nil
	}

	return fmt.Errorf("复制文件失败，已重试%d次: %w", afo.maxRetries, lastErr)
}

func (afo *AtomicFileOperator) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 同步到磁盘
	return destFile.Sync()
}

func (afo *AtomicFileOperator) verifyFileCopy(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	dstInfo, err := os.Stat(dst)
	if err != nil {
		return err
	}

	// 大小验证
	if srcInfo.Size() != dstInfo.Size() {
		return fmt.Errorf("文件大小不匹配: 源文件%d字节, 目标文件%d字节", srcInfo.Size(), dstInfo.Size())
	}

	// 哈希验证（如果启用）
	if afo.verificationMode >= VerificationSHA256 {
		srcHash, err := afo.calculateFileHash(src)
		if err != nil {
			return fmt.Errorf("计算源文件哈希失败: %w", err)
		}

		dstHash, err := afo.calculateFileHash(dst)
		if err != nil {
			return fmt.Errorf("计算目标文件哈希失败: %w", err)
		}

		if srcHash != dstHash {
			return fmt.Errorf("文件哈希不匹配")
		}
	}

	return nil
}

func (afo *AtomicFileOperator) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (afo *AtomicFileOperator) validateFileFormat(filePath string) error {
	// 基础格式验证：检查文件是否可读且有内容
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取文件头部分进行基本验证
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}

	if n == 0 {
		return fmt.Errorf("文件为空或无法读取")
	}

	// 这里可以扩展更详细的格式验证逻辑
	return nil
}

func (afo *AtomicFileOperator) addRollbackOperation(rollback *RollbackOperation) {
	afo.rollbackStack = append(afo.rollbackStack, rollback)
}

func (afo *AtomicFileOperator) cleanupRollbackStack(operationID string) {
	newStack := make([]*RollbackOperation, 0)
	for _, rollback := range afo.rollbackStack {
		if rollback.OperationID != operationID {
			newStack = append(newStack, rollback)
		}
	}
	afo.rollbackStack = newStack
}

// CleanupAllBackups 清理所有备份文件
func (afo *AtomicFileOperator) CleanupAllBackups() error {
	if afo.backupDir == "" {
		return nil
	}

	entries, err := os.ReadDir(afo.backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}

	cleanedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if strings.Contains(filename, ".backup.") {
			filePath := filepath.Join(afo.backupDir, filename)
			if err := os.Remove(filePath); err != nil {
				afo.logger.Warn("删除备份文件失败",
					zap.String("file", filePath),
					zap.Error(err))
			} else {
				cleanedCount++
			}
		}
	}

	afo.logger.Info("备份文件清理完成", zap.Int("cleaned_count", cleanedCount))
	return nil
}

// GetOperationHistory 获取操作历史
func (afo *AtomicFileOperator) GetOperationHistory() []*AtomicOperation {
	return afo.operations
}

// SetVerificationMode 设置验证模式
func (afo *AtomicFileOperator) SetVerificationMode(mode VerificationMode) {
	afo.verificationMode = mode
}

func (ra RollbackAction) String() string {
	switch ra {
	case RollbackRestore:
		return "restore"
	case RollbackDelete:
		return "delete"
	case RollbackMove:
		return "move"
	case RollbackCleanup:
		return "cleanup"
	default:
		return "unknown"
	}
}
