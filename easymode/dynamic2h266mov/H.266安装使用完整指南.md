# Dynamic2H266MOV - H.266安装使用完整指南

**更新日期**: 2025-10-25  
**状态**: ✅ 代码完成，等待FFmpeg安装

---

## 📋 当前状态

### ✅ 已完成

- **工具代码**: 100%完成，编译通过
- **所有依赖**: vvenc/vvdec/pkg-config/cmake/nasm/yasm 已安装
- **编解码器库**: x264/x265/aom/svt-av1/libvpx 已安装
- **安装流程**: 从源码编译FFmpeg（7步骤）

### ⏳ 待完成

- **FFmpeg编译**: 需要用户运行安装流程（10-20分钟）

---

## 🚨 核心问题说明

### 问题

即使您已经安装了所有依赖：
```bash
brew install vvenc vvdec pkg-config cmake nasm yasm
```

**Homebrew的FFmpeg预编译版本仍然不支持libvvenc！**

### 原因

Homebrew在编译FFmpeg时没有启用`--enable-libvvenc`选项。预编译的二进制文件无法使用您安装的vvenc库。

### 解决方案

**唯一可靠的方法：从源码编译FFmpeg**

---

## 🔧 安装流程（2种方法）

### 方法1: 使用工具内置安装（推荐）⭐

```bash
cd easymode/archive/dynamic2h266mov
./bin/dynamic2h266mov-darwin-arm64
```

工具启动后会：
1. 自动检测H.266支持
2. 发现不支持时显示3个选择
3. 选择 **[1] 自动从源码编译FFmpeg**
4. 自动执行完整编译流程
5. 编译完成后重新验证
6. 验证通过后开始使用

**优点**：
- ✅ 全自动，无需手动操作
- ✅ 详细进度显示
- ✅ 失败有容错
- ✅ 完成后直接使用

### 方法2: 使用独立脚本

```bash
cd easymode/archive/dynamic2h266mov
./install_ffmpeg_with_vvenc.sh
```

完成后：
```bash
./bin/dynamic2h266mov-darwin-arm64
```

---

## 📦 编译流程详解（7步骤）

### 步骤1: 检查编译依赖 (~30秒)

检查并安装：
- vvenc (H.266编码器库)
- vvdec (H.266解码器库)
- pkg-config (包配置工具)
- cmake (编译工具)
- nasm, yasm (汇编器)

**您已完成此步骤** ✅

### 步骤2: 安装编解码器库 (~2分钟)

安装：
- x264 (H.264)
- x265 (H.265)
- aom (AV1)
- svt-av1 (快速AV1)
- libvpx (VP8/VP9)

**大部分已安装** ✅

### 步骤3: 下载FFmpeg源码 (~1分钟)

```bash
# 下载FFmpeg 7.1源码（约25MB）
curl -L https://github.com/FFmpeg/FFmpeg/archive/refs/tags/n7.1.tar.gz
tar -xzf ffmpeg.tar.gz
```

保存位置：`~/.pixly_build/FFmpeg-n7.1`

### 步骤4: 配置FFmpeg (~1分钟)

```bash
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig:..."

./configure \
  --prefix=/usr/local \
  --enable-gpl \
  --enable-version3 \
  --enable-nonfree \
  --enable-libvvenc      ← 关键！H.266编码
  --enable-libvvdec      ← 关键！H.266解码
  --enable-libx264       ← H.264
  --enable-libx265       ← H.265
  --enable-libaom        ← AV1
  --enable-libsvtav1     ← 快速AV1
  --enable-libvpx        ← VP8/VP9
  --enable-videotoolbox  ← macOS硬件加速
```

### 步骤5: 编译FFmpeg (~10-15分钟) ⏱️

```bash
make -j10  # 使用10个CPU核心并行编译
```

**这是最耗时的步骤**

### 步骤6: 卸载旧版FFmpeg (~10秒)

```bash
brew uninstall --ignore-dependencies ffmpeg
```

### 步骤7: 安装新版FFmpeg (~30秒)

```bash
sudo make install  # 需要输入密码
```

安装到：`/usr/local/bin/ffmpeg`

---

## ✅ 编译完成后验证

### 检查H.266支持

```bash
ffmpeg -encoders | grep libvvenc
```

**预期输出**：
```
V..... libvvenc             libvvenc H.266 / VVC
```

### 检查FFmpeg版本

```bash
ffmpeg -version
```

**预期输出**（部分）：
```
configuration: ... --enable-libvvenc --enable-libvvdec ...
```

### 测试转换

```bash
# 创建测试GIF
ffmpeg -f lavfi -i "color=c=blue:s=320x240:d=2" test.gif

# 转换为H.266 MOV
ffmpeg -i test.gif -c:v libvvenc -qp 28 -preset medium test.mov

# 检查输出
ffprobe test.mov
```

