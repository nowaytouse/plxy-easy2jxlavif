# `deduplicate_media` - Media Deduplication Tool

## 📖 Introduction

`deduplicate_media` is a utility script that scans a specified directory for media files, identifies content-based duplicates, and moves them to a designated "trash" folder. It also standardizes inconsistent file extensions (e.g., renaming `.jpeg` to `.jpg`).

## 🚀 Features

- ✅ **Broad Format Support** - Supports common image formats (like `.jpg`, `.png`, `.gif`, `.bmp`, `.tif`, `.webp`) and video formats (like `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`).
- ✅ **Standardize Extensions** - Automatically renames extensions like `.jpeg` and `.tiff` to a consistent `.jpg` and `.tif` format.
- ✅ **Accurate Deduplication** - Quickly identifies potential duplicates using SHA-256 hashes and confirms them with a byte-by-byte comparison.
- ✅ **Safe Moving** - Duplicates are moved to a specified folder instead of being permanently deleted, allowing for final review and recovery.
- ✅ **Trash Folder Readme** - Automatically creates a `_readme_about_this_folder.txt` file in the trash folder to explain its purpose.
- ✅ **Clear Logging** - Logs all operations, including extension renaming, discovered duplicates, and moved files.

## 🔧 Usage

### Build the Script

```bash
# Navigate to the script directory
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/deduplicate_media

# Run the build script
./build.sh
```

### Run the Script

```bash
./deduplicate_media -dir /path/to/your/media -trash-dir /path/to/trash
```

### Argument Description

- `-dir`: The path to the directory containing media files to scan (required).
- `-trash-dir`: The path to the directory where duplicate files will be moved (required). If the directory does not exist, the script will create it automatically.

## 📈 Example Output

```
INFO: 2025/10/19 21:25:00 main.go:25: deduplicate_media v1.1.0 starting...
INFO: 2025/10/19 21:25:00 main.go:71: Standardizing extensions...
INFO: 2025/10/19 21:25:00 main.go:86: Renamed image (1).jpeg to image (1).jpg
INFO: 2025/10/19 21:25:00 main.go:92: Finding and moving duplicates...
INFO: 2025/10/19 21:25:01 main.go:110: Potential duplicate found: /path/to/media/image.jpg and /path/to/media/image (1).jpg
INFO: 2025/10/19 21:25:01 main.go:118: Files are identical. Moving image (1).jpg to trash.
INFO: 2025/10/19 21:25:01 main.go:50: Deduplication process complete.
```

---

**Version**: v1.1.0  
**Maintainer**: AI Assistant  
**License**: MIT