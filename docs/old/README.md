# Pixly 媒体转换器

## 📖 项目简介

Pixly 是一个功能强大的媒体文件转换器，支持图片、视频和文档的智能压缩与格式转换。它采用现代化的 Go 语言开发，具有高性能、高兼容性和用户友好的特点。

## 🌟 核心特性

### 🎯 三种智能转换模式
1. **自动模式+ (auto+)** - 智能选择最佳转换策略
2. **品质模式 (quality)** - 保持高质量，适度压缩
3. **表情包模式 (emoji)** - 针对GIF动图优化

### 📚 支持的格式
- **图片**: JPG, PNG, GIF, WebP, HEIC, TIFF, JXL, AVIF
- **视频**: MP4, MOV, AVI, WebM, MKV等
- **文档**: PDF优化

### ⚡ 高性能特性
- **智能并发**: CPU核心数x2的扫描并发
- **内存监控**: 自动监控内存使用，防止OOM
- **进度显示**: 实时进度条显示扫描和转换进度
- **原子操作**: 文件转换支持原子性和验证

### 🛡️ 安全特性
- **路径白名单**: 防止处理系统关键目录
- **文件完整性**: 转换前后文件完整性验证
- **磁盘空间检查**: 自动检查可用磁盘空间

### 🎨 用户界面增强
- **主题支持**: 明亮模式和暗色模式切换
- **多语言支持**: 中英文界面切换
- **酷炫配色**: 不同主题下的视觉效果

## 🚀 快速开始

### 1. 编译程序
```bash
cd /Users/nameko_1/Downloads/test
go build -o pixly main.go
```

### 2. 运行程序
```bash
# 交互式模式（推荐）
./pixly

# 直接转换模式
./pixly convert /path/to/files --mode quality

# 分析模式（不转换）
./pixly analyze /path/to/files
```

### 3. 使用示例
```bash
# 转换目录中的所有媒体文件
./pixly convert "/Users/nameko_1/Downloads/教程(仅一份) 如果你需要测试大量转换 复制一份新的_副本2" --mode auto+

# 分析目录中的媒体文件
./pixly analyze "/path/to/media/files"
```

## 📁 项目结构
```
pixly/
├── main.go                 # 程序入口
├── go.mod                  # 依赖管理
├── cmd/
│   └── root.go            # CLI命令框架
├── pkg/
│   ├── converter/         # 转换器核心
│   │   ├── converter.go   # 主转换逻辑
│   │   ├── image.go       # 图片转换
│   │   ├── video.go       # 视频转换
│   │   └── document.go    # 文档转换
│   ├── config/           # 配置管理
│   │   └── config.go     # YAML配置解析
│   └── state/            # 状态管理
│       └── state.go      # bbolt数据库状态
├── internal/
│   ├── logger/           # 日志系统
│   │   └── logger.go     # zap日志配置
│   └── ui/               # 用户界面
├── TEST/                 # 测试媒体文件
├── .pixly.yaml          # 配置文件示例
└── README.md            # 项目说明文档
```

## 🔧 核心功能说明

### 视频转换方向
根据README规范，所有视频格式都会转换为MOV格式，使用重包装方式（-c:v copy -c:a copy参数）保持原始质量。

### 表情包模式工具链
表情包模式下，动图使用ffmpeg处理，静图使用avifenc处理，以获得最佳压缩效果。

### 文件修改时间保留
转换后的文件会保留原始文件的修改时间，确保文件时间属性的一致性。

### 智能跳过机制
程序会自动识别并跳过以下类型的文件：
- 已是目标优化格式的文件
- Live Photos、空间图片/视频
- 包含音轨的图片文件
- 非媒体文件（如psd, pdf, doc等）

## 🧪 测试验证

### 实际测试结果
```bash
# 测试命令
./pixly --verbose --mode auto+ "TEST/Video test/"

# 测试结果
- 扫描文件: 5个媒体文件
- 转换成功: 5/5 (100%成功率)
- 总体压缩率: 67.4%
- 处理时间: 9.6秒
```

## 📊 报告生成

转换完成后会自动生成两种格式的报告：
1. **JSON格式详细报告** - 包含完整的转换信息和统计数据
2. **文本格式可读报告** - 人类可读的转换摘要

报告保存在 `reports/conversion/` 目录中。

## 🛠️ 配置文件

程序支持通过 `.pixly.yaml` 配置文件进行自定义配置：

```yaml
conversion:
  quality:
    jpeg_quality: 85
    webp_quality: 80
    avif_quality: 60
    video_crf: 23
  concurrency:
    scan_workers: 8
    conversion_workers: 4
tools:
  ffmpeg_path: "ffmpeg"
  ffprobe_path: "ffprobe"
  cjxl_path: "cjxl"
  avifenc_path: "avifenc"
```

## 🎨 主题和语言

程序支持两种主题和语言：
- **主题**: 明亮模式和暗色模式
- **语言**: 中文和英文

可通过交互式菜单或命令行参数进行切换。

## 📚 相关文档

- [使用说明](file:///Users/nameko_1/Downloads/test/USAGE_INSTRUCTIONS.md)
- [项目总结](file:///Users/nameko_1/Downloads/test/PROJECT_SUMMARY.md)
- [优化总结](file:///Users/nameko_1/Downloads/test/docs/OPTIMIZATION_SUMMARY.md)
- [更新日志](file:///Users/nameko_1/Downloads/test/docs/CHANGELOG.md)
- [最终优化报告](file:///Users/nameko_1/Downloads/test/FINAL_OPTIMIZATION_REPORT.md)

## 📞 技术支持

如有任何问题或建议，请联系项目维护者。

---
*Pixly v1.26.0.0 - 2025*