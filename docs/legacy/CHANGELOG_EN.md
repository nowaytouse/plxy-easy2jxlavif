# Changelog

This document records all important changes to the project.

## [2.1.0] - 2025-10-22

### Added

- **Safe Delete Mechanism**: Implemented strict safe delete checks to ensure original files are only deleted after confirming that target files exist and are valid
  - Tools with `-replace` parameter now perform multiple validations before deleting original files
  - Only tools explicitly with `-replace` parameter will delete original files
  - Other tools (such as `static2avif`, `dynamic2avif`, `static2jxl`, `dynamic2jxl`, etc.) will not delete original files by default, only generating converted files
- **Statistics Accuracy**: Fixed savings space calculation errors
  - When converted files are larger than original files, savings space is displayed as 0 instead of negative values
  - Compression ratio now correctly displays conversion efficiency (>100% indicates file size increase)

### Documentation

- **Script README Updates**: The `README.md` files for all conversion scripts within the `easymode` folder have been updated to reflect the latest features, improvements, and usage instructions.
- **Main README Update**: Updated main README to include information about safe delete mechanism and statistics accuracy fixes

## [2.0.7] - 2025-10-20

### Improved

- **Unified File Scanning Logic**: The file scanning logic across all `easymode` conversion scripts (`all2jxl`, `all2avif`, `dynamic2jxl`, `dynamic2avif`, `static2jxl`, `static2avif`) has been unified. Scripts now accurately identify and process only supported media files, ignoring auxiliary files (e.g., `.xmp`).
- **Precise File Count Validation**: All `easymode` conversion scripts now integrate a new, more precise file count validation mechanism. This provides clear reports on the number of original media files, generated target files, and remaining media files, ensuring the accuracy and reliability of the processing results.
- **Optimized HEIC/HEIF Handling**:
    - For `all2jxl`, `dynamic2jxl`, and `static2jxl` scripts: The HEIC/HEIF conversion strategy has been optimized to use a more stable ImageMagick-to-PNG intermediate file approach, resolving issues where `cjxl` previously failed to process certain HEIC files.
    - For `all2avif`, `dynamic2avif`, and `static2avif` scripts: The HEIC/HEIF conversion strategy has been optimized to use a more stable ImageMagick-to-PNG intermediate file approach.
- **Enhanced Metadata Copying**: Metadata copying logic in `all2avif`, `dynamic2avif`, and `static2avif` scripts has been refined to ensure correct metadata transfer from original source files to target AVIF files.
- **Fixed JPEG Parameter Bug in `all2jxl` Series**: Corrected a bug in `all2jxl`, `dynamic2jxl`, and `static2jxl` scripts where the `--lossless_jpeg=1` parameter was incorrectly applied to non-JPEG files. This parameter is now used only when processing JPEG files.

### Documentation

- **Script README Updates**: The `README.md` files for all conversion scripts within the `easymode` folder have been updated to reflect the latest features, improvements, and usage instructions.

## [2.0.7] - 2025-10-20

### Improved

- **Unified File Scanning Logic**: The file scanning logic across all `easymode` conversion scripts (`all2jxl`, `all2avif`, `dynamic2jxl`, `dynamic2avif`, `static2jxl`, `static2avif`) has been unified. Scripts now accurately identify and process only supported media files, ignoring auxiliary files (e.g., `.xmp`).
- **Precise File Count Validation**: All `easymode` conversion scripts now integrate a new, more precise file count validation mechanism. This provides clear reports on the number of original media files, generated target files, and remaining media files, ensuring the accuracy and reliability of the processing results.
- **Optimized HEIC/HEIF Handling**:
    - For `all2jxl`, `dynamic2jxl`, and `static2jxl` scripts: The HEIC/HEIF conversion strategy has been optimized to use a more stable ImageMagick-to-PNG intermediate file approach, resolving issues where `cjxl` previously failed to process certain HEIC files.
    - For `all2avif`, `dynamic2avif`, and `static2avif` scripts: The HEIC/HEIF conversion strategy has been optimized to use a more stable ImageMagick-to-PNG intermediate file approach.
