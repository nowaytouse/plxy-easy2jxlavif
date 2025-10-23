# EasyMode 媒体转换工具集 v2.3.1

> 🚀 **一套强大的Go语言媒体转换工具，支持多种图像和视频格式的批量转换，具备完整的元数据保留、智能性能优化和8层验证系统。**

EasyMode 是一套专为图像收藏家和效率追求者设计的媒体转换工具集，提供专业级的工具来将各种媒体格式转换为现代、高效的格式，并具备完整的元数据保留和智能处理功能。

---

## 🎯 工具套件概览

### 📦 核心工具

| 工具 | 功能 | 输入格式 | 输出格式 | 核心特色 |
|------|------|----------|----------|----------|
| **universal_converter** | 通用媒体转换器 | 所有支持格式 | JXL, AVIF, MOV | 🎯 **一个工具支持所有转换** |
| **media_tools** | 媒体管理工具 | 26+ 格式 | 元数据处理 | 🔧 **XMP合并、去重** |
| **all2jxl** | JPEG XL 转换器 | 图像 | JPEG XL (.jxl) | 🔥 **真正的数学无损** |
| **all2avif** | AVIF 转换器 | 图像 | AVIF (.avif) | ⚡ **高压缩率** |
| **static2jxl** | 静态转 JPEG XL | 静态图像 | JPEG XL (.jxl) | 🖼️ **静态图像优化** |
| **static2avif** | 静态转 AVIF | 静态图像 | AVIF (.avif) | 📸 **静态图像压缩** |
| **dynamic2jxl** | 动态转 JPEG XL | 动画图像 | JPEG XL (.jxl) | 🎬 **动画保留** |
| **dynamic2avif** | 动态转 AVIF | 动画图像 | AVIF (.avif) | 🎭 **动画图像压缩** |
| **video2mov** | 视频转换器 | 视频格式 | MOV | 🎥 **视频重新封装** |

---

## 🌟 主要特性

### 🧠 智能处理
- **通用转换器**：一个工具支持所有转换类型和模式
- **智能格式检测**：增强的 AVIF/HEIC 格式识别
- **Apple Live Photo 检测**：自动跳过 Live Photo 文件以保留配对关系
- **垃圾箱目录排除**：自动跳过 `.trash`、`.Trash`、`Trash` 目录

### 🔒 高级安全
- **8层验证系统**：确保转换质量和数据完整性
- **反作弊机制**：防止硬编码绕过和虚假转换
- **路径安全验证**：防止目录遍历攻击
- **文件类型验证**：验证文件扩展名与实际内容匹配

### ⚡ 高性能
- **智能线程调整**：根据系统负载动态调整处理线程数
- **内存管理**：智能内存使用监控和限制
- **并发控制**：限制外部进程和文件句柄使用
- **文件优先级处理**：优先处理 JPEG 等快速转换格式

### 📋 完整元数据保留
- **EXIF/IPTC/XMP 支持**：所有格式的完整元数据保留
- **专业格式支持**：PSD、PSB 和 8 种 RAW 格式（CR2、CR3、NEF、ARW、DNG、RAF、ORF、RW2）
- **XMP 合并**：自动 XMP 侧边文件合并
- **时间戳保留**：保持原始文件时间戳

---

## 🛠️ 支持的格式

### 📷 图像格式（共26种）

#### 标准格式（12种）
- **JPEG**: .jpg, .jpeg - 最常见的图像格式
- **PNG**: .png - 无损压缩
- **GIF**: .gif - 动画图像
- **BMP**: .bmp - 位图格式
- **TIFF**: .tiff, .tif - 高质量图像
- **WebP**: .webp - Google 格式

#### 现代格式（4种）
- **JPEG XL**: .jxl - 次世代格式
- **AVIF**: .avif - AV1 图像格式
- **HEIC/HEIF**: .heic, .heif - Apple 格式

#### 专业格式（2种）- v2.3.0+
- **Photoshop**: .psd - Photoshop 文档
- **大型 Photoshop**: .psb - 大型 PSD 文件

#### RAW 格式（8种）- v2.3.0+
- **Canon**: .cr2, .cr3 - Canon RAW 格式
- **Nikon**: .nef - Nikon RAW
- **Sony**: .arw - Sony RAW
- **Adobe**: .dng - 通用 RAW
- **Fujifilm**: .raf - Fujifilm RAW
- **Olympus**: .orf - Olympus RAW
- **Panasonic**: .rw2 - Panasonic RAW

### 🎬 视频格式（4种）
- **MP4**: .mp4 - 最常见的视频格式
- **QuickTime**: .mov - Apple 视频格式
- **AVI**: .avi - 旧版视频格式
- **Matroska**: .mkv - 开源容器

---

## 🚀 快速开始

