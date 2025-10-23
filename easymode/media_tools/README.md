# Media Tools - 媒体文件辅助工具集

A unified tool for media file management, combining XMP metadata merging, duplicate detection, and file normalization.

## Features

### 1. XMP Metadata Merging (`merge`)
- Merge XMP sidecar files into corresponding media files
- Support for `.xmp` and `.sidecar.xmp` formats
- Automatic media file detection
- Uses ExifTool for reliable metadata transfer
- Handles paths with spaces and special characters

### 2. Duplicate Media Detection (`dedup`)
- SHA256 hash-based duplicate detection
- Move duplicates to trash directory
- Support for all common media formats
- Safe deletion with verification
- Handles paths with spaces and special characters

### 3. File Extension Normalization (`normalize`)
- Standardize file extensions: `.jpeg` → `.jpg`, `.tiff` → `.tif`
- Case-insensitive detection
- Batch processing
- Dry-run mode available

### 4. Auto Processing (`auto`) - **Recommended**
- Executes all operations in correct order:
  1. Normalize extensions
  2. Merge XMP metadata
  3. Detect and remove duplicates
- One-command solution for complete media management

## Installation

```bash
./build.sh
```

## Usage

### Auto Processing (Recommended)

```bash
# Complete media management in one command
./bin/media_tools auto -dir /path/to/media -trash /path/to/trash

# Dry-run mode (preview only)
./bin/media_tools auto -dir /path/to/media -trash /path/to/trash -dry-run

# Works with paths containing spaces
./bin/media_tools auto -dir "/path/with spaces/media" -trash "/path/to/trash"
```

### Individual Operations

#### Normalize File Extensions
```bash
# Standardize extensions (.jpeg→.jpg, .tiff→.tif)
./bin/media_tools normalize -dir /path/to/media

# Preview changes
./bin/media_tools normalize -dir /path/to/media -dry-run
```

#### Merge XMP Metadata
```bash
# Merge XMP sidecar files
./bin/media_tools merge -dir /path/to/media

# Preview merges
./bin/media_tools merge -dir /path/to/media -dry-run
```

#### Deduplicate Media Files
```bash
# Remove duplicates
./bin/media_tools dedup -dir /path/to/media -trash /path/to/trash

# Preview duplicates
./bin/media_tools dedup -dir /path/to/media -trash /path/to/trash -dry-run
```

## Requirements

- Go 1.25+
- ExifTool (for metadata operations)

## Examples

```bash
# Complete auto processing (recommended)
./bin/media_tools auto -dir ~/Pictures/PhotoLibrary -trash ~/Pictures/.trash

# Works with Chinese characters and spaces
./bin/media_tools auto -dir "~/图片/照片 (2024)" -trash "~/图片/.trash"

# Individual operations
./bin/media_tools normalize -dir ~/Pictures  # Step 1
./bin/media_tools merge -dir ~/Pictures      # Step 2
./bin/media_tools dedup -dir ~/Pictures -trash ~/Pictures/.trash  # Step 3

# Preview all operations without making changes
./bin/media_tools auto -dir ~/Pictures -trash ~/Pictures/.trash -dry-run
```

## Version

2.2.0

## Author

AI Assistant

