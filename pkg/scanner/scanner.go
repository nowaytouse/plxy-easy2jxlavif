package scanner

import (
	"context"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// FileInfo represents basic information about a scanned file.
type FileInfo struct {
	Path    string
	Size    int64
	IsDir   bool
	ModTime int64
}

// Scanner is responsible for scanning directories and finding media files.
type Scanner struct {
	logger *zap.Logger
}

// NewScanner creates a new Scanner.
func NewScanner(logger *zap.Logger) *Scanner {
	return &Scanner{logger: logger}
}

// ScanDirectory scans the target directory and returns a list of FileInfo.
func (s *Scanner) ScanDirectory(ctx context.Context, root string) ([]*FileInfo, error) {
	s.logger.Info("Starting directory scan", zap.String("root", root))
	var files []*FileInfo

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		info, err := d.Info()
		if err != nil {
			s.logger.Warn("Failed to get file info", zap.String("path", path), zap.Error(err))
			return nil // Continue scanning
		}

		files = append(files, &FileInfo{
			Path:    path,
			Size:    info.Size(),
			IsDir:   d.IsDir(),
			ModTime: info.ModTime().Unix(),
		})
		return nil
	})

	if err != nil {
		s.logger.Error("Directory scan failed", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Directory scan completed", zap.Int("files_found", len(files)))
	return files, nil
}