### 系统要求
- **Go 1.25+**：用于构建工具
- **ImageMagick**：用于 AVIF 转换
- **libjxl**：用于 JPEG XL 转换
- **FFmpeg**：用于视频转换
- **ExifTool**：用于元数据处理
- **libavif**：用于静态 AVIF 转换

### 安装

#### macOS
```bash
# 安装依赖
brew install imagemagick libjxl ffmpeg exiftool

# 克隆仓库
git clone <repository-url>
cd easymode
```

#### Ubuntu/Debian
```bash
# 安装依赖
sudo apt-get install imagemagick libjxl-tools ffmpeg exiftool

# 克隆仓库
git clone <repository-url>
cd easymode
```

### 构建工具

```bash
# 构建所有工具
make build

# 或构建单个工具
cd universal_converter && ./build.sh
cd media_tools && ./build.sh
```

---

## 📖 使用指南

### 通用转换器（推荐）

通用转换器是支持所有转换类型的主要工具：

```bash
# 转换所有图像为 JPEG XL
./universal_converter/bin/universal_converter \
  -input /path/to/images \
  -type jxl \
  -mode all \
  -quality 95

# 转换静态图像为 AVIF
./universal_converter/bin/universal_converter \
  -input /path/to/photos \
  -type avif \
  -mode static \
  -quality 90

# 转换视频为 MOV
./universal_converter/bin/universal_converter \
  -input /path/to/videos \
  -type mov \
  -mode video

# 转换动态图像为 JPEG XL
./universal_converter/bin/universal_converter \
  -input /path/to/gifs \
  -type jxl \
  -mode dynamic
```

### 媒体工具

用于元数据管理和文件操作：

```bash
# 自动模式：XMP 合并 + 去重
./media_tools/bin/media_tools auto -dir /path/to/files

# 仅 XMP 合并
./media_tools/bin/media_tools merge -dir /path/to/files

# 仅去重
./media_tools/bin/media_tools dedup -dir /path/to/files

# 自定义垃圾箱目录
./media_tools/bin/media_tools auto \
  -dir /path/to/files \
  -trash /custom/trash/location
```

### 独立工具

```bash
# 转换所有图像为 JPEG XL
./all2jxl/bin/all2jxl -dir /path/to/images -workers 4

# 转换所有图像为 AVIF
./all2avif/bin/all2avif -dir /path/to/images -workers 4

# 转换静态图像为 JPEG XL
./static2jxl/bin/static2jxl -dir /path/to/photos -quality 90

# 转换动态图像为 AVIF
./dynamic2avif/bin/dynamic2avif -dir /path/to/gifs -quality 85
```

---

## 🔧 高级配置

### 通用转换器参数

#### 通用参数
- `-input`: 输入目录路径
- `-output`: 输出目录（默认：与输入相同）
- `-type`: 转换类型（jxl, avif, mov）
- `-mode`: 处理模式（all, static, dynamic, video）
- `-workers`: 工作线程数（0=自动检测）
- `-quality`: 输出质量（1-100）
- `-speed`: 编码速度（0-9）

#### 验证参数
- `-strict`: 严格验证模式
- `-tolerance`: 允许的像素差异百分比
- `-skip-exist`: 跳过已存在的输出文件
- `-dry-run`: 预览模式，不实际转换

#### 性能参数
- `-max-memory`: 最大内存使用（字节）
- `-process-limit`: 最大并发进程数
- `-file-limit`: 最大并发文件数
- `-timeout`: 单文件处理超时（秒）

### 媒体工具参数

#### 通用参数
- `-dir`: 输入目录路径
- `-trash`: 垃圾箱目录（默认：`<input>/.trash`）
- `-workers`: 工作线程数
- `-dry-run`: 预览模式

#### 操作模式
- `auto`: XMP 合并 + 去重
- `merge`: 仅 XMP 合并
- `dedup`: 仅去重

---

## 🛡️ 8层验证系统

为确保转换质量，所有工具都集成了8层验证系统：

1. **基础文件验证**：检查文件存在性和可读性
2. **文件大小验证**：验证转换后文件大小的合理性
3. **格式完整性验证**：确保正确的输出格式
4. **元数据完整性验证**：检查关键元数据字段
5. **图像尺寸验证**：验证图像尺寸一致性
6. **像素级验证**：执行像素级质量检查
7. **质量指标验证**：计算 PSNR、SSIM 质量指标
8. **反作弊验证**：检测硬编码绕过和虚假转换

---

## 📊 性能基准

MacBook Pro M1 测试结果：
- **JPEG 转 JXL**：~50MB/s
- **PNG 转 AVIF**：~30MB/s
- **HEIC 转 JXL**：~20MB/s
- **元数据处理**：~1000 文件/分钟
- **XMP 合并**：~500 文件/分钟
- **去重处理**：~2000 文件/分钟

---

