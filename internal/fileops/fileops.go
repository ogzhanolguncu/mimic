package fileops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ogzhanolguncu/mimic/internal/logger"
)

var (
	ErrRead       = errors.New("file_ops: failed to read a file")
	ErrWrite      = errors.New("file_ops: failed to write a file")
	ErrMkDir      = errors.New("file_ops: failed to make a dir")
	ErrRemoveDir  = errors.New("file_ops: failed to remove a dir")
	ErrPathExists = errors.New("file_ops: path already exists")
	ErrStat       = errors.New("file_ops: failed to stat path")
	ErrBatchRead  = errors.New("file_ops: failed to batch read")
	ErrBatchWrite = errors.New("file_ops: failed to batch write")
)

// CopyFile copies a file from readPath to writePath, preserving permissions
func CopyFile(readPath, writePath string, chunkSize int64) (bool, error) {
	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(readPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStat, err)
	}
	if srcInfo.Size() >= chunkSize {
		logger.Debug("Running batched copy", "file", srcInfo.Name(), "size", srcInfo.Size())
		return copyFileBatching(readPath, writePath, chunkSize)
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
	logger.Debug("File copied successfully", "source", readPath, "destination", writePath, "size", srcInfo.Size())
	return true, nil
}

func copyFileBatching(readPath, writePath string, chunkSize int64) (bool, error) {
	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(readPath)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStat, err)
	}
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(writePath), 0755); err != nil {
		return false, fmt.Errorf("%w: %v", ErrMkDir, err)
	}

	logger.Debug("Starting batch file copy", "source", readPath, "destination", writePath, "size", srcInfo.Size())

	transport := make(chan []byte, 5)
	srcFile, err := os.Open(readPath)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(writePath, os.O_CREATE|os.O_WRONLY, srcInfo.Mode())
	if err != nil {
		return false, fmt.Errorf("failed to open destination file %w", err)
	}
	defer dstFile.Close()

	var readerDone sync.WaitGroup
	readerDone.Add(1)
	errChan := make(chan error, 1)

	logger.Debug("Starting file reader goroutine", "path", readPath)
	go func() {
		defer readerDone.Done()
		defer close(transport)
		buf := make([]byte, chunkSize)
		totalBytesRead := int64(0)

		for {
			n, err := srcFile.Read(buf)
			if n > 0 {
				totalBytesRead += int64(n)
				bufCopy := make([]byte, n)
				copy(bufCopy, buf[:n])
				transport <- bufCopy

				if totalBytesRead%(chunkSize*10) == 0 {
					logger.Debug("Reading progress", "path", readPath, "bytesRead", totalBytesRead, "percentage", float64(totalBytesRead)/float64(srcInfo.Size())*100)
				}
			}
			if err != nil {
				if err == io.EOF {
					logger.Debug("Reached end of file", "path", readPath, "totalBytesRead", totalBytesRead)
					break
				}
				logger.Error("Error reading file", "path", readPath, "error", err)
				select {
				case errChan <- fmt.Errorf("%w: %v", ErrRead, err):
				default:
				}
				return
			}
		}
	}()

	totalBytesWritten := int64(0)
	for data := range transport {
		n, err := dstFile.Write(data)
		if err != nil {
			logger.Error("Error writing to file", "path", writePath, "error", err)
			return false, fmt.Errorf("%w: %v", ErrBatchWrite, err)
		}
		totalBytesWritten += int64(n)

		if totalBytesWritten%(chunkSize*10) == 0 {
			logger.Debug("Writing progress", "path", writePath, "bytesWritten", totalBytesWritten, "percentage", float64(totalBytesWritten)/float64(srcInfo.Size())*100)
		}
	}

	readerDone.Wait()

	select {
	case err := <-errChan:
		return false, fmt.Errorf("%w: %v", ErrBatchRead, err)
	default:
		logger.Debug("Batch file copy completed", "source", readPath, "destination", writePath, "size", totalBytesWritten)
	}

	return true, nil
}

// CreateDir creates a directory and all necessary parent directories
func CreateDir(name string) (bool, error) {
	if err := os.MkdirAll(name, 0755); err != nil {
		logger.Error("Failed to create directory", "path", name, "error", err)
		return false, fmt.Errorf("%w: %v", ErrMkDir, err)
	}
	logger.Debug("Directory created", "path", name)
	return true, nil
}

// DeletePath removes a file or directory and its contents
func DeletePath(name string) (bool, error) {
	fileInfo, err := os.Stat(name)
	if err == nil {
		isDir := fileInfo.IsDir()
		logger.Debug("Removing path", "path", name, "isDirectory", isDir)
	}

	if err := os.RemoveAll(name); err != nil {
		logger.Error("Failed to remove path", "path", name, "error", err)
		return false, fmt.Errorf("%w: %v", ErrRemoveDir, err)
	}

	logger.Debug("Path removed successfully", "path", name)
	return true, nil
}

// PathExists checks if a path exists
func PathExists(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		isDir := fileInfo.IsDir()
		size := int64(0)
		if !isDir {
			size = fileInfo.Size()
		}
		logger.Debug("Path exists", "path", path, "isDirectory", isDir, "size", size)
		return true, nil
	}

	if os.IsNotExist(err) {
		logger.Debug("Path does not exist", "path", path)
		return false, nil
	}

	logger.Error("Error checking if path exists", "path", path, "error", err)
	return false, fmt.Errorf("%w: %v", ErrStat, err)
}
