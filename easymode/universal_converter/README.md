# Universal Converter - 通用媒体转换工具

## 📋 概述

Universal Converter 是一个强大的通用媒体转换工具，支持多种格式的智能转换，具有完整的元数据保留、8层验证系统和智能性能优化功能。

## ✨ 功能特性

### 🎨 支持的转换格式
- **AVIF格式** - 现代高效图像格式
- **JPEG XL格式** - 下一代JPEG标准
- **MOV格式** - 高质量视频容器

### 🔧 处理模式
- **all** - 处理所有文件类型
- **static** - 仅处理静态图像
- **dynamic** - 仅处理动态图像
- **video** - 仅处理视频文件
- **optimized** - 🆕 通用优化模式（智能选择最佳转换方式）

### 🆕 通用优化模式 (v2.3.0)

通用优化模式会根据文件类型智能选择最佳的转换方式：

1. **📸 JPEG文件** → JXL格式（使用 `jpeg_lossless=1` 无损模式）
2. **🎬 动态图片** → AVIF格式（使用现有AVIF动态图片质量参数）
3. **🎥 视频文件** → MOV格式（重新包装，不重新编码）
4. **🚫 其他格式** → 不处理

## 🚀 使用方法

### 基本用法

```bash
# 通用优化模式（推荐）
universal_converter -mode optimized -input /path/to/files

# 传统模式
universal_converter -type jxl -mode all -input /path/to/files
universal_converter -type avif -mode static -input /path/to/files
universal_converter -type mov -mode video -input /path/to/files
```

### 参数说明

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `-input` | 输入目录路径 | 必需 |
| `-output` | 输出目录路径 | 与输入目录相同 |
| `-mode` | 处理模式 | `all` |
| `-type` | 转换类型 | `jxl` |
| `-workers` | 工作线程数 | 自动检测 |
| `-quality` | 输出质量 (1-100) | `90` |
| `-speed` | 编码速度 (0-9) | `4` |
| `-dry-run` | 试运行模式 | `false` |
| `-skip-exist` | 跳过已存在文件 | `false` |

### 通用优化模式示例

```bash
# 基本用法
universal_converter -mode optimized -input ./photos

# 带参数
universal_converter -mode optimized -input ./photos -workers 8 -quality 80

# 试运行
universal_converter -mode optimized -input ./photos -dry-run

# 跳过已存在文件
universal_converter -mode optimized -input ./photos -skip-exist
```

## 🔧 技术特性

### 8层验证系统
1. **文件存在性验证** - 确保输入文件存在
2. **文件大小验证** - 检查文件大小合理性
3. **格式兼容性验证** - 验证输入格式支持
4. **元数据完整性验证** - 检查元数据完整性
5. **尺寸一致性验证** - 验证输出尺寸正确
6. **像素级验证** - 检查像素级差异（可选）
7. **质量指标验证** - 验证输出质量
8. **反作弊验证** - 防止恶意文件

### 智能性能优化
- **自动线程数检测** - 根据CPU核心数调整
- **内存使用监控** - 防止内存溢出
- **文件大小限制** - 避免处理过大文件
- **并发控制** - 智能控制并发处理数量
- **资源管理** - 自动清理临时文件

### 元数据保留
- **EXIF数据** - 完整保留相机信息
- **XMP数据** - 保留编辑软件信息
- **时间戳** - 保留文件创建和修改时间
- **颜色配置文件** - 保留颜色管理信息

## 📊 性能指标

### 处理速度
- **JPEG → JXL**: ~2-5MB/s
- **PNG → AVIF**: ~1-3MB/s
- **GIF → AVIF**: ~0.5-2MB/s
- **MP4 → MOV**: ~10-50MB/s

### 压缩比
- **JXL无损**: 通常比原JPEG小10-30%
- **AVIF有损**: 比JPEG小50-80%
- **MOV重封装**: 文件大小基本不变

## 🛠️ 系统要求

### 依赖工具
- **cjxl/djxl** - JPEG XL编码/解码器
- **ffmpeg** - 视频处理和AVIF编码
- **exiftool** - 元数据处理
- **avifenc** - AVIF静态图像编码

### 安装依赖

```bash
# macOS
brew install libjxl ffmpeg exiftool

# Ubuntu/Debian
sudo apt install libjxl-tools ffmpeg exiftool

# 编译安装
go build -o bin/universal_converter main.go
```

## 📈 版本历史

### v2.3.0 (当前版本)
- ✅ 新增通用优化模式
- ✅ 智能文件类型检测
- ✅ 动态输出格式选择
- ✅ 视频MOV重新包装功能
- ✅ 增强错误处理

### v2.2.0
- ✅ 8层验证系统
- ✅ 智能性能优化
- ✅ 完整元数据保留
- ✅ 批量处理支持

## 🔍 故障排除

### 常见问题

1. **依赖工具缺失**
   ```bash
   # 检查依赖
   which cjxl djxl ffmpeg exiftool
   ```

2. **内存不足**
   ```bash
   # 减少并发数
   universal_converter -mode optimized -input ./files -workers 2
   ```

3. **处理速度慢**
   ```bash
   # 调整质量设置
   universal_converter -mode optimized -input ./files -quality 70 -speed 6
   ```

## 📄 许可证

本项目采用 MIT 许可证。

## 🤝 贡献

欢迎提交问题报告和功能请求！

---

**注意**: 通用优化模式是推荐的使用方式，它会自动为每种文件类型选择最佳的转换策略。
