# all2jxl - Universal Image to JXL Converter

A powerful command-line tool to convert a wide variety of image formats to JPEG XL (JXL) losslessly.

## 🚀 Features

- **Broad Format Support**: Converts all common static and animated image formats, including JPEG, PNG, GIF, WebP, HEIC/HEIF, and more.
- **Intelligent HEIC Handling**: Automatically uses the best conversion strategy for HEIC/HEIF files, ensuring maximum compatibility.
- **Metadata Preservation**: Copies all metadata (EXIF, XMP, etc.) from the source image to the new JXL file.
- **Robust Verification**:
    - For lossless sources (like PNG), verifies the conversion with a pixel-perfect check.
    - For tricky formats (like HEIC), uses a simplified verification to ensure the JXL file is valid and decodable.
- **Safe by Default**: Skips conversion if a JXL file of the same name already exists.
- **Detailed Logging**: Provides comprehensive logs of the entire process in `all2jxl.log`.

## 🛠️ Usage

```bash
# Navigate to the script directory
cd /path/to/easy2jxlavif-beta/easymode/all2jxl

# Run the script on your target directory
go run main.go -dir /path/to/your/images
```

### Command-Line Arguments

| Flag | Description | Default |
|---|---|---|
| `-dir` | **(Required)** Input directory containing images to convert. | |
| `-workers` | Number of concurrent worker threads. | 0 (auto-detect based on CPU cores) |
| `-skip-exist` | If `true`, skips conversion if a `.jxl` file already exists. | `true` |
| `-dry-run` | If `true`, simulates the process without actually converting files. | `false` |
| `-cjxl-threads`| Number of threads for each `cjxl` conversion task. | 1 |
| `-timeout` | Timeout in seconds for each conversion task. | 0 (no limit) |
| `-retries` | Number of times to retry a failed conversion. | 0 |
