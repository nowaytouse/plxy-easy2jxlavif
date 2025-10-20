# EasyMode 工具集

EasyMode 是一套专门为图像格式转换设计的简化工具集，提供简单易用的命令行界面和高效的批量处理能力。

## 🚀 工具概览

### 核心工具
- **all2avif** - 全格式转AVIF工具
- **all2jxl** - 全格式转JXL工具
- **static2avif** - 静态图片转AVIF工具
- **dynamic2avif** - 动态图片转AVIF工具
- **static2jxl** - 静态图片转JXL工具 (新增)
- **dynamic2jxl** - 动态图片转JXL工具 (新增)

### 功能特性
- ✅ **智能格式检测** - 自动识别静态/动态图像
- ✅ **批量处理** - 高效的并发处理能力
- ✅ **安全保护** - 修复了跳过已存在文件时误删原始文件的问题
- ✅ **验证机制** - 完整的处理结果验证和报告生成
- ✅ **元数据保留** - 使用exiftool保留EXIF信息
- ✅ **进度显示** - 实时处理进度和统计信息

## 📁 工具详细说明

### all2avif - 全格式转AVIF
**用途**: 将各种图像格式转换为AVIF格式
**特点**: 支持静态和动态图像，智能参数选择
**使用**: `./all2avif -dir /path/to/images -quality 80 -workers 4`

### all2jxl - 全格式转JXL
**用途**: 将各种图像格式转换为JXL格式
**特点**: 无损压缩，支持动画图像
**使用**: `./all2jxl -dir /path/to/images -workers 4`

### static2avif - 静态图片转AVIF
**用途**: 专门处理静态图像转AVIF
**特点**: 针对静态图像优化，更快的处理速度
**使用**: `./static2avif -input /path/to/images -output /path/to/output -quality 80`

### dynamic2avif - 动态图片转AVIF
**用途**: 专门处理动态图像转AVIF
**特点**: 支持GIF、WebP、APNG等动画格式
**使用**: `./dynamic2avif -input /path/to/images -output /path/to/output -quality 80`

### static2jxl - 静态图片转JXL (新增)
**用途**: 专门处理静态图像转JXL
**特点**: 无损压缩，保持最高质量
**使用**: `go run main.go -input /path/to/images -output /path/to/output -workers 4`

### dynamic2jxl - 动态图片转JXL (新增)
**用途**: 专门处理动态图像转JXL
**特点**: 支持动画图像的JXL转换
**使用**: `go run main.go -input /path/to/images -output /path/to/output -workers 4`

## 🔧 构建说明

### 依赖要求
- Go 1.21+
- ffmpeg (用于AVIF转换)
- cjxl (用于JXL转换)
- exiftool (用于元数据保留)

### 构建步骤
```bash
# 构建所有工具
cd easymode
for dir in all2avif all2jxl static2avif dynamic2avif static2jxl dynamic2jxl; do
    cd $dir
    chmod +x build.sh
    ./build.sh
    cd ..
done
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

## 📈 验证系统

### 自动验证
- 文件数量验证
- 大小压缩验证
- EXIF数据验证
- 格式转换验证

### 报告生成
- JSON格式详细报告
- 用户友好的文本报告
- 失败原因分析

## 🎯 使用建议

### 选择工具
- **全格式处理**: 使用 all2avif 或 all2jxl
- **静态图像**: 使用 static2avif 或 static2jxl
- **动态图像**: 使用 dynamic2avif 或 dynamic2jxl

### 性能调优
- 根据CPU核心数调整工作线程
- 大文件处理时增加超时时间
- 使用试运行模式测试配置

## 🔍 故障排除

### 常见问题
1. **依赖缺失**: 确保安装了ffmpeg、cjxl、exiftool
2. **权限问题**: 检查文件读写权限
3. **空间不足**: 确保有足够的磁盘空间

### 获取帮助
- 查看日志文件了解详细错误
- 检查验证报告中的失败分析
- 使用试运行模式测试配置

## 📝 更新日志

### v2.0.2 (2025-01-27)
- ✅ 修复跳过已存在文件时误删原始文件的问题
- ✅ 新增模块化验证系统
- ✅ 新增动静图分离处理工具
- ✅ 改进错误处理和日志记录
- ✅ 优化性能和内存使用

---

**版本**: v2.0.2  
**维护者**: AI Assistant  
**许可证**: MIT