# EasyMode 用户指南 v2.2.0

## 📖 快速开始

EasyMode是一套强大的媒体转换工具集，支持多种图像和视频格式的批量转换。本指南将帮助您快速上手并充分利用所有功能。

## 🚀 安装和配置

### 系统要求

- **操作系统**: macOS, Linux, Windows
- **Go版本**: 1.19+
- **依赖工具**: cjxl, djxl, ffmpeg, exiftool

### 安装依赖

```bash
# macOS (使用Homebrew)
brew install libjxl ffmpeg exiftool

# Ubuntu/Debian
sudo apt-get install libjxl-tools ffmpeg exiftool

# 验证安装
cjxl --version
djxl --version
ffmpeg -version
exiftool -ver
```

### 编译工具

```bash
# 编译所有工具
make all

# 或使用构建脚本
./build_all.sh
```

## 🎯 主要工具使用

### 1. 通用转换器 (universal_converter)

**功能**: 一个工具支持所有转换类型和模式

#### 基本用法

```bash
# 转换图像为JXL格式
./bin/universal_converter -input /path/to/images -type jxl

# 转换图像为AVIF格式
./bin/universal_converter -input /path/to/images -type avif

# 转换视频为MOV格式
./bin/universal_converter -input /path/to/videos -type mov
```

#### 高级用法

```bash
# 高质量转换（无损）
./bin/universal_converter -input /path/to/images -type jxl -quality 100

# 多线程处理
./bin/universal_converter -input /path/to/images -type jxl -workers 4

# 只处理静态图像
./bin/universal_converter -input /path/to/images -type jxl -mode static

# 只处理动态图像（GIF动画等）
./bin/universal_converter -input /path/to/images -type jxl -mode dynamic

# 试运行模式（查看将要处理的文件）
./bin/universal_converter -input /path/to/images -type jxl -dry-run
```

#### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-input` | 输入目录路径 | 必需 |
| `-type` | 转换类型 (avif/jxl/mov) | jxl |
| `-mode` | 处理模式 (all/static/dynamic/video) | all |
| `-quality` | 输出质量 (1-100) | 90 |
| `-workers` | 工作线程数 (0=自动) | 0 |
| `-dry-run` | 试运行模式 | false |
| `-skip-exist` | 跳过已存在文件 | false |

### 2. 媒体工具集 (media_tools)

**功能**: 元数据管理、文件去重、扩展名标准化

#### 自动模式

```bash
# 自动处理：合并XMP + 去重 + 标准化
./bin/media_tools auto -dir /path/to/media -trash /path/to/trash
```

#### 单独功能

```bash
# 只合并XMP元数据
./bin/media_tools merge -dir /path/to/media

# 只去重文件
./bin/media_tools deduplicate -dir /path/to/media -trash /path/to/trash

# 只标准化扩展名
./bin/media_tools normalize -dir /path/to/media
```

## 🎬 动图处理指南

### GIF动画转JXL动画

```bash
# 转换GIF动画为JXL动画
./bin/universal_converter -input /path/to/gifs -type jxl -mode dynamic

# 高质量动画转换
./bin/universal_converter -input /path/to/gifs -type jxl -mode dynamic -quality 100
```

### 验证动画转换

```bash
# 检查JXL文件是否为动画
file animation.jxl
# 输出: animation.jxl: JPEG XL container

# 使用djxl查看动画信息
djxl animation.jxl -v /dev/null
# 输出包含: Animation: X frames
```

## 📊 性能优化

### 系统配置建议

| 文件数量 | 推荐线程数 | 内存需求 | 处理时间 |
|----------|------------|----------|----------|
| < 100个 | 2-4线程 | 2-4GB | 5-15分钟 |
| 100-1000个 | 4-8线程 | 4-8GB | 15-60分钟 |
| > 1000个 | 8-16线程 | 8-16GB | 1-4小时 |

### 性能调优

```bash
# 高性能服务器配置
./bin/universal_converter -input /path/to/media -type jxl -workers 8 -quality 90

# 低配置机器配置
./bin/universal_converter -input /path/to/media -type jxl -workers 2 -quality 80

# 大文件优化
./bin/universal_converter -input /path/to/media -type jxl -workers 4 -timeout 300
```

## 🔍 质量验证

### 8层验证系统

系统自动执行8层验证确保转换质量：