## 🆕 v2.3.1 新功能

### Universal Converter v2.3.1
- ✅ **Apple Live Photo 智能跳过**：自动检测 HEIC/HEIF + MOV 配对文件
- ✅ **垃圾箱目录自动排除**：自动跳过 `.trash`、`.Trash`、`Trash` 目录
- ✅ **增强文件类型检测**：改进 AVIF/HEIC 格式识别

### Media Tools v2.3.1
- ✅ **扩展格式支持**：添加 PSD、PSB 和 8 种 RAW 格式（共26种格式）
- ✅ **默认垃圾箱目录**：`-trash` 参数现在可选，默认为 `<input>/.trash`
- ✅ **专业格式支持**：Photoshop 和 RAW 格式 XMP 合并

---

## 🎯 使用场景

### 摄影师
- 批量处理带 XMP 元数据的 RAW 图像
- 转换格式同时保留编辑历史
- 整理和去重照片库

### 设计师
- 在保持质量的同时优化图像文件大小
- 转换 Photoshop 文件并保留元数据
- 高效管理大型图像集合

### 内容创作者
- 视频格式转换和优化
- 跨格式元数据管理
- 媒体资产批量处理

### 系统管理员
- 文件去重和存储优化
- 跨系统元数据标准化
- 自动化媒体处理工作流

---

## 🔧 故障排除

### 常见问题

1. **缺少依赖**
```bash
# macOS
brew install imagemagick libjxl ffmpeg exiftool

# Ubuntu/Debian
sudo apt-get install imagemagick libjxl-tools ffmpeg exiftool
```

2. **权限问题**
```bash
chmod +x */build.sh
chmod +x */bin/*
```

3. **内存不足**
```bash
# 减少工作线程
./universal_converter/bin/universal_converter -input ./images -workers 2
```

4. **文件类型识别问题**
```bash
# 使用严格模式进行详细验证
./universal_converter/bin/universal_converter -input ./images -type jxl -strict
```

### Live Photo 检测
- 确保 HEIC 和 MOV 文件具有相同的文件名（除扩展名外）
- 例如：`IMG_0001.heic` + `IMG_0001.mov`

### PSD/RAW 格式支持
- PSD 文件可能很大（>1GB），处理可能需要时间
- RAW 文件应小心处理以保留原始数据
- 先用小文件测试

---

## 📁 项目结构

```
easymode/
├── universal_converter/        # 通用转换工具
│   ├── bin/universal_converter
│   ├── main.go
│   └── build.sh
├── media_tools/               # 媒体管理工具
│   ├── bin/media_tools
│   ├── main.go
│   └── build.sh
├── all2jxl/                   # JPEG XL 转换器
├── all2avif/                  # AVIF 转换器
├── static2jxl/                # 静态转 JPEG XL
├── static2avif/               # 静态转 AVIF
├── dynamic2jxl/               # 动态转 JPEG XL
├── dynamic2avif/              # 动态转 AVIF
├── video2mov/                 # 视频转换器
├── utils/                     # 共享工具
├── docs/                      # 文档
├── archive/                   # 归档工具
├── README.md                  # 英文版本文档
├── README_ZH.md              # 本文件 - 中文版本文档
├── Makefile                   # 构建配置
└── go.mod                     # Go 模块定义
```

---

## 📝 版本历史

### v2.3.1（最新）
- ✅ Universal Converter：添加垃圾箱目录排除
- ✅ Media Tools：trash 参数改为可选，默认为 `.trash`
- ✅ 增强 AVIF/HEIC 文件类型检测
- ✅ Apple Live Photo 智能检测和跳过

### v2.3.0
- ✅ Universal Converter：添加 Live Photo 跳过
- ✅ Media Tools：添加 PSD/PSB 和 8 种 RAW 格式支持
- ✅ 格式支持从 18 种扩展到 26 种
- ✅ 增强文件类型检测

### v2.2.0
- ✅ Universal Converter：一个工具支持所有转换
- ✅ 8层验证系统
- ✅ 模块化设计和统一参数解析
- ✅ 智能性能优化
- ✅ 反作弊机制

---

## 🌐 语言支持

- **English**: [README.md](README.md)
- **简体中文**: [README_ZH.md](README_ZH.md)（当前）

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 🤝 贡献

欢迎贡献！请随时提交 Pull Request。

## 📞 支持

如果遇到任何问题或有疑问，请在 GitHub 上提交 issue。

---

## 🔗 相关链接

- [JPEG XL 官方网站](https://jpeg.org/jpegxl/)
- [AVIF 格式规范](https://aomediacodec.github.io/av1-avif/)
- [ExifTool 文档](https://exiftool.org/)
- [FFmpeg 文档](https://ffmpeg.org/documentation.html)

---

**🎉 开始使用 EasyMode，让媒体转换变得简单高效！**