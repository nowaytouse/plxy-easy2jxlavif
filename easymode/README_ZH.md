# easymode - 简易图像转换工具集

easymode 目录包含专门的命令行工具，用于将图像转换为现代、高效的格式：

1. **all2jxl** - 将各种图像格式转换为 JPEG XL
2. **all2avif** - 将静态和动态图像转换为 AVIF（统一工具）
3. **static2avif** - 专门处理静态图像转AVIF
4. **dynamic2avif** - 专门处理动画图像转AVIF
5. **static2jxl** - 专门处理静态图像转JXL（新增）
6. **dynamic2jxl** - 专门处理动画图像转JXL（新增）
7. **deduplicate_media** - 去除重复图片/视频
8. **merge_xmp** - 合并XMP元数据
9. **video2mov** - 视频格式转换

## 概述

## 🔧 近期修复与改进 (v2.1.0)

- **修复了HEIC转换中的严重错误:** 解决了在HEIC转换过程中临时文件处理不当可能导致转换失败的问题。
- **移除了`merge_xmp`中的硬编码验证:** 将使用硬编码日期的“虚假”验证逻辑替换为动态且更可靠的验证方法。
- **修正了`SafeDelete`逻辑:** 修复了多个脚本中转换成功后未删除原始文件的逻辑错误。
- **改进了文件类型处理:** 根据用户要求，所有转换脚本现在都能正确跳过如 `.psd` 之类的源文件格式。
- **增加了缺失的元数据和日期处理:** 确保所有转换脚本现在都能正确保留文件元数据和原始的创建/修改日期。
- **代码清理和重构:** 重构了重复的代码并移除了冗余文件，以提高代码质量和可维护性。


这些工具旨在提供简单、高效的媒体处理，并具有高质量的结果。每个工具处理特定类型的转换：

- `all2jxl`: 专注于无损或数学上无损转换为 JPEG XL 格式
- `all2avif`: 统一工具，支持静态和动态图像到 AVIF 格式的转换
- `static2avif`: 专门处理静态图像转AVIF格式
- `dynamic2avif`: 专门处理动画图像转AVIF格式
- `static2jxl`: 专门处理静态图像转JXL格式
- `dynamic2jxl`: 专门处理动画图像转JXL格式
- `deduplicate_media`: 用于查找和删除重复的媒体文件
- `merge_xmp`: 合并和管理 XMP 元数据
- `video2mov`: 视频格式转换工具

## 快速开始

### 前提条件

在使用这些工具之前，请确保您具备：
- Go 1.21 或更高版本
- 每个工具的系统依赖项：
  - 对于 `all2jxl`, `static2jxl`, `dynamic2jxl`: `cjxl`, `djxl`, `exiftool`
  - 对于 `all2avif`, `static2avif`, `dynamic2avif`: `ffmpeg`, `exiftool`
  - 对于 `deduplicate_media`, `merge_xmp`, `video2mov`: `exiftool`

在 macOS 上安装依赖项：
```bash
# all2jxl, static2jxl, dynamic2jxl 的依赖
brew install jpeg-xl exiftool

# all2avif, static2avif, dynamic2avif 的依赖
brew install ffmpeg exiftool
```

在 Ubuntu/Debian 上安装依赖项：
```bash
# all2jxl, static2jxl, dynamic2jxl 的依赖
sudo apt install libjxl-tools exiftool

# all2avif, static2avif, dynamic2avif 的依赖
sudo apt install ffmpeg exiftool
```

### 构建和运行

每个工具都可以独立构建和运行：

```bash
# 进入工具目录
cd all2jxl  # 或其他工具

# 构建工具
./build.sh

# 运行工具
./all2jxl -dir /path/to/images
./all2avif -dir /path/to/images
./static2jxl -input /path/to/images -output /path/to/output
./dynamic2jxl -input /path/to/images -output /path/to/output
```

## 工具详细说明

### all2jxl - JPEG XL 转换工具

**用途**: 将各种图像格式转换为 JPEG XL (JXL) 格式