1. **基础文件验证** - 检查文件存在性和可读性
2. **文件大小验证** - 验证转换前后文件大小合理性
3. **格式完整性验证** - 使用专业工具验证文件格式
4. **元数据验证** - 检查EXIF、IPTC、XMP元数据
5. **像素数据验证** - 验证图像像素数据完整性
6. **色彩空间验证** - 检查色彩空间转换正确性
7. **压缩质量验证** - 验证压缩参数和视觉效果
8. **性能验证** - 检查处理时间和资源使用

### 抽样验证

- **抽样率**: 10%（可配置）
- **最少样本**: 5个文件
- **最多样本**: 20个文件
- **验证通过率**: 95%以上为合格

## 📝 日志和监控

### 日志文件

- `universal_converter.log` - 转换器日志
- `media_tools.log` - 媒体工具日志
- 日志自动轮转（50MB限制）

### 监控指标

```bash
# 查看处理统计
tail -f universal_converter.log | grep "📊"

# 查看错误信息
grep "❌" universal_converter.log

# 查看性能统计
grep "⏱️" universal_converter.log
```

## 🛠️ 故障排除

### 常见问题

#### 1. 依赖工具未找到

```bash
# 检查工具是否安装
which cjxl djxl ffmpeg exiftool

# 如果未安装，请安装相应工具
brew install libjxl ffmpeg exiftool  # macOS
```

#### 2. 内存不足

```bash
# 减少线程数
./bin/universal_converter -input /path/to/media -type jxl -workers 2

# 降低质量设置
./bin/universal_converter -input /path/to/media -type jxl -quality 80
```

#### 3. 处理超时

```bash
# 增加超时时间
./bin/universal_converter -input /path/to/media -type jxl -timeout 600
```

#### 4. 动图转换失败

```bash
# 检查源文件是否为动画
file source.gif
# 应该显示: source.gif: GIF image data, animated

# 使用严格模式
./bin/universal_converter -input /path/to/gifs -type jxl -mode dynamic -strict
```

### 调试模式

```bash
# 启用详细日志
./bin/universal_converter -input /path/to/media -type jxl -verbose

# 试运行模式
./bin/universal_converter -input /path/to/media -type jxl -dry-run
```

## 📈 最佳实践

### 1. 预处理

```bash
# 先合并元数据
./bin/media_tools merge -dir /path/to/media

# 再去重文件
./bin/media_tools deduplicate -dir /path/to/media -trash /path/to/trash

# 最后进行转换
./bin/universal_converter -input /path/to/media -type jxl
```

### 2. 批量处理

```bash
# 分批处理大量文件
./bin/universal_converter -input /path/to/batch1 -type jxl -workers 4
./bin/universal_converter -input /path/to/batch2 -type jxl -workers 4
```

### 3. 质量保证

```bash
# 使用严格模式确保质量
./bin/universal_converter -input /path/to/media -type jxl -strict

# 验证转换结果
./bin/universal_converter -input /path/to/media -type jxl -validate
```

## 🔧 高级配置

### 自定义参数

```bash
# 自定义CJXL线程数
./bin/universal_converter -input /path/to/media -type jxl -cjxl-threads 8

# 自定义重试次数
./bin/universal_converter -input /path/to/media -type jxl -retries 3

# 自定义超时时间
./bin/universal_converter -input /path/to/media -type jxl -timeout 300
```

### 环境变量

```bash
# 设置日志级别
export LOG_LEVEL=DEBUG

# 设置最大内存使用
export MAX_MEMORY=8GB

# 设置临时目录
export TMPDIR=/path/to/temp
```

## 📚 更多资源

- [技术架构文档](TECHNICAL_ARCHITECTURE.md)
- [动图处理指南](ANIMATION_PROCESSING_GUIDE.md)
- [验证策略文档](VALIDATION_STRATEGY.md)
- [测试报告](TEST_REPORT_v2.1.0.md)

## 🆘 获取帮助

### 命令行帮助

```bash
# 查看所有参数
./bin/universal_converter -help

# 查看媒体工具帮助
./bin/media_tools -help
```

### 日志分析

```bash
# 分析处理统计
grep "📊" universal_converter.log

# 分析错误信息
grep "❌" universal_converter.log

# 分析性能信息
grep "⏱️" universal_converter.log
```

---

**文档版本**: v2.2.0  
**最后更新**: 2025-10-24  
**维护者**: AI Assistant
