# EasyMode Toolkit

EasyMode is a comprehensive toolkit specifically designed for image format conversion, providing a simple command-line interface with efficient batch processing capabilities.

## 🚀 Tool Overview

### Core Tools
- **all2avif** - Universal format to AVIF converter
- **all2jxl** - Universal format to JXL converter
- **static2avif** - Static image to AVIF converter
- **dynamic2avif** - Animated image to AVIF converter
- **static2jxl** - Static image to JXL converter (new)
- **dynamic2jxl** - Animated image to JXL converter (new)
- **deduplicate_media** - Remove duplicate images/videos
- **merge_xmp** - Merge XMP metadata
- **video2mov** - Video format converter

### Key Features
- ✅ **Smart format detection** - Automatically identifies static/animated images
- ✅ **Batch processing** - Efficient concurrent processing capabilities
- ✅ **Safety protection** - Fixed issue where original files were deleted when skipping existing files
- ✅ **Validation system** - Complete processing result validation and report generation
- ✅ **Metadata preservation** - Uses exiftool to preserve EXIF information
- ✅ **Progress display** - Real-time processing progress and statistics

## 📁 Detailed Tool Descriptions

### all2avif - Universal AVIF Converter
**Purpose**: Convert various image formats to AVIF format  
**Features**: Supports static and animated images, intelligent parameter selection  
**Usage**: `./all2avif -dir /path/to/images -quality 80 -workers 4`

### all2jxl - Universal JXL Converter
**Purpose**: Convert various image formats to JPEG XL format  
**Features**: Lossless compression, supports animated images  
**Usage**: `./all2jxl -dir /path/to/images -workers 4`

### static2avif - Static Image to AVIF
**Purpose**: Specialized static image to AVIF conversion  
**Features**: Optimized for static images, faster processing speed  
**Usage**: `./static2avif -input /path/to/images -output /path/to/output -quality 80`

### dynamic2avif - Animated Image to AVIF
**Purpose**: Specialized animated image to AVIF conversion  
**Features**: Supports GIF, WebP, APNG and other animated formats  
**Usage**: `./dynamic2avif -input /path/to/images -output /path/to/output -quality 80`

### static2jxl - Static Image to JXL (New)
**Purpose**: Specialized static image to JXL conversion  
**Features**: Lossless compression, preserves highest quality  
**Usage**: `./static2jxl -input /path/to/images -output /path/to/output -workers 4`

### dynamic2jxl - Animated Image to JXL (New)
**Purpose**: Specialized animated image to JXL conversion  
**Features**: Supports JXL conversion of animated images  
**Usage**: `./dynamic2jxl -input /path/to/images -output /path/to/output -workers 4`

### deduplicate_media - Media Deduplication
**Purpose**: Remove duplicate images and videos  
**Features**: Compares file content to identify duplicates  
**Usage**: `./deduplicate_media -dir /path/to/media -workers 4`

### merge_xmp - XMP Metadata Merger
**Purpose**: Merge XMP metadata from multiple files  
**Features**: Preserves and combines metadata  
**Usage**: `./merge_xmp -input /path/to/images -output /path/to/output`

### video2mov - Video Format Converter
**Purpose**: Convert various video formats  
**Features**: Supports multiple video format conversions  
**Usage**: `./video2mov -input /path/to/videos -output /path/to/output`

## 🔧 Build Instructions

### Prerequisites
- Go 1.21+
- ffmpeg (for AVIF conversion)
- cjxl (for JXL conversion)
- exiftool (for metadata preservation)

### Build Steps
```bash
# Build all tools
cd easymode
for dir in all2avif all2jxl static2avif dynamic2avif static2jxl dynamic2jxl deduplicate_media merge_xmp video2mov; do
    cd $dir
    chmod +x build.sh
    ./build.sh
    cd ..
done
```

## 📊 Performance Optimization

### Concurrency Control
- Intelligent worker thread configuration
- Resource limiting to prevent system overload
- File handle management

### Memory Management
- Reduced memory footprint
- Optimized file processing flow
- Memory leak prevention

## 🛡️ Safety Features

### File Safety
- Fixed issue where original files were deleted when skipping existing files
- Atomic file operations
- Backup mechanism

### Error Handling
- Comprehensive error recovery mechanism
- Detailed logging
- Automatic retry functionality

## 📈 Validation System

### Automatic Validation
- File count validation
- Size compression validation
- EXIF data validation
- Format conversion validation

### Report Generation
- JSON format detailed reports
- User-friendly text reports
- Failure analysis

## 🎯 Usage Recommendations

### Tool Selection
- **Universal processing**: Use all2avif or all2jxl
- **Static images**: Use static2avif or static2jxl
- **Animated images**: Use dynamic2avif or dynamic2jxl
- **Media cleanup**: Use deduplicate_media
- **Metadata management**: Use merge_xmp

### Performance Tuning
- Adjust worker threads based on CPU cores
- Increase timeout for large files
- Test configurations with dry-run mode

## 🔍 Troubleshooting

### Common Issues
1. **Missing dependencies**: Ensure ffmpeg, cjxl, and exiftool are installed
2. **Permission issues**: Check file read/write permissions
3. **Insufficient space**: Ensure adequate disk space

### Getting Help
- Check log files for detailed errors
- Review failure analysis in validation reports
- Test configurations with dry-run mode

## 📝 Changelog

### v2.0.2 (2025-01-27)
- ✅ Fixed issue where original files were deleted when skipping existing files
- ✅ Added modular validation system
- ✅ Added separate static/animated processing tools
- ✅ Improved error handling and logging
- ✅ Optimized performance and memory usage
- ✅ Added support for new tools: static2jxl, dynamic2jxl, deduplicate_media, merge_xmp, video2mov

---

**Version**: v2.0.2  
**Maintainer**: AI Assistant  
**License**: MIT