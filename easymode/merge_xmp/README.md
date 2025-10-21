# `merge_xmp` - XMP Metadata Merging Tool

## 📖 Introduction

`merge_xmp` is a standalone utility script for merging metadata from `.xmp` files into media files. It automatically finds `.xmp` files with the same name as media files (e.g., `.jpg`, `.png`), merges them using `exiftool`, and deletes the `.xmp` file after successful verification.

## 🚀 Features

- ✅ **Automatic Discovery** - Automatically finds `.xmp` files with the same name as media files (e.g., `.jpg`, `.png`).
- ✅ **Metadata Merging** - Uses `exiftool` to merge all metadata from the `.xmp` file into the media file.
- ✅ **Automatic Deletion** - Automatically deletes the `.xmp` file after successful merging and verification.
- ✅ **Safe Verification** - If verification fails, the `.xmp` file is retained for manual inspection.

## 🔧 Usage

### Dependencies

- **exiftool**: Make sure `exiftool` is installed and in the system's `PATH`.

### Build the Script

```bash
# Navigate to the script directory
cd /Users/nyamiiko/Documents/git/easy2jxlavif-beta/easymode/merge_xmp

# Run the build script
./build.sh
```

### Run the Script

```bash
./merge_xmp -dir /path/to/your/media
```

### Argument Description

- `-dir`: The path to the directory containing the media files to be processed (required).

## 📈 Example Output

```
INFO: 2025/10/19 20:55:03 main.go:33: merge_xmp v1.0.0 starting...
INFO: 2025/10/19 20:55:03 main.go:89: Found media file 'IMG_0429.JPG' with XMP sidecar 'IMG_0429.xmp'
INFO: 2025/10/19 20:55:03 main.go:98: Successfully merged XMP into IMG_0429.JPG
INFO: 2025/10/19 20:55:03 main.go:102: Verification successful for IMG_0429.JPG
INFO: 2025/10/19 20:55:03 main.go:107: Successfully deleted XMP file IMG_0429.xmp
INFO: 2025/10/19 20:55:03 main.go:66: Processing complete.
```

---

**Version**: v1.0.0  
**Maintainer**: AI Assistant  
**License**: MIT