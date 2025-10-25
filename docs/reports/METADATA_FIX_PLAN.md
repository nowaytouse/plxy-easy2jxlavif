# 🔧 元数据保留全面修复计划

**日期**: 2025-10-25  
**严重性**: 🔴 严重 - 所有视频/动图转换都在丢失元数据！  
**影响范围**: 整个plxy-easy2jxlavif项目

---

## 🚨 发现的问题

### 1️⃣ 主程序 Pixly (v3.1.1)

**文件**: `pkg/engine/balance_optimizer.go`  
**问题位置**: `executeMOVRepackage` 函数（第749-769行）

**当前代码** ❌:
```go
args := []string{
    "-i", filePath,
    "-c", "copy",
    "-avoid_negative_ts", "make_zero",
    "-f", "mov",
    "-y", outputPath,
}
```

**问题**: 
- ❌ 缺少 `-map_metadata 0`（复制全部元数据）
- ❌ 缺少 `-movflags use_metadata_tags`（保留MOV元数据标签）
- ❌ 所有EXIF/XMP/GPS/创建时间等元数据**全部丢失**！

**影响**:
- 视频拍摄日期丢失
- GPS位置信息丢失
- 相机/手机型号丢失
- 编辑软件信息丢失

---

### 2️⃣ easymode工具（archive）

#### dynamic2avif（动图→AVIF）
**文件**: `easymode/archive/dynamic2avif/main.go`  
**状态**: ❌ 定义了`copyMetadata`函数，但**从未调用**

#### video2mov（视频→MOV）
**文件**: `easymode/archive/video2mov/main.go`  
**状态**: ❌ 定义了`copyMetadata`函数，但**从未调用**

#### static2jxl/static2avif/dynamic2jxl
**文件**: `easymode/archive/static2*/dynamic2jxl/main.go`  
**状态**: ❌ 同样问题

---

### 3️⃣ easymode工具（已修复）✅

#### universal_converter
**文件**: `easymode/universal_converter/main.go`  
**状态**: ✅ 已正确实现（第552-559行）
```go
if opts.CopyMetadata {
    if err := copyMetadata(filePath, outputPath); err != nil {
        logger.Printf("⚠️  元数据复制失败 %s (非致命): %v", fileName, err)
    }
}
```

#### all2jxl / all2avif
**文件**: `easymode/archive/all2jxl/main.go` 等  
**状态**: ✅ 已正确实现（第553/560行）

---

## 🔧 修复方案

### 阶段一: 修复主程序 Pixly（最高优先级）🔴

#### 1.1 修复 `balance_optimizer.go` - 视频重封装

**位置**: `pkg/engine/balance_optimizer.go:749-769`

**修复代码**:
```go
// executeMOVRepackage 执行MOV重封装（v3.1.1+元数据保留）
func (bo *BalanceOptimizer) executeMOVRepackage(
    ctx context.Context,
    filePath string,
    originalSize int64,
) *OptimizationResult {
    startTime := time.Now()

    dir := filepath.Dir(filePath)
    base := filepath.Base(filePath)
    ext := filepath.Ext(base)
    nameWithoutExt := base[:len(base)-len(ext)]
    outputPath := filepath.Join(dir, nameWithoutExt+".mov")

    // 视频重封装：仅改容器，不重编码（快速！）
    // ✅ 新增：完整保留元数据
    args := []string{
        "-i", filePath,
        "-c", "copy",                      // 复制编码流
        "-map_metadata", "0",              // ✅ 复制所有元数据
        "-movflags", "use_metadata_tags",  // ✅ 保留MOV元数据标签
        "-avoid_negative_ts", "make_zero", // 修复时间戳
        "-f", "mov",                       // MOV格式
        "-y", outputPath,                  // 覆盖输出
    }

    cmd := exec.CommandContext(ctx, bo.toolPaths.FfmpegStablePath, args...)

    bo.logger.Info("🎬 视频重封装（-c copy + 元数据保留）",
        zap.String("file", filepath.Base(filePath)))

    output, err := cmd.CombinedOutput()
    if err != nil {
        bo.logger.Warn("MOV重封装失败",
            zap.String("file", filepath.Base(filePath)),
            zap.String("output", string(output)),
            zap.Error(err))
        return nil
    }

    // 检查输出文件
    outputInfo, err := os.Stat(outputPath)
    if err != nil {
        return nil
    }

    newSize := outputInfo.Size()

    bo.logger.Info("🎬 MOV重封装完成（快速+元数据保留）",
        zap.String("file", filepath.Base(filePath)),
        zap.Duration("time", time.Since(startTime)))

    return &OptimizationResult{
        Success:      true,
        OutputPath:   outputPath,
        OriginalSize: originalSize,
        NewSize:      newSize,
        SpaceSaved:   originalSize - newSize,
        Method:       "mov_repackage_with_metadata",
        ProcessTime:  time.Since(startTime),
    }
}
```

