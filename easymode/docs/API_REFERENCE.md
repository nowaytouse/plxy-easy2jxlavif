# EasyMode API 参考文档

## 概述

本文档详细介绍了EasyMode工具集的API接口、数据结构、函数签名和使用示例。

## 📦 核心模块

### utils/parameters.go - 参数处理模块

#### 类型定义

```go
// ConversionType 转换类型枚举
type ConversionType int

const (
    ConvertToAVIF ConversionType = iota // 转换为AVIF格式
    ConvertToJXL                        // 转换为JPEG XL格式
    ConvertToMOV                        // 转换为MOV格式
)

// ProcessingMode 处理模式枚举
type ProcessingMode int

const (
    ProcessAll ProcessingMode = iota // 处理所有文件类型
    ProcessStatic                    // 仅处理静态图像
    ProcessDynamic                   // 仅处理动态图像
    ProcessVideo                     // 仅处理视频文件
)
```

#### UniversalOptions 结构体

```go
type UniversalOptions struct {
    // 基础参数
    InputDir       string // 输入目录路径
    OutputDir      string // 输出目录路径
    Workers        int    // 工作线程数
    DryRun         bool   // 试运行模式
    SkipExist      bool   // 跳过已存在文件
    Retries        int    // 重试次数
    TimeoutSeconds int    // 超时时间

    // 转换参数
    ConversionType ConversionType // 转换类型
    ProcessingMode ProcessingMode // 处理模式

    // 质量参数
    Quality     int // 输出质量 (1-100)
    Speed       int // 处理速度 (1-10)
    CJXLThreads int // CJXL线程数

    // 验证参数
    StrictMode     bool    // 严格模式
    AllowTolerance float64 // 允许误差
    CopyMetadata   bool    // 复制元数据
    PreserveTimes  bool    // 保留时间戳
}
```

#### 主要函数

```go
// DefaultOptions 获取默认配置
func DefaultOptions() UniversalOptions

// ParseUniversalFlags 解析命令行参数
func ParseUniversalFlags() UniversalOptions

// Validate 验证参数有效性
func (opts *UniversalOptions) Validate() error

// GetOutputExtension 获取输出文件扩展名
func (opts *UniversalOptions) GetOutputExtension() string

// GetConversionCommand 获取转换命令
func (opts *UniversalOptions) GetConversionCommand(inputPath, outputPath string) (string, []string, error)

// IsSupportedInputFormat 检查是否为支持的输入格式
func (opts *UniversalOptions) IsSupportedInputFormat(filePath string) bool
```

### utils/validation.go - 8层验证系统

#### 核心结构

```go
// ValidationResult 验证结果
type ValidationResult struct {
    Success   bool                   // 是否成功
    Message   string                 // 消息
    Details   map[string]interface{} // 详细信息
    Layer     int                    // 验证层级
    LayerName string                 // 层级名称
}

// ValidationOptions 验证选项
type ValidationOptions struct {
    TimeoutSeconds int     // 超时时间
    CJXLThreads    int     // CJXL线程数
    StrictMode     bool    // 严格模式
    AllowTolerance float64 // 允许误差
}

// EightLayerValidator 8层验证器
type EightLayerValidator struct {
    options ValidationOptions
}
```

#### 主要函数

```go
// NewEightLayerValidator 创建8层验证器
func NewEightLayerValidator(options ValidationOptions) *EightLayerValidator

// ValidateConversion 执行8层验证
func (v *EightLayerValidator) ValidateConversion(originalPath, convertedPath string, fileType EnhancedFileType) (*ValidationResult, error)
```

#### 验证层级

| 层级 | 名称 | 功能描述 |
|------|------|----------|
| 1 | 基础文件验证 | 检查文件存在性、可读性、权限 |
| 2 | 文件大小验证 | 验证转换前后文件大小合理性 |
| 3 | 格式完整性验证 | 使用专业工具验证文件格式 |
| 4 | 元数据验证 | 检查EXIF、IPTC、XMP元数据 |
| 5 | 像素数据验证 | 验证图像像素数据完整性 |
| 6 | 色彩空间验证 | 检查色彩空间转换正确性 |
| 7 | 压缩质量验证 | 验证压缩参数和视觉效果 |
| 8 | 性能验证 | 检查处理时间和资源使用 |

### utils/post_validation.go - 转换后验证

#### 核心结构

