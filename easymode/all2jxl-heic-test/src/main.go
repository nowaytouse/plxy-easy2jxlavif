package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/karrick/godirwalk"
)

// Config holds the configuration for the conversion process
type Config struct {
	InputDir        string
	OutputDir       string
	Quality         float64
	Effort          int
	NumThreads      int
	Verbose         bool
	ReplaceOriginal bool
	SkipCorrupted   bool
	Verify          bool
	LosslessJPEG    bool
	StrictDynamic   bool
	Mode            string // "auto", "jpeg", "dynamic", "lossless"
}

// calculateHash computes the SHA256 hash of a file
func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// validateJXLFile checks if the JXL file is valid
func validateJXLFile(jxlPath string) error {
	// Use djxl to decode the JXL file to a temporary PPM file to verify it's valid
	tempOutput := jxlPath + ".tmp.ppm"
	
	// Create a context with timeout to prevent hanging during validation
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // 1 minute for validation
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "djxl", jxlPath, tempOutput)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start validation for %s: %v", jxlPath, err)
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill validation process for %s after timeout: %v, original error: %v", jxlPath, killErr, ctx.Err())
		}
		return fmt.Errorf("validation timed out for %s (possibly hanging or very large file)", jxlPath)
	case err := <-done:
		if err != nil {
			// Since cmd.Wait() doesn't provide output directly, we'll report just the error
			// Clean up temp file regardless of success or failure
			os.Remove(tempOutput)
			return fmt.Errorf("invalid JXL file: %v", err)
		}
	}

	// Clean up temp file regardless of success or failure
	os.Remove(tempOutput)
	return nil
}

// validateJXLInfo checks if the JXL file has valid metadata
func validateJXLInfo(jxlPath string) error {
	// Use jxlinfo to check the file with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30 seconds for metadata validation
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "jxlinfo", jxlPath)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start jxlinfo validation for %s: %v", jxlPath, err)
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill jxlinfo process for %s after timeout: %v, original error: %v", jxlPath, killErr, ctx.Err())
		}
		return fmt.Errorf("jxlinfo validation timed out for %s (possibly hanging)", jxlPath)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("jxlinfo validation failed: %v", err)
		}

		// If command succeeded, we need to run it again to get the output for validation
		cmd := exec.CommandContext(context.Background(), "jxlinfo", jxlPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("jxlinfo validation failed when retrieving output: %v", err)
		}

		// Check if the output contains basic info indicating a valid file
		outputStr := string(output)
		if !strings.Contains(outputStr, "JPEG XL") {
			return fmt.Errorf("jxlinfo reported invalid file: %s", outputStr)
		}
	}

	return nil
}

// isJPEGFile checks if the file is a JPEG format
func isJPEGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".jfif"
}

// isDynamicImage checks if an image file is actually dynamic (has animation)
func isDynamicImage(filePath string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".gif":
		// Check if it's a real animated GIF
		return isAnimatedGIF(filePath)
	case ".png":
		// Check if it's an animated PNG (APNG)
		return isAnimatedPNG(filePath)
	case ".webp":
		// Check if it's an animated WebP
		return isAnimatedWebP(filePath)
	default:
		return false, fmt.Errorf("unsupported format for dynamic check: %s", ext)
	}
}

// isAnimatedGIF checks if a GIF file is actually animated
func isAnimatedGIF(filePath string) (bool, error) {
	// Use ImageMagick identify to check for animation frames
	cmd := exec.Command("identify", "-format", "%%n\n", filePath)
	output, err := cmd.Output()
	if err != nil {
		// If ImageMagick is not available, assume it's animated if it's a GIF
		return true, nil
	}
	
	// Check if it has more than 1 frame
	frameCountStr := strings.TrimSpace(string(output))
	frameCount, err := strconv.Atoi(frameCountStr)
	if err != nil {
		return false, err
	}
	return frameCount > 1, nil
}

// isAnimatedPNG checks if a PNG file is actually animated (APNG format)
func isAnimatedPNG(filePath string) (bool, error) {
	// Check for APNG by using ImageMagick
	cmd := exec.Command("identify", "-format", "%%n\n", filePath)
	output, err := cmd.Output()
	if err != nil {
		// If ImageMagick is not available, return true for any PNG to be safe
		return true, nil
	}

	// Check if it has more than 1 frame
	frameCountStr := strings.TrimSpace(string(output))
	frameCount, err := strconv.Atoi(frameCountStr)
	if err != nil {
		return false, err
	}
	return frameCount > 1, nil
}