**关键修复**:
1. ✅ 添加 `-map_metadata 0` - 复制所有元数据流
2. ✅ 添加 `-movflags use_metadata_tags` - 保留MOV特有的元数据标签
3. ✅ 记录output用于调试

---

#### 1.2 修复 `simple_converter.go` - 视频重封装

**位置**: `pkg/engine/simple_converter.go:237`

**当前代码** ❌:
```go
args := []string{"-i", sourcePath, "-c", "copy", "-y", targetPath}
```

**修复代码** ✅:
```go
args := []string{
    "-i", sourcePath,
    "-c", "copy",
    "-map_metadata", "0",              // ✅ 复制元数据
    "-movflags", "use_metadata_tags",  // ✅ MOV元数据标签
    "-y", targetPath,
}
```

---

#### 1.3 修复 `conversion_engine.go` - 视频重封装

**位置**: `pkg/engine/conversion_engine.go:1519-1537`

**当前代码** ❌:
```go
var args []string
args = append(args, "-i", task.SourcePath)
args = append(args, "-c", "copy")
args = append(args, "-avoid_negative_ts", "make_zero")
// ... 缺少元数据参数
```

**修复代码** ✅:
```go
var args []string
args = append(args, "-i", task.SourcePath)
args = append(args, "-c", "copy")
args = append(args, "-map_metadata", "0")              // ✅ 元数据
args = append(args, "-movflags", "use_metadata_tags")  // ✅ MOV标签
args = append(args, "-avoid_negative_ts", "make_zero")
```

---

### 阶段二: 修复 easymode archive 工具

#### 2.1 创建统一的元数据复制函数

**文件**: `easymode/utils/metadata.go`

**当前实现** ✅（已存在）:
```go
// CopyMetadataWithTimeout 使用exiftool在超时内复制元数据
func CopyMetadataWithTimeout(ctx context.Context, src, dst string, timeoutSec int) error {
    c, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
    defer cancel()

    cmd := exec.CommandContext(c, "exiftool", "-overwrite_original", "-TagsFromFile", src, dst)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("exiftool failed: %v, out=%s", err, string(out))
    }
    return nil
}
```

**改进**: 添加更详细的日志和错误处理

---

#### 2.2 修复所有 archive 工具

需要修改的文件列表:
1. `easymode/archive/dynamic2avif/main.go`
2. `easymode/archive/video2mov/main.go`
3. `easymode/archive/static2jxl/main.go`
4. `easymode/archive/static2avif/main.go`
5. `easymode/archive/dynamic2jxl/main.go`

**统一修复模式**:

在每个工具的 `processFileByType` 函数中，**转换成功后**立即调用元数据复制：

```go
func processFileByType(filePath string, opts Options) (string, string, string, error) {
    // ... 执行转换 ...
    
    // ✅ 转换成功后，立即复制元数据
    if err := copyMetadata(filePath, outputPath); err != nil {
        logger.Printf("⚠️  元数据复制失败: %s -> %s: %v", 
            filepath.Base(filePath), filepath.Base(outputPath), err)
        // 不返回错误，因为转换本身成功了
    } else {
        logger.Printf("✅ 元数据复制成功: %s", filepath.Base(outputPath))
    }
    
    return conversionMode, outputPath, "", nil
}
```

---

### 阶段三: 验证和测试

#### 3.1 创建元数据测试脚本

**文件**: `tests/metadata_validation_test.sh`

```bash
#!/bin/bash

# 元数据保留验证测试
# 测试所有转换工具是否正确保留元数据

echo "🔍 元数据保留验证测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 检查依赖
if ! command -v exiftool &> /dev/null; then
    echo "❌ exiftool 未安装"
    exit 1
fi

# 创建测试文件夹
TEST_DIR="/tmp/pixly_metadata_test"
mkdir -p "$TEST_DIR"

# 测试1: 视频元数据保留（主程序）
echo ""
echo "📹 测试1: 视频元数据保留（Pixly主程序）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TEST_VIDEO="$TEST_DIR/test_video.mp4"

# 创建测试视频并添加元数据
ffmpeg -f lavfi -i testsrc=duration=1:size=320x240:rate=30 \
    -metadata title="Test Video" \
    -metadata comment="Metadata Test" \
    -metadata creation_time="2025-10-25T08:00:00Z" \
    -y "$TEST_VIDEO" 2>/dev/null

# 提取原始元数据
echo "📊 原始元数据:"
exiftool -Title -Comment -CreateDate "$TEST_VIDEO"

# 使用Pixly转换
# ... (调用pixly_interactive或直接调用balance_optimizer)

# 提取转换后元数据
# echo "📊 转换后元数据:"
# exiftool -Title -Comment -CreateDate "$TEST_DIR/test_video.mov"

# 对比元数据
# ...

# 测试2: 图片元数据保留（easymode）
echo ""
echo "🖼️  测试2: 图片元数据保留（universal_converter）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 测试3: 动图元数据保留
echo ""
echo "🎞️  测试3: 动图元数据保留"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 清理
rm -rf "$TEST_DIR"
echo ""
echo "✅ 测试完成"
```

