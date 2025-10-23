# Universal Converter v2.2.1 优化报告

## 🎯 优化目标
解决大规模转换时出现的 `signal: killed` 错误,提升系统稳定性和资源利用率。

## ❌ 问题分析

### 原始问题
- **184个文件失败**: 全部报错 `signal: killed`
- **成功处理**: 670个文件
- **失败率**: 21.5% (184/854)

### 根本原因
1. **固定effort=9**: 对大文件(>10MB)极其消耗内存
2. **固定超时30秒**: 大文件需要更长时间处理
3. **过度并发**: 6个worker + 10核心CPU可能导致内存耗尽
4. **系统OOM杀进程**: 内存不足时系统强制终止进程

## ✅ 优化方案

### 1. 智能Effort策略 (Dynamic Effort)

**位置**: `utils/parameters.go` - `getSmartEffort()`

```go
// 动态effort策略:
< 500KB:   effort 9 (最高压缩)
< 2MB:     effort 8
< 5MB:     effort 7
< 10MB:    effort 6
>= 10MB:   effort 5 (避免内存耗尽)
```

**效果**:
- ✅ 小文件保持最高压缩率
- ✅ 大文件降低内存占用
- ✅ 自适应文件大小

### 2. 智能超时机制 (Dynamic Timeout)

**位置**: `universal_converter/main.go` - `getSmartTimeout()`

```go
// 动态超时策略:
< 500KB:   30秒
< 2MB:     60秒
< 5MB:     120秒 (2分钟)
< 10MB:    300秒 (5分钟)
>= 10MB:   600秒 (10分钟)
```

**效果**:
- ✅ 小文件快速失败
- ✅ 大文件有足够处理时间
- ✅ 避免超时导致的进程杀死

### 3. 智能并发控制 (Smart Concurrency)

**位置**: `utils/parameters.go` - `getSmartProcessLimit()` & `getSmartCJXLThreads()`

#### 并发进程数 (ProcessLimit)
```go
CPU核心 >= 10 (M4/M2等): 4个并发进程
CPU核心 >= 8:            3个并发进程
CPU核心 >= 4:            2个并发进程
CPU核心 < 4:             1个并发进程
```

#### 每进程线程数 (CJXLThreads)
```go
CPU核心 >= 10:  4线程/进程
CPU核心 >= 8:   3线程/进程
CPU核心 >= 4:   2线程/进程
CPU核心 < 4:    1线程/进程
```

**效果**:
- ✅ **Mac Mini M4 (10核)**: 4进程 × 4线程 = 16并发任务
- ✅ 预留资源给系统和其他任务
- ✅ 避免过度并发导致内存耗尽
- ✅ 自动适配不同CPU型号

### 4. 通用CPU检测

**特性**:
- ✅ 自动检测CPU核心数(`runtime.NumCPU()`)
- ✅ 适配各种CPU架构(M1/M2/M4, Intel, AMD)
- ✅ 不依赖硬编码配置

## 📊 优化效果预测

### 内存占用
- **优化前**: 6进程 × effort9 × 大文件 = 极高内存
- **优化后**: 4进程 × effort5-9 × 智能分级 = 可控内存

### 处理速度
- **小文件(<2MB)**: 保持最高压缩质量(effort9)
- **大文件(>10MB)**: 牺牲少量压缩率,换取稳定性(effort5)

### 成功率
- **预期**: 失败率从21.5%降低到 <5%
- **原因**: 动态资源分配 + 充足超时时间

## 🚀 使用建议

### 默认配置(推荐)
```bash
./universal_converter -input "your_folder" -type jxl
```
- 自动检测CPU并优化配置
- 适合大多数场景

### 手动调优(高级)
```bash
# 更保守的配置(低内存机器)
./universal_converter -input "your_folder" -type jxl \
  -process-limit 2 -cjxl-threads 2

# 更激进的配置(高内存机器)
./universal_converter -input "your_folder" -type jxl \
  -process-limit 6 -cjxl-threads 4
```

### Mac Mini M4专用配置
```bash
# 默认配置已优化为:
# - ProcessLimit: 4
# - CJXLThreads: 4
# - FileLimit: 8
# 无需手动调整
```

## 🔬 测试验证

### 建议测试步骤
1. **小规模测试**: 先测试50-100个文件
2. **观察资源**: 使用`Activity Monitor`监控内存/CPU
3. **检查日志**: 查找`signal: killed`错误
4. **调整参数**: 根据实际情况微调

### 监控命令
```bash
# 实时监控内存
watch -n 1 "ps aux | grep universal_converter | grep -v grep"

# 查看错误日志
tail -f universal_converter.log | grep "❌"
```

## 📝 技术细节

### 数学无损保证
- ✅ **JPEG**: `--lossless_jpeg=1` (保留DCT系数)
- ✅ **PNG/GIF**: `-d 0` (distance=0, 数学无损)
- ✅ **Quality参数被忽略**: 代码直接使用无损参数

### Effort对压缩的影响
- **Effort 9**: 最高压缩,最慢速度,最高内存(适合<2MB文件)
- **Effort 7**: 平衡压缩/速度/内存(适合<5MB文件)
- **Effort 5**: 快速压缩,较低内存(适合>10MB文件)
- **压缩率差异**: effort 5 vs 9 通常差异<5%

## 🛠️ 故障排查

### 如果仍出现killed错误

1. **降低并发**:
   ```bash
   ./universal_converter -input "folder" -type jxl \
     -process-limit 2 -cjxl-threads 2
   ```

2. **增加超时**:
   ```bash
   ./universal_converter -input "folder" -type jxl \
     -timeout 120
   ```

3. **查看系统日志**:
   ```bash
   # macOS
   log show --predicate 'eventMessage contains "killed"' --last 1h
   ```

4. **检查可用内存**:
   ```bash
   vm_stat | grep "Pages free"
   ```

## 📈 版本历史

### v2.2.1 (当前版本)
- ✅ 智能effort动态调整
- ✅ 智能超时机制
- ✅ 智能并发控制
- ✅ 通用CPU检测

### v2.2.0
- XMP合并与验证
- 8层验证系统
- 后处理验证

### v2.1.0
- 初始版本
- 基础转换功能

---

**作者**: AI Assistant  
**日期**: 2025-10-23  
**适用平台**: macOS (M1/M2/M4), Linux, Windows

