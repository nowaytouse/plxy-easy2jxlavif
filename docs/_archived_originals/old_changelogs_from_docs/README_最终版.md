# 🎨 Pixly 智能图像转换工具套件

> **现代化图像格式转换解决方案** - 支持 JXL 和 AVIF 格式的智能批量转换工具

[![版本](https://img.shields.io/badge/版本-v2.1.0-blue.svg)](https://github.com/your-repo)
[![Go版本](https://img.shields.io/badge/Go-1.21+-green.svg)](https://golang.org)
[![许可证](https://img.shields.io/badge/许可证-MIT-yellow.svg)](LICENSE)

## 📋 目录

- [🎯 项目概述](#-项目概述)
- [✨ 核心特性](#-核心特性)
- [🏗️ 项目结构](#️-项目结构)
- [🚀 快速开始](#-快速开始)
- [📖 详细使用指南](#-详细使用指南)
- [🔧 工具组件](#-工具组件)
- [⚙️ 配置选项](#️-配置选项)
- [📊 性能优化](#-性能优化)
- [🛡️ 安全策略](#️-安全策略)
- [📈 更新日志](#-更新日志)
- [🤝 贡献指南](#-贡献指南)
- [📄 许可证](#-许可证)

## 🎯 项目概述

**Pixly** 是一套完整的现代化图像格式转换解决方案，专为处理大量图像文件而设计。支持将传统图像格式（JPG、PNG、GIF等）智能转换为下一代图像格式（JXL、AVIF），在保持图像质量的同时显著减少文件大小。

### 🌟 主要优势

- **🎯 智能策略**: 自动分析图像特征，选择最优转换格式
- **⚡ 高性能**: 多线程并发处理，支持大规模批量转换
- **🛡️ 安全可靠**: 多重验证机制，确保转换质量
- **🎨 用户友好**: 美观的界面设计，丰富的交互体验
- **🔧 高度可配置**: 支持多种质量模式和安全策略

## ✨ 核心特性

### 🧠 智能转换引擎

- **自动格式选择**: 根据图像类型智能选择 JXL 或 AVIF 格式
- **质量评估**: 基于文件大小和内容特征进行质量分析
- **尝试引擎**: 测试不同参数组合，找到最佳转换策略
- **无损转换**: 支持无损和有损转换模式

### 🎛️ 多种工作模式

- **质量模式**: `high`、`medium`、`low`、`auto`
- **表情包模式**: 优化小文件处理
- **交互模式**: 美观的命令行界面
- **批处理模式**: 非交互式批量处理

### 🛡️ 安全与可靠性

- **多重验证**: 转换前后文件完整性验证
- **元数据保护**: 完整保留原始文件元数据
- **备份机制**: 可选的原始文件备份
- **错误恢复**: 智能错误处理和重试机制

## 🏗️ 项目结构

```
easy2jxlavif-beta/
├── 📁 cmd/                          # 启动器
│   └── launcher.go
├── 📁 config/                       # 配置管理
│   └── config.go
├── 📁 easymode/                     # 核心转换工具
│   ├── 📁 all2jxl/                 # JXL 转换工具
│   │   ├── main.go                 # 主程序
│   │   ├── build.sh               # 构建脚本
│   │   ├── README.md              # 说明文档
│   │   └── bin/                   # 编译输出
│   ├── 📁 all2avif/               # AVIF 转换工具
│   │   ├── main.go               # 主程序
│   │   ├── build.sh             # 构建脚本
│   │   ├── README.md            # 说明文档
│   │   └── bin/                 # 编译输出
│   └── 📁 all2jxl/               # 原始 JXL 工具
├── 📁 pkg/                        # 核心功能包
│   ├── 📁 atomic/                # 原子操作
│   ├── 📁 batchdecision/         # 批处理决策
│   ├── 📁 concurrency/           # 并发控制
│   ├── 📁 conversion/            # 转换引擎
│   ├── 📁 engine/                # 处理引擎
│   ├── 📁 errorhandling/         # 错误处理
│   ├── 📁 monitor/               # 监控系统
│   └── 📁 ui/                    # 用户界面
├── 📁 tests/                      # 测试套件
│   ├── 📁 automation/            # 自动化测试
│   ├── 📁 benchmark/             # 性能测试
│   ├── 📁 integration/           # 集成测试
│   └── 📁 unit/                  # 单元测试
├── main.go                       # Pixly 主程序
├── pixly                        # 编译后的主程序
└── README_最终版.md              # 本文档
```

## 🚀 快速开始

### 📋 系统要求

- **操作系统**: macOS、Linux、Windows
- **Go 版本**: 1.21 或更高版本
- **依赖工具**: 
  - `cjxl` / `djxl` (JPEG XL 工具)
  - `avifenc` (AVIF 编码器)
  - `ffmpeg` (视频处理)
  - `exiftool` (元数据处理)
  - `ImageMagick` (图像预处理)

### 🔧 安装步骤

1. **克隆项目**
   ```bash
   git clone https://github.com/your-repo/easy2jxlavif-beta.git
   cd easy2jxlavif-beta
   ```

2. **安装依赖工具**
   ```bash
   # macOS
   brew install libjxl avifenc ffmpeg exiftool imagemagick
   
   # Ubuntu/Debian
   sudo apt install libjxl-tools avif-tools ffmpeg exiftool imagemagick
   ```

3. **构建项目**
   ```bash
   # 构建主程序
   go build -o pixly main.go
   
   # 构建转换工具
   cd easymode/all2jxl && ./build.sh
   cd ../all2avif && ./build.sh
   ```

4. **验证安装**
   ```bash
   ./pixly -help
   ```

### 🎯 快速使用

```bash
# 智能转换（推荐）
./pixly -dir /path/to/images

# 高质量模式
./pixly -dir /path/to/images -quality high

# 表情包模式
./pixly -dir /path/to/images -sticker

# 非交互模式
./pixly -dir /path/to/images -non-interactive
```

## 📖 详细使用指南

### 🎛️ 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-dir` | string | - | 目标目录路径 |
| `-quality` | string | auto | 质量模式: auto, high, medium, low |
| `-format` | string | auto | 输出格式: jxl, avif, auto |
| `-sticker` | bool | false | 启用表情包模式 |
| `-non-interactive` | bool | false | 非交互模式 |
| `-emoji` | bool | true | 启用表情符号 |
| `-try-engine` | bool | true | 启用尝试引擎 |
| `-security` | string | medium | 安全级别: high, medium, low |

### 🎯 使用场景

#### 1. 个人照片管理
```bash
# 高质量照片转换
./pixly -dir ~/Pictures -quality high -security high
```

#### 2. 表情包处理
```bash
# 表情包优化
./pixly -dir ~/Stickers -sticker -quality medium
```

#### 3. 批量处理
```bash
# 非交互式批量处理
./pixly -dir /data/images -non-interactive -format jxl
```

#### 4. 专业工作流
```bash
# 高安全级别处理
./pixly -dir /work/images -quality high -security high -format auto
```

## 🔧 工具组件

### 🎨 Pixly 主程序

**功能**: 智能转换协调器
- 智能格式选择
- 用户界面管理
- 配置管理
- 进度监控

**特点**:
- 🧠 智能策略引擎
- 🎯 自动格式选择
- 🎨 美观的用户界面
- ⚙️ 灵活的配置选项

### 📸 all2jxl 工具

**功能**: JPEG XL 格式转换
- 支持静态图像转换
- 无损和有损转换
- 元数据保护
- 批量处理

**支持格式**:
- 输入: JPG, PNG, GIF, BMP, TIFF, WebP, HEIC
- 输出: JXL

**特点**:
- 🔄 无损转换支持
- 📊 智能压缩优化
- 🛡️ 质量验证机制
- ⚡ 多线程处理

### 🎬 all2avif 工具

**功能**: AVIF 格式转换
- 支持静态和动态图像
- 动画 GIF 转换
- 高质量压缩
- 现代格式优化

**支持格式**:
- 输入: JPG, PNG, GIF, WebP, HEIC
- 输出: AVIF

**特点**:
- 🎬 动画支持
- 🎯 现代格式优化
- 📱 移动设备友好
- 🌐 Web 标准兼容

## ⚙️ 配置选项

### 📝 配置文件

配置文件位置: `~/.pixly/config.json`

```json
{
  "quality_mode": "auto",
  "emoji_mode": true,
  "interactive": true,
  "output_format": "auto",
  "replace_originals": true,
  "create_backup": true,
  "sticker_mode": false,
  "try_engine": true,
  "security_level": "medium"
}
```

### 🎛️ 质量模式详解

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `auto` | 自动选择 | 通用场景 |
| `high` | 高质量 | 专业摄影、重要文档 |
| `medium` | 中等质量 | 日常使用、社交媒体 |
| `low` | 低质量 | 快速预览、临时文件 |

### 🛡️ 安全级别

| 级别 | 说明 | 特性 |
|------|------|------|
| `high` | 高安全 | 完整备份、严格验证 |
| `medium` | 中等安全 | 标准验证、可选备份 |
| `low` | 低安全 | 基础验证、快速处理 |

## 📊 性能优化

### ⚡ 性能特性

- **多线程处理**: 自动检测 CPU 核心数，优化并发
- **内存管理**: 智能内存分配，避免内存泄漏
- **资源限制**: 防止系统过载的保护机制
- **超时控制**: 防止单个任务阻塞整个流程

### 📈 性能指标

| 指标 | 数值 | 说明 |
|------|------|------|
| 并发线程 | CPU核心数 | 自动优化 |
| 内存使用 | < 2GB | 智能控制 |
| 处理速度 | 5-10文件/秒 | 取决于文件大小 |
| 压缩率 | 30-70% | 根据图像内容 |

### 🔧 优化建议

1. **大文件处理**: 使用 `-sample` 参数进行小规模测试
2. **内存优化**: 处理大量文件时使用 `-non-interactive` 模式
3. **质量平衡**: 根据需求选择合适的质量模式
4. **安全策略**: 重要文件使用高安全级别

## 🛡️ 安全策略

### 🔒 安全特性

- **文件验证**: 转换前后完整性检查
- **元数据保护**: 完整保留原始元数据
- **备份机制**: 可选的原始文件备份
- **错误恢复**: 智能错误处理和重试
- **权限控制**: 安全的文件操作权限

### 🚨 安全警告

- ⚠️ **备份重要文件**: 转换前请备份重要文件
- ⚠️ **测试小规模**: 首次使用请先测试少量文件
- ⚠️ **检查结果**: 转换后请验证文件完整性
- ⚠️ **权限管理**: 确保有足够的文件操作权限

## 📈 更新日志

### v2.1.1 (2025-10-19)

#### 🐛 修复
- **严重错误修复**: 修复了在处理同名但扩展名不同的文件时（例如 `image.jpg` 和 `image.jpeg`），程序会错误地删除原始文件的严重问题。现在程序会正确跳过重复文件，而不会删除原始数据。详情请参阅 [CHANGELOG.md](CHANGELOG.md)。

### v2.1.0 (2024-10-19)

#### ✨ 新功能
- 🎨 **Pixly 主程序**: 全新的智能转换协调器
- 🧠 **智能策略引擎**: 自动分析图像特征选择最优格式
- 🎯 **尝试引擎**: 测试不同参数组合找到最佳策略
- 🎛️ **多种工作模式**: 质量模式、表情包模式、交互模式
- 🛡️ **安全策略**: 多层次安全保护机制

#### 🔧 改进
- ⚡ **性能优化**: 大幅提升处理速度
- 🎨 **界面美化**: 丰富的表情符号和进度显示
- 🔧 **配置管理**: 灵活的 JSON 配置文件
- 📊 **统计信息**: 详细的处理统计和报告

#### 🐛 修复
- 🔧 **文件清理**: 修复临时文件清理问题
- 🛡️ **系统稳定性**: 防止大规模处理时系统崩溃
- 📁 **文件数量验证**: 确保处理后文件数量正确
- 🔄 **错误处理**: 改进错误恢复机制

### v2.0.0 (2024-10-18)

#### ✨ 新功能
- 📸 **all2jxl 工具**: 专业的 JPEG XL 转换工具
- 🎬 **all2avif 工具**: 现代化的 AVIF 转换工具
- 🔄 **批量处理**: 支持大规模文件批量转换
- 📊 **进度监控**: 实时处理进度显示

#### 🔧 改进
- ⚡ **并发处理**: 多线程并发转换
- 🛡️ **质量验证**: 转换质量自动验证
- 📋 **元数据保护**: 完整保留原始元数据
- 🎯 **智能压缩**: 根据图像特征优化压缩

### v1.0.0 (2024-10-17)

#### 🎉 初始版本
- 🏗️ **基础架构**: 项目基础结构搭建
- 📸 **图像转换**: 基础图像格式转换功能
- 🔧 **工具集成**: 集成必要的转换工具
- 📖 **文档编写**: 基础使用文档

## 🤝 贡献指南

### 🛠️ 开发环境

1. **克隆项目**
   ```bash
   git clone https://github.com/your-repo/easy2jxlavif-beta.git
   cd easy2jxlavif-beta
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **运行测试**
   ```bash
   go test ./...
   ```

### 📝 贡献流程

1. **Fork 项目**
2. **创建功能分支**: `git checkout -b feature/amazing-feature`
3. **提交更改**: `git commit -m 'Add amazing feature'`
4. **推送分支**: `git push origin feature/amazing-feature`
5. **创建 Pull Request**

### 📋 代码规范

- 使用 `gofmt` 格式化代码
- 添加适当的注释
- 编写单元测试
- 更新相关文档

### 🐛 问题报告

请使用 GitHub Issues 报告问题，包含：
- 操作系统和版本
- Go 版本
- 错误信息和日志
- 复现步骤

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

感谢以下开源项目的支持：
- [libjxl](https://github.com/libjxl/libjxl) - JPEG XL 编码库
- [libavif](https://github.com/AOMediaCodec/libavif) - AVIF 编码库
- [ffmpeg](https://ffmpeg.org/) - 多媒体处理框架
- [exiftool](https://exiftool.org/) - 元数据处理工具
- [ImageMagick](https://imagemagick.org/) - 图像处理库

---

**🎨 Pixly - 让图像转换更智能、更高效！**

如有问题或建议，欢迎提交 Issue 或 Pull Request。
