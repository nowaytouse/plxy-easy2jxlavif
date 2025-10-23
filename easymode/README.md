# EasyMode Media Conversion Toolkit v2.2.0

A powerful Go-based media conversion toolkit supporting batch conversion of multiple image and video formats, featuring complete metadata preservation, intelligent performance optimization, and an 8-layer validation system.

## 🚀 Key Features

- **🎨 Multi-format Support**: Supports mainstream image formats including JPG, PNG, GIF, WebP, AVIF, HEIC, TIFF, BMP
- **🔒 Lossless Conversion**: Provides lossless conversion for JPEG XL and AVIF modern image formats
- **📋 Metadata Preservation**: Complete preservation of EXIF, IPTC, XMP metadata information
- **⚡ Intelligent Performance Optimization**: Dynamically adjusts processing threads based on system load
- **🛡️ 8-Layer Validation System**: Ensures conversion quality and data integrity, prevents cheating bypass
- **🏞️ Live Photo Detection**: Automatically identifies and skips Apple Live Photo files
- **📝 Smart Log Management**: Log rotation and detailed processing records
- **🔧 Modular Design**: Unified parameter parsing and validation modules
- **🎯 Universal Converter**: One tool supports all conversion types and modes

## 📦 Tool List

### 🎨 Image Conversion Tools
- `all2avif` - Batch convert to AVIF format
- `all2jxl` - Batch convert to JPEG XL format
- `static2avif` - Static images to AVIF
- `static2jxl` - Static images to JPEG XL
- `dynamic2avif` - Dynamic images to AVIF
- `dynamic2jxl` - Dynamic images to JPEG XL

### 🎬 Video Processing Tools
- `video2mov` - Video re-encapsulation to MOV format

### 🔧 Media Management Tools
- `media_tools` - Metadata management, file deduplication, extension normalization
- `universal_converter` - Universal conversion tool supporting all formats and modes

## 📚 Documentation Resources

### User Guides
- [User Guide v2.2.0](docs/USER_GUIDE_v2.2.0.md) - Complete usage tutorial
- [Animation Processing Guide](docs/ANIMATION_PROCESSING_GUIDE.md) - Detailed animation conversion guide
- [Technical Architecture](docs/TECHNICAL_ARCHITECTURE.md) - System architecture and design

### Developer Resources
- [API Reference](docs/API_REFERENCE.md) - Complete API interface documentation
- [Validation Strategy](docs/VALIDATION_STRATEGY.md) - 8-layer validation system details
- [Test Report](docs/TEST_REPORT_v2.1.0.md) - Functional testing and performance benchmarks

### Historical Versions
- [Comprehensive Test Report](docs/COMPREHENSIVE_TEST_REPORT.md)
- [Final Comprehensive Report v2.2.0](docs/FINAL_COMPREHENSIVE_REPORT_v2.2.0.md)
- [Optimization Report v2.2.1](docs/OPTIMIZATION_v2.2.1.md)

## 🛠️ Installation and Usage

### System Requirements
- Go 1.25+
- ImageMagick (for AVIF conversion)
- libjxl (for JPEG XL conversion)
- FFmpeg (for video conversion)
- ExifTool (for metadata processing)
- libavif (for static AVIF conversion with avifenc)

### Quick Start

1. **Clone Repository**
```bash
git clone <repository-url>
cd easymode
```

2. **Install Dependencies**
```bash
# macOS
brew install imagemagick libjxl ffmpeg exiftool

# Ubuntu/Debian
sudo apt-get install imagemagick libjxl-tools ffmpeg exiftool
```

3. **Build All Tools**
```bash
./build_all.sh
```

