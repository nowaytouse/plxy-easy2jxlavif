# `deduplicate_media` - 媒体文件去重工具

## 📖 简介

`deduplicate_media` 是一个辅助工具，用于扫描指定目录中的媒体文件，识别内容重复的文件，并将重复项移动到指定的“垃圾”文件夹中。它还可以规范化不一致的文件扩展名（例如，将 `.jpeg` 重命名为 `.jpg`）。

## 🚀 功能特性

- ✅ **广泛的格式支持** - 支持常见的图片格式 (如 `.jpg`, `.png`, `.gif`, `.bmp`, `.tif`, `.webp`) 和视频格式 (如 `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`)。
- ✅ **规范扩展名** - 自动将 `.jpeg`, `.tiff` 等扩展名重命名为统一的 `.jpg`, `.tif` 格式。
- ✅ **精确去重** - 通过 SHA-256 哈希值快速识别潜在的重复文件，并通过逐字节比较进行最终确认。
- ✅ **安全移动** - 重复文件将被移动到指定的文件夹，而不是永久删除，以便用户进行最终检查和恢复。
- ✅ **垃圾文件夹注释** - 在垃圾文件夹中自动创建一个 `_readme_about_this_folder.txt` 文件，说明其用途。
- ✅ **清晰日志** - 记录所有操作，包括扩展名重命名、发现的重复项以及移动的文件。

## 🔧 使用方法

### 编译脚本

```bash
# 进入脚本目录
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/deduplicate_media

# 运行构建脚本
./build.sh
```

### 运行脚本

```bash
./deduplicate_media -dir /path/to/your/media -trash-dir /path/to/trash
```

### 参数说明

- `-dir`: 要扫描的媒体文件所在目录的路径 (必需)。
- `-trash-dir`: 用于存放重复文件的目录路径 (必需)。如果该目录不存在，脚本将自动创建。

## 📈 输出示例

```
INFO: 2025/10/19 21:25:00 main.go:25: deduplicate_media v1.1.0 starting...
INFO: 2025/10/19 21:25:00 main.go:71: Standardizing extensions...
INFO: 2025/10/19 21:25:00 main.go:86: Renamed image (1).jpeg to image (1).jpg
INFO: 2025/10/19 21:25:00 main.go:92: Finding and moving duplicates...
INFO: 2025/10/19 21:25:01 main.go:110: Potential duplicate found: /path/to/media/image.jpg and /path/to/media/image (1).jpg
INFO: 2025/10/19 21:25:01 main.go:118: Files are identical. Moving image (1).jpg to trash.
INFO: 2025/10/19 21:25:01 main.go:50: Deduplication process complete.
```

---

**版本**: v1.1.0  
**维护者**: AI Assistant  
**许可证**: MIT
