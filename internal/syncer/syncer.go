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
	"github.com/ogzhanolguncu/mimic/internal/fileops"
)

const (
	ActionNone   = 0x00
	ActionCreate = 0x01
	ActionUpdate = 0x02
	ActionDelete = 0x03
)

type SyncAction struct {
	Type         int
	RelativePath string
	SourceInfo   EntryInfo
}

type EntryInfo struct {
	RelativePath string      // Path relative to the root sync directory.
	Mtime        time.Time   // Last modification timestamp.
	Size         int64       // File size in bytes (0 for directories).
	IsDir        bool        // True if this entry is a directory.
	Checksum     string      // Hash of file contents (empty for directories).
	Permissions  os.FileMode // Full file mode bits (type + permissions).
}

var (
	ErrSyncerRead          = errors.New("syncer: read error")
	ErrSyncerNotExist      = errors.New("syncer: path does not exist")
	ErrSyncerNoDir         = errors.New("syncer: path is not a dir")
	ErrSyncerSrcNotExists  = errors.New("syncer: src dir does not exist")
	ErrSyncerChecksum      = errors.New("syncer: checksum calculation failed")
	ErrEmptySrcDir         = errors.New("syncer: src dir is empty")
	ErrEmptySrcNotADir     = errors.New("syncer: src is not a dir")
	ErrSyncerFaultyRelPath = errors.New("syncer: rel path cannot be calculated")
	ErrSyncerDirWalk       = errors.New("syncer: dir walk failed")
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
func ScanSource(rootDir string) (map[string]EntryInfo, error) {
	op := "ScanSource"
	Logger.Debug("starting scan", "operation", op, "dir", rootDir)

	if rootDir == "" {
		return nil, ErrEmptySrcDir
	}
	rootDir = filepath.Clean(rootDir)

	fileInfo, err := retryableOpWithResult("exists", rootDir, func() (os.FileInfo, error) {
		return exists(rootDir)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncerSrcNotExists, err)
	}
	if !fileInfo.IsDir() {
		return nil, ErrEmptySrcNotADir
	}

	entries := make(map[string]EntryInfo)

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErrIn error) error {
		if walkErrIn != nil {
			if errors.Is(walkErrIn, fs.ErrPermission) {
				Logger.Warn("permission denied during scan, skipping", "path", path, "error", walkErrIn)
				return fs.SkipDir
			}
			Logger.Error("access error during scan", "path", path, "error", walkErrIn)
			return walkErrIn // Halt the walk for other errors
		}

		relPath, err := retryableOpWithResult("rel_path", rootDir, func() (string, error) {
			return filepath.Rel(rootDir, path)
		})
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSyncerFaultyRelPath, err) // Halt the walk
		}
		relPath = filepath.Clean(relPath)

		// TODO: Later pass config exclude path here
		if relPath == "." || shouldExclude(relPath, []string{".DS_Store"}) {
			Logger.Debug("skipping entry", "path", relPath)
			return nil // Continue walking
		}

		info, err := retryableOpWithResult("file_info", rootDir, func() (fs.FileInfo, error) {
			return d.Info()
		})
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				Logger.Warn("file disappeared after detection, skipping entry", "path", path)
				return nil
			}
			Logger.Error("cannot get file info, skipping entry", "path", path, "error", err)
			return nil
		}

		isDir := d.IsDir()
		entry := EntryInfo{
			RelativePath: relPath,
			Mtime:        info.ModTime(),
			Size:         info.Size(), // Size is 0 or irrelevant for dirs, but store anyway
			IsDir:        isDir,
			Permissions:  info.Mode(), // Store the full FileMode
			Checksum:     "",
		}

		if !isDir {
			checksumBytes, csErr := retryableOpWithResult("checksum", rootDir, func() ([]byte, error) {
				return generateChecksum(path)
			})
			if csErr != nil {
				if errors.Is(csErr, ErrSyncerNotExist) {
					Logger.Warn("file disappeared before checksum, skipping entry", "path", path)
					return nil
				}
				Logger.Warn("checksum failed, skipping file", "path", path, "error", csErr)
			}
			entry.Checksum = hex.EncodeToString(checksumBytes)
		}

		entries[relPath] = entry
		Logger.Debug("scanned entry", "path", relPath, "isDir", isDir)
		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncerDirWalk, walkErr)
	}

	Logger.Info("scan finished successfully", "operation", op, "dir", rootDir, "entries_found", len(entries))
	return entries, nil
}

