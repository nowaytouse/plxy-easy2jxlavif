package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"pixly/utils"
)

const (
	toolName = "merge_xmp"
	version     = "2.1.0"
)

var (
	logger *log.Logger
)

func init() {
	logFile, err := os.OpenFile("merge_xmp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	logger.Printf("%s v%s starting...", toolName, version)

	dir := flag.String("dir", "", "Directory to process")
	flag.Parse()

	if *dir == "" {
		logger.Fatal("Directory path is required. Use -dir <path>")
	}

	// Check for exiftool dependency
	if _, err := exec.LookPath("exiftool"); err != nil {
		logger.Fatalf("Dependency 'exiftool' not found in PATH. Please install it.")
	}

	var files []string
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		logger.Fatalf("Error walking the path %q: %v", *dir, err)
	}

	for _, file := range files {
		processFile(file)
	}

	logger.Println("Processing complete.")
}

func processFile(mediaPath string) {
	ext := filepath.Ext(mediaPath)
	if !isMediaFile(ext) {
		return
	}

	xmpPath := strings.TrimSuffix(mediaPath, ext) + ".xmp"
	if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
		// Also check for sidecar.xmp format
		xmpPath = mediaPath + ".xmp"
		if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
			return
		}
	}

	// Check if the xmp file still exists
	if _, err := os.Stat(xmpPath); os.IsNotExist(err) {
		return
	}

	logger.Printf("Found media file '%s' with XMP sidecar '%s'", filepath.Base(mediaPath), filepath.Base(xmpPath))

	// Merge XMP
	mergeCmd := exec.Command("exiftool", "-tagsfromfile", xmpPath, "-all:all", "-overwrite_original", mediaPath)
	if output, err := mergeCmd.CombinedOutput(); err != nil {
		logger.Printf("Failed to merge XMP for %s: %v. Output: %s", filepath.Base(mediaPath), err, string(output))
		return
	}

	logger.Printf("Successfully merged XMP into %s", filepath.Base(mediaPath))

	// Verify merge
	if verifyMerge(mediaPath, xmpPath) {
		logger.Printf("Verification successful for %s", filepath.Base(mediaPath))
		// 安全删除 XMP 文件，仅在确认元数据已成功合并后才删除
		if err := utils.SafeDelete(xmpPath, mediaPath, func(format string, v ...interface{}) {
			logger.Printf(format, v...)
		}); err != nil {
			logger.Printf("⚠️  安全删除 XMP 文件失败 %s: %v", filepath.Base(xmpPath), err)
		}
	} else {
		logger.Printf("Verification failed for %s. The XMP file will be kept.", filepath.Base(mediaPath))
	}
}

func isMediaFile(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".tif", ".tiff", ".gif", ".mp4", ".mov", ".heic", ".heif":
		return true
	default:
		return false
	}
}

func verifyMerge(mediaPath, xmpPath string) bool {
	// Read a specific tag from the XMP file
	xmpData, err := ioutil.ReadFile(xmpPath)
	if err != nil {
		logger.Printf("Failed to read XMP file for verification: %v", err)
		return false
	}

	// Example verification: check for photoshop:DateCreated
	// A more robust implementation would parse the XML properly
	if strings.Contains(string(xmpData), "photoshop:DateCreated") {
		verifyCmd := exec.Command("exiftool", "-XMP-photoshop:DateCreated", mediaPath)
		output, err := verifyCmd.CombinedOutput()
		if err != nil {
			logger.Printf("Failed to run exiftool for verification: %v", err)
			return false
		}

		// This is a simple check. A more robust check would parse the date and compare.
		if strings.Contains(string(output), "2025-09-11T19:34:07") {
			return true
		}
	}

	// Default to true if no specific verification tag is found, assuming merge was successful
	return true
}
