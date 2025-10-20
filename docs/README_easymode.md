# easymode - 简易图像转换工具

easymode 目录包含专门的命令行工具，用于将图像转换为现代、高效的格式：

1. **all2jxl** - 将各种图像格式（包括HEIC/HEIF）转换为 JPEG XL
2. **all2avif** - 统一 AVIF 转换工具，支持多种图像格式（包括HEIC/HEIF）
3. **dynamic2jxl** - 将动画图像（GIF、WebP、APNG、HEIF动画）转换为 JPEG XL
4. **static2jxl** - 将静态图像（JPEG、PNG、HEIC、HEIF等）转换为 JPEG XL
5. **dynamic2avif** - 将动画图像（GIF、WebP、APNG）转换为 AVIF
6. **static2avif** - 将静态图像（JPEG、PNG 等）转换为 AVIF
7. **merge_xmp** - 合并 XMP 元数据到图像文件

## 快速开始

### 前提条件

在使用这些工具之前，请确保您具备：
- Go 1.19 或更高版本
- 每个工具的系统依赖项：
  - 对于 `all2jxl`: `cjxl`, `djxl`, `exiftool`
  - 对于 `dynamic2avif` 和 `static2avif`: `ffmpeg`

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

## 更多信息

有关综合使用说明，请参阅 [USAGE_TUTORIAL_ZH.md](USAGE_TUTORIAL_ZH.md) 文件，其中包含：

- 每个工具的所有命令行选项
- 性能和质量的最佳实践
- 故障排除常见问题
- 不同用例的示例
- 安全性和可靠性建议

每个单独的工具在其各自目录中也有自己的 README，提供更具体的信息。