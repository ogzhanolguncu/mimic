package syncer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

type EntryInfo struct {
	RelativePath string      // Path relative to the root sync directory.
	Mtime        time.Time   // Last modification timestamp.
	Size         int64       // File size in bytes (0 for directories).
	IsDir        bool        // True if this entry is a directory.
	Checksum     string      // Hash of file contents (empty for directories).
	Permissions  os.FileMode // Full file mode bits (type + permissions).
	// Add IsSymlink/LinkTarget fields here when symlink support is added
}

var (
	ErrRead     = errors.New("syncer: read error")
	ErrNotExist = errors.New("syncer: path does not exist")
	ErrNoDir    = errors.New("syncer: path is not a directory")
	ErrChecksum = errors.New("syncer: checksum calculation failed")
)

var Logger *slog.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func SetLogger(l *slog.Logger) {
	if l != nil {
		Logger = l
	}
}

// ScanSource scans the root directory recursively and returns a map of all entries
// keyed by their relative path, containing their metadata.
// It skips the root directory itself and .DS_Store files.
// Errors during scanning of individual files (e.g., checksum failure) are logged,
// and the file is skipped, allowing the scan to continue. More critical errors
// (e.g., cannot read root directory, permission denied on subdirectory traversal)
// will halt the scan and return an error.
func ScanSource(rootDir string) (map[string]*EntryInfo, error) {
	op := "ScanSource"
	Logger.Debug("starting scan", "operation", op, "dir", rootDir)

	fileInfo, err := exists(rootDir)
	if err != nil {
		return nil, fmt.Errorf("%s: initial check failed for %q: %w", op, rootDir, err)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s: root path %q is not a directory: %w", op, rootDir, ErrNoDir)
	}

	entries := make(map[string]*EntryInfo)

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErrIn error) error {
		if walkErrIn != nil {
			if errors.Is(walkErrIn, fs.ErrPermission) {
				Logger.Warn("permission denied during scan, skipping", "path", path, "error", walkErrIn)
				return fs.SkipDir
			}
			Logger.Error("access error during scan", "path", path, "error", walkErrIn)
			return walkErrIn // Halt the walk for other errors
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("calculate relative path for %q: %w", path, err) // Halt the walk
		}

		// TODO: Later pass config exclude path here
		if relPath == "." || shouldExclude(relPath, []string{".DS_Store"}) {
			Logger.Debug("skipping entry", "path", relPath)
			return nil // Continue walking
		}

		info, err := d.Info()
		if err != nil {
			Logger.Error("cannot get file info, skipping entry", "path", path, "error", err)
			return nil
		}

		isDir := d.IsDir()
		entry := &EntryInfo{
			RelativePath: relPath,
			Mtime:        info.ModTime(),
			Size:         info.Size(), // Size is 0 or irrelevant for dirs, but store anyway
			IsDir:        isDir,
			Permissions:  info.Mode(), // Store the full FileMode
			Checksum:     "",
		}

		if !isDir {
			checksumBytes, csErr := generateChecksum(path)
			if csErr != nil {
				Logger.Warn("checksum failed, skipping file", "path", path, "error", csErr)
				return nil
			}
			entry.Checksum = hex.EncodeToString(checksumBytes)
		}

		entries[relPath] = entry
		Logger.Debug("scanned entry", "path", relPath, "isDir", isDir)
		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("%s: directory walk failed for %q: %w", op, rootDir, walkErr)
	}

	Logger.Info("scan finished successfully", "operation", op, "dir", rootDir, "entries_found", len(entries))
	return entries, nil
}

// exists checks if a path exists and returns its FileInfo.
// Returns wrapped ErrNotExist or ErrRead on failure.
func exists(path string) (os.FileInfo, error) {
	op := "exists"
	fileInfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", op, ErrNotExist)
		}
		return nil, fmt.Errorf("%s: check failed for %q: %w", op, path, ErrRead)
	}
	return fileInfo, nil
}

// generateChecksum calculates the xxHash checksum for a given file path.
// Returns wrapped ErrRead or ErrChecksum on failure.
func generateChecksum(filePath string) ([]byte, error) {
	op := "generateChecksum"
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s: open failed for %q: %w", op, filePath, ErrRead)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("error closing file: %v", err)
		}
	}()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("%s: copy failed for %q: %w", op, filePath, ErrChecksum)
	}
	return hash.Sum(nil), nil
}

func shouldExclude(relPath string, matchers []string) bool {
	baseName := filepath.Base(relPath)
	for _, pattern := range matchers {
		if strings.HasSuffix(pattern, "/") { // Treat as directory prefix/exact match
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Check if path is exactly this directory or inside it
			if relPath == dirPattern || strings.HasPrefix(relPath, dirPattern+"/") {
				return true
			}
		} else {
			// Use filepath.Match for glob patterns against the base name
			matched, _ := filepath.Match(pattern, baseName)
			if matched {
				return true
			}
			// Also handle exact matches for the whole path
			if pattern == relPath {
				return true
			}
		}
	}
	return false
}
