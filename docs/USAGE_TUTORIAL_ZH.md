# easymode - 简易图像转换工具使用教程

easymode 目录包含三个专门的命令行工具，用于将图像转换为现代、高效的格式：

1. **all2jxl** - 将各种图像格式转换为 JPEG XL
2. **dynamic2avif** - 将动画图像（GIF、WebP、APNG）转换为 AVIF
3. **static2avif** - 将静态图像（JPEG、PNG 等）转换为 AVIF

## 快速开始

### 前提条件

在使用这些工具之前，请确保您具备：
- Go 1.19 或更高版本
- 每个工具的系统依赖项：
  - 对于 `all2jxl`：`cjxl`、`djxl`、`exiftool`
  - 对于 `dynamic2avif` 和 `static2avif`：`ffmpeg`

在 macOS 上安装依赖项：
```bash
# all2jxl 的依赖
brew install jpeg-xl exiftool

# dynamic2avif 和 static2avif 的依赖
brew install ffmpeg
```

### 构建工具

所有工具都可以从各自的子目录构建：

```bash
# 构建 all2jxl
cd easymode/all2jxl
./build.sh

# 构建 dynamic2avif
cd easymode/dynamic2avif
./build.sh

# 构建 static2avif
cd easymode/static2avif
./build.sh
```

### 运行工具

可执行文件创建在 `bin` 子目录中：

```bash
# 转换为 JPEG XL
./easymode/all2jxl/bin/all2jxl -dir "/path/to/images"

# 将动画图像转换为 AVIF
./easymode/dynamic2avif/bin/dynamic2avif -input "/path/to/animated" -output "/path/to/avif"

# 将静态图像转换为 AVIF
./easymode/static2avif/bin/static2avif -input "/path/to/static" -output "/path/to/avif"
```

## 详细用法

### all2jxl - 将图像转换为 JPEG XL

#### 基本用法
```bash
# 转换目录中的所有图像为 JPEG XL
./easymode/all2jxl/bin/all2jxl -dir "/path/to/your/images"

# 带常用选项的示例
./easymode/all2jxl/bin/all2jxl -dir "/path/to/your/images" -workers 8 -verify strict
```

#### 所有选项
- `-dir STRING`: 输入目录路径（必选）
- `-workers INT`: 工作线程数（0=自动检测）
- `-verify STRING`: 验证模式：`strict|fast`
- `-copy`: 复制目录到 *_work 然后处理
- `-sample INT`: 测试模式：仅处理 N 个中等大小文件
- `-skip-exist`: 跳过现有的 .jxl 文件（默认值：true）
- `-dry-run`: 试运行模式：仅记录操作，不转换
- `-cjxl-threads INT`: 每个转换任务的线程数（默认值：1）
- `-timeout INT`: 单个任务超时秒数（0=无限制）
- `-retries INT`: 失败重试次数（默认值：0）

#### 示例
```bash
# 基本转换
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum"

# 高性能转换，使用 16 个线程
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum" -workers 16

# 试运行，查看将要转换的文件
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum" -dry-run

# 跳过已有 .jxl 版本的文件
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum" -skip-exist

# 启用快速验证以加快处理速度
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum" -verify fast

# 每个转换任务使用 4 个线程以加快处理
./bin/all2jxl -dir "/Users/username/Pictures/MyAlbum" -cjxl-threads 4
```

### dynamic2avif - 将动态图像转换为 AVIF

#### 基本用法
```bash
# 从输入目录转换动画图像到输出目录
./bin/dynamic2avif -input "/path/to/input/animated" -output "/path/to/output/avif"

# 带常用选项的示例
./bin/dynamic2avif -input "/path/to/input" -output "/path/to/output" -quality 80 -workers 4
```

#### 所有选项
- `-input STRING`: 输入目录（必选）
- `-output STRING`: 输出目录（必选）
- `-quality INT`: AVIF 质量（0-100，默认值：50）
- `-speed INT`: 编码速度（0-10，默认值：6）
- `-workers INT`: 并发工作线程数
- `-skip-exist`: 跳过输出中已存在的文件
- `-dry-run`: 试运行模式：仅记录操作
- `-timeout INT`: 每个文件的超时秒数（默认值：120）
- `-retries INT`: 失败重试次数（默认值：2）

