package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	toolName = "deduplicate_media"
	version     = "2.1.0"
)

var (
	logger *log.Logger
)

func init() {
	logFile, err := os.OpenFile("deduplicate_media.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	logger.Printf("%s v%s starting...", toolName, version)

	dir := flag.String("dir", "", "Directory to scan for duplicates")
	trashDir := flag.String("trash-dir", "", "Directory to move duplicates to")
	flag.Parse()

	if *dir == "" || *trashDir == "" {
		logger.Fatal("Both -dir and -trash-dir flags are required.")
	}

	if err := os.MkdirAll(*trashDir, 0755); err != nil {
		logger.Fatalf("Failed to create trash directory: %v", err)
	}

	// Create a readme file in the trash directory
	readmePath := filepath.Join(*trashDir, "_readme_about_this_folder.txt")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readmeContent := "This folder contains files that were identified as duplicates by the deduplicate_media script. You can review them and delete them permanently if you are sure they are not needed."
		if err := ioutil.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			logger.Printf("Failed to create readme file in trash directory: %v", err)
		}
	}

	files := findFiles(*dir)
	standardizeExtensions(files)

	// Re-read files after standardization
	files = findFiles(*dir)
	findAndMoveDuplicates(files, *trashDir)

	logger.Println("Deduplication process complete.")
}

func findFiles(dir string) []string {
	var fileList []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		logger.Printf("Error walking the path %q: %v", dir, err)
	}
	return fileList
}

func standardizeExtensions(files []string) {
	logger.Println("Standardizing extensions...")
	for _, path := range files {
		oldExt := filepath.Ext(path)
		newExt := strings.ToLower(oldExt)

		switch newExt {
		case ".jpeg":
			newExt = ".jpg"
		case ".tiff":
			newExt = ".tif"
		}

		if oldExt == newExt {
			continue
		}

		newPath := strings.TrimSuffix(path, oldExt) + newExt
		if err := os.Rename(path, newPath); err != nil {
			logger.Printf("Failed to rename %s to %s: %v", path, newPath, err)
		} else {
			logger.Printf("Renamed %s to %s", filepath.Base(path), filepath.Base(newPath))
		}
	}
}

func findAndMoveDuplicates(files []string, trashDir string) {
	logger.Println("Finding and moving duplicates...")
	hashes := make(map[string]string)

	for _, path := range files {
		if !isMediaFile(filepath.Ext(path)) {
			continue
		}

		hash, err := calculateHash(path)
		if err != nil {
			logger.Printf("Failed to calculate hash for %s: %v", path, err)
			continue
		}

		if originalPath, ok := hashes[hash]; ok {
			// Potential duplicate found, verify byte-by-byte
			logger.Printf("Potential duplicate found: %s and %s", originalPath, path)
			areIdentical, err := compareFiles(originalPath, path)
			if err != nil {
				logger.Printf("Failed to compare files: %v", err)
				continue
			}

			if areIdentical {
				logger.Printf("Files are identical. Moving %s to trash.", filepath.Base(path))
				moveToTrash(path, trashDir)
			} else {
				logger.Printf("Files have the same hash but are not identical. Keeping both.")
			}
		} else {
			hashes[hash] = path
		}
	}
}

func isMediaFile(ext string) bool {
	switch strings.ToLower(ext) {
	// Image formats
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tif", ".tiff", ".webp", ".heic", ".heif":
		return true
	// Video formats
	case ".mp4", ".mov", ".mkv", ".avi", ".webm", ".flv", ".wmv":
		return true
	default:
		return false
	}
}

func calculateHash(path string) (string, error) {
	file, err := os.Open(path)
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

func compareFiles(path1, path2 string) (bool, error) {
	file1, err := ioutil.ReadFile(path1)
	if err != nil {
		return false, err
	}
	file2, err := ioutil.ReadFile(path2)
	if err != nil {
		return false, err
	}

	if len(file1) != len(file2) {
		return false, nil
	}

	for i := range file1 {
		if file1[i] != file2[i] {
			return false, nil
		}
	}

	return true, nil
}

func moveToTrash(path, trashDir string) {
	destPath := filepath.Join(trashDir, filepath.Base(path))
	if err := os.Rename(path, destPath); err != nil {
		logger.Printf("Failed to move %s to %s: %v", path, destPath, err)
	}
}