// isAnimatedWebP checks if a WebP file is actually animated
func isAnimatedWebP(filePath string) (bool, error) {
	// Use dwebp to check if WebP is animated
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "dwebp", "-info", filePath)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return false, err
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return false, fmt.Errorf("failed to kill dwebp process: %v", killErr)
		}
		return false, fmt.Errorf("dwebp info timeout for %s", filePath)
	case err := <-done:
		if err != nil {
			// If dwebp fails, check using ffprobe as fallback
			cmd := exec.Command("ffprobe", "-v", "quiet", "-show_streams", filePath)
			output, probeErr := cmd.CombinedOutput()
			if probeErr != nil {
				// If neither tool works, assume it's not animated
				return false, nil
			}
			
			// Look for animation keywords in ffprobe output
			outputStr := string(output)
			return strings.Contains(outputStr, "animated") || strings.Contains(outputStr, "nb_frames=") && !strings.Contains(outputStr, "nb_frames=1"), nil
		}
		// dwebp succeeded but doesn't output clear animation info
		// If it didn't error, likely it's a static image
		return false, nil
	}
}

// isLosslessFormat checks if the file format is a lossless format suitable for lossless conversion
func isLosslessFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Supported formats for lossless conversion
	supportedFormats := map[string]bool{
		".png":  true,
		".bmp":  true,
		".tga":  true,
		".ppm":  true,
		".pgm":  true,
		".pbm":  true,
		".pam":  true,
		".pfm":  true,
		".heic": true,  // HEIC is lossless and should be treated as such
		".heif": true,  // HEIF is lossless and should be treated as such
	}
	
	// Exclude source files like PSD, CLIP, KRA, SAI2, etc.
	excludedFormats := map[string]bool{
		".psd":  true,
		".clip": true,
		".kra":  true,
		".sai2": true,
		".xcf":  true,
		".ora":  true,
		".pdn":  true,
	}
	
	// Return true if it's a supported format and not an excluded one
	return supportedFormats[ext] && !excludedFormats[ext]
}

// convertImage converts an image file to JXL format based on its type
func convertImage(inputPath, outputPath string, config Config) error {
	// Determine the appropriate conversion method based on file type
	if isJPEGFile(inputPath) {
		return convertJPEGToJXL(inputPath, outputPath, config)
	} else if isHEICFile(inputPath) {
		return convertHEICToJXL(inputPath, outputPath, config)
	} else if isLosslessFormat(inputPath) {
		return convertLosslessToJXL(inputPath, outputPath, config)
	} else if dynamicCheck, _ := isDynamicImage(inputPath); dynamicCheck {
		return convertDynamicToJXL(inputPath, outputPath, config)
	} else {
		// For unknown formats, treat as lossless
		return convertLosslessToJXL(inputPath, outputPath, config)
	}
}

// convertJPEGToJXL converts a JPEG file to JXL format
func convertJPEGToJXL(inputPath, outputPath string, config Config) error {
	// Prepare the command arguments
	args := []string{inputPath, outputPath}

	// Add lossless JPEG parameter if enabled
	if config.LosslessJPEG {
		args = append(args, "--lossless_jpeg=1")
	} else {
		args = append(args, "--lossless_jpeg=0")
	}

	// Add quality parameter if lossless JPEG is disabled
	if !config.LosslessJPEG {
		if config.Quality >= 0 {
			args = append(args, fmt.Sprintf("--distance=%.2f", config.Quality))
		}
	}

	// Add effort parameter
	if config.Effort >= 1 && config.Effort <= 10 {
		args = append(args, fmt.Sprintf("--effort=%d", config.Effort))
	}

	// Add verbose flag if needed
	if config.Verbose {
		args = append(args, "-v")
	}

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 2 minutes for conversion
	defer cancel()

	// Execute the cjxl command with timeout
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start JPEG conversion for %s: %v", inputPath, err)
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill JPEG conversion process for %s after timeout: %v, original error: %v", inputPath, killErr, ctx.Err())
		}
		return fmt.Errorf("JPEG conversion timed out for %s (possibly hanging or very large file)", inputPath)
	case err := <-done:
		if err != nil {
			// Since cmd.Wait() doesn't provide output directly, we'll report just the error
			return fmt.Errorf("failed to convert JPEG %s: %v", inputPath, err)
		}
	}

	if config.Verbose {
		fmt.Printf("Successfully converted JPEG: %s -> %s\n", inputPath, outputPath)
	}

	return nil
}