#### 示例
```bash
# 基本动画图像转换
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF"

# 高质量转换
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF" -quality 90

# 低质量快速转换
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF" -quality 30 -speed 2

# 使用 8 个工作线程以加快处理
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF" -workers 8

# 跳过已转换的文件
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF" -skip-exist

# 试运行，查看将要转换的文件
./bin/dynamic2avif -input "/Users/username/GIFs" -output "/Users/username/GIFs_AVIF" -dry-run
```

### static2avif - 将静态图像转换为 AVIF

#### 基本用法
```bash
# 从输入目录转换静态图像到输出目录
./bin/static2avif -input "/path/to/input/static" -output "/path/to/output/avif"

# 带常用选项的示例
./bin/static2avif -input "/path/to/input" -output "/path/to/output" -quality 80 -workers 4
```

#### 所有选项
- `-input STRING`: 输入目录（必选）
- `-output STRING`: 输出目录（必选）
- `-quality INT`: AVIF 质量（0-100，默认值：50）
- `-speed INT`: 编码速度（0-10，默认值：6）
- `-workers INT`: 并发工作线程数
- `-skip-exist`: 跳过输出中已存在的文件
- `-dry-run`: 试运行模式：仅记录操作
- `-timeout INT`: 每个文件的超时秒数（默认值：120）
- `-retries INT`: 失败重试次数（默认值：2）

#### 示例
```bash
# 基本静态图像转换
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF"

# 高质量转换
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF" -quality 90

# 低质量快速转换
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF" -quality 30 -speed 2

# 使用 8 个工作线程以加快处理
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF" -workers 8

# 跳过已转换的文件
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF" -skip-exist

# 试运行，查看将要转换的文件
./bin/static2avif -input "/Users/username/Photos" -output "/Users/username/Photos_AVIF" -dry-run
```

## 最佳实践

### 性能优化
- 根据 CPU 核心数调整 `-workers`（通常是 1-2 倍核数）
- 对于 CPU 密集型任务，考虑使用较少的工作线程以避免系统变慢
- 使用适合您需求的 `-quality` 设置：
  - 高质量（80-100）：更好的视觉质量，更大的文件
  - 中等质量（50-79）：质量和文件大小的良好平衡
  - 低质量（0-49）：较小的文件，降低的视觉质量

### 安全和可靠性
- 始终先在图像副本上测试
- 使用 `-dry-run` 预览将要转换的内容
- 工具执行验证并在转换期间保留元数据
- 对于重要图像，请在转换的图像之外保留原始副本

### 目录结构
- 按图像类型组织输入目录（如果可能）
- 使用描述性输出目录名称
- 考虑按格式、日期或内容类型进行组织

### 质量设置
- 对于 JPEG XL (all2jxl)：专注于尽可能无损转换
- 对于 AVIF (dynamic2avif/static2avif)：较高的质量设置（70-90）通常适用于大多数用途
- 对于网络使用：质量设置 60-80 通常提供良好的压缩
- 对于存档：考虑较高的质量设置，结合适当的 speed 设置

## 故障排除

### 常见问题
- **"command not found"**: 验证所有依赖项都已安装并在您的 PATH 中
- **转换失败**: 尝试降低质量设置或增加超时值
- **高内存使用**: 减少工作线程数
- **处理缓慢**: 尝试不同的 speed 设置（0-10）

### 验证安装
检查所有依赖项是否可用：
```bash
# all2jxl
which cjxl && which djxl && which exiftool

# dynamic2avif 和 static2avif
which ffmpeg
```

### 获取帮助
获取任何工具的详细帮助：
```bash
./bin/all2jxl --help
./bin/dynamic2avif --help
./bin/static2avif --help
```

## 日志和监控

所有程序都向控制台和日志文件输出详细日志：
- `all2jxl.log` - all2jxl 的日志文件
- `dynamic2avif.log` - dynamic2avif 的日志文件
- `static2avif.log` - static2avif 的日志文件

日志包含有关以下内容的详细信息：
- 处理进度
- 文件大小和压缩比
- 错误信息
- 性能指标
- 统计摘要

这些日志对于跟踪处理效率和诊断问题很有用。

## 主程序使用

主程序 `pixly` 提供了一个统一的界面来处理图像转换：

```bash
# 交互模式运行
./pixly

# 非交互模式运行
./pixly -non-interactive /path/to/images
```