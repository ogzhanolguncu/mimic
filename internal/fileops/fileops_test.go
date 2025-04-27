package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ogzhanolguncu/mimic/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "fileops_test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Test case 1: Copy a small file successfully
	testContent := []byte("This is test content")
	sourcePath := filepath.Join(tempDir, "source.txt")
	destPath := filepath.Join(tempDir, "destination.txt")

	// Create source file
	err = os.WriteFile(sourcePath, testContent, 0644)
	require.NoError(t, err, "Failed to create source file")

	// Copy file
	success, err := CopyFile(sourcePath, destPath, config.DefaultChunkSize)
	require.NoError(t, err, "CopyFile should not return error")
	require.True(t, success, "CopyFile should return success")

	// Verify content is the same
	destContent, err := os.ReadFile(destPath)
	require.NoError(t, err, "Failed to read destination file")
	require.Equal(t, testContent, destContent, "File content should be identical")

	// Test case 2: Copy to a destination in a non-existent directory
	nestedDestPath := filepath.Join(tempDir, "subdir", "nested", "destination.txt")
	success, err = CopyFile(sourcePath, nestedDestPath, config.DefaultChunkSize)
	require.NoError(t, err, "CopyFile should create parent directories")
	require.True(t, success, "CopyFile should return success")

	// Verify content
	destContent, err = os.ReadFile(nestedDestPath)
	require.NoError(t, err, "Failed to read destination file")
	require.Equal(t, testContent, destContent, "File content should be identical")

	// Test case 3: Source file doesn't exist
	nonExistPath := filepath.Join(tempDir, "nonexistent.txt")
	success, err = CopyFile(nonExistPath, destPath, config.DefaultChunkSize)
	require.Error(t, err, "CopyFile should return error for non-existent source")
	require.False(t, success, "CopyFile should not return success")
}

func TestLargeFileCopy(t *testing.T) {
	// Skip this test if in short mode
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "fileops_large_test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a large file (slightly larger than chunkSize)
	largeSize := config.DefaultChunkSize + 1024
	sourcePath := filepath.Join(tempDir, "large_source.bin")
	destPath := filepath.Join(tempDir, "large_dest.bin")

	// Create and prepare test data - using a pattern rather than random data
	file, err := os.Create(sourcePath)
	require.NoError(t, err, "Failed to create large source file")

	// Write a repeating pattern to fill the file
	chunk := make([]byte, 1024)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	remaining := largeSize
	for remaining > 0 {
		writeSize := min(remaining, len(chunk))
		_, err = file.Write(chunk[:writeSize])
		require.NoError(t, err, "Failed to write to large source file")
		remaining -= writeSize
	}
	file.Close()

	// Verify the file size
	info, err := os.Stat(sourcePath)
	require.NoError(t, err, "Failed to stat large file")
	require.GreaterOrEqual(t, info.Size(), int64(config.DefaultChunkSize), "Test file should be larger than chunk size")

	// Perform the copy
	success, err := CopyFile(sourcePath, destPath, config.DefaultChunkSize)
	require.NoError(t, err, "Failed to copy large file")
	require.True(t, success, "CopyFile should return success")

	// Verify destination file size matches source
	destInfo, err := os.Stat(destPath)
	require.NoError(t, err, "Failed to stat destination file")
	require.Equal(t, info.Size(), destInfo.Size(), "Destination file size should match source")

	// Compare file contents (using checksums or reading chunks)
	srcFile, err := os.Open(sourcePath)
	require.NoError(t, err, "Failed to open source file for verification")
	defer srcFile.Close()

	dstFile, err := os.Open(destPath)
	require.NoError(t, err, "Failed to open destination file for verification")
	defer dstFile.Close()

	// Compare files in chunks
	srcBuf := make([]byte, 8192)
	dstBuf := make([]byte, 8192)

	for {
		srcN, srcErr := srcFile.Read(srcBuf)
		dstN, dstErr := dstFile.Read(dstBuf)

		require.Equal(t, srcN, dstN, "Files should have the same content length")
		require.Equal(t, srcBuf[:srcN], dstBuf[:dstN], "File contents should match")

		if srcErr == nil && dstErr == nil {
			continue
		}

		// Both should reach EOF at the same time
		require.Equal(t, srcErr, dstErr, "Files should end at the same time")
		if srcErr != nil {
			break
		}
	}
}

func TestCreateDir(t *testing.T) {
	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "fileops_dir_test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a simple directory
	dirPath := filepath.Join(tempDir, "testdir")
	success, err := CreateDir(dirPath)
	require.NoError(t, err, "CreateDir should not return error")
	require.True(t, success, "CreateDir should return success")

	// Verify directory exists
	exists, err := PathExists(dirPath)
	require.NoError(t, err, "PathExists should not return error")
	require.True(t, exists, "Directory should exist")

	// Create nested directories
	nestedPath := filepath.Join(tempDir, "nested", "deeply", "path")
	success, err = CreateDir(nestedPath)
	require.NoError(t, err, "CreateDir should handle nested directories")
	require.True(t, success, "CreateDir should return success")

	// Verify nested directory exists
	exists, err = PathExists(nestedPath)
	require.NoError(t, err, "PathExists should not return error")
	require.True(t, exists, "Nested directory should exist")

	// Create a directory that already exists
	success, err = CreateDir(dirPath)
	require.NoError(t, err, "CreateDir should not error on existing directory")
	require.True(t, success, "CreateDir should return success for existing directory")
}

func TestDeletePath(t *testing.T) {
	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "fileops_delete_test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Test case 1: Delete a file
	filePath := filepath.Join(tempDir, "test_file.txt")
	err = os.WriteFile(filePath, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	success, err := DeletePath(filePath)
	require.NoError(t, err, "DeletePath should not return error")
	require.True(t, success, "DeletePath should return success")

	// Verify file doesn't exist
	exists, err := PathExists(filePath)
	require.NoError(t, err, "PathExists should not return error")
	require.False(t, exists, "File should be deleted")

	// Delete a directory with contents
	dirPath := filepath.Join(tempDir, "test_dir")
	nestedFilePath := filepath.Join(dirPath, "nested", "file.txt")

	// Create directory structure
	err = os.MkdirAll(filepath.Dir(nestedFilePath), 0755)
	require.NoError(t, err, "Failed to create directory structure")

	// Create nested file
	err = os.WriteFile(nestedFilePath, []byte("nested content"), 0644)
	require.NoError(t, err, "Failed to create nested file")

	// Delete directory
	success, err = DeletePath(dirPath)
	require.NoError(t, err, "DeletePath should not return error on directory")
	require.True(t, success, "DeletePath should return success")

	// Verify directory doesn't exist
	exists, err = PathExists(dirPath)
	require.NoError(t, err, "PathExists should not return error")
	require.False(t, exists, "Directory should be deleted")

	// Delete non-existent path
	nonExistPath := filepath.Join(tempDir, "nonexistent")
	success, err = DeletePath(nonExistPath)
	require.NoError(t, err, "DeletePath should not error on non-existent path")
	require.True(t, success, "DeletePath should return success for non-existent path")
}