// convertLosslessToJXL converts a lossless image (PNG, BMP, etc.) to JXL format
func convertLosslessToJXL(inputPath, outputPath string, config Config) error {
	// Prepare the command arguments
	args := []string{inputPath, outputPath}

	// For lossless formats, we want to maintain maximum quality
	// Set quality to 0.0 for mathematically lossless conversion
	args = append(args, "--distance=0.0")
	
	// Add effort parameter
	if config.Effort >= 1 && config.Effort <= 10 {
		args = append(args, fmt.Sprintf("--effort=%d", config.Effort))
	}
	
	// For lossless formats, enable lossless JPEG transcoding even though it's not JPEG
	// This helps preserve metadata better
	args = append(args, "--lossless_jpeg=1")
	
	// Use container format to preserve all metadata
	args = append(args, "--container=1")
	args = append(args, "--compress_boxes=1")

	// Add verbose flag if needed
	if config.Verbose {
		args = append(args, "-v")
	}

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes for large lossless files
	defer cancel()

	// Execute the cjxl command with timeout
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start lossless conversion for %s: %v", inputPath, err)
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill lossless conversion process for %s after timeout: %v, original error: %v", inputPath, killErr, ctx.Err())
		}
		return fmt.Errorf("lossless conversion timed out for %s (possibly hanging or very large file)", inputPath)
	case err := <-done:
		if err != nil {
			// Since cmd.Wait() doesn't provide output directly, we'll report just the error
			return fmt.Errorf("failed to convert lossless image %s: %v", inputPath, err)
		}
	}

	if config.Verbose {
		fmt.Printf("Successfully converted lossless image: %s -> %s\n", inputPath, outputPath)
	}

	return nil
}

// convertDynamicToJXL converts a dynamic image (GIF, APNG, etc.) to JXL format
func convertDynamicToJXL(inputPath, outputPath string, config Config) error {
	// Prepare the command arguments
	args := []string{inputPath, outputPath}

	// Add quality parameter
	if config.Quality >= 0 {
		args = append(args, fmt.Sprintf("--distance=%.2f", config.Quality))
	}

	// Add effort parameter
	if config.Effort >= 1 && config.Effort <= 10 {
		args = append(args, fmt.Sprintf("--effort=%d", config.Effort))
	}

	// For dynamic images, we want to preserve animation and metadata
	// Use container format to preserve all metadata and animation
	args = append(args, "--container=1")
	args = append(args, "--compress_boxes=1")

	// Add verbose flag if needed
	if config.Verbose {
		args = append(args, "-v")
	}

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 2 minutes for conversion
	defer cancel()

	// Execute the cjxl command with timeout
	cmd := exec.CommandContext(ctx, "cjxl", args...)
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dynamic conversion for %s: %v", inputPath, err)
	}

	// Wait for the command to finish with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled (timeout), try to kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill dynamic conversion process for %s after timeout: %v, original error: %v", inputPath, killErr, ctx.Err())
		}
		return fmt.Errorf("dynamic conversion timed out for %s (possibly hanging or very large file)", inputPath)
	case err := <-done:
		if err != nil {
			// Since cmd.Wait() doesn't provide output directly, we'll report just the error
			return fmt.Errorf("failed to convert dynamic image %s: %v", inputPath, err)
		}
	}

	if config.Verbose {
		fmt.Printf("Successfully converted dynamic image: %s -> %s\n", inputPath, outputPath)
	}

	return nil
}

// isHEICFile checks if the file is a HEIC or HEIF format
func isHEICFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".heic" || ext == ".heif"
}