```go
// PostValidationResult 验证结果
type PostValidationResult struct {
    TotalFiles      int                    // 总文件数
    SampledFiles    int                    // 抽样文件数
    PassedFiles     int                    // 通过验证的文件数
    FailedFiles     int                    // 未通过验证的文件数
    ValidationItems []ValidationItemResult  // 每个文件的验证结果
    Summary         string                 // 验证摘要
}

// ValidationItemResult 单个文件验证结果
type ValidationItemResult struct {
    OriginalPath  string   // 原始文件路径
    ConvertedPath string   // 转换后文件路径
    FileType      string   // 文件类型
    Passed        bool     // 是否通过验证
    Checks        []string // 检查项列表
    Issues        []string // 发现的问题
}

// MediaProperties 媒体属性
type MediaProperties struct {
    Width      int     // 宽度
    Height     int     // 高度
    FrameCount int     // 帧数
    FPS        float64 // 帧率
    Duration   float64 // 时长
    Format     string  // 格式
}
```

#### 主要函数

```go
// NewPostValidator 创建转换后验证器
func NewPostValidator(sampleRate float64, minSamples, maxSamples int) *PostValidator

// ValidateConversions 验证转换结果
func (pv *PostValidator) ValidateConversions(pairs []FilePair) *PostValidationResult

// validateAnimated 验证动图
func (pv *PostValidator) validateAnimated(result *ValidationItemResult, orig, conv *MediaProperties)

// validateVideo 验证视频
func (pv *PostValidator) validateVideo(result *ValidationItemResult, orig, conv *MediaProperties)

// validateStatic 验证静图
func (pv *PostValidator) validateStatic(result *ValidationItemResult, orig, conv *MediaProperties)
```

### utils/filetype_enhanced.go - 文件类型检测

#### 核心结构

```go
// EnhancedFileType 增强文件类型
type EnhancedFileType struct {
    Extension    string // 文件扩展名
    MimeType     string // MIME类型
    IsAnimated   bool   // 是否为动画
    IsVideo      bool   // 是否为视频
    IsStatic     bool   // 是否为静态图像
    Priority     int    // 处理优先级
}
```

#### 主要函数

```go
// DetectFileType 检测文件类型
func DetectFileType(filePath string) (EnhancedFileType, error)

// IsImageFile 检查是否为图像文件
func IsImageFile(filePath string) bool

// IsVideoFile 检查是否为视频文件
func IsVideoFile(filePath string) bool

// IsAnimatedFile 检查是否为动画文件
func IsAnimatedFile(filePath string) bool
```

## 🔧 使用示例

### 基本转换

```go
package main

import (
    "fmt"
    "pixly/utils"
)

func main() {
    // 创建默认配置
    opts := utils.DefaultOptions()
    
    // 设置输入目录
    opts.InputDir = "/path/to/images"
    
    // 设置转换类型
    opts.ConversionType = utils.ConvertToJXL
    
    // 设置处理模式
    opts.ProcessingMode = utils.ProcessAll
    
    // 验证配置
    if err := opts.Validate(); err != nil {
        fmt.Printf("配置错误: %v\n", err)
        return
    }
    
    // 获取转换命令
    cmd, args, err := opts.GetConversionCommand("input.jpg", "output.jxl")
    if err != nil {
        fmt.Printf("获取转换命令失败: %v\n", err)
        return
    }
    
    fmt.Printf("转换命令: %s %v\n", cmd, args)
}
```

### 文件类型检测

```go
package main

import (
    "fmt"
    "pixly/utils"
)

func main() {
    // 检测文件类型
    fileType, err := utils.DetectFileType("image.gif")
    if err != nil {
        fmt.Printf("检测失败: %v\n", err)
        return
    }
    
    fmt.Printf("文件类型: %s\n", fileType.Extension)
    fmt.Printf("是否为动画: %t\n", fileType.IsAnimated)
    fmt.Printf("是否为视频: %t\n", fileType.IsVideo)
}
```

### 验证系统使用

```go
package main

import (
    "fmt"
    "pixly/utils"
)

func main() {
    // 创建验证器
    validator := utils.NewEightLayerValidator(utils.ValidationOptions{
        TimeoutSeconds: 30,
        CJXLThreads:    4,
        StrictMode:     true,
        AllowTolerance: 0.1,
    })
    
    // 执行验证
    result, err := validator.ValidateConversion("input.jpg", "output.jxl", fileType)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }
    
    if result.Success {
        fmt.Printf("验证通过: %s\n", result.Message)
    } else {
        fmt.Printf("验证失败: %s\n", result.Message)
    }
}
```

### 转换后验证

```go
package main

import (
    "fmt"
    "pixly/utils"
)

func main() {
    // 创建转换后验证器
    validator := utils.NewPostValidator(0.1, 5, 20) // 10%抽样率
    
    // 准备文件对
    pairs := []utils.FilePair{
        {OriginalPath: "input1.jpg", ConvertedPath: "output1.jxl"},
        {OriginalPath: "input2.png", ConvertedPath: "output2.jxl"},
    }
    
    // 执行验证
    result := validator.ValidateConversions(pairs)
    
    fmt.Printf("验证结果: %s\n", result.Summary)
    fmt.Printf("通过率: %.1f%%\n", float64(result.PassedFiles)/float64(result.SampledFiles)*100)
}
```

