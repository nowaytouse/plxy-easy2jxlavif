# static2jxl - 静态图片转JXL工具

## 📖 简介

static2jxl 是一个专门用于静态图像转JXL格式的工具。针对静态图像进行了优化，提供更快的处理速度和更好的压缩效果。

## 🚀 功能特性

- ✅ **静态图像优化** - 专门针对静态图像设计
- ✅ **无损压缩** - 使用JXL格式实现无损压缩
- ✅ **智能检测** - 自动识别静态图像类型
- ✅ **批量处理** - 高效的并发处理能力
- ✅ **安全保护** - 修复了跳过已存在文件时误删原始文件的问题
- ✅ **元数据保留** - 使用exiftool保留EXIF信息
- ✅ **进度显示** - 实时处理进度和统计信息
- ✅ **精确的文件数量验证** - 转换完成后，提供详细的文件数量验证报告，确保处理过程的准确性和可靠性。
- ✅ **优化HEIC/HEIF处理** - 采用更稳定的中间格式转换策略，提高HEIC/HEIF文件的转换成功率。
- ✅ **修复JPEG参数错误** - 修正了 `--lossless_jpeg=1` 参数被错误应用于非JPEG文件的Bug。

## 🔧 使用方法

### 基本用法
```bash
go run main.go -input /path/to/images -output /path/to/output -workers 4
```

### 参数说明
- `-input`: 输入目录路径 (必需)
- `-output`: 输出目录路径 (必需)
- `-workers`: 并发工作线程数 (默认: CPU核心数)
- `-skip-exist`: 跳过已存在的文件 (默认: true)
- `-dry-run`: 试运行模式，只显示将要处理的文件
- `-retries`: 失败重试次数 (默认: 2)
- `-timeout`: 单个文件处理超时秒数 (默认: 300)
- `-cjxl-threads`: 每个转换任务的线程数 (默认: 1)

### 高级用法
```bash
# 高并发处理
go run main.go -input /path/to/images -output /path/to/output -workers 8

# 试运行模式
go run main.go -input /path/to/images -output /path/to/output -dry-run

# 跳过已存在文件
go run main.go -input /path/to/images -output /path/to/output -skip-exist

# 自定义重试次数
go run main.go -input /path/to/images -output /path/to/output -retries 3 -timeout 600
```

## 📊 性能优化

### 并发控制
- 智能工作线程配置
- 资源限制防止系统过载
- 文件句柄管理

### 内存管理
- 减少内存占用
- 优化文件处理流程
- 防止内存泄漏

## 🛡️ 安全特性

### 文件安全
- 修复了跳过已存在文件时误删原始文件的问题
- 原子性文件操作
- 备份机制

### 错误处理
- 完善的错误恢复机制
- 详细的日志记录
- 自动重试功能

## 🔍 故障排除

### 常见问题
1. **依赖缺失**: 确保安装了cjxl和exiftool
2. **权限问题**: 检查文件读写权限
3. **空间不足**: 确保有足够的磁盘空间

### 获取帮助
- 查看日志文件了解详细错误
- 使用试运行模式测试配置
- 检查文件权限和磁盘空间

### 支持的文件格式

- **JPEG**: .jpg, .jpeg
- **PNG**: .png
- **BMP**: .bmp
- **TIFF**: .tiff, .tif
- **HEIC/HEIF**: .heic, .heif

## 📝 更新日志

### v2.0.1 (2025-01-27)
- ✅ 新增静态图片转JXL工具
- ✅ 修复跳过已存在文件时误删原始文件的问题
- ✅ 改进错误处理和日志记录
- ✅ 优化性能和内存使用
- ✅ 增强安全保护机制

---

**版本**: v2.0.1  
**维护者**: AI Assistant  
**许可证**: MIT