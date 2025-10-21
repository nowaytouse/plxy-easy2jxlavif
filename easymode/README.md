# 媒体文件转换工具集

这是一套完整的媒体文件转换工具，支持多种图像和视频格式的批量转换，并保留原始文件的元数据。

## 🚀 工具列表

### 图像转换工具

#### 1. all2jxl - 批量图像转JPEG XL格式工具
- **功能**: 支持多种图像格式批量转换为JPEG XL格式
- **特点**: 保留原始文件的元数据和系统时间戳，支持动画图像和静态图像的无损转换
- **编译**: `go build -o bin/all2jxl main.go`
- **使用**: `./bin/all2jxl -dir <目录路径> [选项]`

#### 2. static2jxl - 静态图像转JPEG XL格式工具
- **功能**: 专门处理静态图像文件转换为JPEG XL格式
- **特点**: 支持多种静态图像格式（JPEG、PNG、BMP、TIFF等），使用CJXL编码器进行高质量转换
- **编译**: `go build -o bin/static2jxl main.go`
- **使用**: `./bin/static2jxl -input <输入目录> -output <输出目录> [选项]`

#### 3. dynamic2jxl - 动态图像转JPEG XL格式工具
- **功能**: 专门处理动态图像文件转换为JPEG XL格式
- **特点**: 支持多种动态图像格式（GIF、APNG、WebP、AVIF、HEIF等），使用CJXL编码器进行高质量转换
- **编译**: `go build -o bin/dynamic2jxl main.go`
- **使用**: `./bin/dynamic2jxl -input <输入目录> -output <输出目录> [选项]`

#### 4. static2avif - 静态图像转AVIF格式工具
- **功能**: 专门处理静态图像文件转换为AVIF格式
- **特点**: 支持多种静态图像格式，使用ImageMagick进行高质量转换
- **编译**: `go build -o bin/static2avif main.go`
- **使用**: `./bin/static2avif -input <输入目录> -output <输出目录> [选项]`

#### 5. dynamic2avif - 动态图像转AVIF格式工具
- **功能**: 专门处理动态图像文件转换为AVIF格式
- **特点**: 支持多种动态图像格式，使用ImageMagick进行高质量转换
- **编译**: `go build -o bin/dynamic2avif main.go`
- **使用**: `./bin/dynamic2avif -input <输入目录> -output <输出目录> [选项]`

### 视频转换工具

#### 6. video2mov - 批量视频转MOV格式工具
- **功能**: 支持多种视频格式批量转换为MOV格式
- **特点**: 使用ffmpeg进行视频重新封装，不重新编码，保留原始文件的元数据
- **编译**: `go build -o bin/video2mov main.go`
- **使用**: `./bin/video2mov -input <输入目录> -output <输出目录> [选项]`

### 元数据处理工具

#### 7. merge_xmp - XMP元数据合并工具
- **功能**: 将XMP侧边文件合并到对应的媒体文件中
- **特点**: 支持多种媒体格式（图像、视频等），自动检测XMP文件
- **编译**: `go build -o bin/merge_xmp main.go`
- **使用**: `./bin/merge_xmp -dir <目录路径>`

### 文件管理工具

#### 8. deduplicate_media - 媒体文件去重工具
- **功能**: 扫描目录中的重复媒体文件
- **特点**: 使用SHA256哈希值进行文件内容比较，标准化文件扩展名，将重复文件移动到垃圾箱目录
- **编译**: `go build -o bin/deduplicate_media main.go`
- **使用**: `./bin/deduplicate_media -dir <扫描目录> -trash-dir <垃圾箱目录>`

## 🛠️ 系统依赖

所有工具都需要以下系统依赖：

### 图像转换工具依赖
- **cjxl**: JPEG XL编码器
- **djxl**: JPEG XL解码器
- **exiftool**: 元数据处理工具
- **magick**: ImageMagick图像处理工具（用于AVIF转换）

### 视频转换工具依赖
- **ffmpeg**: 视频处理工具
- **exiftool**: 元数据处理工具

### 元数据处理工具依赖
- **exiftool**: 元数据处理工具

## 📦 安装依赖

### macOS (使用Homebrew)
```bash
# 安装JPEG XL工具
brew install libjxl

# 安装ImageMagick
brew install imagemagick

# 安装FFmpeg
brew install ffmpeg

# 安装ExifTool
brew install exiftool
```

### Ubuntu/Debian
```bash
# 安装JPEG XL工具
sudo apt-get install libjxl-tools

# 安装ImageMagick
sudo apt-get install imagemagick

# 安装FFmpeg
sudo apt-get install ffmpeg

# 安装ExifTool
sudo apt-get install libimage-exiftool-perl
```

## 🚀 快速开始

1. **克隆仓库**
```bash
git clone <repository-url>
cd easymode
```

2. **编译所有工具**
```bash
# 编译所有工具
for dir in all2jxl video2mov merge_xmp deduplicate_media static2jxl dynamic2jxl static2avif dynamic2avif; do
    cd $dir
    go build -o bin/$dir main.go
    cd ..
done
```

3. **使用工具**
```bash
# 转换图像为JPEG XL
./all2jxl/bin/all2jxl -dir /path/to/images

# 转换视频为MOV
./video2mov/bin/video2mov -input /path/to/videos -output /path/to/output

# 合并XMP元数据
./merge_xmp/bin/merge_xmp -dir /path/to/media

# 去重媒体文件
./deduplicate_media/bin/deduplicate_media -dir /path/to/media -trash-dir /path/to/trash
```

## ⚙️ 通用选项

所有工具都支持以下通用选项：

- `-workers int`: 并发工作线程数（默认：CPU核心数）
- `-timeout int`: 单个文件处理超时秒数
- `-retries int`: 转换失败时的重试次数
- `-skip-exist`: 跳过已存在的目标文件
- `-dry-run`: 试运行模式，只显示将要处理的文件而不实际转换

## 📊 功能特点

### 统一的基础架构
- 统一的错误处理和日志记录
- 统一的并发控制和资源管理
- 统一的元数据保留机制
- 统一的统计和报告系统

### 元数据保留
- 使用exiftool复制EXIF数据
- 在macOS上使用mdls和exiftool保留文件创建/修改时间
- 支持XMP侧边文件的自动合并

### 并发处理
- 智能线程数配置，根据CPU核心数动态调整
- 资源限制机制，防止系统过载
- 优雅的中断处理

### 详细日志
- 同时输出到控制台和文件
- 详细的处理统计和进度报告
- 按格式统计的处理结果
- 处理时间最长的文件信息

## 🔧 开发说明

### 代码结构
每个工具都遵循统一的结构：
- 详细的简体中文注释
- 统一的错误处理
- 统一的日志格式
- 统一的并发控制

### 编译要求
- Go 1.19+
- 相关系统依赖工具

### 测试
每个工具都支持`-h`参数查看帮助信息，`-dry-run`参数进行试运行测试。

## 📝 版本信息

- **版本**: 2.1.0
- **作者**: AI Assistant
- **更新日期**: 2024年10月22日

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这些工具。

## 📄 许可证

本项目采用MIT许可证。