4. **Use Universal Converter (Recommended)**
```bash
# Convert all images to JPEG XL
./universal_converter/bin/universal_converter -dir ./images -type jxl -mode all

# Convert static images to AVIF
./universal_converter/bin/universal_converter -dir ./photos -type avif -mode static

# Convert videos to MOV
./universal_converter/bin/universal_converter -dir ./videos -type mov -mode video

# Convert dynamic images to JPEG XL
./universal_converter/bin/universal_converter -dir ./gifs -type jxl -mode dynamic
```

5. **Use Individual Tools**
```bash
# Convert all images to JPEG XL
./all2jxl/bin/all2jxl -dir ./images -workers 4

# Convert all images to AVIF
./all2avif/bin/all2avif -dir ./images -workers 4
```

## 📋 Detailed Parameters

### General Parameters
- `-dir`: Input directory path
- `-output`: Output directory (default: same as input)
- `-workers`: Number of worker threads (0=auto-detect)
- `-dry-run`: Dry run mode
- `-skip-exist`: Skip existing output files
- `-retries`: Number of retries on conversion failure
- `-timeout`: Timeout for single file processing (seconds)

### Conversion Parameters
- `-type`: Conversion type (avif, jxl, mov)
- `-mode`: Processing mode (all, static, dynamic, video)

### Quality Parameters
- `-quality`: Output quality (1-100)
- `-speed`: Encoding speed (0-9)
- `-cjxl-threads`: CJXL encoder thread count

### Validation Parameters
- `-strict`: Strict validation mode
- `-tolerance`: Allowed pixel difference percentage

### Metadata Parameters
- `-copy-metadata`: Copy metadata
- `-preserve-times`: Preserve file timestamps

### Logging Parameters
- `-log-level`: Log level (DEBUG, INFO, WARN, ERROR)
- `-log-file`: Log file path
- `-log-max-size`: Maximum log file size (bytes)

### Performance Parameters
- `-max-memory`: Maximum memory usage (bytes)
- `-process-limit`: Maximum concurrent processes
- `-file-limit`: Maximum concurrent files

## 🔍 8-Layer Validation System

To ensure conversion quality, all tools integrate an 8-layer validation system:

1. **Basic File Validation**: Check file existence and readability
2. **File Size Validation**: Verify converted file size reasonableness
3. **Format Integrity Validation**: Ensure correct output format
4. **Metadata Integrity Validation**: Check critical metadata fields
5. **Image Dimension Validation**: Verify image dimension consistency
6. **Pixel-Level Validation**: Perform pixel-level quality checks
7. **Quality Metrics Validation**: Calculate PSNR, SSIM quality metrics
8. **Anti-Cheat Validation**: Detect hardcoded bypasses and fake conversions

## 🚀 Performance Optimization

- **Smart Thread Adjustment**: Dynamically adjust worker threads based on system memory usage
- **File Type Priority**: Prioritize fast conversion formats like JPEG
- **Memory Management**: Intelligent memory usage monitoring and limiting
- **Concurrency Control**: Limit external processes and file handle usage
- **Enhanced File Type Detection**: Solve AVIF/HEIC format recognition issues

## 📊 Usage Examples

### Universal Converter Examples
```bash
# Convert entire photo library to JPEG XL
./universal_converter/bin/universal_converter \
  -dir /Users/username/Pictures \
  -type jxl \
  -mode all \
  -workers 8 \
  -quality 95 \
  -strict

# Convert static images to AVIF
./universal_converter/bin/universal_converter \
  -dir /Users/username/Photos \
  -type avif \
  -mode static \
  -workers 4 \
  -quality 90

# Convert videos to MOV
./universal_converter/bin/universal_converter \
  -dir /Users/username/Videos \
  -type mov \
  -mode video \
  -workers 2
```

### Individual Tool Examples
```bash
# Convert all images to JPEG XL
./all2jxl/bin/all2jxl -dir ./images -workers 4 -strict

# Convert static images to AVIF
./static2avif/bin/static2avif -dir ./photos -quality 90

# Convert dynamic images to JPEG XL
./dynamic2jxl/bin/dynamic2jxl -dir ./gifs -workers 2
```

