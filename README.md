# Pixly - 智能化媒体优化解决方案 beta

简单易用版本：
[https://github.com/nowaytouse/easy2jxlavif/tree/main/easymode](https://github.com/nowaytouse/plxy-easy2jxlavif/tree/main/easymode)

## 🚀 项目简介

Pixly 是一款强大的命令行工具，旨在帮助用户智能化地优化图片和视频文件，在尽可能保持视觉质量的前提下，大幅减小文件体积。无论是静态图片、动态 GIF 还是视频文件，Pixly 都能提供高效、安全的处理方案。

## ✨ 核心特性

*   **智能自动模式 (Auto Mode+)**：
    *   根据文件原始品质（高、中、低）智能选择最佳优化策略。
    *   对高品质文件优先采用无损压缩，确保画质。
    *   对中等品质文件启用"尝试引擎"，通过多轮尝试（无损重新包装、数学无损、有损探测）寻找最佳平衡点。
    *   对低品质文件采用更激进的压缩策略，以实现最大体积减小。
*   **品质优先模式 (Quality Mode)**：
    *   专注于最高画质，对静态图片进行 JXL 无损压缩，对动态图片进行 AVIF 无损压缩，对视频进行无损重封装。
*   **表情包模式 (Emoji Mode)**：
    *   追求极致小体积，适用于表情包或对画质要求不高的场景，采用激进的有损压缩。
*   **GIF 动图到 AVIF 转换**：
    *   支持将传统的 GIF 动图转换为更现代、更高效的 AVIF 格式，显著减小文件大小并提升加载速度。
*   **原始文件安全删除**：
    *   在文件成功优化并生成新格式后，自动删除原始文件，避免占用额外存储空间。
*   **多工具链支持**：
    *   集成 `ffmpeg`、`cjxl`、`avifenc` (可选) 等业界领先的媒体处理工具。
*   **进程监控与防卡死机制**：
    *   内置智能进程监控器，根据文件属性动态估算处理时限。
    *   实时监控子进程活动，若发现长时间无响应（30 秒无有效计算），则判断为卡死。
    *   提供优雅终止、强制终止和用户决策（等待、终止、忽略）的三级终止策略，确保程序稳定运行。
*   **安全检查**：
    *   在处理前对目标目录进行安全检查，防止对系统关键区域进行误操作。
*   **详细统计报告**：
    *   处理完成后生成详细的统计报告，展示成功处理的文件数量和总共节省的存储空间（精确到 MB）。

## 🛠️ 安装与依赖

Pixly 依赖于一些外部命令行工具来执行媒体转换。请确保您的系统已安装以下工具：

*   **Go 语言环境**：用于编译和运行 Pixly。
*   **FFmpeg**：用于视频处理和 AVIF 转换。
*   **cjxl**：用于 JPEG XL (JXL) 格式的编码和解码。
*   **exiftool**：用于读取和写入媒体文件的元数据。
*   **avifenc** (可选)：如果安装，Pixly 将优先使用它进行静态图片的 AVIF 编码，通常效果更优。

### macOS (使用 Homebrew)

```bash
# 安装 Go 语言环境
brew install go

# 安装 FFmpeg
brew install ffmpeg

# 安装 cjxl (JPEG XL 工具)
brew install libjxl

# 安装 exiftool
brew install exiftool

# 安装 avifenc (可选，用于更好的 AVIF 编码)
brew install libavif
```

### Ubuntu/Debian

```bash
# 更新包列表
sudo apt update

# 安装 Go 语言环境
sudo apt install golang-go

# 安装 FFmpeg
sudo apt install ffmpeg

# 安装 exiftool
sudo apt install exiftool

# 安装 libjxl (cjxl 工具)
sudo apt install libjxl-tools

# 安装 libavif (avifenc 工具)
sudo apt install libavif-bin
```

### CentOS/RHEL

```bash
# 安装 Go 语言环境
sudo yum install golang

# 安装 FFmpeg
sudo yum install ffmpeg

# 安装 exiftool
sudo yum install perl-Image-ExifTool

# 安装 libjxl (需要从源码编译或使用第三方仓库)
# 安装 libavif (需要从源码编译或使用第三方仓库)
```

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/your-username/pixly.git
cd pixly
```

### 2. 编译项目

```bash
go build -o pixly main.go
```

### 3. 运行 Pixly

```bash
# 基本用法 - 智能自动模式
./pixly -mode auto -dir /path/to/your/images

# 品质优先模式
./pixly -mode quality -dir /path/to/your/images

# 表情包模式
./pixly -mode emoji -dir /path/to/your/images

# 查看帮助信息
./pixly -h
```

## 📋 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-mode` | 处理模式：auto, quality, emoji | auto |
| `-dir` | 目标目录路径 | 必需 |
| `-workers` | 并发工作线程数 | 10 |
| `-timeout` | 单个文件处理超时时间（秒） | 300 |
| `-retries` | 失败重试次数 | 3 |
| `-dry-run` | 试运行模式（不实际处理文件） | false |
| `-verbose` | 详细输出模式 | false |

## 🎯 使用示例

### 处理图片文件夹

```bash
# 智能优化图片文件夹
./pixly -mode auto -dir ~/Pictures

# 高品质优化
./pixly -mode quality -dir ~/Pictures

# 表情包优化
./pixly -mode emoji -dir ~/Pictures/emojis
```

### 处理视频文件夹

```bash
# 优化视频文件
./pixly -mode auto -dir ~/Videos
```

### 批量处理多个目录

```bash
# 处理多个目录
for dir in ~/Pictures/*/; do
    ./pixly -mode auto -dir "$dir"
done
```

## 📊 输出示例

```
🎨 Pixly - 智能化媒体优化解决方案 v1.0.0
✨ 作者: AI Assistant
🔧 开始初始化...
✅ 系统依赖检查通过
📁 准备处理目录: /path/to/images
🔍 扫描文件...
📊 发现 150 个候选文件
⚡ 配置处理性能...
🚀 开始并行处理 - 工作线程: 10, 文件数: 150
🛑 设置信号处理...

🔄 开始处理: image1.jpg (2.5 MB)
✅ 识别为高品质文件
🎯 采用无损压缩策略
✅ 处理成功: image1.jpg
📊 大小变化: 2.50 MB -> 1.20 MB (节省: 1.30 MB, 压缩率: 48.0%)

...

⏱️  总处理时间: 2m30.5s
🎯 ===== 处理摘要 =====
✅ 成功处理文件: 145
❌ 处理失败文件: 3
⏭️  跳过文件: 2
📊 ===== 大小统计 =====
📥 原始总大小: 500.00 MB
📤 优化后大小: 250.00 MB
💾 节省空间: 250.00 MB (压缩率: 50.0%)
🎉 ===== 处理完成 =====
```

## 辅助工具

### `deduplicate_media` - 媒体文件去重工具

`deduplicate_media` 是一个独立的辅助脚本，位于 `easymode/deduplicate_media/` 目录下。它的主要功能是扫描指定目录，查找内容重复的媒体文件，并将重复项移动到指定的文件夹中。

**功能:**

*   支持多种媒体格式，包括常见的图片和视频。
*   自动将 `.jpeg`, `.tiff` 等扩展名规范为 `.jpg`, `.tif`。
*   通过文件哈希和字节对比，精确查找重复文件。
*   将重复文件移动到指定的“垃圾”文件夹，而不是直接删除。
*   在“垃圾”文件夹中创建说明文件，解释其用途。

**使用方法:**

```bash
# 编译
cd easymode/deduplicate_media
./build.sh

# 运行
./easymode/deduplicate_media/deduplicate_media -dir /path/to/your/media -trash-dir /path/to/trash
```

### `merge_xmp` - 合并 XMP 元数据

`merge_xmp` 是一个独立的辅助脚本，位于 `easymode/merge_xmp/` 目录下。它的主要功能是将在与媒体文件同名的 `.xmp` 文件中的元数据合并到媒体文件中。

**功能:**

*   自动查找与媒体文件（如 `.jpg`, `.png`, `.heic`, `.heif` 等）同名的 `.xmp` 文件。
*   使用 `exiftool` 将 `.xmp` 文件中的所有元数据合并到媒体文件中。
*   在合并成功并验证后，自动删除 `.xmp` 文件。
*   如果验证失败，将保留 `.xmp` 文件以便进行手动检查。

### `all2jxl` - HEIC/HEIF 转换支持

**easymode 工具现在完美支持 HEIC/HEIF 格式转换!**

*   **智能多重转换策略**: 自动在 ImageMagick、FFmpeg 和宽松模式间切换以处理 HEIC/HEIF 文件
*   **统一验证流程**: 支持 HEIC/HEIF 文件的验证和像素级准确性检查
*   **动画检测**: 支持 HEIF 动画检测和转换
*   **Live Photo 保护**: 自动检测并跳过 Apple Live Photos（.mov 配对文件），避免损坏 Live Photo 组合
*   **元数据保留**: 完整保留 HEIC/HEIF 文件的元数据
*   **错误处理**: 多层错误处理机制，确保转换的稳定性

**使用方法:**

1.  **编译脚本:**
    ```bash
    cd easymode/merge_xmp
    ./build.sh
    ```

2.  **运行脚本:**
    ```bash
    ./easymode/merge_xmp/merge_xmp -dir /path/to/your/media
    ```

**示例:**

假设您有以下文件结构:

```
/path/to/your/media/
├── image1.jpg
├── image1.xmp
└── image2.png
```

运行脚本后:

```bash
./easymode/merge_xmp/merge_xmp -dir /path/to/your/media
```

脚本会将 `image1.xmp` 的元数据合并到 `image1.jpg` 中，然后删除 `image1.xmp`。`image2.png` 因为没有对应的 `.xmp` 文件，所以不会被处理。

## 辅助工具

### `deduplicate_media` - 媒体文件去重工具

`deduplicate_media` 是一个独立的辅助脚本，位于 `easymode/deduplicate_media/` 目录下。它的主要功能是扫描指定目录，查找内容重复的媒体文件，并将重复项移动到指定的文件夹中。

**功能:**

*   支持多种媒体格式，包括常见的图片和视频。
*   自动将 `.jpeg`, `.tiff` 等扩展名规范为 `.jpg`, `.tif`。
*   通过文件哈希和字节对比，精确查找重复文件。
*   将重复文件移动到指定的“垃圾”文件夹，而不是直接删除。
*   在“垃圾”文件夹中创建说明文件，解释其用途。

**使用方法:**

```bash
# 编译
cd easymode/deduplicate_media
./build.sh

# 运行
./easymode/deduplicate_media/deduplicate_media -dir /path/to/your/media -trash-dir /path/to/trash
```

### `merge_xmp` - 合并 XMP 元数据

`merge_xmp` 是一个独立的辅助脚本，位于 `easymode/merge_xmp/` 目录下。它的主要功能是将在与媒体文件同名的 `.xmp` 文件中的元数据合并到媒体文件中。

**功能:**

*   自动查找与媒体文件（如 `.jpg`, `.png`, `.heic`, `.heif` 等）同名的 `.xmp` 文件。
*   使用 `exiftool` 将 `.xmp` 文件中的所有元数据合并到媒体文件中。
*   在合并成功并验证后，自动删除 `.xmp` 文件。
*   如果验证失败，将保留 `.xmp` 文件以便进行手动检查。

### `all2jxl` - HEIC/HEIF 转换支持

**easymode 工具现在完美支持 HEIC/HEIF 格式转换!**

*   **智能多重转换策略**: 自动在 ImageMagick、FFmpeg 和宽松模式间切换以处理 HEIC/HEIF 文件
*   **统一验证流程**: 支持 HEIC/HEIF 文件的验证和像素级准确性检查
*   **动画检测**: 支持 HEIF 动画检测和转换
*   **Live Photo 保护**: 自动检测并跳过 Apple Live Photos（.mov 配对文件），避免损坏 Live Photo 组合
*   **元数据保留**: 完整保留 HEIC/HEIF 文件的元数据
*   **错误处理**: 多层错误处理机制，确保转换的稳定性

**使用方法:**

1.  **编译脚本:**
    ```bash
    cd easymode/merge_xmp
    ./build.sh
    ```

2.  **运行脚本:**
    ```bash
    ./easymode/merge_xmp/merge_xmp -dir /path/to/your/media
    ```

**示例:**

假设您有以下文件结构:

```
/path/to/your/media/
├── image1.jpg
├── image1.xmp
└── image2.png
```

运行脚本后:

```bash
./pixly -mode auto -dir /path/to/your/media
```

脚本会将 `image1.xmp` 的元数据合并到 `image1.jpg` 中，然后删除 `image1.xmp`。`image2.png` 因为没有对应的 `.xmp` 文件，所以不会被处理。

### `all2jxl` - HEIC/HEIF 转换支持

**easymode 工具现在完美支持 HEIC/HEIF 格式转换!**

*   **智能多重转换策略**: 自动在 ImageMagick、FFmpeg 和宽松模式间切换以处理 HEIC/HEIF 文件
*   **统一验证流程**: 支持 HEIC/HEIF 文件的验证和像素级准确性检查
*   **动画检测**: 支持 HEIF 动画检测和转换
*   **Live Photo 保护**: 自动检测并跳过 Apple Live Photos（.mov 配对文件），避免损坏 Live Photo 组合
*   **元数据保留**: 完整保留 HEIC/HEIF 文件的元数据
*   **错误处理**: 多层错误处理机制，确保转换的稳定性

**使用方法:**

1.  **编译脚本:**
    ```bash
    cd easymode/merge_xmp
    ./build.sh
    ```

2.  **运行脚本:**
    ```bash
    ./easymode/merge_xmp/merge_xmp -dir /path/to/your/media
    ```

**示例:**

假设您有以下文件结构:

```
/path/to/your/media/
├── image1.jpg
├── image1.xmp
└── image2.png
```

运行脚本后:

```bash
./pixly -mode auto -dir /path/to/your/media
```

脚本会将 `image1.xmp` 的元数据合并到 `image1.jpg` 中，然后删除 `image1.xmp`。`image2.png` 因为没有对应的 `.xmp` 文件，所以不会被处理。

### `video2mov` - 视频重新包装工具

`video2mov` 是一个独立的辅助脚本，位于 `easymode/video2mov/` 目录下。它的主要功能是将各种视频格式**无损地重新包装**为 `.mov` 容器格式。此工具不进行视频编码，而是通过流复制（stream copy）的方式，确保原始视频和音频流的质量完全保留，同时提供更好的兼容性和元数据处理能力。

**功能:**

*   **无损重新包装**: 使用 `ffmpeg -c copy` 进行流复制，不进行任何视频或音频的重新编码，确保原始质量。
*   **广泛视频格式支持**: 支持常见的视频格式，如 `.mp4`, `.avi`, `.mkv` 等。
*   **元数据保留**: 使用 `exiftool` 将原始视频文件的元数据完整复制到新的 `.mov` 文件中。
*   **精确的文件数量验证**: 重新包装完成后，提供详细的文件数量验证报告，确保处理过程的准确性和可靠性。

**使用方法:**

```bash
# 编译
cd easymode/video2mov
./build.sh

# 运行
./easymode/video2mov/video2mov -input /path/to/your/videos -output /path/to/mov/output
```

## 深入技术细节：转换流程与中间格式

为了确保最佳的兼容性和转换成功率，尤其是在处理复杂或新兴的图像格式时，我们的工具采用了智能化的多阶段转换策略。以下是其核心技术细节：

### 1. 统一的输入处理

所有转换脚本（`all2jxl`, `all2avif`, `dynamic2jxl`, `dynamic2avif`, `static2jxl`, `static2avif`）现在都使用统一的文件扫描逻辑，并支持广泛的媒体文件类型。

### 2. 智能化的中间格式转换

对于某些源格式（特别是 HEIC/HEIF），直接将其转换为目标格式（JXL 或 AVIF）可能会遇到兼容性问题或导致质量损失。为了解决这个问题，我们引入了**稳定的中间格式转换**：

-   **HEIC/HEIF 文件处理**:
    -   当源文件是 HEIC/HEIF 时，工具会首先使用 `ImageMagick` 将其**无损地转换为 PNG 格式**。PNG 是一种广泛支持的无损图像格式，作为中间格式能有效避免在后续处理中出现兼容性问题或像素数据丢失。
    -   **重要提示**: 这一中间转换步骤是无损的，确保了原始图像数据的完整性。
    -   转换完成后，原始的 HEIC/HEIF 文件路径会被更新为这个临时的 PNG 文件路径，供后续的 JXL/AVIF 编码器使用。临时 PNG 文件会在处理完成后自动清理。

### 3. 目标格式编码

-   **JXL 编码器 (`cjxl`)**:
    -   对于 JPEG 源文件（或经过无损转换为 PNG 的 HEIC/HEIF 文件），`cjxl` 会被调用以生成 JXL 文件。
    -   **关键参数**:
        -   `--lossless_jpeg=1`: **仅当源文件是 JPEG 时**，此参数会被启用。它指示 `cjxl` 对 JPEG 数据进行无损重包装，而不是先解码再重新编码像素，从而确保了 JPEG 到 JXL 的真正无损转换。
        -   `-d 0 -e 9`: 这些参数用于控制 JXL 的压缩级别和编码效率，通常设置为无损模式和最高效率。
-   **AVIF 编码器 (`ffmpeg` with `libsvtav1`)**:
    -   对于所有支持的静态和动态图像格式（或经过中间转换的 HEIC/HEIF 文件），`ffmpeg` 会被调用以生成 AVIF 文件。
    -   **关键参数**:
        -   `-c:v libsvtav1`: 指定使用 SVT-AV1 编码器进行 AVIF 编码。
        -   `-crf`: 控制 AVIF 的质量，根据用户设定的 `-quality` 参数动态调整。
        -   `-preset`: 控制编码速度与文件大小的平衡，根据用户设定的 `-speed` 参数动态调整。

### 4. 元数据保留

-   所有转换脚本都集成了 `exiftool`，以确保原始文件的所有元数据（包括 EXIF、XMP 等）都能**完整、无损地复制**到最终生成的 JXL 或 AVIF 文件中。这对于摄影师和内容创作者来说至关重要。

### 5. 鲁棒的验证机制

-   **JXL 脚本**:
    -   对于 JPEG 源文件，验证过程会将生成的 JXL 文件解码回 JPEG，并进行像素级比对，确保真正的无损转换。
    -   对于 PNG 等其他无损源文件，验证过程会将生成的 JXL 文件解码回 PNG，并进行像素级比对。
    -   对于 HEIC/HEIF 等有损源文件，验证过程会简化为检查生成的 JXL 文件是否能被成功解码，以避免有损格式带来的像素差异导致验证失败。
-   **AVIF 脚本**:
    -   由于 AVIF 编码通常是有损的，验证机制主要侧重于确保生成的 AVIF 文件是有效且可解码的，并检查文件数量的准确性。

通过这些精细化的处理和验证流程，我们的工具旨在提供一个既高效又可靠的媒体文件优化解决方案。

## 🔧 高级配置

### 自定义工作线程数

```bash
# 使用 20 个工作线程
./pixly -mode auto -dir /path/to/images -workers 20
```

### 调整超时时间

```bash
# 设置 10 分钟超时
./pixly -mode auto -dir /path/to/images -timeout 600
```

### 试运行模式

```bash
# 试运行，不实际处理文件
./pixly -mode auto -dir /path/to/images -dry-run
```

## 🐛 故障排除

### 常见问题

**Q: 提示"缺少依赖工具"**
A: 请确保已安装所有必需的依赖工具，运行 `ffmpeg -version`、`cjxl -h`、`exiftool -ver` 验证安装。

**Q: 处理速度慢**
A: 可以增加工作线程数，使用 `-workers 20` 参数。

**Q: 某些文件处理失败**
A: 检查文件是否损坏，或尝试使用 `-retries 5` 增加重试次数。

**Q: 内存使用过高**
A: 减少工作线程数，使用 `-workers 5` 参数。

### 调试模式

```bash
# 启用详细输出
./pixly -mode auto -dir /path/to/images -verbose
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 支持

如有问题，请提交 Issue 或联系维护者。

## 📝 更新日志

### [2.1.1] - 2025-10-21
- **代码优化**: 消除重复函数，合并 `easymode` 脚本中的重复函数定义，提升代码质量和维护性
    - `static2avif` 脚本：将重复的 `validateFileCount`、`findTempFiles` 和 `cleanupTempFiles` 函数合并为单一定义
    - `dynamic2avif` 脚本：将重复的 `getFileTimesDarwin` 和 `setFinderDates` 函数合并为单一定义

### [2.0.8] - 2025-10-20
- **新增功能**: `video2mov` 视频重新包装工具
- **文档更新**: 主README更新，包含 `video2mov` 脚本介绍

### [2.0.7] - 2025-10-20
- **统一文件扫描逻辑**: 所有 `easymode` 转换脚本的文件扫描逻辑已统一
- **精确文件数量验证**: 所有 `easymode` 转换脚本均已集成精确的文件数量验证机制
- **优化HEIC/HEIF处理**: 改进HEIC/HEIF转换策略，采用更稳定的ImageMagick转PNG中间文件方案
- **修复JPEG参数错误**: 修正了 `--lossless_jpeg=1` 参数被错误应用于非JPEG文件的问题

### [2.0.6] - 2025-10-20
- **HEIC/HEIF格式支持**: 为所有转换工具完美添加 HEIC/HEIF 格式支持
- **多重转换策略**: 实现智能多重转换策略，自动在 ImageMagick、FFmpeg 和宽松模式间切换
- **Live Photo 保护**: 自动检测并跳过 Apple Live Photos，保护 Live Photo 组合完整性
- **元数据保留**: 完整保留 HEIC/HEIF 文件的元数据信息

### [2.0.1] - 2025-10-19
- **错误修复**: 修复了在处理同名但扩展名不同的文件时，可能导致原始文件被错误删除的严重问题。详情请参阅 [CHANGELOG.md](CHANGELOG.md)。

### v1.0.0-beta
- 初始版本发布
- 支持智能自动模式
- 支持品质优先模式
- 支持表情包模式
- 集成多工具链支持
- 实现进程监控与防卡死机制
- 添加安全检查功能
- 提供详细统计报告
