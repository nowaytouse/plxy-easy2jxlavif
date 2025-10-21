# dynamic2avif - Dynamic Image to AVIF Converter

`dynamic2avif` is a command-line tool designed for image collectors and efficiency seekers, aiming to convert dynamic images (GIF, WebP, APNG, HEIF, etc.) to the next-generation image format AVIF (.avif) in a high-quality, safe, and reliable manner.

## Core Features

- **Fully Automatic Intelligent Processing:** Without any complex configuration, the tool runs in a unique "fully automatic mode", intelligently identifying each file and adopting the optimal strategy for processing.
- **Visually Lossless Conversion:** Guarantees high-quality conversion, ensuring that your images maintain excellent visual quality during the conversion process.
- **High-Performance Concurrent Processing:** Fully utilizes the multi-core performance of modern CPUs to process multiple files concurrently, significantly reducing waiting time.
- **Safe and Reliable:** Adopts transactional operations and automatically rolls back on failure, ensuring that original files are not affected.
- **Intelligent Error Recovery:** Supports a retry mechanism, so network fluctuations or temporary failures will not cause the entire task to fail.
- **Accurate File Count Verification:** After the conversion is complete, a detailed file count verification report is provided to ensure the accuracy and reliability of the processing.
- **Code Optimization:** Eliminates duplicate functions and merges duplicate `getFileTimesDarwin` and `setFinderDates` function definitions to improve code quality and maintainability.

## Technical Advantages

### Intelligent Strategy Selection

The tool automatically selects the optimal conversion strategy based on the file type:

- **For GIF files:**
  - **High-Quality Conversion:** The program uses the `libsvtav1` encoder of `ffmpeg` for conversion, preserving animation information.
- **For WebP files:**
  - **High-Quality Conversion:** The program uses the `libsvtav1` encoder of `ffmpeg` for conversion, preserving animation information.
- **For APNG files:**
  - **High-Quality Conversion:** The program uses the `libsvtav1` encoder of `ffmpeg` for conversion, preserving animation information.
- **For HEIC/HEIF files:**
  - **High-Quality Conversion:** The program uses multiple strategies (including ImageMagick, ffmpeg, etc.) to try to convert HEIC/HEIF to an intermediate format, and then to AVIF, supporting the detection and conversion of animated HEIC/HEIF.

### Advantages of AVIF Format

1. **High Compression Ratio:** The AVIF format has a higher compression ratio than GIF/WebP, significantly reducing file size while maintaining visual quality.
2. **Modern Feature Support:** Supports modern features such as HDR, wide color gamut, transparency, and animation.
3. **Wide Compatibility:** Modern browsers and devices all support the AVIF format.

## Installation Requirements

### System Dependencies
- Go 1.19 or higher
- FFmpeg 4.0 or higher (for image conversion)

### Install FFmpeg
```bash
# macOS (using Homebrew)
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg

# Windows (using Chocolatey)
choco install ffmpeg
```

## Build the Project

### Method 1: Using go build
```bash
cd /path/to/dynamic2avif
go build -o bin/dynamic2avif main.go
```

## Usage

The executable is located at `bin/dynamic2avif`. For detailed usage, please see [USAGE_TUTORIAL_ZH.md](../USAGE_TUTORIAL_ZH.md).

### Basic Conversion
```bash
# Convert an entire directory
./bin/dynamic2avif -input /path/to/images -output /path/to/avif/output
```

### Advanced Configuration
```bash
# Convert with high-quality settings
./bin/dynamic2avif -input /input -output /output -quality 80 -speed 5

# Specify the number of concurrent threads
./bin/dynamic2avif -input /input -output /output -workers 4

# Skip existing files
./bin/dynamic2avif -input /input -output /output -skip-exist
```

### Command-Line Arguments

| Argument | Type | Default | Description |
|---|---|---|---|
| `-input` | string | none | Input directory (required) |
| `-output` | string | none | Output directory (required) |
| `-quality` | integer | 50 | AVIF quality (0-100) |
| `-speed` | integer | 6 | Encoding speed (0-10) |
| `-workers` | integer | CPU cores | Number of concurrent worker threads |
| `-skip-exist` | boolean | false | Skip existing files |
| `-dry-run` | boolean | false | Dry run mode |
| `-timeout` | integer | 120 | Timeout in seconds for a single file |
| `-retries` | integer | 2 | Number of retries on failure |
| `-replace` | boolean | false | Delete original files after conversion **⚠️ Safety Note**: Only deletes original files after verifying that the target file exists and is valid. |

## Usage Examples

### Simple Conversion
```bash
./dynamic2avif -input ./images -output ./avif_output
```

### High-Quality Conversion
```bash
./dynamic2avif -input ./images -output ./avif_output -quality 80 -speed 4
```

### Conversion with Limited Concurrency
```bash
./dynamic2avif -input ./images -output ./avif_output -workers 2
```

## Log Interpretation

The program will output the processing progress to the console and generate a `dynamic2avif.log` log file in the current directory. The main log messages include:

- `🔄 开始处理`: Start processing a file
- `🎬 检测到动画图像`: Detected an animated image file
- `✅ 转换完成`: File conversion successful
- `❌ 转换失败`: File conversion failed
- `⏭️  跳过已存在的文件`: Skipped an existing file (when using `-skip-exist`)
- `⚠️  动画检测失败`: Animation detection failed

## Troubleshooting

### Common Problems

1. **"command not found: ffmpeg"**
   - Make sure FFmpeg is installed correctly and is in the PATH

2. **Slow conversion speed**
   - Lower the speed parameter value (0-3)
   - Reduce the workers parameter value
   - Check system resource usage

### Supported File Formats

- **GIF**: .gif (including animation)
- **WebP**: .webp (including animation)
- **APNG**: .png (PNG with animation)
- **HEIC/HEIF**: .heic, .heif (including animation, supports Live Photo detection)

## License

This project is licensed under the MIT License.
