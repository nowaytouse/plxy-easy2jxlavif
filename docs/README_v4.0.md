# Pixly v4.0 🚀

**专业级图像转换工具 - 模块化 • 智能化 • 国际化**

[![Version](https://img.shields.io/badge/version-4.0.0-blue.svg)](https://github.com/your-repo/pixly)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)

---

## ✨ 特性

### 🎯 核心功能

- **智能转换**: 自动分析图像质量并调整转换参数
- **性能监控**: 实时CPU/内存/磁盘监控，动态调整工作线程
- **断点续传**: BoltDB存储，支持崩溃恢复
- **多语言**: 简体中文和English双语界面
- **灵活配置**: 200+可配置参数，YAML/ENV/CLI三级优先级

### 📦 支持格式

**输入格式**:
- 静态图片: JPG, PNG, BMP, TIFF, WebP, HEIC
- 动态图片: GIF, APNG, WebP动图
- 视频: MP4, AVI, MKV, MOV, FLV

**输出格式**:
- 现代图像: JPEG XL, AVIF
- 视频编码: AV1, H.265, H.266/VVC
- 容器: MP4, MOV

---

## 📊 v4.0 新特性

### 1️⃣ 性能监控系统

```go
// 自动监控并调整工作线程
monitor := monitor.NewSystemMonitor()
optimizer := optimizer.NewDynamicOptimizer()

// 实时显示监控面板
panel := ui.NewMonitorPanel()
panel.Start()
```

**特性**:
- 实时CPU/内存/磁盘监控
- 动态工作线程调整
- 性能报告生成

### 2️⃣ YAML配置系统

```yaml
# ~/.pixly/config.yaml
project:
  name: "我的图片库"
  target_dir: "~/Pictures"

concurrency:
  workers: 8
  auto_adjust: true

conversion:
  default_mode: "auto"
  quality_analysis: true
```

**特性**:
- 200+可配置参数
- 多级优先级（YAML < ENV < CLI）
- 自动验证和迁移

### 3️⃣ 质量评估增强

```go
// 分析图像质量
analyzer := quality.NewAnalyzer()
metrics, _ := analyzer.Analyze("photo.jpg")

// 基于质量调整参数
adjuster := predictor.NewQualityAdjuster()
prediction := adjuster.AdjustParams(prediction, metrics)
```

**特性**:
- BytesPerPixel分析
- 内容类型识别
- 自动参数优化

### 4️⃣ BoltDB断点续传

```go
// 创建会话
manager := checkpoint.NewManager("sessions.db", 10)
manager.CreateSession(sessionID, targetDir, outputDir, mode, inPlace)

// 处理文件（自动保存）
manager.RecordFileComplete(...)

// 恢复中断会话
manager.LoadSession(sessionID)
```

**特性**:
- ACID事务保证
- 多会话并行
- 自动崩溃恢复

### 5️⃣ 多语言支持

```go
// 初始化（自动检测系统语言）
i18n.Init(i18n.ZhCN)

// 翻译消息
fmt.Println(i18n.T(i18n.MsgWelcome))

// 切换语言
i18n.SetLocale(i18n.EnUS)
```

**特性**:
- 双语支持（中英）
- 100+条翻译
- 零性能开销

---

## 🏗️ 架构

```
pixly/
├── cmd/pixly/           # 主程序入口
├── pkg/                 # 核心模块
│   ├── monitor/         # 性能监控
│   ├── optimizer/       # 动态优化
│   ├── config/          # 配置系统
│   ├── quality/         # 质量评估
│   ├── checkpoint/      # 断点续传
│   └── i18n/            # 多语言
├── easymode/archive/    # 归档工具集
│   ├── shared/          # 共享模块
│   ├── dynamic2mov/     # 动图→视频
│   ├── dynamic2avif/    # 动图→AVIF
│   ├── dynamic2jxl/     # 动图→JXL
│   ├── static2avif/     # 静图→AVIF
│   ├── static2jxl/      # 静图→JXL
│   └── video2mov/       # 视频重编码
└── docs/                # 文档
```

---

## 📖 使用指南

### 主程序使用

```bash
# 基础转换
./pixly convert /path/to/images

# 指定输出目录
./pixly convert /path/to/images -o /path/to/output

# 指定格式和质量
./pixly convert images/ --format jxl --quality 90

# 启用监控
./pixly convert images/ --monitor --workers 16

# 恢复会话
./pixly convert --resume
```

### 归档工具使用

所有归档工具支持两种模式：

**交互模式**（无参数启动）:
```bash
./dynamic2mov-darwin-arm64
# 按提示操作：拖入文件夹 → 选择选项 → 开始转换
```

**命令行模式**（带参数）:
```bash
./dynamic2mov-darwin-arm64 \
  -dir /path/to/gifs \
  --codec av1 \
  --format mp4 \
  --workers 8 \
  --in-place
```

---

## 🎨 归档工具详解

### dynamic2mov - 动图转视频

**特点**: 最全面的动图转换工具

```bash
# AV1编码（最高压缩率）
./dynamic2mov-darwin-arm64 -dir gifs/ --codec av1 --format mp4

# H.265编码（广泛兼容）
./dynamic2mov-darwin-arm64 -dir gifs/ --codec h265 --format mov

# 自动选择（推荐）
./dynamic2mov-darwin-arm64 -dir gifs/ --codec auto
```

### dynamic2avif / dynamic2jxl - 动图转现代格式

```bash
# 转换为AVIF
./dynamic2avif-darwin-arm64 -dir gifs/ --workers 8

# 转换为JPEG XL
./dynamic2jxl-darwin-arm64 -dir gifs/ --effort 9
```

### static2avif / static2jxl - 静图转换

```bash
# 批量转换照片为JPEG XL
./static2jxl-darwin-arm64 -dir photos/ --effort 9

# 批量转换PNG为AVIF
./static2avif-darwin-arm64 -dir screenshots/ --workers 12
```

### video2mov - 视频重编码

```bash
# 重编码为H.265 MOV
./video2mov-darwin-arm64 -dir videos/ --workers 4
```

---

## 🔍 性能优化建议

### CPU密集型任务

```yaml
# config.yaml
concurrency:
  workers: 16  # 根据CPU核心数调整
  cpu_threshold: 85
```

### 内存受限环境

```yaml
concurrency:
  workers: 4
  memory_threshold: 70
  auto_adjust: true
```

### 大文件转换

```yaml
conversion:
  effort_auto: true  # 大文件自动降低effort
  timeout: 3600      # 增加超时时间
```

---

## 📈 性能指标

### 转换速度（参考值）

| 任务 | 配置 | 速度 |
|------|------|------|
| 1000张照片→JXL | 8 workers | ~10分钟 |
| 100个GIF→AV1 | 4 workers | ~5分钟 |
| 50个视频→H.265 | 2 workers | ~20分钟 |

*实际速度取决于文件大小、硬件配置和质量设置*

### 压缩效果（参考值）

| 格式转换 | 空间节省 | 质量损失 |
|---------|---------|---------|
| PNG→JXL | 70-85% | 无损 |
| JPG→AVIF | 30-50% | 极小 |
| GIF→AV1 | 75-90% | 极小 |

---

## 🛠️ 开发

### 编译主程序

```bash
cd cmd/pixly
go build -o pixly

# 或使用Makefile
make build
```

### 编译归档工具

```bash
cd easymode/archive

# 编译所有工具
for tool in dynamic2mov dynamic2avif dynamic2jxl static2avif static2jxl video2mov; do
    cd $tool
    go build -o bin/${tool}-darwin-arm64 .
    cd ..
done
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./pkg/checkpoint/...
go test ./pkg/i18n/...

# 详细输出
go test -v ./pkg/...
```

---

## 📝 更新日志

### v4.0.0 (2025-10-25)

**新增**:
- ✅ 性能监控系统（1061行）
- ✅ YAML配置系统（2000行）
- ✅ 质量评估增强（835行）
- ✅ BoltDB断点续传（945行）
- ✅ 多语言支持（760行）

**改进**:
- ✅ 归档工具共享模块化（-1400行重复代码）
- ✅ 连续转换模式（无需重启）
- ✅ 单文件支持
- ✅ 原地转换选项
- ✅ 失败保护机制

**总计**:
- 28个核心模块
- ~5,600行新代码
- ~2,900行文档

---

## 🙏 致谢

感谢所有贡献者和用户！

---

## 📄 许可

MIT License

---

**Pixly v4.0 - 让图像转换更专业** 🎨