// exists checks if a path exists and returns its FileInfo.
func exists(path string) (os.FileInfo, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrSyncerNotExist
		}
		return nil, ErrSyncerRead
	}
	return fileInfo, nil
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

// generateChecksum calculates the xxHash checksum for a given file path.
// Returns wrapped ErrRead or ErrChecksum on failure.
func generateChecksum(filePath string) ([]byte, error) {
	initialInfo, err := exists(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncerSrcNotExists, err)
	}

	initialMtime := initialInfo.ModTime()
	initialSize := initialInfo.Size()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, ErrSyncerRead
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("error closing file: %v", err)
		}
	}()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, ErrSyncerChecksum
	}

	currentInfo, err := exists(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncerSrcNotExists, err)
	} else if currentInfo.ModTime() != initialMtime || currentInfo.Size() != initialSize {
		// File changed during scan
		Logger.Warn("file modified during checksum calculation",
			"path", filePath,
			"initial_mtime", initialMtime,
			"current_mtime", currentInfo.ModTime())
		return nil, ErrSyncerChecksum
		//  Mark the file with a special flag in its entry (better approach)
		// Return the checksum anyway, and handle in the caller with a flag
	}

	return hash.Sum(nil), nil
}

const maxRetries = 5

// Generic retryable operation that returns a value and an error
func retryableOpWithResult[T any](operation string, path string, op func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		r, err := op()
		if err == nil {
			return r, nil
		}

		// If the file disappeared, no point retrying
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrSyncerNotExist) {
			return result, err
		}

		lastErr = err
		Logger.Warn("operation failed, retrying",
			"operation", operation,
			"path", path,
			"attempt", attempt+1,
			"error", err)

		time.Sleep(time.Millisecond * 10 * time.Duration(attempt+1))
	}

	return result, lastErr
}

// ------- SYNC ACTIONS -------

func CompareStates(sourceScan, loadedStateEntries map[string]EntryInfo) []SyncAction {
	var syncActions []SyncAction
	const timeDiffThreshold = 1 * time.Second

	// Process source entries (creates and updates)
	for path, source := range sourceScan {
		entry, found := loadedStateEntries[path]

		if !found {
			// New file - create action
			syncActions = append(syncActions, SyncAction{
				Type: ActionCreate, RelativePath: path, SourceInfo: source,
			})
			continue
		}

		// Check if file is unchanged
		timeDiff := source.Mtime.Sub(entry.Mtime)
		sameTime := timeDiff < timeDiffThreshold && timeDiff > -timeDiffThreshold
		sameSize := source.Size == entry.Size

		if sameTime && sameSize {
			syncActions = append(syncActions, SyncAction{
				Type: ActionNone, RelativePath: path, SourceInfo: EntryInfo{},
			})
		} else {
			syncActions = append(syncActions, SyncAction{
				Type: ActionUpdate, RelativePath: path, SourceInfo: source,
			})
		}
	}

	// Process loaded entries (deletes)
	for path := range loadedStateEntries {
		if _, exists := sourceScan[path]; !exists {
			syncActions = append(syncActions, SyncAction{
				Type: ActionDelete, RelativePath: path, SourceInfo: EntryInfo{},
			})
		}
	}

	return syncActions
}

func ExecuteActions(srcRoot, dstRoot string, actions []SyncAction) error {
	for _, action := range actions {
		readPath := filepath.Join(srcRoot, action.RelativePath)
		writePath := filepath.Join(dstRoot, action.RelativePath)

		switch action.Type {
		case ActionNone:
			continue
		case ActionCreate:
			isDir := action.SourceInfo.IsDir
			if isDir {
				_, err := fileops.CreateDir(writePath)
				if err != nil {
					return err
				}
			} else {
				_, err := fileops.CopyFile(readPath, writePath)
				if err != nil {
					return err
				}
			}
		case ActionDelete:
			_, err := fileops.DeletePath(writePath)
			if err != nil {
				return err
			}
		case ActionUpdate:
			_, err := fileops.CopyFile(readPath, writePath)
			if err != nil {
				return err
			}
		default:
			Logger.Error("unknown action",
				"action", action.Type)

		}

	}
	return nil
}