**特性**:
- 支持多种输入格式：JPEG、PNG、GIF、WebP、BMP、TIFF、HEIC、HEIF、AVIF
- 智能动画检测（支持HEIF动画）
- Live Photo 保护：自动检测并跳过 Apple Live Photos（.mov 配对文件）
- 多重转换策略：自动在ImageMagick、FFmpeg和宽松模式间切换以处理HEIC/HEIF文件
- 统一验证流程：支持HEIC/HEIF文件的验证和像素级准确性检查
- 无损和数学上无损转换
- 完整的元数据保留
- 高性能并行处理

**使用示例**:
```bash
# 基本用法
./all2jxl -dir /path/to/images

# 高质量转换
./all2jxl -dir /path/to/images -quality 95

# 转换后删除原始文件
./all2jxl -dir /path/to/images -replace

# 试运行模式
./all2jxl -dir /path/to/images -dry-run
```

### all2avif - AVIF 转换工具（统一工具）

**用途**: 将静态和动态图像转换为 AVIF 格式

**特性**:
- 支持静态图像：JPEG、PNG、BMP、TIFF、WebP、HEIC、HEIF、AVIF
- 支持动画图像：GIF、WebP 动画、HEIF 动画
- 智能动画检测（支持HEIF动画检测）
- Live Photo 保护：自动检测并跳过 Apple Live Photos（.mov 配对文件）
- 多重转换策略：自动在ImageMagick、FFmpeg和宽松模式间切换以处理HEIC/HEIF文件
- 可配置的质量和速度设置
- 完整的元数据保留

**使用示例**:
```bash
# 基本用法
./all2avif -dir /path/to/images

# 高质量转换
./all2avif -dir /path/to/images -quality 90

# 快速转换
./all2avif -dir /path/to/images -speed 6

# 转换后删除原始文件
./all2avif -dir /path/to/images -replace

# 试运行模式
./all2avif -dir /path/to/images -dry-run
```

### static2avif - 静态图像转AVIF工具

**用途**: 专门处理静态图像转AVIF的工具

**特性**:
- 针对静态图像优化
- 更快的处理速度
- 支持JPEG、PNG、BMP、TIFF、WebP、HEIC、HEIF、AVIF等格式
- 完整的元数据保留

**使用示例**:
```bash
# 基本用法
./static2avif -input /path/to/images -output /path/to/output

# 高质量转换
./static2avif -input /path/to/images -output /path/to/output -quality 90
```

### dynamic2avif - 动画图像转AVIF工具

**用途**: 专门处理动画图像转AVIF的工具

**特性**:
- 支持GIF、WebP动画、APNG、HEIF动画
- 智能动画检测
- 保持动画质量
- 完整的元数据保留

**使用示例**:
```bash
# 基本用法
./dynamic2avif -input /path/to/images -output /path/to/output

# 高质量转换
./dynamic2avif -input /path/to/images -output /path/to/output -quality 90
```

### static2jxl - 静态图像转JXL工具 (新增)

**用途**: 专门处理静态图像转JXL的工具

**特性**:
- 针对静态图像优化
- 无损压缩
- 保持最高质量
- 完整的元数据保留

**使用示例**:
```bash
# 基本用法
./static2jxl -input /path/to/images -output /path/to/output

# 高质量转换
./static2jxl -input /path/to/images -output /path/to/output -quality 95
```

### dynamic2jxl - 动画图像转JXL工具 (新增)

**用途**: 专门处理动画图像转JXL的工具

**特性**:
- 支持GIF、WebP动画、APNG、HEIF动画
- 智能动画检测
- 保持动画质量
- 完整的元数据保留

**使用示例**:
```bash
# 基本用法
./dynamic2jxl -input /path/to/images -output /path/to/output

# 高质量转换
./dynamic2jxl -input /path/to/images -output /path/to/output -quality 95
```

### deduplicate_media - 媒体文件去重工具

**用途**: 检测并移除重复的图片和视频文件

**特性**:
- 比较文件内容识别重复项
- 高效的哈希算法
- 安全删除机制

**使用示例**:
```bash
# 基本用法
./deduplicate_media -dir /path/to/media -workers 4
```

### merge_xmp - XMP元数据合并工具

**用途**: 合并和管理XMP元数据