### Metadata Processing Examples
```bash
# Merge XMP metadata
./merge_xmp/bin/merge_xmp -dir /Users/username/Photos

# Detect duplicate files
./deduplicate_media/bin/deduplicate_media -dir /Users/username/Photos -trash ./trash
```

## 🔧 Troubleshooting

### Common Issues

1. **Missing Dependencies**
```bash
# macOS
brew install imagemagick libjxl ffmpeg exiftool

# Ubuntu/Debian
sudo apt-get install imagemagick libjxl-tools ffmpeg exiftool
```

2. **Permission Issues**
```bash
chmod +x build_all.sh
chmod +x */build.sh
```

3. **Insufficient Memory**
```bash
# Reduce worker threads
./universal_converter/bin/universal_converter -dir ./images -workers 2
```

4. **File Type Recognition Issues**
```bash
# Use strict mode for detailed validation
./universal_converter/bin/universal_converter -dir ./images -type jxl -strict
```

## 📈 Performance Benchmarks

Test results on MacBook Pro M1:
- JPEG to JXL: ~50MB/s
- PNG to AVIF: ~30MB/s
- HEIC to JXL: ~20MB/s
- Metadata processing: ~1000 files/minute

## 🆕 v2.2.0 New Features

- ✅ **Universal Converter**: One tool supports all conversion types and modes
- ✅ **Enhanced File Type Detection**: Solve AVIF/HEIC format recognition issues
- ✅ **8-Layer Validation System**: Ensure conversion quality and data integrity
- ✅ **Modular Design**: Unified parameter parsing and validation modules
- ✅ **Smart Performance Optimization**: Dynamic adjustment based on system load
- ✅ **Live Photo Detection**: Automatically skip Apple Live Photo files
- ✅ **Anti-Cheat Mechanism**: Prevent hardcoded bypasses and fake conversions

## 📝 Version History

### v2.2.0 (Latest)
- **Universal Converter**: One tool supports all conversion types and modes
- **Enhanced File Type Detection**: Solve AVIF/HEIC format recognition issues
- **Modular Design**: Unified parameter parsing and validation modules
- **8-Layer Validation System**: Ensure conversion quality and data integrity
- **Smart Performance Optimization**: Dynamic adjustment based on system load
- **Live Photo Detection**: Automatically skip Apple Live Photo files
- **Anti-Cheat Mechanism**: Prevent hardcoded bypasses and fake conversions

### v2.1.1
- **8-Layer Validation System** - Multi-layer protection against various bypass attacks
- **HEIC/HEIF Support** - Comprehensive support for modern image formats
- **Smart Performance Optimization** - Dynamic thread adjustment and file priority processing
- **Live Photo Detection** - Automatically skip Live Photo files
- **Log Management Optimization** - Automatic log rotation to prevent oversized files
- **Anti-Cheat Mechanism** - Prevent hardcoded and demo code bypassing validation
- **Enhanced Chinese Comments** - Detailed technical annotations

### v2.1.0
- Enhanced security validation mechanisms
- Improved error handling and logging
- Optimized performance and memory usage
- Added XMP format validation
- Enhanced documentation and examples

## 🌐 Language Support

- **English**: [README.md](README.md) (Current)
- **简体中文**: [README_ZH.md](README_ZH.md)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📞 Support

If you encounter any issues or have questions, please open an issue on GitHub.

## 🎯 Use Cases

- **Photographers** - Batch process RAW images, convert formats
- **Designers** - Optimize image file sizes while maintaining quality
- **Content Creators** - Video format conversion, metadata management
- **System Administrators** - File deduplication, storage optimization

## 🔗 Related Links

- [JPEG XL Official Website](https://jpeg.org/jpegxl/)
- [AVIF Format Specification](https://aomediacodec.github.io/av1-avif/)
- [ExifTool Documentation](https://exiftool.org/)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)