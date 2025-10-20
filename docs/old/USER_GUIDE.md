# Pixly 媒体转换工具使用指南

## 📖 概述

Pixly 是一个高性能的媒体文件转换工具，专注于现代格式的转换和存储效率优化。支持智能转换策略，能够在保持质量的同时显著减少文件大小。

## 🚀 快速开始

### 安装依赖

在使用Pixly之前，请确保系统已安装以下依赖：

```bash
# macOS (使用Homebrew)
brew install ffmpeg imagemagick cjxl avifenc exiftool

# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg imagemagick libjxl-tools libavif-bin exiftool

# 检查依赖状态
./pixly deps
```

### 基本使用

```bash
# 编译程序
go build -o pixly main.go

# 查看帮助
./pixly --help

# 转换指定目录的文件
./pixly convert /path/to/your/images

# 使用特定模式转换
./pixly convert /path/to/your/images --mode emoji

# 详细日志模式
./pixly convert /path/to/your/images --verbose
```

## 🎯 转换模式详解

### 1. 🤖 自动模式+ (auto+) - 默认推荐

**适用场景**: 日常使用，追求质量与体积的最佳平衡

**转换策略**:
- 静态图片 → JXL格式
- 动态图片 → AVIF格式（无损）
- 视频文件 → MOV格式（重包装）

**智能决策**:
- 高品质文件：自动路由至quality模式处理
- 中等品质文件：应用平衡优化算法
- 低品质文件：有损压缩探测

```bash
./pixly convert /path/to/images --mode auto+
```

### 2. 🔥 品质模式 (quality)

**适用场景**: 专业用途，归档存储，要求最高保真度

**转换策略**:
- 静态图片 → JXL格式（最高压缩等级 -e 9）
- 动态图片 → AVIF格式（无损）
- 视频文件 → MOV格式（重包装）

**特点**:
- 无损优先
- 最大保真度
- 适合长期存储

```bash
./pixly convert /path/to/images --mode quality
```

### 3. 🚀 表情包模式 (emoji)

**适用场景**: 网络分享，社交媒体，追求极限压缩

**转换策略**:
- 所有图片 → AVIF格式
- 视频文件 → 跳过处理

**压缩目标**:
- 体积减小7%-13%或更多时视为成功
- 针对网络传输优化
- 保持可接受的视觉质量

```bash
./pixly convert /path/to/images --mode emoji
```

## 📁 支持的文件格式

### 输入格式
- **图片**: JPG, PNG, GIF, WebP, HEIC, TIFF, JXL, AVIF
- **视频**: MP4, MOV, AVI, WebM, MKV
- **文档**: PDF

### 输出格式
- **现代图片格式**: JXL, AVIF
- **视频容器**: MOV (重包装)

## ⚙️ 高级选项

### 并发控制
```bash
# 设置并发数（默认为CPU核心数）
./pixly convert /path/to/images --concurrent 8
```

### 输出目录
```bash
# 指定输出目录（默认为原目录）
./pixly convert /path/to/images --output /path/to/output
```

### 配置文件
```bash
# 使用自定义配置文件
./pixly convert /path/to/images --config /path/to/config.yaml
```

## 📊 转换报告

每次转换完成后，Pixly会生成详细的转换报告：

### 报告位置
- **详细报告**: `reports/conversion/pixly_detailed_report_YYYYMMDD_HHMMSS.json`
- **可读报告**: `reports/conversion/pixly_report_YYYYMMDD_HHMMSS.txt`

### 报告内容
- 转换统计（成功/失败/跳过文件数）
- 格式分布和压缩率
- 文件处理详情
- 空间节省统计
- 处理时间分析

## 🔧 故障排除

### 常见问题

#### 1. 依赖缺失
```bash
# 检查依赖状态
./pixly deps

# 根据提示安装缺失的依赖
```

#### 2. 转换失败
```bash
# 使用详细日志查看错误信息
./pixly convert /path/to/images --verbose

# 检查日志文件
tail -f logs/pixly.log
```

#### 3. 权限问题
```bash
# 确保对目标目录有写权限
ls -la /path/to/images

# 如需要，调整权限
chmod 755 /path/to/images
```

#### 4. 内存不足
```bash
# 减少并发数
./pixly convert /path/to/images --concurrent 2

# 分批处理大量文件
```

### 错误代码
- **退出码 0**: 成功完成
- **退出码 1**: 一般错误
- **退出码 2**: 配置错误
- **退出码 3**: 依赖缺失
- **退出码 4**: 权限错误
- **退出码 5**: 内存不足

## 🧪 测试套件

### 运行测试
```bash
# 运行完整测试套件
./pixly testsuite

# 运行特定测试
./unified_test_executor
```

### 测试类型
1. **UI交互测试**: 验证用户界面功能
2. **转换器测试**: 验证核心转换逻辑
3. **依赖检查测试**: 验证系统依赖
4. **性能测试**: 验证处理性能

## 📈 性能优化建议

### 1. 硬件优化
- **CPU**: 多核处理器，支持更高并发
- **内存**: 8GB+推荐，处理大文件时需要更多内存
- **存储**: SSD硬盘，提升I/O性能

### 2. 软件配置
```bash
# 根据系统配置调整并发数
./pixly convert /path/to/images --concurrent $(nproc)

# 大文件处理时减少并发
./pixly convert /path/to/large/files --concurrent 2
```

### 3. 批处理策略
- 按文件大小分组处理
- 优先处理小文件
- 大文件单独处理

## 🔒 安全注意事项

### 文件安全
- Pixly使用原子操作确保文件安全
- 转换过程中创建临时文件
- 失败时自动回滚到原始状态

### 权限控制
- 默认仅允许在用户主目录操作
- 不允许直接操作系统级目录
- 操作前检查权限和磁盘空间

### 数据备份
- 重要文件请提前备份
- 使用测试目录验证转换效果
- 批量处理前先小规模测试

## 📞 技术支持

### 获取帮助
```bash
# 查看命令帮助
./pixly --help
./pixly convert --help

# 查看版本信息
./pixly --version
```

### 报告问题
1. 收集错误日志
2. 记录系统环境信息
3. 提供复现步骤
4. 包含转换报告

### 日志位置
- **应用日志**: `logs/pixly.log`
- **转换报告**: `reports/conversion/`
- **调试信息**: 使用`--verbose`标志

---

**提示**: 首次使用建议先在小规模测试目录中验证转换效果，确认满意后再进行大批量处理。