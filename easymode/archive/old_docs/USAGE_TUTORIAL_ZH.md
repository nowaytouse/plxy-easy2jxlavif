# EasyMode 使用教程

本教程将指导您如何使用EasyMode工具集进行媒体文件处理。

## 📋 目录

1. [环境准备](#环境准备)
2. [工具构建](#工具构建)
3. [图像转换](#图像转换)
4. [视频处理](#视频处理)
5. [元数据管理](#元数据管理)
6. [文件去重](#文件去重)
7. [高级用法](#高级用法)
8. [故障排除](#故障排除)

## 🔧 环境准备

### 1. 安装依赖工具

```bash
# 使用Homebrew安装所有依赖
brew install ffmpeg exiftool imagemagick libjxl

# 验证安装
ffmpeg -version
exiftool -ver
magick -version
cjxl -V
```

### 2. 检查系统兼容性

```bash
# 检查macOS版本
sw_vers

# 检查Go版本
go version

# 检查架构（Apple Silicon vs Intel）
uname -m
```

## 🛠️ 工具构建

### 构建所有工具

```bash
# 进入easymode目录
cd easymode

# 构建所有工具
for tool in all2avif all2jxl static2avif static2jxl dynamic2avif dynamic2jxl video2mov merge_xmp deduplicate_media; do
    echo "构建 $tool..."
    cd $tool
    ./build.sh
    cd ..
done
```

### 构建单个工具

```bash
# 构建all2avif
cd all2avif
./build.sh

# 或手动构建
go build -o bin/all2avif main.go
```

## 🖼️ 图像转换

### 批量转换为AVIF

```bash
# 基本用法
./all2avif/bin/all2avif -dir /path/to/images

# 指定输出目录
./all2avif/bin/all2avif -dir /path/to/images -output /path/to/output

# 调整质量和速度
./all2avif/bin/all2avif -dir /path/to/images -quality 90 -speed 2

# 试运行模式
./all2avif/bin/all2avif -dir /path/to/images -dry-run
```

### 批量转换为JPEG XL

```bash
# 基本用法
./all2jxl/bin/all2jxl -dir /path/to/images

# 严格验证模式
./all2jxl/bin/all2jxl -dir /path/to/images -verify strict

# 快速验证模式
./all2jxl/bin/all2jxl -dir /path/to/images -verify fast
```

### 静态图像转换

```bash
# 静态图像转AVIF
./static2avif/bin/static2avif -dir /path/to/static/images

# 静态图像转JPEG XL
./static2jxl/bin/static2jxl -dir /path/to/static/images
```

### 动态图像转换

```bash
# 动态图像转AVIF
./dynamic2avif/bin/dynamic2avif -dir /path/to/animated/images

# 动态图像转JPEG XL
./dynamic2jxl/bin/dynamic2jxl -dir /path/to/animated/images
```

## 🎬 视频处理

### 视频重新封装

```bash
# 基本用法
./video2mov/bin/video2mov -dir /path/to/videos

# 指定输出目录
./video2mov/bin/video2mov -dir /path/to/videos -output /path/to/output

# 跳过已存在的文件
./video2mov/bin/video2mov -dir /path/to/videos -skip-exist
```

## 📝 元数据管理

### XMP元数据合并

```bash
# 基本用法
./merge_xmp/bin/merge_xmp -dir /path/to/media

# 处理特定格式
./merge_xmp/bin/merge_xmp -dir /path/to/photos
```

## 🗂️ 文件去重

### 媒体文件去重

```bash
# 基本用法
./deduplicate_media/bin/deduplicate_media -dir /path/to/media -trash-dir /path/to/trash

# 查看去重结果
ls -la /path/to/trash/
```

## 🚀 高级用法

### 1. 批量处理多个目录

```bash
#!/bin/bash
# 处理多个目录的脚本

directories=(
    "/Users/username/Photos/2023"
    "/Users/username/Photos/2024"
    "/Users/username/Downloads/Images"
)

for dir in "${directories[@]}"; do
    echo "处理目录: $dir"
    ./all2avif/bin/all2avif -dir "$dir" -output "$dir/avif"
done
```

### 2. 自动化工作流

```bash
#!/bin/bash
# 完整的媒体处理工作流

INPUT_DIR="/path/to/raw/media"
OUTPUT_DIR="/path/to/processed"
TRASH_DIR="/path/to/trash"

# 1. 去重
echo "步骤1: 去重..."
./deduplicate_media/bin/deduplicate_media -dir "$INPUT_DIR" -trash-dir "$TRASH_DIR"

# 2. 合并XMP元数据
echo "步骤2: 合并元数据..."
./merge_xmp/bin/merge_xmp -dir "$INPUT_DIR"

# 3. 转换为AVIF
echo "步骤3: 转换为AVIF..."
./all2avif/bin/all2avif -dir "$INPUT_DIR" -output "$OUTPUT_DIR"

# 4. 转换为JPEG XL
echo "步骤4: 转换为JPEG XL..."
./all2jxl/bin/all2jxl -dir "$INPUT_DIR" -output "$OUTPUT_DIR"
```

### 3. 性能优化

```bash
# 使用更多工作线程
./all2avif/bin/all2avif -dir /path/to/images -workers 16

# 调整CJXL线程数
./all2jxl/bin/all2jxl -dir /path/to/images -cjxl-threads 4

# 设置超时时间
./all2avif/bin/all2avif -dir /path/to/images -timeout 600
```

## 🔍 故障排除

### 常见问题

#### 1. 依赖工具未找到

```bash
# 检查工具是否在PATH中
which ffmpeg
which exiftool
which magick
which cjxl

# 如果未找到，重新安装
brew reinstall ffmpeg exiftool imagemagick libjxl
```

#### 2. 权限问题

```bash
# 给脚本执行权限
chmod +x build.sh

# 检查输出目录权限
ls -la /path/to/output/
```

#### 3. 内存不足

```bash
# 减少工作线程数
./all2avif/bin/all2avif -dir /path/to/images -workers 4

# 使用试运行模式检查
./all2avif/bin/all2avif -dir /path/to/images -dry-run
```

#### 4. 文件格式不支持

```bash
# 检查文件类型
file /path/to/image.jpg

# 使用exiftool检查元数据
exiftool /path/to/image.jpg
```

### 调试模式

```bash
# 启用详细日志
export DEBUG=1
./all2avif/bin/all2avif -dir /path/to/images

# 查看日志文件
tail -f all2avif.log
```

### 性能监控

```bash
# 监控系统资源
top -pid $(pgrep all2avif)

# 监控磁盘使用
df -h

# 监控内存使用
vm_stat
```

## 📊 最佳实践

### 1. 文件组织

```
project/
├── raw/           # 原始文件
├── processed/     # 处理后的文件
├── trash/         # 重复文件
└── logs/          # 日志文件
```

### 2. 备份策略

```bash
# 处理前备份
cp -r /path/to/images /path/to/backup/images_$(date +%Y%m%d)

# 使用版本控制
git add .
git commit -m "处理前备份"
```

### 3. 质量设置

- **高质量**: quality=95, speed=0
- **平衡**: quality=80, speed=4
- **快速**: quality=60, speed=6

### 4. 批量处理建议

- 小批量处理（<1000文件）
- 定期检查日志
- 监控磁盘空间
- 使用试运行模式验证

## 📞 获取帮助

如果遇到问题，请：

1. 检查日志文件
2. 使用试运行模式
3. 查看工具帮助：`./tool/bin/tool -h`
4. 提交Issue到项目仓库

---

**注意**: 本教程基于EasyMode v2.1.0，某些功能可能在旧版本中不可用。