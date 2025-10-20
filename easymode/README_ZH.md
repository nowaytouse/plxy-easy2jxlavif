# easymode - 简易图像转换工具

easymode 目录包含四个专门的命令行工具，用于将图像转换为现代、高效的格式：

1. **all2jxl** - 将各种图像格式转换为 JPEG XL
2. **all2avif** - 将静态和动态图像转换为 AVIF（统一工具）
3. **dynamic2avif** - 将动画图像（GIF、WebP、APNG）转换为 AVIF（已合并到all2avif）
4. **static2avif** - 将静态图像（JPEG、PNG 等）转换为 AVIF（已合并到all2avif）

## 概述

这些工具旨在提供简单、高效的图像转换，并具有高质量的结果。每个工具处理特定类型的转换：

- `all2jxl`: 专注于无损或数学上无损转换为 JPEG XL 格式
- `all2avif`: 统一工具，支持静态和动态图像到 AVIF 格式的转换
- `dynamic2avif` 和 `static2avif`: 已合并到 `all2avif` 中，提供统一的 AVIF 转换体验

## 快速开始

### 前提条件

在使用这些工具之前，请确保您具备：
- Go 1.19 或更高版本
- 每个工具的系统依赖项：
  - 对于 `all2jxl`: `cjxl`, `djxl`, `exiftool`
  - 对于 `all2avif`: `ffmpeg`, `exiftool`

在 macOS 上安装依赖项：
```bash
# all2jxl 的依赖
brew install jpeg-xl exiftool

# all2avif 的依赖
brew install ffmpeg exiftool
```

在 Ubuntu/Debian 上安装依赖项：
```bash
# all2jxl 的依赖
sudo apt install libjxl-tools exiftool

# all2avif 的依赖
sudo apt install ffmpeg exiftool
```

### 构建和运行

每个工具都可以独立构建和运行：

```bash
# 进入工具目录
cd all2jxl  # 或 all2avif

# 构建工具
./build.sh

# 运行工具
./all2jxl -dir /path/to/images
./all2avif -dir /path/to/images
```

## 工具详细说明

### all2jxl - JPEG XL 转换工具

**用途**: 将各种图像格式转换为 JPEG XL (JXL) 格式

**特性**:
- 支持多种输入格式：JPEG、PNG、GIF、WebP、BMP、TIFF、HEIC、AVIF
- 智能动画检测和处理
- 无损和数学上无损转换
- 完整的元数据保留
- 高性能并行处理
- 智能跳过已存在文件
- 自动删除原始文件选项

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
- 支持静态图像：JPEG、PNG、BMP、TIFF、WebP、HEIC、AVIF
- 支持动画图像：GIF、WebP 动画
- 智能动画检测
- 可配置的质量和速度设置
- 完整的元数据保留
- 高性能并行处理
- 智能跳过已存在文件
- 自动删除原始文件选项

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

## 使用场景

### 图片优化
- **个人照片**: 使用 `all2jxl` 进行无损压缩
- **网页图片**: 使用 `all2avif` 进行现代格式转换
- **表情包**: 使用 `all2avif` 进行动画优化

### 批量处理
- **大量图片**: 使用高并发设置处理大量文件
- **格式统一**: 将不同格式统一转换为目标格式
- **存储优化**: 通过压缩减少存储空间使用

## 性能优化

### 并发设置
```bash
# 使用更多工作线程（适用于多核CPU）
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20
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

# 查看详细日志
tail -f all2jxl.log
tail -f all2avif.log
```

## 更新日志

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