# Pixly 媒体转换引擎 - API参考文档

## 📋 目录

- [命令行接口](#命令行接口)
- [配置文件API](#配置文件api)
- [编程接口](#编程接口)
- [REST API](#rest-api)
- [错误代码](#错误代码)
- [示例代码](#示例代码)

---

## 🖥️ 命令行接口

### 基本语法

```bash
pixly [路径] [选项]
```

### 全局选项

#### 基本选项

| 选项 | 短选项 | 类型 | 默认值 | 描述 |
|------|--------|------|--------|------|
| `--mode` | `-m` | string | `auto` | 转换模式: auto, quality, emoji |
| `--output-dir` | `-o` | string | `./output` | 输出目录 |
| `--config` | `-c` | string | `~/.pixly.yaml` | 配置文件路径 |
| `--verbose` | `-v` | bool | `false` | 详细输出 |
| `--quiet` | `-q` | bool | `false` | 静默模式 |
| `--help` | `-h` | bool | `false` | 显示帮助信息 |
| `--version` | | bool | `false` | 显示版本信息 |

#### 转换选项

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `--quality` | int | `85` | JPEG质量 (1-100) |
| `--effort` | int | `7` | JXL编码努力程度 (1-9) |
| `--lossless` | bool | `false` | 无损压缩 |
| `--progressive` | bool | `true` | 渐进式JPEG |
| `--optimize` | bool | `true` | 优化输出 |
| `--strip-metadata` | bool | `false` | 移除元数据 |

#### 并发选项

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `--scan-workers` | int | `4` | 扫描工作线程数 |
| `--conversion-workers` | int | `4` | 转换工作线程数 |
| `--max-memory` | string | `1GB` | 最大内存使用 |
| `--cpu-limit` | float | `0.8` | CPU使用限制 (0.0-1.0) |

#### 输出选项

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `--format` | string | `auto` | 输出格式: auto, jxl, avif, webp |
| `--suffix` | string | `""` | 文件名后缀 |
| `--preserve-structure` | bool | `true` | 保持目录结构 |
| `--overwrite` | bool | `false` | 覆盖现有文件 |
| `--backup` | bool | `false` | 创建备份 |

#### 过滤选项

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `--include` | []string | `[]` | 包含的文件模式 |
| `--exclude` | []string | `[]` | 排除的文件模式 |
| `--min-size` | string | `1KB` | 最小文件大小 |
| `--max-size` | string | `100MB` | 最大文件大小 |
| `--extensions` | []string | `["jpg","png","gif"]` | 支持的扩展名 |

### 使用示例

#### 基本使用

```bash
# 转换当前目录的所有图片
pixly .

# 使用品质模式转换
pixly /path/to/images --mode quality

# 指定输出目录
pixly /input --output-dir /output

# 静默模式运行
pixly /input --quiet
```

#### 高级使用

```bash
# 自定义质量和并发
pixly /input --quality 90 --conversion-workers 8

# 只转换特定格式
pixly /input --extensions jpg,png --format jxl

# 排除某些文件
pixly /input --exclude "*_backup.*,temp/*"

# 限制文件大小范围
pixly /input --min-size 100KB --max-size 50MB
```

#### 配置文件使用

```bash
# 使用自定义配置文件
pixly /input --config /path/to/config.yaml

# 生成默认配置文件
pixly --generate-config > pixly.yaml
```

---

## ⚙️ 配置文件API

### 配置文件格式

配置文件使用YAML格式，支持以下结构：

```yaml
# Pixly 配置文件
version: "1.0"

# 转换设置
conversion:
  mode: "auto"              # 转换模式
  quality: 85               # 默认质量
  effort: 7                 # JXL编码努力程度
  lossless: false           # 无损压缩
  progressive: true         # 渐进式JPEG
  optimize: true            # 优化输出
  strip_metadata: false     # 移除元数据

# 并发设置
concurrency:
  scan_workers: 4           # 扫描工作线程
  conversion_workers: 4     # 转换工作线程
  max_memory: "1GB"         # 最大内存
  cpu_limit: 0.8           # CPU限制
  enable_watchdog: true     # 启用看门狗

# 输出设置
output:
  directory: "./output"     # 输出目录
  format: "auto"           # 输出格式
  suffix: ""               # 文件后缀
  preserve_structure: true  # 保持目录结构
  overwrite: false         # 覆盖文件
  backup: false            # 创建备份

# 工具设置
tools:
  ffmpeg_path: "ffmpeg"     # FFmpeg路径
  ffprobe_path: "ffprobe"   # FFprobe路径
  cjxl_path: "cjxl"         # CJXL路径
  avifenc_path: "avifenc"   # avifenc路径
  exiftool_path: "exiftool" # ExifTool路径
  timeout: "30s"           # 工具超时

# 安全设置
security:
  allowed_paths: []         # 允许的路径
  blocked_paths: []         # 禁止的路径
  max_file_size: "100MB"    # 最大文件大小
  enable_sandbox: false     # 启用沙箱

# 主题设置
theme:
  name: "default"          # 主题名称
  dark_mode: false         # 暗色模式
  colors:
    primary: "#007acc"     # 主色调
    secondary: "#6c757d"   # 次色调
    success: "#28a745"     # 成功色
    warning: "#ffc107"     # 警告色
    error: "#dc3545"       # 错误色

# 问题文件处理
problem_file_handling:
  skip_corrupted: true      # 跳过损坏文件
  skip_low_quality: false   # 跳过低质量文件
  retry_count: 3           # 重试次数
  retry_delay: "1s"        # 重试延迟

# 日志设置
logging:
  level: "info"            # 日志级别
  format: "json"           # 日志格式
  output: "stderr"         # 输出目标
  file: ""                 # 日志文件
  max_size: "100MB"        # 最大文件大小
  max_backups: 3           # 最大备份数
  max_age: 28              # 最大保存天数

# 性能设置
performance:
  enable_profiling: false   # 启用性能分析
  memory_limit: "2GB"      # 内存限制
  gc_percent: 100          # GC百分比
  max_procs: 0             # 最大进程数

# 高级设置
advanced:
  enable_experimental: false # 启用实验功能
  debug_mode: false         # 调试模式
  checkpoint_interval: "5m" # 检查点间隔
  temp_dir: "/tmp"          # 临时目录
```

### 配置验证

#### 配置验证规则

```yaml
# 验证规则
validation:
  conversion:
    quality:
      min: 1
      max: 100
    effort:
      min: 1
      max: 9
  concurrency:
    scan_workers:
      min: 1
      max: 32
    conversion_workers:
      min: 1
      max: 32
    cpu_limit:
      min: 0.1
      max: 1.0
```

#### 配置热重载

```yaml
# 热重载设置
hot_reload:
  enabled: true
  watch_interval: "1s"
  debounce_delay: "500ms"
```

---

## 🔧 编程接口

### Go语言API

#### 核心接口

```go
package pixly

import (
    "context"
    "time"
)

// Converter 转换器接口
type Converter interface {
    // ConvertFiles 转换文件
    ConvertFiles(ctx context.Context, inputPath string) error
    
    // GetStats 获取统计信息
    GetStats() *ConversionStats
    
    // GetResults 获取转换结果
    GetResults() []*ConversionResult
    
    // Stop 停止转换
    Stop() error
    
    // SetProgressCallback 设置进度回调
    SetProgressCallback(callback ProgressCallback)
}

// ProgressCallback 进度回调函数
type ProgressCallback func(current, total int64, message string)

// ConversionOptions 转换选项
type ConversionOptions struct {
    Mode                ConversionMode
    Quality             int
    Effort              int
    Lossless            bool
    Progressive         bool
    Optimize            bool
    StripMetadata       bool
    OutputDir           string
    Format              string
    ScanWorkers         int
    ConversionWorkers   int
    MaxMemory           int64
    CPULimit            float64
}

// NewConverter 创建转换器
func NewConverter(options *ConversionOptions) (Converter, error)

// ConvertWithOptions 使用选项转换
func ConvertWithOptions(ctx context.Context, inputPath string, options *ConversionOptions) (*ConversionStats, error)
```

#### 使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/your-org/pixly"
)

func main() {
    // 创建转换选项
    options := &pixly.ConversionOptions{
        Mode:              pixly.ModeAuto,
        Quality:           85,
        Effort:            7,
        OutputDir:         "./output",
        ScanWorkers:       4,
        ConversionWorkers: 4,
    }
    
    // 创建转换器
    converter, err := pixly.NewConverter(options)
    if err != nil {
        log.Fatal(err)
    }
    
    // 设置进度回调
    converter.SetProgressCallback(func(current, total int64, message string) {
        fmt.Printf("进度: %d/%d - %s\n", current, total, message)
    })
    
    // 创建上下文
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    // 执行转换
    err = converter.ConvertFiles(ctx, "/path/to/images")
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取统计信息
    stats := converter.GetStats()
    fmt.Printf("转换完成: %d/%d 文件成功\n", stats.SuccessfulFiles, stats.TotalFiles)
    fmt.Printf("压缩比: %.2f%%\n", stats.CompressionRatio*100)
}
```

### 配置管理API

```go
// ConfigManager 配置管理器接口
type ConfigManager interface {
    // Load 加载配置
    Load(configFile string) error
    
    // Save 保存配置
    Save() error
    
    // Get 获取配置值
    Get(key string) interface{}
    
    // Set 设置配置值
    Set(key string, value interface{}) error
    
    // Watch 监听配置变化
    Watch(callback ConfigChangeCallback) error
    
    // Validate 验证配置
    Validate() error
}

// ConfigChangeCallback 配置变化回调
type ConfigChangeCallback func(key string, oldValue, newValue interface{})

// 使用示例
func configExample() {
    // 创建配置管理器
    configManager := pixly.NewConfigManager()
    
    // 加载配置文件
    err := configManager.Load("pixly.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取配置值
    quality := configManager.Get("conversion.quality").(int)
    fmt.Printf("当前质量设置: %d\n", quality)
    
    // 设置配置值
    err = configManager.Set("conversion.quality", 90)
    if err != nil {
        log.Fatal(err)
    }
    
    // 监听配置变化
    configManager.Watch(func(key string, oldValue, newValue interface{}) {
        fmt.Printf("配置变化: %s = %v -> %v\n", key, oldValue, newValue)
    })
    
    // 保存配置
    err = configManager.Save()
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## 🌐 REST API

### API端点

#### 转换管理

```http
# 开始转换任务
POST /api/v1/convert
Content-Type: application/json

{
  "input_path": "/path/to/images",
  "options": {
    "mode": "auto",
    "quality": 85,
    "output_dir": "./output"
  }
}

# 响应
{
  "task_id": "task-123456",
  "status": "started",
  "created_at": "2025-01-04T10:30:00Z"
}
```

```http
# 获取任务状态
GET /api/v1/convert/{task_id}

# 响应
{
  "task_id": "task-123456",
  "status": "running",
  "progress": {
    "current": 50,
    "total": 100,
    "percentage": 50.0
  },
  "stats": {
    "processed_files": 50,
    "successful_files": 48,
    "failed_files": 2,
    "compression_ratio": 0.35
  }
}
```

```http
# 停止转换任务
DELETE /api/v1/convert/{task_id}

# 响应
{
  "task_id": "task-123456",
  "status": "stopped",
  "stopped_at": "2025-01-04T10:35:00Z"
}
```

#### 配置管理

```http
# 获取配置
GET /api/v1/config

# 响应
{
  "conversion": {
    "mode": "auto",
    "quality": 85
  },
  "concurrency": {
    "scan_workers": 4,
    "conversion_workers": 4
  }
}
```

```http
# 更新配置
PUT /api/v1/config
Content-Type: application/json

{
  "conversion": {
    "quality": 90
  }
}

# 响应
{
  "message": "配置已更新",
  "updated_at": "2025-01-04T10:30:00Z"
}
```

#### 系统信息

```http
# 获取系统状态
GET /api/v1/status

# 响应
{
  "version": "1.65.6.6",
  "uptime": "2h30m15s",
  "memory_usage": {
    "used": 512000000,
    "total": 8000000000,
    "percentage": 6.4
  },
  "cpu_usage": 25.5,
  "active_tasks": 2
}
```

```http
# 获取支持的格式
GET /api/v1/formats

# 响应
{
  "input_formats": [
    "jpg", "jpeg", "png", "gif", "webp", "bmp", "tiff"
  ],
  "output_formats": [
    "jxl", "avif", "webp", "jpg", "png"
  ]
}
```

### WebSocket API

#### 实时进度更新

```javascript
// 连接WebSocket
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/progress/{task_id}');

// 监听进度更新
ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('进度更新:', data);
    
    // 数据格式
    // {
    //   "type": "progress",
    //   "task_id": "task-123456",
    //   "current": 75,
    //   "total": 100,
    //   "message": "正在处理 image075.jpg"
    // }
};

// 监听任务完成
ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    if (data.type === 'completed') {
        console.log('任务完成:', data.stats);
    }
};
```

---

## ❌ 错误代码

### 系统错误代码

| 代码 | 名称 | 描述 | 解决方案 |
|------|------|------|----------|
| 1000 | `INVALID_INPUT_PATH` | 输入路径无效 | 检查路径是否存在且可访问 |
| 1001 | `PERMISSION_DENIED` | 权限不足 | 检查文件/目录权限 |
| 1002 | `DISK_SPACE_INSUFFICIENT` | 磁盘空间不足 | 清理磁盘空间或更改输出目录 |
| 1003 | `MEMORY_LIMIT_EXCEEDED` | 内存限制超出 | 减少并发数或增加内存限制 |
| 1004 | `CPU_LIMIT_EXCEEDED` | CPU限制超出 | 减少并发数或调整CPU限制 |

### 配置错误代码

| 代码 | 名称 | 描述 | 解决方案 |
|------|------|------|----------|
| 2000 | `CONFIG_FILE_NOT_FOUND` | 配置文件未找到 | 创建配置文件或指定正确路径 |
| 2001 | `CONFIG_PARSE_ERROR` | 配置文件解析错误 | 检查YAML语法 |
| 2002 | `CONFIG_VALIDATION_ERROR` | 配置验证失败 | 检查配置值是否在有效范围内 |
| 2003 | `CONFIG_PERMISSION_ERROR` | 配置文件权限错误 | 检查配置文件读写权限 |

### 转换错误代码

| 代码 | 名称 | 描述 | 解决方案 |
|------|------|------|----------|
| 3000 | `FILE_NOT_SUPPORTED` | 文件格式不支持 | 检查支持的格式列表 |
| 3001 | `FILE_CORRUPTED` | 文件已损坏 | 使用原始文件或修复文件 |
| 3002 | `CONVERSION_FAILED` | 转换失败 | 检查工具安装和文件完整性 |
| 3003 | `OUTPUT_WRITE_ERROR` | 输出写入错误 | 检查输出目录权限和磁盘空间 |
| 3004 | `TOOL_NOT_FOUND` | 转换工具未找到 | 安装所需的转换工具 |
| 3005 | `TOOL_EXECUTION_ERROR` | 工具执行错误 | 检查工具版本和参数 |

### 网络错误代码

| 代码 | 名称 | 描述 | 解决方案 |
|------|------|------|----------|
| 4000 | `API_ENDPOINT_NOT_FOUND` | API端点未找到 | 检查API路径 |
| 4001 | `API_METHOD_NOT_ALLOWED` | HTTP方法不允许 | 使用正确的HTTP方法 |
| 4002 | `API_RATE_LIMIT_EXCEEDED` | API速率限制超出 | 减少请求频率 |
| 4003 | `API_AUTHENTICATION_FAILED` | API认证失败 | 检查认证凭据 |
| 4004 | `API_AUTHORIZATION_FAILED` | API授权失败 | 检查用户权限 |

### 错误响应格式

```json
{
  "error": {
    "code": 3002,
    "name": "CONVERSION_FAILED",
    "message": "转换失败: 无法处理文件 image.jpg",
    "details": {
      "file": "/path/to/image.jpg",
      "tool": "cjxl",
      "exit_code": 1,
      "stderr": "Invalid JPEG file"
    },
    "timestamp": "2025-01-04T10:30:00Z",
    "request_id": "req-123456"
  }
}
```

---

## 📝 示例代码

### 批量转换脚本

```bash
#!/bin/bash
# 批量转换脚本

set -e

# 配置
INPUT_DIR="/path/to/input"
OUTPUT_DIR="/path/to/output"
CONFIG_FILE="./pixly.yaml"

# 检查输入目录
if [ ! -d "$INPUT_DIR" ]; then
    echo "错误: 输入目录不存在: $INPUT_DIR"
    exit 1
fi

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 执行转换
echo "开始转换..."
pixly "$INPUT_DIR" \
    --output-dir "$OUTPUT_DIR" \
    --config "$CONFIG_FILE" \
    --mode auto \
    --quality 85 \
    --conversion-workers 8 \
    --verbose

echo "转换完成!"
```

### Python集成示例

```python
#!/usr/bin/env python3
# Python集成示例

import subprocess
import json
import sys
from pathlib import Path

class PixlyConverter:
    def __init__(self, pixly_path="pixly"):
        self.pixly_path = pixly_path
    
    def convert(self, input_path, output_dir=None, **options):
        """转换文件"""
        cmd = [self.pixly_path, str(input_path)]
        
        if output_dir:
            cmd.extend(["--output-dir", str(output_dir)])
        
        for key, value in options.items():
            if isinstance(value, bool):
                if value:
                    cmd.append(f"--{key.replace('_', '-')}")
            else:
                cmd.extend([f"--{key.replace('_', '-')}", str(value)])
        
        try:
            result = subprocess.run(
                cmd, 
                capture_output=True, 
                text=True, 
                check=True
            )
            return {
                "success": True,
                "stdout": result.stdout,
                "stderr": result.stderr
            }
        except subprocess.CalledProcessError as e:
            return {
                "success": False,
                "error": str(e),
                "stdout": e.stdout,
                "stderr": e.stderr
            }
    
    def get_version(self):
        """获取版本信息"""
        try:
            result = subprocess.run(
                [self.pixly_path, "--version"],
                capture_output=True,
                text=True,
                check=True
            )
            return result.stdout.strip()
        except subprocess.CalledProcessError:
            return None

# 使用示例
if __name__ == "__main__":
    converter = PixlyConverter()
    
    # 检查版本
    version = converter.get_version()
    print(f"Pixly版本: {version}")
    
    # 转换文件
    result = converter.convert(
        input_path="./input",
        output_dir="./output",
        mode="auto",
        quality=85,
        conversion_workers=4,
        verbose=True
    )
    
    if result["success"]:
        print("转换成功!")
        print(result["stdout"])
    else:
        print("转换失败:")
        print(result["error"])
        sys.exit(1)
```

### Node.js集成示例

```javascript
// Node.js集成示例
const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs').promises;

class PixlyConverter {
    constructor(pixlyPath = 'pixly') {
        this.pixlyPath = pixlyPath;
    }
    
    async convert(inputPath, options = {}) {
        return new Promise((resolve, reject) => {
            const args = [inputPath];
            
            // 构建命令行参数
            Object.entries(options).forEach(([key, value]) => {
                const argName = `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
                
                if (typeof value === 'boolean') {
                    if (value) args.push(argName);
                } else {
                    args.push(argName, String(value));
                }
            });
            
            const child = spawn(this.pixlyPath, args);
            
            let stdout = '';
            let stderr = '';
            
            child.stdout.on('data', (data) => {
                stdout += data.toString();
            });
            
            child.stderr.on('data', (data) => {
                stderr += data.toString();
            });
            
            child.on('close', (code) => {
                if (code === 0) {
                    resolve({
                        success: true,
                        stdout,
                        stderr
                    });
                } else {
                    reject(new Error(`转换失败，退出代码: ${code}\n${stderr}`));
                }
            });
            
            child.on('error', (error) => {
                reject(error);
            });
        });
    }
    
    async convertWithProgress(inputPath, options = {}, onProgress) {
        // 实现带进度回调的转换
        const args = [inputPath, '--verbose'];
        
        Object.entries(options).forEach(([key, value]) => {
            const argName = `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
            if (typeof value === 'boolean') {
                if (value) args.push(argName);
            } else {
                args.push(argName, String(value));
            }
        });
        
        return new Promise((resolve, reject) => {
            const child = spawn(this.pixlyPath, args);
            
            let stdout = '';
            
            child.stdout.on('data', (data) => {
                const text = data.toString();
                stdout += text;
                
                // 解析进度信息
                const progressMatch = text.match(/进度: (\d+)\/(\d+)/g);
                if (progressMatch && onProgress) {
                    const [, current, total] = progressMatch[0].match(/(\d+)\/(\d+)/);
                    onProgress({
                        current: parseInt(current),
                        total: parseInt(total),
                        percentage: (parseInt(current) / parseInt(total)) * 100
                    });
                }
            });
            
            child.on('close', (code) => {
                if (code === 0) {
                    resolve({ success: true, stdout });
                } else {
                    reject(new Error(`转换失败，退出代码: ${code}`));
                }
            });
        });
    }
}

// 使用示例
async function main() {
    const converter = new PixlyConverter();
    
    try {
        console.log('开始转换...');
        
        const result = await converter.convertWithProgress(
            './input',
            {
                outputDir: './output',
                mode: 'auto',
                quality: 85,
                conversionWorkers: 4
            },
            (progress) => {
                console.log(`进度: ${progress.percentage.toFixed(1)}% (${progress.current}/${progress.total})`);
            }
        );
        
        console.log('转换完成!');
        console.log(result.stdout);
        
    } catch (error) {
        console.error('转换失败:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = PixlyConverter;
```

### Docker集成示例

```dockerfile
# Dockerfile
FROM golang:1.19-alpine AS builder

# 安装依赖
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制源代码
COPY . .

# 编译
RUN go build -ldflags "-s -w" -o pixly .

# 运行时镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# 创建用户
RUN adduser -D -s /bin/sh pixly

# 设置工作目录
WORKDIR /home/pixly

# 复制二进制文件
COPY --from=builder /app/pixly /usr/local/bin/pixly

# 设置权限
RUN chmod +x /usr/local/bin/pixly

# 切换用户
USER pixly

# 设置入口点
ENTRYPOINT ["pixly"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  pixly:
    build: .
    volumes:
      - ./input:/input:ro
      - ./output:/output
      - ./config:/config:ro
    environment:
      - PIXLY_CONFIG_FILE=/config/pixly.yaml
      - PIXLY_LOG_LEVEL=info
    command: ["/input", "--output-dir", "/output", "--config", "/config/pixly.yaml"]
    restart: unless-stopped
    
  # 可选: Web界面
  pixly-web:
    image: pixly-web:latest
    ports:
      - "8080:8080"
    environment:
      - PIXLY_API_URL=http://pixly:8080
    depends_on:
      - pixly
```

---

*本API参考文档提供了 Pixly 媒体转换引擎的完整接口说明。如需更多信息，请参考技术规格文档和用户指南。*