---

## 🎬 使用Dynamic2H266MOV工具

### 交互模式（推荐）

```bash
cd easymode/archive/dynamic2h266mov
./bin/dynamic2h266mov-darwin-arm64
```

按提示操作：
1. 拖入GIF/WebP/APNG文件夹
2. 选择是否原地转换
3. 等待转换完成
4. 选择是否继续下一个

### 命令行模式

```bash
./bin/dynamic2h266mov-darwin-arm64 -dir /path/to/gifs
```

参数：
- `-dir` - 输入目录或文件
- `-output` - 输出目录
- `--in-place` - 原地转换（删除原文件）
- `--dry-run` - 试运行
- `--workers` - 并发数

---

## 🔍 故障排查

### Q: 编译失败怎么办？

**检查1**: 确认所有依赖已安装
```bash
brew list vvenc vvdec pkg-config cmake nasm yasm
```

**检查2**: 验证pkg-config能找到vvenc
```bash
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig"
pkg-config --exists libvvenc && echo "✅ 找到" || echo "❌ 未找到"
```

**检查3**: 查看configure输出
```bash
cd ~/.pixly_build/FFmpeg-n7.1
cat config.log | grep vvenc
```

### Q: 编译后仍不支持H.266？

**检查1**: 确认使用正确的FFmpeg
```bash
which ffmpeg
# 应该是: /usr/local/bin/ffmpeg
```

**检查2**: 重启终端
```bash
# 关闭并重新打开终端
ffmpeg -encoders | grep libvvenc
```

**检查3**: 检查PATH优先级
```bash
echo $PATH | tr ':' '\n' | grep -n "local\|homebrew"
# /usr/local/bin 应该在 /opt/homebrew/bin 之前
```

### Q: 编译太慢怎么办？

**正常现象！** 从源码编译FFmpeg需要10-20分钟。

可以：
- 使用其他归档工具（dynamic2mov等）
- 或耐心等待（仅需一次）

---

## 🆚 替代方案

如果不想等待编译，可以使用这些同样出色的工具：

### dynamic2mov - H.265/AV1编码 ⭐⭐⭐⭐⭐

```bash
cd easymode/archive/dynamic2mov
./bin/dynamic2mov-darwin-arm64
```

**优势**：
- ✅ 无需编译，立即可用
- ✅ AV1压缩率接近H.266
- ✅ 广泛兼容

**使用场景**：
```bash
# AV1编码（最高压缩率）
./dynamic2mov-darwin-arm64 -dir gifs/ --codec av1 --format mp4

# H.265编码（广泛兼容）
./dynamic2mov-darwin-arm64 -dir gifs/ --codec h265 --format mov
```

### dynamic2avif - AVIF格式 ⭐⭐⭐⭐⭐

```bash
cd easymode/archive/dynamic2avif
./bin/dynamic2avif-darwin-arm64 -dir gifs/
```

**优势**：
- ✅ 基于AV1，压缩率极高
- ✅ 现代浏览器支持
- ✅ 立即可用

### dynamic2jxl - JPEG XL格式 ⭐⭐⭐⭐

```bash
cd easymode/archive/dynamic2jxl
./bin/dynamic2jxl-darwin-arm64 -dir gifs/
```

**优势**：
- ✅ 新一代图像格式
- ✅ 压缩率优秀
- ✅ 立即可用

---

## 📊 格式对比

| 格式 | 压缩率 | 速度 | 兼容性 | 需要编译 |
|------|--------|------|--------|----------|
| H.266/VVC | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ✅ 是 |
| AV1 | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ❌ 否 |
| H.265 | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ 否 |
| AVIF | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ❌ 否 |
| JPEG XL | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ❌ 否 |

**建议**：
- 追求极致压缩且愿意等待 → H.266（需编译）
- 追求高压缩且立即可用 → AV1或AVIF
- 追求广泛兼容 → H.265

---

## 🎯 总结

### Dynamic2H266MOV工具现状

✅ **代码**: 100%完成，编译通过，逻辑正确  
✅ **依赖**: 所有必要组件已安装  
✅ **安装流程**: 可靠的从源码编译方案  
⏳ **FFmpeg**: 需要用户运行安装（10-20分钟）

### 使用建议

**立即使用**：
- 使用dynamic2mov（AV1/H.265）
- 使用dynamic2avif（AVIF）
- 使用dynamic2jxl（JPEG XL）

**愿意等待**：
- 运行dynamic2h266mov选择自动安装
- 10-20分钟后获得H.266支持

---

**版本**: v1.0.0  
**作者**: Pixly Team  
**最后更新**: 2025-10-25
