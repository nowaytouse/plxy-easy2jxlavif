# video2mov - Video Repackaging Tool

`video2mov` is a command-line tool designed for video files, aiming to **losslessly repackage** various video formats into the `.mov` container format. This tool does not re-encode video, but instead uses stream copy to ensure that the quality of the original video and audio streams is fully preserved, while providing better compatibility and metadata handling capabilities.

## 🚀 Core Features

- **Lossless Repackaging**: Uses `ffmpeg -c copy` for stream copying, without any re-encoding of video or audio, ensuring original quality.
- **Wide Video Format Support**: Supports common video formats such as `.mp4`, `.avi`, `.mkv`, `.wmv`, `.flv`, `.webm`, `.m4v`, `.3gp`, etc.
- **Metadata Preservation**: Uses `exiftool` to completely copy the metadata of the original video file to the new `.mov` file.
- **Accurate File Count Verification**: After repackaging is complete, a detailed file count verification report is provided to ensure the accuracy and reliability of the processing.
- **Safe and Reliable**: Supports a retry mechanism and verifies that the output file exists and is valid before deleting the original file, ensuring the safety of the processing.
- **Detailed Logging**: Provides a comprehensive processing log `video2mov.log`.

## 🛠️ Usage

### Basic Usage
```bash
# Navigate to the script directory
cd /path/to/easy2jxlavif-beta/easymode/video2mov

# Run the script to repackage videos in the specified directory
go run main.go -input /path/to/your/videos -output /path/to/mov/output
```

### Command-Line Arguments

| Argument | Type | Default | Description |
|---|---|---|---|
| `-input` | string | none | Input directory (required) |
| `-output` | string | none | Output directory (defaults to the input directory) |
| `-workers` | integer | CPU cores | Number of concurrent worker threads |
| `-skip-exist` | boolean | `true` | Skip existing target `.mov` files |
| `-dry-run` | boolean | `false` | Dry run mode, only prints the files to be processed |
| `-timeout` | integer | 300 | Timeout in seconds for a single file |
| `-retries` | integer | 2 | Number of retries on failure |
| `-replace` | boolean | `false` | Delete original video files after repackaging |

## 🔍 Troubleshooting

### Common Problems
1. **Missing dependencies**: Make sure `ffmpeg` and `exiftool` are installed.
2. **Permission issues**: Check file read/write permissions.
3. **Insufficient space**: Make sure there is enough disk space.

## 📝 Update Log

### v1.0.0 - 2025-10-20
- ✅ Added video repackaging tool `video2mov`.
- ✅ Implemented lossless stream copying of videos to `.mov` format.
- ✅ Integrated accurate file count verification.
- ✅ Supports metadata preservation.