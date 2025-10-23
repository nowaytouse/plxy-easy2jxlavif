# dynamic2jxl - Dynamic Image to JXL Converter

## 📖 Introduction

`dynamic2jxl` is a tool specifically designed for converting dynamic images to the JXL format. It supports animated formats such as GIF, WebP, APNG, and HEIC/HEIF, providing lossless compression and batch processing capabilities.

## 🚀 Features

- ✅ **Dynamic Image Support** - Supports animated formats such as GIF, WebP, APNG, and HEIC/HEIF.
- ✅ **Lossless Compression** - Achieves lossless compression using the JXL format.
- ✅ **Intelligent Detection** - Automatically identifies dynamic image types.
- ✅ **Batch Processing** - Efficient concurrent processing capabilities.
- ✅ **Safety Protection** - Fixed an issue where original files were mistakenly deleted when skipping existing files.
- ✅ **Metadata Preservation** - Retains EXIF information using exiftool.
- ✅ **Progress Display** - Real-time processing progress and statistics.
- ✅ **Accurate File Count Verification** - After conversion, a detailed file count verification report is provided to ensure the accuracy and reliability of the processing.
- ✅ **Optimized HEIC/HEIF Handling** - Adopts a more stable intermediate format conversion strategy to improve the success rate of HEIC/HEIF file conversion.
- ✅ **Fixed JPEG Parameter Error** - Corrected a bug where the `--lossless_jpeg=1` parameter was incorrectly applied to non-JPEG files.

## 🔧 Usage

### Basic Usage
```bash
go run main.go -input /path/to/images -output /path/to/output -workers 4
```

### Argument Description
- `-input`: Input directory path (required).
- `-output`: Output directory path (required).
- `-workers`: Number of concurrent worker threads (default: number of CPU cores).
- `-skip-exist`: Skip existing files (default: true).
- `-dry-run`: Dry run mode, only prints the files to be processed.
- `-retries`: Number of retries on failure (default: 2).
- `-timeout`: Timeout in seconds for a single file (default: 300).
- `-cjxl-threads`: Number of threads for each conversion task (default: 1).
- `-replace`: Delete original files after conversion. **⚠️ Safety Note**: Only deletes original files after verifying that the target file exists and is valid.

### Advanced Usage
```bash
# High-concurrency processing
go run main.go -input /path/to/images -output /path/to/output -workers 8

# Dry run mode
go run main.go -input /path/to/images -output /path/to/output -dry-run

# Skip existing files
go run main.go -input /path/to/images -output /path/to/output -skip-exist

# Custom number of retries
go run main.go -input /path/to/images -output /path/to/output -retries 3 -timeout 600
```

## 📊 Performance Optimization

### Concurrency Control
- Intelligent worker thread configuration.
- Resource limits to prevent system overload.
- File handle management.

### Memory Management
- Reduced memory footprint.
- Optimized file processing flow.
- Prevention of memory leaks.

## 🛡️ Safety Features

### File Safety
- Fixed an issue where original files were mistakenly deleted when skipping existing files.
- Atomic file operations.
- Backup mechanism.

### Error Handling
- Comprehensive error recovery mechanism.
- Detailed logging.
- Automatic retry function.

## 🔍 Troubleshooting

### Common Problems
1. **Missing dependencies**: Make sure `cjxl` and `exiftool` are installed.
2. **Permission issues**: Check file read/write permissions.
3. **Insufficient space**: Make sure there is enough disk space.

### Getting Help
- Check the log file for detailed errors.
- Use the dry run mode to test the configuration.
- Check file permissions and disk space.

### Supported File Formats

- **GIF**: .gif (including animation)
- **WebP**: .webp (including animation)
- **APNG**: .png (PNG with animation)
- **HEIC/HEIF**: .heic, .heif (including animation)

## 📝 Update Log

### v2.0.1 (2025-01-27)
- ✅ Added dynamic image to JXL conversion tool.
- ✅ Fixed an issue where original files were mistakenly deleted when skipping existing files.
- ✅ Improved error handling and logging.
- ✅ Optimized performance and memory usage.
- ✅ Enhanced security protection mechanisms.

---

**Version**: v2.0.1  
**Maintainer**: AI Assistant  
**License**: MIT