**特性**:
- 保留和合并元数据信息
- 支持多种图像格式
- 安全的元数据操作

**使用示例**:
```bash
# 基本用法
./merge_xmp -input /path/to/images -output /path/to/output
```

### video2mov - 视频格式转换工具

**用途**: 转换各种视频格式

**特性**:
- 支持多种视频格式转换
- 保持视频质量
- 高效处理

**使用示例**:
```bash
# 基本用法
./video2mov -input /path/to/videos -output /path/to/output
```

## 命令行参数

### all2jxl 参数

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

### all2avif 参数

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

### static2jxl, dynamic2jxl, static2avif, dynamic2avif 参数

这些工具共享类似的参数结构：
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-input` | 必需 | 输入目录路径 |
| `-output` | 输入目录 | 输出目录路径 |
| `-workers` | 10 | 工作线程数 |
| `-quality` | 80 (95 for JXL) | 图像质量 (1-100) |
| `-speed` | 4 (仅AVIF) | 编码速度 (0-6) |
| `-skip-exist` | true | 跳过已存在的目标文件 |
| `-replace` | true | 转换后删除原始文件 |
| `-dry-run` | false | 试运行模式 |

## 使用场景

### 图片优化
- **个人照片**: 使用 `all2jxl` 或 `static2jxl` 进行无损压缩
- **网页图片**: 使用 `all2avif` 或 `static2avif` 进行现代格式转换
- **表情包**: 使用 `all2avif` 或 `dynamic2avif` 进行动画优化

### 批量处理
- **大量图片**: 使用高并发设置处理大量文件
- **格式统一**: 将不同格式统一转换为目标格式
- **存储优化**: 通过压缩减少存储空间使用
- **媒体整理**: 使用 `deduplicate_media` 清理重复文件，使用 `merge_xmp` 管理元数据

## 性能优化

### 并发设置
```bash
# 使用更多工作线程（适用于多核CPU）
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20
./static2jxl -input /path/to/images -workers 20
./dynamic2jxl -input /path/to/images -workers 20
```

### 质量与速度平衡
```bash
# 高质量设置
./all2avif -dir /path/to/images -quality 95 -speed 1

# 快速设置
./all2avif -dir /path/to/images -quality 70 -speed 6
```

## 故障排除

### 常见问题

**Q: 转换失败，提示"缺少依赖工具"**
A: 请确保已安装所有必需的依赖工具，运行相应命令验证安装。

**Q: 处理速度慢**
A: 可以增加工作线程数，使用 `-workers 20` 参数。

**Q: 某些文件处理失败**
A: 检查文件是否损坏，或尝试使用 `-retries 5` 增加重试次数。

**Q: 内存使用过高**
A: 减少工作线程数，使用 `-workers 5` 参数。

### 调试技巧

```bash
# 使用试运行模式检查文件
./all2jxl -dir /path/to/images -dry-run
./all2avif -dir /path/to/images -dry-run
./static2jxl -input /path/to/images -dry-run
./dynamic2jxl -input /path/to/images -dry-run

# 查看详细日志
tail -f all2jxl.log
tail -f all2avif.log
```

## 更新日志

### v2.0.2 (2025-01-27)
- ✅ 修复跳过已存在文件时误删原始文件的问题
- ✅ 新增模块化验证系统
- ✅ 增强 dynamic2avif 中的 HEIC/HEIF 支持，使其具有与 dynamic2jxl 相同的稳健性
- ✅ 新增动静图分离处理工具 (static2jxl, dynamic2jxl)
- ✅ 新增更多工具 (deduplicate_media, merge_xmp, video2mov)
- ✅ 改进错误处理和日志记录
- ✅ 优化性能和内存使用

### v2.0.0
- 合并 `dynamic2avif` 和 `static2avif` 为统一的 `all2avif` 工具
- 改进错误处理和统计功能
- 优化性能和内存使用
- 更新所有文档为简体中文

### v1.0.0
- 初始版本发布
- 支持 `all2jxl` JPEG XL 转换
- 支持 `dynamic2avif` 动画转换
- 支持 `static2avif` 静态图像转换

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 支持

如有问题，请提交 Issue 或联系维护者。