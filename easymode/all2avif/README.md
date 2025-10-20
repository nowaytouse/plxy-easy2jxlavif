# all2avif - Universal Image to AVIF Converter

A powerful command-line tool to convert a wide variety of static image formats to AVIF.

## 🚀 Features

- **Broad Format Support**: Converts all common static image formats, including JPEG, PNG, BMP, TIFF, HEIC/HEIF, and more.
- **Intelligent HEIC Handling**: Automatically uses the best conversion strategy for HEIC/HEIF files, ensuring maximum compatibility.
- **Metadata Preservation**: Copies all metadata (EXIF, XMP, etc.) from the source image to the new AVIF file.
- **Robust Verification**: Ensures the conversion process is clean and file counts are accurate.
- **Safe by Default**: Skips conversion if an AVIF file of the same name already exists.
- **Detailed Logging**: Provides comprehensive logs of the entire process in `all2avif.log`.

## 🛠️ Usage

```bash
# Navigate to the script directory
cd /path/to/easy2jxlavif-beta/easymode/all2avif

# Run the script on your target directory
go run main.go -dir /path/to/your/images
```

### Command-Line Arguments

| Flag | Description | Default |
|---|---|---|
| `-dir` | **(Required)** Input directory containing images to convert. | |
| `-workers` | Number of concurrent worker threads. | 0 (auto-detect based on CPU cores) |
| `-quality` | AVIF quality (0-100). Higher value = higher quality. | 80 |
| `-speed` | Encoding speed (0-6). Higher value = faster encoding, but larger file size. | 4 |
| `-skip-exist` | If `true`, skips conversion if an `.avif` file already exists. | `true` |
| `-dry-run` | If `true`, simulates the process without actually converting files. | `false` |
| `-timeout` | Timeout in seconds for each conversion task. | 300 |
| `-retries` | Number of times to retry a failed conversion. | 1 |
| `-replace` | If `true`, deletes original files after successful conversion. | `true` |