## 🎬 动图处理API

### 动图转换

```go
// 动图转换配置
opts := utils.DefaultOptions()
opts.ConversionType = utils.ConvertToJXL
opts.ProcessingMode = utils.ProcessDynamic
opts.Quality = 100 // 无损压缩

// 获取动图转换命令
cmd, args, err := opts.GetConversionCommand("animation.gif", "animation.jxl")
// 返回: "cjxl", ["animation.gif", "-d", "0", "-e", "7", "--num_threads", "4", "--container=1", "animation.jxl"], nil
```

### 动图验证

```go
// 动图验证配置
validator := utils.NewPostValidator(0.1, 5, 20)

// 验证动图转换
result := validator.ValidateConversions(animationPairs)

// 检查动图特性
for _, item := range result.ValidationItems {
    if item.FileType == "animated" {
        // 执行动图特定验证
        // 1. 分辨率检查
        // 2. 帧数检查
        // 3. 帧率检查
        // 4. 动图特性验证
    }
}
```

## 📊 性能监控API

### 统计信息

```go
// 处理统计结构
type ProcessingStats struct {
    Processed       int              // 成功处理数量
    Failed          int              // 失败数量
    Skipped         int              // 跳过数量
    TotalSizeBefore int64            // 处理前总大小
    TotalSizeAfter  int64            // 处理后总大小
    DetailedLogs    []FileProcessInfo // 详细日志
    StartTime       time.Time        // 开始时间
}

// 获取处理统计
func (s *ProcessingStats) GetStats() map[string]interface{} {
    return map[string]interface{}{
        "processed": s.Processed,
        "failed": s.Failed,
        "skipped": s.Skipped,
        "compression_ratio": float64(s.TotalSizeAfter) / float64(s.TotalSizeBefore),
        "processing_time": time.Since(s.StartTime),
    }
}
```

### 性能指标

```go
// 性能指标结构
type PerformanceMetrics struct {
    FilesPerSecond    float64 // 每秒处理文件数
    BytesPerSecond    int64   // 每秒处理字节数
    AverageFileTime   time.Duration // 平均文件处理时间
    MemoryUsage       int64   // 内存使用量
    CPUUsage          float64 // CPU使用率
}
```

## 🔍 错误处理

### 错误类型

```go
// 自定义错误类型
type ConversionError struct {
    FilePath    string
    ErrorType   string
    Message     string
    RetryCount  int
    Timestamp   time.Time
}

// 错误处理方法
func (e *ConversionError) Error() string {
    return fmt.Sprintf("转换失败 [%s]: %s (重试次数: %d)", e.ErrorType, e.Message, e.RetryCount)
}
```

### 错误恢复

```go
// 错误恢复策略
func HandleConversionError(err error, filePath string, retryCount int) error {
    switch {
    case strings.Contains(err.Error(), "timeout"):
        return &ConversionError{FilePath: filePath, ErrorType: "timeout", RetryCount: retryCount}
    case strings.Contains(err.Error(), "memory"):
        return &ConversionError{FilePath: filePath, ErrorType: "memory", RetryCount: retryCount}
    default:
        return &ConversionError{FilePath: filePath, ErrorType: "unknown", RetryCount: retryCount}
    }
}
```

## 📝 日志API

### 日志配置

```go
// 日志配置结构
type LogConfig struct {
    Level      string // 日志级别
    FilePath   string // 日志文件路径
    MaxSize    int64  // 最大文件大小
    MaxBackups int    // 最大备份数
    MaxAge     int    // 最大保存天数
}

// 创建日志记录器
func NewLogger(config LogConfig) (*log.Logger, error) {
    // 实现日志轮转和级别控制
}
```

### 日志记录

```go
// 结构化日志记录
func LogConversionStart(filePath string, fileSize int64) {
    logger.Printf("🔄 开始转换: %s (大小: %d bytes)", filePath, fileSize)
}

func LogConversionSuccess(filePath string, processingTime time.Duration) {
    logger.Printf("✅ 转换成功: %s (耗时: %v)", filePath, processingTime)
}

func LogConversionError(filePath string, err error) {
    logger.Printf("❌ 转换失败: %s (错误: %v)", filePath, err)
}
```

---

**文档版本**: v2.2.0  
**最后更新**: 2025-10-24  
**维护者**: AI Assistant
