# video2mov - 视频转MOV格式工具

## 📋 功能描述

视频转MOV格式工具，基于 universal_converter 和 media_tools 功能进行深入优化。

## 🔧 输入输出格式

- **输入格式**: MP4, AVI, MKV, MOV
- **输出格式**: MOV

## 🚀 使用方法

### 构建工具
```bash
./build.sh
```

### 基本用法
```bash
./bin/video2mov -dir /path/to/input -workers 4
```

### 参数说明
- `-dir`: 输入目录路径（必需）
- `-output`: 输出目录路径（默认为输入目录）
- `-workers`: 工作线程数（0=自动检测）
- `-skip-exist`: 跳过已存在的文件
- `-dry-run`: 试运行模式
- `-timeout`: 处理超时时间（秒）
- `-retries`: 重试次数
- `-max-memory`: 最大内存使用量
- `-health-check`: 启用健康检查

## ✨ 优化特性

- **增强错误处理和恢复机制**
- **改进资源管理和内存控制**
- **优化并发控制和性能**
- **增强日志记录和监控**
- **添加信号处理和优雅关闭**
- **改进参数验证和配置**
- **增强统计和报告功能**
- **添加健康监控和错误分类**
- **实现智能性能调优**
- **增强安全性和稳定性**

## 📊 性能特性

- 智能线程数检测
- 内存使用监控
- 文件大小限制
- 并发控制
- 详细统计报告
- 错误分类分析

## 🔧 技术依赖

- Go 1.25.3+
- 系统工具: cjxl, djxl, avifenc, ffmpeg, exiftool
- Go模块: godirwalk, gopsutil

## 📈 版本信息

- **当前版本**: v2.3.1 (优化版)
- **作者**: AI Assistant
- **基于**: universal_converter 和 media_tools 功能优化
