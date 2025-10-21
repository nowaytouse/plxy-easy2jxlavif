# `merge_xmp` - XMP 元数据合并工具

## 📖 简介

`merge_xmp` 是一个独立的辅助脚本，用于将 `.xmp` 文件中的元数据合并到媒体文件中。它会自动查找与媒体文件同名的 `.xmp` 文件，使用 `exiftool` 进行合并，并在验证成功后删除 `.xmp` 文件。

## 🚀 功能特性

- ✅ **自动查找** - 自动查找与媒体文件（如 `.jpg`, `.png` 等）同名的 `.xmp` 文件。
- ✅ **元数据合并** - 使用 `exiftool` 将 `.xmp` 文件中的所有元数据合并到媒体文件中。
- ✅ **自动删除** - 在合并成功并验证后，自动删除 `.xmp` 文件。
- ✅ **安全验证** - 如果验证失败，将保留 `.xmp` 文件以便进行手动检查。

## 🔧 使用方法

### 依赖

- **exiftool**: 确保 `exiftool` 已安装并在系统的 `PATH` 中。

### 编译脚本

```bash
# 进入脚本目录
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/merge_xmp

# 运行构建脚本
./build.sh
```

### 运行脚本

```bash
./merge_xmp -dir /path/to/your/media
```

### 参数说明

- `-dir`: 要处理的媒体文件所在目录的路径 (必需)。

## 📈 输出示例

```
INFO: 2025/10/19 20:55:03 main.go:33: merge_xmp v1.0.0 starting...
INFO: 2025/10/19 20:55:03 main.go:89: Found media file 'IMG_0429.JPG' with XMP sidecar 'IMG_0429.xmp'
INFO: 2025/10/19 20:55:03 main.go:98: Successfully merged XMP into IMG_0429.JPG
INFO: 2025/10/19 20:55:03 main.go:102: Verification successful for IMG_0429.JPG
INFO: 2025/10/19 20:55:03 main.go:107: Successfully deleted XMP file IMG_0429.xmp
INFO: 2025/10/19 20:55:03 main.go:66: Processing complete.
```

---

**版本**: v1.0.0  
**维护者**: AI Assistant  
**许可证**: MIT
