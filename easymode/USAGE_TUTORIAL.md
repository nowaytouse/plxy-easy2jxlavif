# easymode 程序使用教程

本教程解释如何使用四个 easymode 程序：`all2jxl`、`all2avif`、`dynamic2avif` 和 `static2avif`。这些是用于将图像转换为现代格式的高质量、高效率的命令行工具。

## 目录
1. [概述](#概述)
2. [前提条件](#前提条件)
3. [all2jxl - 将图像转换为 JPEG XL](#all2jxl---将图像转换为-jpeg-xl)
4. [all2avif - 统一 AVIF 转换工具](#all2avif---统一-avif-转换工具)
5. [dynamic2avif - 将动态图像转换为 AVIF（已合并）](#dynamic2avif---将动态图像转换为-avif已合并)
6. [static2avif - 将静态图像转换为 AVIF（已合并）](#static2avif---将静态图像转换为-avif已合并)
7. [最佳实践](#最佳实践)

## 概述

easymode 程序提供简单、高质量的图像转换工具：

- **all2jxl**: 将各种图像格式转换为 JPEG XL（尽可能进行无损转换）
- **all2avif**: 统一工具，将静态和动态图像转换为 AVIF 格式
- **dynamic2avif**: 将动画图像（GIF、WebP、APNG）转换为 AVIF 格式（已合并到all2avif）
- **static2avif**: 将静态图像（JPEG、PNG、BMP、TIFF）转换为 AVIF 格式（已合并到all2avif）

所有程序都支持并发处理并包含健全的错误处理。

## 前提条件

在使用这些工具之前，请确保您已安装所需的依赖项：

### 系统要求
- Go 1.19 或更高版本
- macOS、Linux 或 Windows

### 依赖工具

#### all2jxl 依赖
- `cjxl` - JPEG XL 编码器
- `djxl` - JPEG XL 解码器
- `exiftool` - 元数据处理工具

#### all2avif 依赖
- `ffmpeg` - 视频和图像处理工具
- `exiftool` - 元数据处理工具

### 安装依赖

#### macOS (使用 Homebrew)
```bash
# all2jxl 的依赖
brew install jpeg-xl exiftool

# all2avif 的依赖
brew install ffmpeg exiftool
```

#### Ubuntu/Debian
```bash
# all2jxl 的依赖
sudo apt install libjxl-tools exiftool

# all2avif 的依赖
sudo apt install ffmpeg exiftool
```

#### CentOS/RHEL
```bash
# all2jxl 的依赖
sudo yum install libjxl-tools perl-Image-ExifTool

# all2avif 的依赖
sudo yum install ffmpeg perl-Image-ExifTool
```

## all2jxl - 将图像转换为 JPEG XL

### 概述
`all2jxl` 是一个高性能的 JPEG XL 转换器，支持多种图像格式的无损转换。

### 特性
- 支持格式：JPEG、PNG、GIF、WebP、BMP、TIFF、HEIC、HEIF、AVIF
- 智能动画检测（支持HEIF动画）
- Live Photo 保护：自动检测并跳过 Apple Live Photos（.mov 配对文件）
- 多重转换策略：自动在ImageMagick、FFmpeg和宽松模式间切换以处理HEIC/HEIF文件
- 统一验证流程：支持HEIC/HEIF文件的验证和像素级准确性检查
- 无损和数学上无损转换
- 完整的元数据保留
- 高性能并行处理

### 基本用法

```bash
# 进入工具目录
cd easymode/all2jxl

# 构建工具
./build.sh

# 基本转换
./all2jxl -dir /path/to/images

# 查看帮助
./all2jxl -h
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-dir` | 必需 | 输入目录路径 |
| `-workers` | 10 | 工作线程数 |
| `-quality` | 80 | 图像质量 (1-100) |
| `-skip-exist` | true | 跳过已存在的 JXL 文件 |
| `-replace` | true | 转换后删除原始文件 |
| `-dry-run` | false | 试运行模式 |
| `-timeout` | 300 | 单个文件超时时间（秒） |
| `-retries` | 1 | 重试次数 |

### 使用示例

```bash
# 基本转换
./all2jxl -dir ~/Pictures

# 高质量转换
./all2jxl -dir ~/Pictures -quality 95

# 使用更多工作线程
./all2jxl -dir ~/Pictures -workers 20

# 试运行模式
./all2jxl -dir ~/Pictures -dry-run

# 转换后保留原始文件
./all2jxl -dir ~/Pictures -replace=false
```

### 输出示例

```
🎨 JPEG XL 批量转换工具 v2.0.0
✨ 作者: AI Assistant
🔧 开始初始化...
✅ cjxl 已就绪
✅ djxl 已就绪
✅ exiftool 已就绪
📁 准备处理目录...
📂 直接处理目录: /path/to/images
🔍 扫描图像文件...
📊 发现 150 个候选文件
⚡ 配置处理性能...
🚀 开始并行处理 - 工作线程: 10, 文件数: 150

🔄 开始处理: image1.jpg (2.5 MB)
✅ 识别为图像格式: image1.jpg (jpg)
🖼️  静态图像: image1.jpg
✅ 转换完成: image1.jpg (JPEG Lossless Re-encode)
✅ 验证通过: image1.jpg 无损转换正确
🎉 处理成功: image1.jpg
📊 大小变化: 2.50 MB -> 2.00 MB (节省: 0.50 MB, 压缩率: 80.0%)

...

⏱️  总处理时间: 2m30.5s
🎯 ===== 处理摘要 =====
✅ 成功处理图像: 150
❌ 转换失败图像: 0
📊 ===== 大小统计 =====
📥 原始总大小: 500.00 MB
📤 转换后大小: 350.00 MB
💾 节省空间: 150.00 MB (压缩率: 70.0%)
🎉 ===== 处理完成 =====
```

## all2avif - 统一 AVIF 转换工具

### 概述
`all2avif` 是一个统一的 AVIF 转换工具，支持静态和动态图像的转换。

### 特性
- 支持静态图像：JPEG、PNG、BMP、TIFF、WebP、HEIC、HEIF、AVIF
- 支持动画图像：GIF、WebP 动画、HEIF 动画
- 智能动画检测（支持HEIF动画检测）
- Live Photo 保护：自动检测并跳过 Apple Live Photos（.mov 配对文件）
- 多重转换策略：自动在ImageMagick、FFmpeg和宽松模式间切换以处理HEIC/HEIF文件
- 可配置的质量和速度设置
- 完整的元数据保留

### 基本用法

```bash
# 进入工具目录
cd easymode/all2avif

# 构建工具
./build.sh

# 基本转换
./all2avif -dir /path/to/images

# 查看帮助
./all2avif -h
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-dir` | 必需 | 输入目录路径 |
| `-output` | 输入目录 | 输出目录路径 |
| `-workers` | 10 | 工作线程数 |
| `-quality` | 80 | 图像质量 (1-100) |
| `-speed` | 4 | 编码速度 (0-6) |
| `-skip-exist` | true | 跳过已存在的 AVIF 文件 |
| `-replace` | true | 转换后删除原始文件 |
| `-dry-run` | false | 试运行模式 |
| `-timeout` | 300 | 单个文件超时时间（秒） |
| `-retries` | 1 | 重试次数 |

### 使用示例

```bash
# 基本转换
./all2avif -dir ~/Pictures

# 高质量转换
./all2avif -dir ~/Pictures -quality 90

# 快速转换
./all2avif -dir ~/Pictures -speed 6

# 指定输出目录
./all2avif -dir ~/Pictures -output ~/Pictures/avif

# 试运行模式
./all2avif -dir ~/Pictures -dry-run
```

### 质量与速度设置

#### 质量设置 (1-100)
- **90-100**: 最高质量，文件较大
- **80-89**: 高质量，平衡质量和大小
- **70-79**: 中等质量，较小文件
- **60-69**: 低质量，小文件
- **1-59**: 最低质量，最小文件

#### 速度设置 (0-6)
- **0-1**: 最慢，质量最好
- **2-3**: 较慢，质量较好
- **4**: 默认设置，平衡速度和质量
- **5-6**: 最快，质量一般

### 输出示例

```
🎨 AVIF 批量转换工具 v2.0.0
✨ 作者: AI Assistant
🔧 开始初始化...
✅ ffmpeg 已就绪
✅ exiftool 已就绪
📁 准备处理目录...
📂 直接处理目录: /path/to/images
🔍 扫描图像文件...
📊 发现 150 个候选文件
⚡ 配置处理性能...
🚀 开始并行处理 - 工作线程: 10, 文件数: 150

🔄 开始处理: image1.jpg (2.5 MB)
🖼️  静态图像: image1.jpg
✅ 转换完成: image1.jpg (Static Image Conversion)
📋 元数据复制成功: image1.jpg
🎉 处理成功: image1.jpg
📊 大小变化: 2.50 MB -> 1.20 MB (节省: 1.30 MB, 压缩率: 48.0%)

🔄 开始处理: animation.gif (1.2 MB)
🎬 检测到动画图像: animation.gif
✅ 转换完成: animation.gif (Animated Image Conversion)
📋 元数据复制成功: animation.gif
🎉 处理成功: animation.gif
📊 大小变化: 1.20 MB -> 0.80 MB (节省: 0.40 MB, 压缩率: 66.7%)

...

⏱️  总处理时间: 3m15.2s
🎯 ===== 处理摘要 =====
✅ 成功处理图像: 150
❌ 转换失败图像: 0
📊 ===== 大小统计 =====
📥 原始总大小: 500.00 MB
📤 转换后大小: 300.00 MB
💾 节省空间: 200.00 MB (压缩率: 60.0%)
🎉 ===== 处理完成 =====
```

## dynamic2avif - 将动态图像转换为 AVIF（已合并）

### 概述
`dynamic2avif` 功能已合并到 `all2avif` 中。请使用 `all2avif` 进行动画图像转换。

### 迁移指南
- 使用 `all2avif` 替代 `dynamic2avif`
- 所有功能都已集成到 `all2avif` 中
- 命令行参数保持兼容

## static2avif - 将静态图像转换为 AVIF（已合并）

### 概述
`static2avif` 功能已合并到 `all2avif` 中。请使用 `all2avif` 进行静态图像转换。

### 迁移指南
- 使用 `all2avif` 替代 `static2avif`
- 所有功能都已集成到 `all2avif` 中
- 命令行参数保持兼容

## 最佳实践

### 1. 选择合适的工具

- **使用 all2jxl**: 当您需要无损压缩和最高质量时
- **使用 all2avif**: 当您需要现代格式和良好的压缩率时

### 2. 性能优化

#### 工作线程设置
```bash
# 对于多核CPU，使用更多工作线程
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20

# 对于内存受限的系统，减少工作线程
./all2jxl -dir /path/to/images -workers 4
./all2avif -dir /path/to/images -workers 4
```

#### 质量与速度平衡
```bash
# 高质量设置（适合最终输出）
./all2avif -dir /path/to/images -quality 95 -speed 1

# 快速设置（适合预览或测试）
./all2avif -dir /path/to/images -quality 70 -speed 6
```

### 3. 批量处理

#### 处理多个目录
```bash
# 使用循环处理多个目录
for dir in ~/Pictures/*/; do
    ./all2jxl -dir "$dir"
done

for dir in ~/Pictures/*/; do
    ./all2avif -dir "$dir"
done
```

#### 使用脚本自动化
```bash
#!/bin/bash
# 批量转换脚本

# 设置目录
INPUT_DIR="/path/to/images"
OUTPUT_DIR="/path/to/output"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 转换到JXL
echo "开始JXL转换..."
./all2jxl -dir "$INPUT_DIR" -output "$OUTPUT_DIR/jxl"

# 转换到AVIF
echo "开始AVIF转换..."
./all2avif -dir "$INPUT_DIR" -output "$OUTPUT_DIR/avif"

echo "转换完成！"
```

### 4. 错误处理

#### 试运行模式
```bash
# 在正式转换前先试运行
./all2jxl -dir /path/to/images -dry-run
./all2avif -dir /path/to/images -dry-run
```

#### 重试机制
```bash
# 对于不稳定的文件，增加重试次数
./all2jxl -dir /path/to/images -retries 5
./all2avif -dir /path/to/images -retries 5
```

#### 超时设置
```bash
# 对于大文件，增加超时时间
./all2jxl -dir /path/to/images -timeout 600
./all2avif -dir /path/to/images -timeout 600
```

### 5. 存储管理

#### 磁盘空间检查
```bash
# 在转换前检查可用空间
df -h /path/to/images

# 使用du命令查看目录大小
du -sh /path/to/images
```

#### 备份重要文件
```bash
# 在转换前备份重要文件
cp -r /path/to/images /path/to/backup

# 或者使用rsync进行增量备份
rsync -av /path/to/images/ /path/to/backup/
```

### 6. 监控和日志

#### 查看处理进度
```bash
# 在另一个终端中监控日志
tail -f all2jxl.log
tail -f all2avif.log
```

#### 检查系统资源
```bash
# 监控CPU和内存使用
top -p $(pgrep all2jxl)
top -p $(pgrep all2avif)
```

### 7. 故障排除

#### 常见问题解决

**问题**: 转换失败，提示"缺少依赖工具"
```bash
# 检查依赖工具安装
which cjxl djxl exiftool
which ffmpeg exiftool

# 重新安装依赖
brew install jpeg-xl exiftool ffmpeg
```

**问题**: 内存不足
```bash
# 减少工作线程数
./all2jxl -dir /path/to/images -workers 4
./all2avif -dir /path/to/images -workers 4
```

**问题**: 处理速度慢
```bash
# 增加工作线程数（如果CPU和内存允许）
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20
```

**问题**: 某些文件处理失败
```bash
# 检查文件是否损坏
file /path/to/problematic/file

# 尝试单独处理问题文件
./all2jxl -dir /path/to/single/file
./all2avif -dir /path/to/single/file
```

### 8. 性能基准测试

#### 测试不同设置的效果
```bash
# 测试不同工作线程数的性能
for workers in 1 4 8 16 20; do
    echo "测试 $workers 个工作线程..."
    time ./all2jxl -dir /path/to/test/images -workers $workers
done
```

#### 比较不同质量设置
```bash
# 测试不同质量设置
for quality in 60 70 80 90 95; do
    echo "测试质量 $quality..."
    time ./all2avif -dir /path/to/test/images -quality $quality
done
```

## 总结

easymode 程序提供了一套完整的图像转换解决方案：

1. **all2jxl**: 适合需要无损压缩的场景
2. **all2avif**: 适合需要现代格式和良好压缩率的场景

通过合理使用这些工具和遵循最佳实践，您可以高效地将图像转换为现代格式，同时保持高质量和良好的性能。

记住：
- 总是先进行试运行
- 根据系统资源调整工作线程数
- 定期备份重要文件
- 监控处理进度和系统资源
- 根据需求选择合适的质量和速度设置

## 更新日志

### v2.0.1
- **重要修复**: 添加文件数量验证功能，防止临时文件残留
- **自动清理**: 自动检测和清理未清理的临时文件
- **质量保证**: 确保处理前后文件数量符合预期

### v2.0.0
- 合并 `dynamic2avif` 和 `static2avif` 为统一的 `all2avif` 工具
- 改进错误处理和统计功能
- 优化性能和内存使用
- 更新所有文档为简体中文