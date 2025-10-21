# easymode Program Usage Tutorial

This tutorial explains how to use the easymode programs, including tools for image format conversion, media deduplication, metadata management, and video conversion. These are high-quality, high-efficiency command-line tools for converting media files to modern formats.

## Table of Contents
1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [all2jxl - Convert Images to JPEG XL](#all2jxl---convert-images-to-jpeg-xl)
4. [all2avif - Unified AVIF Conversion Tool](#all2avif---unified-avif-conversion-tool)
5. [static2avif - Static Images to AVIF](#static2avif---static-images-to-avif)
6. [dynamic2avif - Animated Images to AVIF](#dynamic2avif---animated-images-to-avif)
7. [static2jxl - Static Images to JXL](#static2jxl---static-images-to-jxl)
8. [dynamic2jxl - Animated Images to JXL](#dynamic2jxl---animated-images-to-jxl)
9. [deduplicate_media - Deduplicate Media Files](#deduplicate_media---deduplicate-media-files)
10. [merge_xmp - Merge XMP Metadata](#merge_xmp---merge-xmp-metadata)
11. [video2mov - Video Format Conversion](#video2mov---video-format-conversion)
12. [Best Practices](#best-practices)

## Overview

The easymode programs provide a simple, high-quality set of media processing tools:

- **all2jxl**: Converts various image formats to JPEG XL (with lossless conversion where possible).
- **all2avif**: A unified tool for converting static and dynamic images to the AVIF format.
- **static2avif**: Specialized tool for converting static images to AVIF format.
- **dynamic2avif**: Specialized tool for converting animated images to AVIF format.
- **static2jxl**: Specialized tool for converting static images to JXL format.
- **dynamic2jxl**: Specialized tool for converting animated images to JXL format.
- **deduplicate_media**: Detects and deletes duplicate media files.
- **merge_xmp**: Merges and manages XMP metadata.
- **video2mov**: Converts various video formats.

All programs support concurrent processing and include robust error handling.

## Prerequisites

Before using these tools, please ensure you have the required dependencies installed:

### System Requirements
- Go 1.19 or higher
- macOS, Linux, or Windows

### Dependency Tools

#### all2jxl Dependencies
- `cjxl` - JPEG XL encoder
- `djxl` - JPEG XL decoder
- `exiftool` - Metadata processing tool

#### all2avif Dependencies
- `ffmpeg` - Video and image processing tool
- `exiftool` - Metadata processing tool

### Installing Dependencies

#### macOS (using Homebrew)
```bash
# Dependencies for all2jxl
brew install jpeg-xl exiftool

# Dependencies for all2avif
brew install ffmpeg exiftool
```

#### Ubuntu/Debian
```bash
# Dependencies for all2jxl
sudo apt install libjxl-tools exiftool

# Dependencies for all2avif
sudo apt install ffmpeg exiftool
```

#### CentOS/RHEL
```bash
# Dependencies for all2jxl
sudo yum install libjxl-tools perl-Image-ExifTool

# Dependencies for all2avif
sudo yum install ffmpeg perl-Image-ExifTool
```

## all2jxl - Convert Images to JPEG XL

### Overview
`all2jxl` is a high-performance JPEG XL converter that supports lossless conversion for a variety of image formats.

### Features
- Supported formats: JPEG, PNG, GIF, WebP, BMP, TIFF, HEIC, HEIF, AVIF
- Intelligent animation detection (supports HEIF animation)
- Live Photo protection: Automatically detects and skips Apple Live Photos (.mov sidecar files)
- Multiple conversion strategies: Automatically switches between ImageMagick, FFmpeg, and a relaxed mode to handle HEIC/HEIF files
- Unified verification process: Supports verification of HEIC/HEIF files and pixel-perfect accuracy checks
- Lossless and mathematically lossless conversion
- Complete metadata preservation
- High-performance parallel processing

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/all2jxl

# Build the tool
./build.sh

# Basic conversion
./all2jxl -dir /path/to/images

# View help
./all2jxl -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-dir` | Required | Input directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 80 | Image quality (1-100) |
| `-skip-exist` | true | Skip existing JXL files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./all2jxl -dir ~/Pictures

# High-quality conversion
./all2jxl -dir ~/Pictures -quality 95

# Use more worker threads
./all2jxl -dir ~/Pictures -workers 20

# Dry run mode
./all2jxl -dir ~/Pictures -dry-run

# Keep original files after conversion
./all2jxl -dir ~/Pictures -replace=false
```

### Example Output

```
🎨 JPEG XL Batch Conversion Tool v2.0.0
✨ Author: AI Assistant
🔧 Initializing...
✅ cjxl is ready
✅ djxl is ready
✅ exiftool is ready
📁 Preparing processing directory...
📂 Processing directory directly: /path/to/images
🔍 Scanning image files...
📊 Found 150 candidate files
⚡ Configuring processing performance...
🚀 Starting parallel processing - Workers: 10, Files: 150

🔄 Starting to process: image1.jpg (2.5 MB)
✅ Identified as image format: image1.jpg (jpg)
🖼️  Static image: image1.jpg
✅ Conversion complete: image1.jpg (JPEG Lossless Re-encode)
✅ Verification successful: image1.jpg lossless conversion correct
🎉 Processing successful: image1.jpg
📊 Size change: 2.50 MB -> 2.00 MB (Saved: 0.50 MB, Compression ratio: 80.0%)

...

⏱️  Total processing time: 2m30.5s
🎯 ===== Processing Summary =====
✅ Successfully processed images: 150
❌ Failed to convert images: 0
📊 ===== Size Statistics =====
📥 Original total size: 500.00 MB
📤 Converted size: 350.00 MB
💾 Space saved: 150.00 MB (Compression ratio: 70.0%)
🎉 ===== Processing Complete =====
```

## all2avif - Unified AVIF Conversion Tool

### Overview
`all2avif` is a unified AVIF conversion tool that supports the conversion of both static and dynamic images.

### Features
- Supports static images: JPEG, PNG, BMP, TIFF, WebP, HEIC, HEIF, AVIF
- Supports animated images: GIF, animated WebP, APNG, animated HEIF
- Intelligent animation detection (supports HEIF animation detection)
- Live Photo protection: Automatically detects and skips Apple Live Photos (.mov sidecar files)
- Multiple conversion strategies: Automatically switches between ImageMagick, FFmpeg, and a relaxed mode to handle HEIC/HEIF files
- Configurable quality and speed settings
- Complete metadata preservation

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/all2avif

# Build the tool
./build.sh

# Basic conversion
./all2avif -dir /path/to/images

# View help
./all2avif -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-dir` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 80 | Image quality (1-100) |
| `-speed` | 4 | Encoding speed (0-6) |
| `-skip-exist` | true | Skip existing AVIF files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./all2avif -dir ~/Pictures

# High-quality conversion
./all2avif -dir ~/Pictures -quality 90

# Fast conversion
./all2avif -dir ~/Pictures -speed 6

# Specify output directory
./all2avif -dir ~/Pictures -output ~/Pictures/avif

# Dry run mode
./all2avif -dir ~/Pictures -dry-run
```

### Quality and Speed Settings

#### Quality Settings (1-100)
- **90-100**: Highest quality, larger file size
- **80-89**: High quality, balanced quality and size
- **70-79**: Medium quality, smaller file size
- **60-69**: Low quality, small file size
- **1-59**: Lowest quality, smallest file size

#### Speed Settings (0-6)
- **0-1**: Slowest, best quality
- **2-3**: Slower, better quality
- **4**: Default setting, balanced speed and quality
- **5-6**: Fastest, average quality

### Example Output

```
🎨 AVIF Batch Conversion Tool v2.0.0
✨ Author: AI Assistant
🔧 Initializing...
✅ ffmpeg is ready
✅ exiftool is ready
📁 Preparing processing directory...
📂 Processing directory directly: /path/to/images
🔍 Scanning image files...
📊 Found 150 candidate files
⚡ Configuring processing performance...
🚀 Starting parallel processing - Workers: 10, Files: 150

🔄 Starting to process: image1.jpg (2.5 MB)
🖼️  Static image: image1.jpg
✅ Conversion complete: image1.jpg (Static Image Conversion)
📋 Metadata copied successfully: image1.jpg
🎉 Processing successful: image1.jpg
📊 Size change: 2.50 MB -> 1.20 MB (Saved: 1.30 MB, Compression ratio: 48.0%)

🔄 Starting to process: animation.gif (1.2 MB)
🎬 Detected animated image: animation.gif
✅ Conversion complete: animation.gif (Animated Image Conversion)
📋 Metadata copied successfully: animation.gif
🎉 Processing successful: animation.gif
📊 Size change: 1.20 MB -> 0.80 MB (Saved: 0.40 MB, Compression ratio: 66.7%)

...

⏱️  Total processing time: 3m15.2s
🎯 ===== Processing Summary =====
✅ Successfully processed images: 150
❌ Failed to convert images: 0
📊 ===== Size Statistics =====
📥 Original total size: 500.00 MB
📤 Converted size: 300.00 MB
💾 Space saved: 200.00 MB (Compression ratio: 60.0%)
🎉 ===== Processing Complete =====
```

## static2avif - Static Images to AVIF

### Overview
`static2avif` is a specialized AVIF conversion tool for static images, providing an optimized processing flow.

### Features
- Supports static images: JPEG, PNG, BMP, TIFF, WebP, HEIC, HEIF, AVIF
- Optimized for static images, faster processing speed
- Configurable quality and speed settings
- Complete metadata preservation

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/static2avif

# Build the tool
./build.sh

# Basic conversion
./static2avif -input /path/to/images

# View help
./static2avif -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 80 | Image quality (1-100) |
| `-speed` | 4 | Encoding speed (0-6) |
| `-skip-exist` | true | Skip existing AVIF files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./static2avif -input ~/Pictures

# High-quality conversion
./static2avif -input ~/Pictures -quality 90

# Fast conversion
./static2avif -input ~/Pictures -speed 6

# Specify output directory
./static2avif -input ~/Pictures -output ~/Pictures/avif

# Dry run mode
./static2avif -input ~/Pictures -dry-run
```

## dynamic2avif - Animated Images to AVIF

### Overview
`dynamic2avif` is a specialized AVIF conversion tool for animated images, supporting a variety of animated formats.

### Features
- Supports animated images: GIF, animated WebP, APNG, animated HEIF
- Intelligent animation detection (supports HEIF animation detection)
- Live Photo protection: Automatically detects and skips Apple Live Photos (.mov sidecar files)
- Multiple conversion strategies: Automatically switches between ImageMagick, FFmpeg, and a relaxed mode to handle HEIC/HEIF files
- Configurable quality settings
- Complete metadata preservation

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/dynamic2avif

# Build the tool
./build.sh

# Basic conversion
./dynamic2avif -input /path/to/images

# View help
./dynamic2avif -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 80 | Image quality (1-100) |
| `-skip-exist` | true | Skip existing AVIF files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./dynamic2avif -input ~/Animations

# High-quality conversion
./dynamic2avif -input ~/Animations -quality 90

# Specify output directory
./dynamic2avif -input ~/Animations -output ~/Animations/avif

# Dry run mode
./dynamic2avif -input ~/Animations -dry-run
```

## static2jxl - Static Images to JXL

### Overview
`static2jxl` is a specialized JPEG XL conversion tool for static images, providing lossless and lossy conversion options.

### Features
- Supports static images: JPEG, PNG, GIF, WebP, BMP, TIFF, HEIC, HEIF, AVIF
- Optimized processing flow for static images
- Lossless and mathematically lossless conversion
- Complete metadata preservation
- High-performance parallel processing

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/static2jxl

# Build the tool
./build.sh

# Basic conversion
./static2jxl -input /path/to/images

# View help
./static2jxl -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 95 | Image quality (1-100) |
| `-skip-exist` | true | Skip existing JXL files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./static2jxl -input ~/Pictures

# High-quality conversion
./static2jxl -input ~/Pictures -quality 98

# Specify output directory
./static2jxl -input ~/Pictures -output ~/Pictures/jxl

# Dry run mode
./static2jxl -input ~/Pictures -dry-run
```

## dynamic2jxl - Animated Images to JXL

### Overview
`dynamic2jxl` is a specialized JPEG XL conversion tool for animated images, supporting a variety of animated formats.

### Features
- Supports animated images: GIF, animated WebP, APNG, animated HEIF
- Intelligent animation detection (supports HEIF animation detection)
- Live Photo protection: Automatically detects and skips Apple Live Photos (.mov sidecar files)
- Lossless and mathematically lossless conversion
- Complete metadata preservation
- High-performance parallel processing

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/dynamic2jxl

# Build the tool
./build.sh

# Basic conversion
./dynamic2jxl -input /path/to/images

# View help
./dynamic2jxl -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-quality` | 95 | Image quality (1-100) |
| `-skip-exist` | true | Skip existing JXL files |
| `-replace` | true | Delete original files after conversion |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |
| `-retries` | 1 | Number of retries |

### Usage Examples

```bash
# Basic conversion
./dynamic2jxl -input ~/Animations

# High-quality conversion
./dynamic2jxl -input ~/Animations -quality 98

# Specify output directory
./dynamic2jxl -input ~/Animations -output ~/Animations/jxl

# Dry run mode
./dynamic2jxl -input ~/Animations -dry-run
```

## deduplicate_media - Deduplicate Media Files

### Overview
`deduplicate_media` is a tool for detecting and deleting duplicate media files.

### Features
- Compares file content to identify duplicates
- Efficient hashing algorithm
- Safe deletion mechanism
- Supports both image and video files

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/deduplicate_media

# Build the tool
./build.sh

# Basic deduplication
./deduplicate_media -dir /path/to/media

# View help
./deduplicate_media -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-dir` | Required | Input directory path |
| `-workers` | 10 | Number of worker threads |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |

### Usage Examples

```bash
# Basic deduplication
./deduplicate_media -dir ~/Photos

# Use more worker threads
./deduplicate_media -dir ~/Photos -workers 20

# Dry run mode
./deduplicate_media -dir ~/Photos -dry-run
```

## merge_xmp - Merge XMP Metadata

### Overview
`merge_xmp` is a tool for merging and managing XMP metadata.

### Features
- Preserves and merges metadata information
- Supports a variety of image formats
- Safe metadata operations

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/merge_xmp

# Build the tool
./build.sh

# Basic merge
./merge_xmp -input /path/to/images

# View help
./merge_xmp -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-dry-run` | false | Dry run mode |

### Usage Examples

```bash
# Basic merge
./merge_xmp -input ~/Photos

# Specify output directory
./merge_xmp -input ~/Photos -output ~/Photos/xmp-merged

# Dry run mode
./merge_xmp -input ~/Photos -dry-run
```

## video2mov - Video Format Conversion

### Overview
`video2mov` is a tool for converting various video formats.

### Features
- Supports a variety of video format conversions
- Preserves video quality
- Efficient processing

### Basic Usage

```bash
# Navigate to the tool directory
cd easymode/video2mov

# Build the tool
./build.sh

# Basic conversion
./video2mov -input /path/to/videos

# View help
./video2mov -h
```

### Command-Line Arguments

| Argument | Default | Description |
|---|---|---|
| `-input` | Required | Input directory path |
| `-output` | Input directory | Output directory path |
| `-workers` | 10 | Number of worker threads |
| `-dry-run` | false | Dry run mode |
| `-timeout` | 300 | Timeout in seconds for a single file |

### Usage Examples

```bash
# Basic conversion
./video2mov -input ~/Videos

# Specify output directory
./video2mov -input ~/Videos -output ~/Videos/converted

# Dry run mode
./video2mov -input ~/Videos -dry-run
```

## Best Practices

### 1. Choose the Right Tool

- **Use all2jxl**: When you need lossless compression and the highest quality.
- **Use all2avif**: When you need a modern format and good compression.

### 2. Performance Optimization

#### Worker Thread Settings
```bash
# For multi-core CPUs, use more worker threads
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20

# For memory-constrained systems, reduce worker threads
./all2jxl -dir /path/to/images -workers 4
./all2avif -dir /path/to/images -workers 4
```

#### Quality vs. Speed Balance
```bash
# High-quality settings (suitable for final output)
./all2avif -dir /path/to/images -quality 95 -speed 1

# Fast settings (suitable for previews or testing)
./all2avif -dir /path/to/images -quality 70 -speed 6
```

### 3. Batch Processing

#### Processing Multiple Directories
```bash
# Use a loop to process multiple directories
for dir in ~/Pictures/*/; do
    ./all2jxl -dir "$dir"
done

for dir in ~/Pictures/*/; do
    ./all2avif -dir "$dir"
done
```

#### Automation with a Script
```bash
#!/bin/bash
# Batch conversion script

# Set directories
INPUT_DIR="/path/to/images"
OUTPUT_DIR="/path/to/output"

# Create output directories
mkdir -p "$OUTPUT_DIR"

# Convert to JXL
echo "Starting JXL conversion..."
./all2jxl -dir "$INPUT_DIR" -output "$OUTPUT_DIR/jxl"

# Convert to AVIF
echo "Starting AVIF conversion..."
./all2avif -dir "$INPUT_DIR" -output "$OUTPUT_DIR/avif"

echo "Conversion complete!"
```

### 4. Error Handling

#### Dry Run Mode
```bash
# Dry run before actual conversion
./all2jxl -dir /path/to/images -dry-run
./all2avif -dir /path/to/images -dry-run
```

#### Retry Mechanism
```bash
# For unstable files, increase the number of retries
./all2jxl -dir /path/to/images -retries 5
./all2avif -dir /path/to/images -retries 5
```

#### Timeout Setting
```bash
# For large files, increase the timeout
./all2jxl -dir /path/to/images -timeout 600
./all2avif -dir /path/to/images -timeout 600
```

### 5. Storage Management

#### Disk Space Check
```bash
# Check available space before conversion
df -h /path/to/images

# Check directory size with du
du -sh /path/to/images
```

#### Backing Up Important Files
```bash
# Back up important files before conversion
cp -r /path/to/images /path/to/backup

# Or use rsync for incremental backups
rsync -av /path/to/images/ /path/to/backup/
```

### 6. Monitoring and Logging

#### Checking Processing Progress
```bash
# Monitor logs in another terminal
tail -f all2jxl.log
tail -f all2avif.log
```

#### Checking System Resources
```bash
# Monitor CPU and memory usage
top -p $(pgrep all2jxl)
top -p $(pgrep all2avif)
```

### 7. Troubleshooting

#### Common Problem Solving

**Problem**: Conversion fails with "missing dependency tool"
```bash
# Check dependency installation
which cjxl djxl exiftool
which ffmpeg exiftool

# Reinstall dependencies
brew install jpeg-xl exiftool ffmpeg
```

**Problem**: Insufficient memory
```bash
# Reduce the number of worker threads
./all2jxl -dir /path/to/images -workers 4
./all2avif -dir /path/to/images -workers 4
```

**Problem**: Slow processing speed
```bash
# Increase the number of worker threads (if CPU and memory allow)
./all2jxl -dir /path/to/images -workers 20
./all2avif -dir /path/to/images -workers 20
```

**Problem**: Some files fail to process
```bash
# Check if the file is corrupted
file /path/to/problematic/file

# Try to process the problematic file individually
./all2jxl -dir /path/to/single/file
./all2avif -dir /path/to/single/file
```

### 8. Performance Benchmarking

#### Testing the effect of different settings
```bash
# Test the performance of different numbers of worker threads
for workers in 1 4 8 16 20; do
    echo "Testing $workers worker threads..."
    time ./all2jxl -dir /path/to/test/images -workers $workers
done
```

#### Comparing different quality settings
```bash
# Test different quality settings
for quality in 60 70 80 90 95; do
    echo "Testing quality $quality..."
    time ./all2avif -dir /path/to/test/images -quality $quality
done
```

## Summary

The easymode programs provide a complete media processing solution:

1. **all2jxl**: Suitable for scenarios requiring lossless compression.
2. **all2avif**: Suitable for scenarios requiring modern formats and good compression ratios.
3. **static2avif/static2jxl**: Suitable for scenarios requiring specialized processing of static images.
4. **dynamic2avif/dynamic2jxl**: Suitable for scenarios requiring specialized processing of animated images.
5. **deduplicate_media**: Suitable for scenarios requiring cleanup of duplicate media files.
6. **merge_xmp**: Suitable for scenarios requiring management of XMP metadata.
7. **video2mov**: Suitable for scenarios requiring video format conversion.

By using these tools appropriately and following best practices, you can efficiently process a variety of media files while maintaining high quality and good performance.

Remember:
- Always do a dry run first.
- Adjust the number of worker threads based on system resources.
- Back up important files regularly.
- Monitor processing progress and system resources.
- Choose the appropriate quality and speed settings based on your needs.

## Changelog

### v2.0.2 (2025-01-27)
- **New Tools**: Added `static2jxl`, `dynamic2jxl`, `deduplicate_media`, `merge_xmp`, `video2mov` tools.
- **Functional Enhancements**: Improved the security and performance of all tools.
- **HEIC/HEIF Support**: Enhanced HEIC/HEIF support in the `dynamic2avif` tool to have the same robust processing as `dynamic2jxl`.
- **Documentation Updates**: Improved the documentation for all tools.

### v2.0.1
- **Important Fix**: Added file count verification to prevent residual temporary files.
- **Automatic Cleanup**: Automatically detects and cleans up uncleared temporary files.
- **Quality Assurance**: Ensures that the number of files before and after processing meets expectations.

### v2.0.0
- Merged `dynamic2avif` and `static2avif` into a unified `all2avif` tool.
- Improved error handling and statistics.
- Optimized performance and memory usage.
- Updated all documentation to Simplified Chinese.
