# dynamic2avif - 动态图片转AVIF转换器

`dynamic2avif` 是一款专为图像收藏者和效率追求者设计的命令行工具，旨在将动态图片（GIF, WebP, APNG等）以高质量、安全可靠的方式转换为下一代图像格式AVIF (.avif)。

## 核心功能

- **全自动智能处理:** 无需任何复杂配置，工具以唯一的"全自动模式"运行，智能识别每一种文件并采用最优策略处理。
- **视觉无损转换:** 保证高质量转换，确保您的图片在转换过程中保持优秀的视觉质量。
- **高性能并发处理:** 充分利用现代CPU的多核性能，并发处理多个文件，大幅缩短等待时间。
- **安全可靠:** 采用事务性操作，失败时自动回滚，确保原始文件不受影响。
- **智能错误恢复:** 支持重试机制，网络波动或临时故障不会导致整个任务失败。
- **精确的文件数量验证**: 转换完成后，提供详细的文件数量验证报告，确保处理过程的准确性和可靠性。
- **代码优化** - 消除重复函数，合并重复的 `getFileTimesDarwin` 和 `setFinderDates` 函数定义，提升代码质量和维护性。

## 技术优势

### 智能策略选择

工具会根据文件类型自动选择最优的转换策略：

- **对于 GIF 文件:**
  - **执行高质量转换:** 程序会使用`ffmpeg`的`libsvtav1`编码器进行转换，保留动画信息。
- **对于 WebP 文件:**
  - **执行高质量转换:** 程序会使用`ffmpeg`的`libsvtav1`编码器进行转换，保留动画信息。
- **对于 APNG 文件:**
  - **执行高质量转换:** 程序会使用`ffmpeg`的`libsvtav1`编码器进行转换，保留动画信息。

### AVIF格式优势

1. **高压缩率:** AVIF格式相比GIF/WebP具有更高的压缩率，在保持视觉质量的同时显著减小文件大小。
2. **现代特性支持:** 支持HDR、宽色域、透明度、动画等现代特性。
3. **广泛兼容性:** 现代浏览器和设备都支持AVIF格式。

## 安装要求

### 系统依赖
- Go 1.19 或更高版本
- FFmpeg 4.0 或更高版本（用于图像转换）

### 安装FFmpeg
```bash
# macOS (使用Homebrew)
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg

# Windows (使用Chocolatey)
choco install ffmpeg
```

## 构建项目

### 方法1：使用go build
```bash
cd /path/to/dynamic2avif
go build -o bin/dynamic2avif main.go
```

## 使用方法

可执行文件位于 `bin/dynamic2avif`。详细使用方法请参见 [USAGE_TUTORIAL_ZH.md](../USAGE_TUTORIAL_ZH.md)。

### 基础转换
```bash
# 转换整个目录
./bin/dynamic2avif -input /path/to/images -output /path/to/avif/output
```

### 高级配置
```bash
# 使用高质量设置转换
./bin/dynamic2avif -input /input -output /output -quality 80 -speed 5

# 指定并发线程数
./bin/dynamic2avif -input /input -output /output -workers 4

# 跳过已存在的文件
./bin/dynamic2avif -input /input -output /output -skip-exist
```

### 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-input` | 字符串 | 无 | 输入目录（必需） |
| `-output` | 字符串 | 无 | 输出目录（必需） |
| `-quality` | 整数 | 50 | AVIF质量 (0-100) |
| `-speed` | 整数 | 6 | 编码速度 (0-10) |
| `-workers` | 整数 | CPU核心数 | 并发工作线程数 |
| `-skip-exist` | 布尔 | false | 跳过已存在的文件 |
| `-dry-run` | 布尔 | false | 试运行模式 |
| `-timeout` | 整数 | 120 | 单个文件处理超时秒数 |
| `-retries` | 整数 | 2 | 失败重试次数 |

## 使用示例

### 简单转换
```bash
./dynamic2avif -input ./images -output ./avif_output
```

### 高质量转换
```bash
./dynamic2avif -input ./images -output ./avif_output -quality 80 -speed 4
```

### 限制并发数转换
```bash
./dynamic2avif -input ./images -output ./avif_output -workers 2
```

## 日志解读

程序会在控制台输出处理进度，并在当前目录生成 `dynamic2avif.log` 日志文件。主要日志消息包括：

- `🔄 开始处理`: 开始处理一个文件
- `🎬 检测到动画图像`: 检测到动态图像文件
- `✅ 转换完成`: 文件转换成功
- `❌ 转换失败`: 文件转换失败
- `⏭️  跳过已存在的文件`: 跳过已存在的文件（使用 `-skip-exist` 时）
- `⚠️  动画检测失败`: 动画检测出现问题

## 故障排除

### 常见问题

1. **"command not found: ffmpeg"**
   - 确保FFmpeg已正确安装并在PATH中

2. **转换速度慢**
   - 降低speed参数值（0-3）
   - 减少workers参数值
   - 检查系统资源使用情况

### 支持的文件格式

- **GIF**: .gif (包括动画)
- **WebP**: .webp (包括动画)
- **APNG**: .png (包含动画的PNG)

## 许可证

本项目采用MIT许可证。