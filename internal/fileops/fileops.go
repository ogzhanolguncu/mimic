package fileops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrRead       = errors.New("file_ops: failed to read a file")
	ErrWrite      = errors.New("file_ops: failed to write a file")
	ErrMkDir      = errors.New("file_ops: failed to make a dir")
	ErrRemoveDir  = errors.New("file_ops: failed to remove a dir")
	ErrPathExists = errors.New("file_ops: path already exists")
	ErrStat       = errors.New("file_ops: failed to stat path")
)

// CopyFile copies a file from readPath to writePath, preserving permissions
func CopyFile(readPath, writePath string) (bool, error) {
	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(readPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStat, err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(writePath), 0755); err != nil {
		return false, fmt.Errorf("%w: %v", ErrMkDir, err)
	}

	// Read source file
	file, err := os.ReadFile(readPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrRead, err)
	}

	// Write to destination with original permissions
	if err := os.WriteFile(writePath, file, srcInfo.Mode()); err != nil {
		return false, fmt.Errorf("%w: %v", ErrWrite, err)
	}

	return true, nil
}

// CreateDir creates a directory and all necessary parent directories
func CreateDir(name string) (bool, error) {
	if err := os.MkdirAll(name, 0755); err != nil {
		return false, fmt.Errorf("%w: %v", ErrMkDir, err)
	}
	return true, nil
}

// DeletePath removes a file or directory and its contents
func DeletePath(name string) (bool, error) {
	if err := os.RemoveAll(name); err != nil {
		return false, fmt.Errorf("%w: %v", ErrRemoveDir, err)
	}
	return true, nil
}

// PathExists checks if a path exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("%w: %v", ErrStat, err)
}