// convertHEICToJXL converts a HEIC file to JXL format by first converting to PNG
func convertHEICToJXL(inputPath, outputPath string, config Config) error {
	// First convert HEIC to PNG using ImageMagick
	tempPNGPath := inputPath + ".temp.png"
	
	// Use ImageMagick to convert HEIC to PNG
	convertCmd := exec.Command("magick", inputPath, tempPNGPath)
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("failed to convert HEIC to PNG: %v", err)
	}
	
	// Now convert the temporary PNG to JXL using existing function
	if err := convertLosslessToJXL(tempPNGPath, outputPath, config); err != nil {
		// Clean up temp file even if conversion fails
		os.Remove(tempPNGPath)
		return fmt.Errorf("failed to convert intermediate PNG to JXL: %v", err)
	}
	
	// Clean up temp file after successful conversion
	if err := os.Remove(tempPNGPath); err != nil {
		if config.Verbose {
			fmt.Printf("Warning: Failed to remove temporary file %s: %v\n", tempPNGPath, err)
		}
	}
	
	if config.Verbose {
		fmt.Printf("Successfully converted HEIC: %s -> %s\n", inputPath, outputPath)
	}
	
	return nil
}

// findFiles finds all supported image files in the input directory and its subdirectories
func findFiles(inputDir string, config Config) (map[string][]string, error) {
	fileTypes := make(map[string][]string)
	
	err := godirwalk.Walk(inputDir, &godirwalk.Options{
		Callback: func(osPathname string, entry *godirwalk.Dirent) error {
			if entry.IsDir() {
				return nil
			}

			// Check if the file is a supported format
			ext := strings.ToLower(filepath.Ext(osPathname))
			
			// Determine file type and add to appropriate category
			if isJPEGFile(osPathname) {
				fileTypes["jpeg"] = append(fileTypes["jpeg"], osPathname)
			} else if isHEICFile(osPathname) {
				fileTypes["lossless"] = append(fileTypes["lossless"], osPathname) // Treat HEIC as lossless
			} else if isLosslessFormat(osPathname) {
				fileTypes["lossless"] = append(fileTypes["lossless"], osPathname)
			} else if dynamicCheck, _ := isDynamicImage(osPathname); dynamicCheck {
				fileTypes["dynamic"] = append(fileTypes["dynamic"], osPathname)
			} else if config.Mode == "auto" {
				// For auto mode, try to guess file type based on extension
				switch ext {
				case ".jpg", ".jpeg", ".jfif":
					fileTypes["jpeg"] = append(fileTypes["jpeg"], osPathname)
				case ".heic", ".heif":
					fileTypes["lossless"] = append(fileTypes["lossless"], osPathname) // Treat HEIC as lossless
				case ".png", ".bmp", ".tga", ".ppm", ".pgm", ".pbm", ".pam", ".pfm":
					fileTypes["lossless"] = append(fileTypes["lossless"], osPathname)
				case ".gif", ".webp":
					fileTypes["dynamic"] = append(fileTypes["dynamic"], osPathname)
				}
			}

			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the directory: %v", err)
	}

	return fileTypes, nil
}

// ensureDir ensures that the output directory exists
func ensureDir(outputPath string) error {
	dir := filepath.Dir(outputPath)
	return os.MkdirAll(dir, 0755)
}

// replaceOriginal replaces the original image file with the converted JXL file
func replaceOriginal(imagePath, jxlPath string, verbose bool) error {
	// Remove the original image file
	if err := os.Remove(imagePath); err != nil {
		return fmt.Errorf("failed to remove original image: %v", err)
	}

	// Move the JXL file to the original image location (but with .jxl extension)
	imageDir := filepath.Dir(imagePath)
	imageBase := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	newJxlPath := filepath.Join(imageDir, imageBase+".jxl")

	if err := os.Rename(jxlPath, newJxlPath); err != nil {
		return fmt.Errorf("failed to move JXL to original location: %v", err)
	}

	if verbose {
		fmt.Printf("Replaced %s with %s\n", imagePath, newJxlPath)
	}

	return nil
}

// processFile processes a single file for conversion
func processFile(inputPath string, config Config, wg *sync.WaitGroup, errors chan<- error, corruptedFiles chan<- string) {
	defer wg.Done()

	// Generate output path with .jxl extension
	relPath, err := filepath.Rel(config.InputDir, inputPath)
	if err != nil {
		select {
		case errors <- fmt.Errorf("failed to get relative path for %s: %v", inputPath, err):
		default: // Non-blocking send in case errors channel is closed
		}
		return
	}

	var outputPath string
	if config.ReplaceOriginal {
		// For in-place replacement, output to a temporary location first
		tempDir := filepath.Join(filepath.Dir(inputPath), ".jxl_temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			select {
			case errors <- fmt.Errorf("failed to create temp directory for %s: %v", inputPath, err):
			default: // Non-blocking send in case channel is full
			}
			return
		}
		outputPath = filepath.Join(tempDir, filepath.Base(inputPath)+".jxl")
	} else {
		outputPath = filepath.Join(config.OutputDir, relPath)
		outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".jxl"
	}

	// Ensure output directory exists
	if err := ensureDir(outputPath); err != nil {
		select {
		case errors <- fmt.Errorf("failed to create output directory for %s: %v", outputPath, err):
		default: // Non-blocking send in case channel is full
			fmt.Fprintf(os.Stderr, "Could not add output dir error to channel: %v\n", err)
		}
		return
	}

	// Convert the image
	if err := convertImage(inputPath, outputPath, config); err != nil {
		// If requested, skip corrupted files by adding to the corrupted list
		if config.SkipCorrupted {
			select {
			case corruptedFiles <- inputPath:
			default: // Non-blocking send in case channel is full
				// If channel is full, log to stderr but don't block
				fmt.Fprintf(os.Stderr, "Could not add corrupted file to channel: %s\n", inputPath)
			}
			if config.Verbose {
				fmt.Printf("Skipped corrupted file: %s (%v)\n", inputPath, err)
			}
		} else {
			select {
			case errors <- err:
			default: // Non-blocking send in case channel is full
				// If channel is full, log to stderr but don't block
				fmt.Fprintf(os.Stderr, "Could not add error to channel: %v\n", err)
			}
		}
		return
	}

	// Verify the output file if requested
	if config.Verify {
		if err := validateJXLFile(outputPath); err != nil {
			select {
			case errors <- fmt.Errorf("verification failed for %s: %v", outputPath, err):
			default: // Non-blocking send in case channel is full
				fmt.Fprintf(os.Stderr, "Could not add verification error to channel: %v\n", err)
			}
			// Clean up the invalid file
			os.Remove(outputPath)
			return
		}

		if err := validateJXLInfo(outputPath); err != nil {
			select {
			case errors <- fmt.Errorf("info validation failed for %s: %v", outputPath, err):
			default: // Non-blocking send in case channel is full
				fmt.Fprintf(os.Stderr, "Could not add validation error to channel: %v\n", err)
			}
			// Clean up the invalid file
			os.Remove(outputPath)
			return
		}

		if config.Verbose {
			fmt.Printf("Verified: %s (size: %d bytes)\n", outputPath, getFileSize(outputPath))
		}
	}

	// If replace original is enabled, perform the replacement
	if config.ReplaceOriginal {
		if err := replaceOriginal(inputPath, outputPath, config.Verbose); err != nil {
			select {
			case errors <- fmt.Errorf("failed to replace original file %s: %v", inputPath, err):
			default: // Non-blocking send in case channel is full
				fmt.Fprintf(os.Stderr, "Could not add replacement error to channel: %v\n", err)
			}
			// If we couldn't replace, keep the converted file
			return
		}
	}
}

// getFileSize returns the size of the file in bytes
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// processFiles processes all image files found in the input directory
func processFiles(fileTypes map[string][]string, config Config) ([]string, []error) {
	var errors []error
	var corruptedFilesList []string

	// Calculate total number of files
	totalFiles := 0
	for _, files := range fileTypes {
		totalFiles += len(files)
	}

	// Create channels to collect results
	errChan := make(chan error, totalFiles)
	corruptedChan := make(chan string, totalFiles)

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Limit the number of concurrent conversions
	var maxWorkers int
	if config.NumThreads > 0 {
		maxWorkers = config.NumThreads
	} else {
		// Use conservative threading to prevent system overload
		maxWorkers = runtime.NumCPU() / 4
		if maxWorkers < 1 {
			maxWorkers = 1
		}
		if maxWorkers > 4 {  // Cap to 4 to prevent too much resource usage
			maxWorkers = 4
		}
	}

	// Create a semaphore to limit concurrent workers
	semaphore := make(chan struct{}, maxWorkers)

	// Process each file type
	for _, files := range fileTypes {
		for _, file := range files {
			wg.Add(1)
			go func(f string) {
				// Acquire semaphore
				semaphore <- struct{}{}
				// Add a substantial delay to prevent resource overload
				time.Sleep(50 * time.Millisecond)
				defer func() { 
					time.Sleep(20 * time.Millisecond)  // Small delay when releasing to reduce thrashing
					<-semaphore // Release semaphore
				}()

				processFile(f, config, &wg, errChan, corruptedChan)
			}(file)
		}
	}

	// Use a goroutine to signal when all work is done or timeout occurs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for all work to complete or timeout after a reasonable time
	select {
	case <-done:
		// All work completed successfully
		close(errChan)
		close(corruptedChan)
	case <-time.After(60 * time.Minute): // 60 minute timeout as safety measure
		fmt.Fprintf(os.Stderr, "Timeout waiting for conversions to complete. Some operations may still be running.\n")
		close(errChan)
		close(corruptedChan)
	}

	// Collect errors from the channel
	for err := range errChan {
		errors = append(errors, err)
		if config.Verbose {
			fmt.Printf("Error: %v\n", err)
		}
	}

	// Collect corrupted files from the channel
	for corruptedFile := range corruptedChan {
		corruptedFilesList = append(corruptedFilesList, corruptedFile)
	}

	return corruptedFilesList, errors
}

// printUsage prints the usage information
func printUsage() {
	fmt.Println("Universal Image to JXL Converter")
	fmt.Println("Converts JPEG, PNG, GIF, BMP, and other image formats to JXL format with optimal settings for each type.")
	fmt.Println("Usage: all2jxl [OPTIONS] <input_directory> [output_directory]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -q, --quality FLOAT      Set quality (distance) value (default 0.0 for lossless)")
	fmt.Println("  -e, --effort INT         Set encoder effort (1-10, default 7)")
	fmt.Println("  -t, --threads INT        Number of concurrent threads (default: conservative to prevent overload)")
	fmt.Println("  --lossless-jpeg BOOL     Whether to use lossless JPEG transcoding (default true)")
	fmt.Println("  --strict-dynamic         Only convert files that are actually animated (default: false)")
	fmt.Println("  --mode STRING            Processing mode: auto, jpeg, dynamic, lossless (default: auto)")
	fmt.Println("  --replace                Replace original image files with JXL files (default: false)")
	fmt.Println("  --skip-corrupted         Skip files that cause errors during conversion (default: false)")
	fmt.Println("  --verify                 Verify output files after conversion (default: false)")
	fmt.Println("  -v, --verbose            Enable verbose output")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  all2jxl /path/to/image/dir /path/to/jxl/output")
	fmt.Println("  all2jxl --replace --verify -e 9 -t 2 /path/to/image/dir")
	fmt.Println("  all2jxl --skip-corrupted --verify --mode jpeg /path/to/image/dir /path/to/output")
	fmt.Println("  all2jxl --replace --mode lossless /path/to/image/dir")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse command line arguments
	config := Config{
		Quality:       0.0, // Default to lossless
		Effort:        7,
		NumThreads:    0, // Default to conservative threading
		Verbose:       false,
		LosslessJPEG:  true,
		StrictDynamic: false,
		Mode:          "auto",
		ReplaceOriginal: false,
		SkipCorrupted:   false,
		Verify:          false,
	}

	// Find input and output directories
	var inputDir, outputDir string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		switch arg {
		case "-q", "--quality":
			if i+1 < len(os.Args) {
				quality, err := strconv.ParseFloat(os.Args[i+1], 64)
				if err != nil {
					fmt.Println("Error: -q/--quality requires a numeric value")
					os.Exit(1)
				}
				config.Quality = quality
				i++
			} else {
				fmt.Println("Error: -q/--quality requires a value")
				os.Exit(1)
			}
		case "-e", "--effort":
			if i+1 < len(os.Args) {
				effort, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					fmt.Println("Error: -e/--effort requires an integer value")
					os.Exit(1)
				}
				config.Effort = effort
				i++
			} else {
				fmt.Println("Error: -e/--effort requires a value")
				os.Exit(1)
			}
		case "-t", "--threads":
			if i+1 < len(os.Args) {
				threads, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					fmt.Println("Error: -t/--threads requires a value")
					os.Exit(1)
				}
				config.NumThreads = threads
				i++
			} else {
				fmt.Println("Error: -t/--threads requires a value")
				os.Exit(1)
			}
		case "--lossless-jpeg":
			if i+1 < len(os.Args) {
				val := strings.ToLower(os.Args[i+1])
				config.LosslessJPEG = val == "true" || val == "1" || val == "yes"
				i++
			} else {
				fmt.Println("Error: --lossless-jpeg requires a value (true/false, 1/0, or yes/no)")
				os.Exit(1)
			}
		case "--strict-dynamic":
			config.StrictDynamic = true
		case "--mode":
			if i+1 < len(os.Args) {
				mode := strings.ToLower(os.Args[i+1])
				if mode == "auto" || mode == "jpeg" || mode == "dynamic" || mode == "lossless" {
					config.Mode = mode
				} else {
					fmt.Println("Error: --mode requires one of: auto, jpeg, dynamic, lossless")
					os.Exit(1)
				}
				i++
			} else {
				fmt.Println("Error: --mode requires a value")
				os.Exit(1)
			}
		case "--replace":
			config.ReplaceOriginal = true
		case "--skip-corrupted":
			config.SkipCorrupted = true
		case "--verify":
			config.Verify = true
		case "-v", "--verbose":
			config.Verbose = true
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			if inputDir == "" {
				inputDir = arg
			} else if outputDir == "" && !config.ReplaceOriginal {
				outputDir = arg
			}
		}
	}

	// Validate input directory
	if inputDir == "" {
		fmt.Println("Error: Input directory must be specified")
		printUsage()
		os.Exit(1)
	}

	// If not replacing originals, output directory is required
	if !config.ReplaceOriginal && outputDir == "" {
		fmt.Println("Error: Output directory must be specified when not using --replace")
		printUsage()
		os.Exit(1)
	}

	// Check if input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Printf("Error: Input directory does not exist: %s\n", inputDir)
		os.Exit(1)
	}

	// If not replacing originals, create output directory if it doesn't exist
	if !config.ReplaceOriginal {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error: Failed to create output directory: %v\n", err)
			os.Exit(1)
		}
		config.OutputDir = outputDir
	} else {
		config.OutputDir = inputDir // Use input dir as "output" for in-place replacement
	}

	config.InputDir = inputDir

	if config.Verbose {
		fmt.Printf("Input directory: %s\n", config.InputDir)
		if !config.ReplaceOriginal {
			fmt.Printf("Output directory: %s\n", config.OutputDir)
		} else {
			fmt.Println("Mode: In-place replacement (original images will be replaced with JXLs)")
		}
		fmt.Printf("Quality: %.2f\n", config.Quality)
		fmt.Printf("Effort: %d\n", config.Effort)
		fmt.Printf("Lossless JPEG: %t\n", config.LosslessJPEG)
		fmt.Printf("Strict Dynamic Check: %t\n", config.StrictDynamic)
		fmt.Printf("Mode: %s\n", config.Mode)
		fmt.Printf("Threads: %d\n", config.NumThreads)
		fmt.Printf("Skip Corrupted: %t\n", config.SkipCorrupted)
		fmt.Printf("Verify Output: %t\n", config.Verify)
		fmt.Println()
	}

	fmt.Println("Searching for image files...")
	fileTypes, err := findFiles(config.InputDir, config)
	if err != nil {
		fmt.Printf("Error finding image files: %v\n", err)
		os.Exit(1)
	}

	// Count total files
	totalFiles := 0
	for _, files := range fileTypes {
		totalFiles += len(files)
	}

	if totalFiles == 0 {
		fmt.Println("No supported image files found in the input directory.")
		os.Exit(0)
	}

	// Count file types
	jpegCount := len(fileTypes["jpeg"])
	losslessCount := len(fileTypes["lossless"])
	dynamicCount := len(fileTypes["dynamic"])
	
	fmt.Printf("Found %d image files to convert (%d JPEG, %d lossless/HEIC, %d dynamic).\n", 
		totalFiles, jpegCount, losslessCount, dynamicCount)
	fmt.Println("Starting conversion...")

	corruptedFiles, errors := processFiles(fileTypes, config)

	if len(corruptedFiles) > 0 {
		fmt.Printf("\nFound %d corrupted/skipped files:\n", len(corruptedFiles))
		for _, file := range corruptedFiles {
			fmt.Printf("  - %s\n", file)
		}
	}

	if len(errors) > 0 {
		fmt.Printf("\nConversion completed with %d error(s).\n", len(errors))
		if config.Verbose {
			for _, err := range errors {
				fmt.Printf("Error: %v\n", err)
			}
		}
	} else {
		fmt.Println("\nAll image files converted successfully!")
	}

	if len(corruptedFiles) == 0 && len(errors) == 0 {
		fmt.Println("All operations completed successfully!")
	}
}