---

#### 3.2 预期测试结果

**成功标准**:
- ✅ 视频: 所有EXIF/XMP字段100%保留
- ✅ 图片: EXIF/IPTC/XMP/ICC 100%保留
- ✅ 动图: 帧数+元数据100%保留

**失败示例**:
```
❌ 元数据丢失:
  - Title: "Test Video" → (空)
  - CreateDate: "2025:10:25 08:00:00" → (空)
  - GPS: 存在 → (空)
```

---

## 📋 修复优先级

| 优先级 | 文件 | 类型 | 影响 | 状态 |
|--------|------|------|------|------|
| 🔴 最高 | `pkg/engine/balance_optimizer.go` | 视频 | Pixly主程序 | ⏳ 待修复 |
| 🔴 最高 | `pkg/engine/simple_converter.go` | 视频 | Pixly主程序 | ⏳ 待修复 |
| 🔴 最高 | `pkg/engine/conversion_engine.go` | 视频 | Pixly主程序 | ⏳ 待修复 |
| 🟠 高 | `easymode/archive/dynamic2avif/` | 动图 | easymode | ⏳ 待修复 |
| 🟠 高 | `easymode/archive/video2mov/` | 视频 | easymode | ⏳ 待修复 |
| 🟡 中 | `easymode/archive/static2jxl/` | 图片 | easymode | ⏳ 待修复 |
| 🟡 中 | `easymode/archive/static2avif/` | 图片 | easymode | ⏳ 待修复 |
| 🟡 中 | `easymode/archive/dynamic2jxl/` | 动图 | easymode | ⏳ 待修复 |
| ✅ 已修复 | `easymode/universal_converter/` | 全部 | easymode | ✅ 已实现 |
| ✅ 已修复 | `easymode/archive/all2jxl/` | JXL | easymode | ✅ 已实现 |
| ✅ 已修复 | `easymode/archive/all2avif/` | AVIF | easymode | ✅ 已实现 |

---

## 🎯 修复后的效果

### 修复前 ❌
```bash
# 视频转换
ffmpeg -i video.mp4 -c copy -f mov output.mov
→ 元数据全部丢失 ❌

# 图片转换
cjxl input.png output.jxl
→ 元数据全部丢失 ❌
```

### 修复后 ✅
```bash
# 视频转换
ffmpeg -i video.mp4 -c copy -map_metadata 0 -movflags use_metadata_tags -f mov output.mov
→ 元数据100%保留 ✅

# 图片转换
cjxl input.png output.jxl
exiftool -overwrite_original -TagsFromFile input.png output.jxl
→ 元数据100%保留 ✅
```

---

## 📊 元数据保留清单

### EXIF (图片/视频)
- ✅ Make (厂商)
- ✅ Model (型号)
- ✅ DateTime (拍摄时间)
- ✅ Orientation (方向)
- ✅ ExposureTime (曝光)
- ✅ FNumber (光圈)
- ✅ ISO
- ✅ FocalLength (焦距)
- ✅ LensModel (镜头)

### GPS
- ✅ GPSLatitude (纬度)
- ✅ GPSLongitude (经度)
- ✅ GPSAltitude (海拔)
- ✅ GPSTimeStamp (GPS时间)

### XMP
- ✅ Creator (创作者)
- ✅ Rights (版权)
- ✅ Description (描述)
- ✅ Subject (主题)
- ✅ Rating (评分)
- ✅ Label (标签)

### ICC Profile
- ✅ ColorSpace (色彩空间)
- ✅ ProfileDescription (配置文件描述)

### 视频特有
- ✅ Duration (时长)
- ✅ FrameRate (帧率)
- ✅ VideoCodec (视频编码)
- ✅ AudioCodec (音频编码)
- ✅ Bitrate (比特率)

---

## 🚀 立即行动

**下一步**:
1. ✅ 创建此修复计划文档
2. 🔴 修复 Pixly 主程序（3个文件）
3. 🟠 修复 easymode archive 工具（5个文件）
4. 🟡 创建测试脚本验证
5. ✅ 更新文档和README

**预计时间**: 2-3小时  
**测试时间**: 1小时  
**总计**: 3-4小时

**完成后**: 整个plxy-easy2jxlavif项目的所有转换都将**100%保留元数据**！

