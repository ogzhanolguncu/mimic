package fileops

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrRead      = errors.New("file_ops: failed to read a file")
	ErrWrite     = errors.New("file_ops: failed to write a file")
	ErrMkDir     = errors.New("file_ops: failed to make a dir")
	ErrRemoveDir = errors.New("file_ops: failed to remove a dir")
)

func CopyFile(path string) (bool, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrRead, err)
	}

	if err := os.WriteFile(path, file, 0644); err != nil {
		return false, fmt.Errorf("%w: %v", ErrWrite, err)
	}
	return true, nil
}

func CreateDir(name string) (bool, error) {
	if err := os.Mkdir(name, 0755); err != nil {
		return false, fmt.Errorf("%w: %v", ErrMkDir, err)
	}
	return true, nil
}

func DeletePath(name string) (bool, error) {
	if err := os.Remove(name); err != nil {
		return false, fmt.Errorf("%w: %v", ErrRemoveDir, err)
	}
	return true, nil
}
