# EasyMode 媒体转换工具集 v2.2.0

一套强大的Go语言媒体转换工具，支持多种图像和视频格式的批量转换，具备完整的元数据保留、智能性能优化和8层验证系统。

## 🚀 主要特性

- **🎨 多格式支持**: 支持JPG、PNG、GIF、WebP、AVIF、HEIC、TIFF、BMP等主流图像格式
- **🔒 无损转换**: 提供JPEG XL和AVIF两种现代图像格式的无损转换
- **📋 元数据保留**: 完整保留EXIF、IPTC、XMP等元数据信息
- **⚡ 智能性能优化**: 根据系统负载动态调整处理线程数
- **🛡️ 8层验证系统**: 确保转换质量和数据完整性，防止作弊绕过
- **🏞️ Live Photo检测**: 自动识别并跳过Apple Live Photo文件
- **📝 智能日志管理**: 日志轮转和详细处理记录
- **🔧 模块化设计**: 统一的参数解析和验证模块
- **🎯 通用转换器**: 一个工具支持所有转换类型和模式

## 📦 工具列表

### 🎨 图像转换工具
- `all2avif` - 批量转换为AVIF格式
- `all2jxl` - 批量转换为JPEG XL格式
- `static2avif` - 静态图像转AVIF
- `static2jxl` - 静态图像转JPEG XL
- `dynamic2avif` - 动态图像转AVIF
- `dynamic2jxl` - 动态图像转JPEG XL

### 🎬 视频处理工具
- `video2mov` - 视频重新封装为MOV格式

### 🔧 媒体管理工具
- `media_tools` - 元数据管理、文件去重、扩展名标准化
- `universal_converter` - 统一转换工具，支持所有格式和模式

### 文件管理工具
- **deduplicate_media** - 媒体文件去重工具

## 🔧 系统要求

- macOS 10.15+ (支持Apple Silicon和Intel)
- Go 1.19+
- 依赖工具：
  - `ffmpeg` - 视频/音频处理
  - `exiftool` - 元数据处理
  - `magick` (ImageMagick) - 图像处理
  - `cjxl`/`djxl` - JPEG XL编解码器

## 📦 安装依赖

### 使用Homebrew安装依赖工具：
```bash
brew install ffmpeg exiftool imagemagick libjxl
```

## 🛠️ 构建和安装

每个工具都可以独立构建：

```bash
# 构建单个工具
cd all2avif
./build.sh

# 或手动构建
go build -o bin/all2avif main.go
```

## 📖 使用说明

### all2avif - 批量转AVIF
```bash
./all2avif/bin/all2avif -dir /path/to/images -output /path/to/output
```

### all2jxl - 批量转JPEG XL
```bash
./all2jxl/bin/all2jxl -dir /path/to/images -output /path/to/output
```

### universal_converter - 统一转换
```bash
./universal_converter/bin/universal_converter -mode all -type jxl -input /path/to/images -output /path/to/output
```

### merge_xmp - XMP元数据合并
```bash
./merge_xmp/bin/merge_xmp -dir /path/to/media
```

### deduplicate_media - 文件去重
```bash
./deduplicate_media/bin/deduplicate_media -dir /path/to/media -trash-dir /path/to/trash
```

## 🔒 安全特性

所有工具都包含以下安全验证机制：

- **路径安全验证** - 防止路径遍历攻击
- **文件类型验证** - 验证文件扩展名与实际内容匹配
- **文件大小验证** - 防止处理异常大小的文件
- **XMP格式验证** - 验证XMP文件格式的有效性
- **参数范围验证** - 限制输入参数在合理范围内
- **8层验证系统** - 质量优先的转换结果验证，防止作弊绕过
- **Live Photo跳过** - 自动跳过苹果Live Photos

## 📊 性能特性

- **并发处理** - 多线程并行处理
- **内存优化** - 流式处理大文件
- **进度监控** - 实时处理进度显示
- **错误恢复** - 自动重试机制
- **智能线程调整** - 基于系统负载动态调整线程数
- **处理优先级** - 优先处理JPEG等常见格式

## 🎯 使用场景

- **摄影师** - 批量处理RAW图像，转换格式
- **设计师** - 优化图像文件大小，保持质量
- **内容创作者** - 视频格式转换, 元数据管理
- **系统管理员** - 文件去重, 存储优化

## 📝 版本历史

### v2.2.0 (最新)
- 添加统一转换工具
- 增强文件类型检测
- 添加8层验证系统
- 智能性能优化
- Live Photo检测和跳过
- 日志轮转管理
- 更多简体中文注释

### v2.1.0
- 增强安全验证机制
- 改进错误处理和日志记录
- 优化性能和内存使用
- 添加XMP格式验证
- 完善文档和示例

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这些工具。

## 📄 许可证

MIT License - 详见各工具的LICENSE文件。

## 🔗 相关链接

- [JPEG XL官方网站](https://jpeg.org/jpegxl/)
- [AVIF格式规范](https://aomediacodec.github.io/av1-avif/)
- [ExifTool文档](https://exiftool.org/)
- [FFmpeg文档](https://ffmpeg.org/documentation.html)