- **Enhanced Metadata Copying**: Metadata copying logic in `all2avif`, `dynamic2avif`, and `static2avif` scripts has been refined to ensure correct metadata transfer from original source files to target AVIF files.
- **Fixed JPEG Parameter Bug in `all2jxl` Series**: Corrected a bug in `all2jxl`, `dynamic2jxl`, and `static2jxl` scripts where the `--lossless_jpeg=1` parameter was incorrectly applied to non-JPEG files. This parameter is now used only when processing JPEG files.

### Documentation

- **Script README Updates**: The `README.md` files for all conversion scripts within the `easymode` folder have been updated to reflect the latest features, improvements, and usage instructions.

## [2.0.8] - 2025-10-20

### Added

- **Video Re-packaging Tool (`video2mov`)**: Added a new independent helper script `easymode/video2mov` to **losslessly re-package** various video formats into the `.mov` container format. This tool uses `ffmpeg -c copy` for stream copying, ensuring full preservation of original video and audio stream quality, and supports metadata retention and precise file count validation.

### Documentation

- **Main README Update**: The project's main `README.md` file has been updated to include an introduction to the `video2mov` script.

## [2.0.2] - 2025-01-27

### Added Features

- **Modular Validation System**: Added comprehensive post-processing validation mechanism including file count, size, EXIF data validation and failure report generation
- **Static/Dynamic Image Separation**: Added separate tools for static image to JXL (`static2jxl`) and dynamic image to JXL (`dynamic2jxl`) conversion
- **Smart Validation Reports**: Automatically generates user-friendly validation reports in both JSON and text formats to help users understand processing results
- **Unprocessed File Analysis**: Automatically detects and analyzes unprocessed files with detailed reason analysis
- **Main Program Optimization**: Integrated advanced scanning engine, smart strategy, watchdog, state management and security enhancements

### Fixed

- Fixed a critical bug that would incorrectly delete original files when processing files with the same name but different extensions (e.g., `image.jpg` and `image.jpeg`). After converting the first file, the program would detect that the target file already exists and skip the second file's conversion, but then incorrectly delete the second original file.
- Now, when skipping existing target files, the program will no longer delete the corresponding original files, ensuring data safety.

### Improved

- **easymode Tool Consistency**: All easymode tools now use the same fix logic, ensuring consistent behavior
- **Validation Mechanism Integration**: Validation modules have been integrated into all easymode tools, providing unified validation experience
- **Enhanced Error Handling**: Improved error handling and logging with more detailed processing information
- **Test Validation Completed**: All new features have been validated for stability and correctness through actual testing

### Test Results

- **Emoji Folder Test**: Successfully converted 31 JXL files, all original image files processed
- **Menchazi Folder Test**: Successfully converted 31 JXL files, all original image files processed
- **Feature Stability**: All new features performed stably in real-world usage with no data loss

## [2.0.1] - 2025-10-19

### Fixed

- Fixed a critical bug where original files were incorrectly deleted when target files with the same name but different extensions already existed
- Improved error handling and logging throughout the application

## [2.0.0] - 2025-10-19

### Added Features

- **Easy Mode Tools**: Added simplified command-line tools for common conversion tasks
  - `all2avif`: Convert all supported images to AVIF format
  - `all2jxl`: Convert all supported images to JXL format
  - `static2avif`: Convert static images to AVIF format
  - `dynamic2avif`: Convert dynamic/animated images to AVIF format
  - `static2jxl`: Convert static images to JXL format
  - `dynamic2jxl`: Convert dynamic/animated images to JXL format
- **Advanced Main Program**: Enhanced main program with advanced features
  - Two-stage scanning engine
  - Smart conversion strategy
  - Watchdog monitoring
  - State management with persistent storage
  - Security enhancements
- **Comprehensive Documentation**: Added detailed documentation in both Chinese and English

### Improved

- **Performance**: Optimized conversion performance with smart concurrency control
- **User Experience**: Enhanced user interface with progress tracking and detailed logging
- **Error Handling**: Comprehensive error handling and recovery mechanisms
- **Code Quality**: Improved code structure and maintainability

### Technical Details

- **Language**: Go 1.21+
- **Dependencies**: Minimal external dependencies for better reliability
- **Architecture**: Modular design with clear separation of concerns
- **Testing**: Comprehensive test coverage including unit, integration, and performance tests