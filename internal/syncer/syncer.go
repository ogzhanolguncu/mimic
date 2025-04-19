package syncer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"
)

type EntryInfo struct {
	RelativePath string      // Path relative to the root sync directory.
	Mtime        time.Time   // Last modification timestamp.
	Size         int64       // File size in bytes
	IsDir        bool        // True if this entry is a directory.
	Checksum     string      // Hash of file contents
	Permissions  os.FileMode // File mode bits
}

var (
	ErrRead     = errors.New("syncer: error reading a file or directory")
	ErrNotExist = errors.New("syncer: file directory does not exist")
	ErrNoDir    = errors.New("syncer: path is not a directory")
	ErrChecksum = errors.New("syncer: failed to generate checksum")
)

var Logger *slog.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func SetLogger(l *slog.Logger) {
	if l != nil {
		Logger = l
	}
}

// ScanSource scans the root directory and returns a map of all entries with their information
func ScanSource(rootDir string) (map[string]*EntryInfo, error) {
	Logger.Debug("starting scan of source directory", "dir", rootDir)

	// Check if directory exists
	fileInfo, err := exists(rootDir)
	if err != nil {
		Logger.Error("root directory does not exist or is not accessible",
			"dir", rootDir,
			"error", err)
		return nil, err
	}

	if !fileInfo.IsDir() {
		Logger.Error("path is not a directory", "path", rootDir)
		return nil, ErrNoDir
	}

	entries := make(map[string]*EntryInfo)

	filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			Logger.Error("could not read",
				"entry", path,
				"error", err)
			return err
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			Logger.Error("could not find relative path",
				"entry", path,
				"error", err)
			return err
		}

		hash, err := generateChecksum(path)
		if err != nil {
			return err
		}

		entries[path] = &EntryInfo{
			Mtime:        info.ModTime(),
			Size:         info.Size(),
			IsDir:        false,
			Permissions:  info.Mode().Perm(),
			RelativePath: relPath,
			Checksum:     hex.EncodeToString(hash),
		}
		return nil
	})

	return entries, nil
}

func exists(path string) (os.FileInfo, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			Logger.Debug("path does not exist", "path", path)
			return nil, ErrNotExist
		}
		Logger.Error("error reading path", "path", path, "error", err)
		return nil, ErrRead
	}
	return fileInfo, nil
}

func generateChecksum(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		Logger.Error("failed to open file for checksum", "path", filePath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrRead, err)
	}
	defer file.Close()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		Logger.Error("failed to calculate checksum", "path", filePath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrChecksum, err)
	}

	Logger.Debug("checksum generated successfully", "path", filePath)
	return hash.Sum(nil), nil
}
