package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	tempDir2, err := os.MkdirTemp("", "fileops-test2")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir2)

	t.Run("SuccessfulCopy", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "test.txt")
		testFileDest := filepath.Join(tempDir2, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Copy the file
		success, err := CopyFile(testFile, testFileDest)
		require.NoError(t, err, "CopyFile failed")
		require.True(t, success, "CopyFile reported failure despite no error")

		// Verify the content
		content, err := os.ReadFile(testFileDest)
		require.NoError(t, err, "Failed to read copied file")
		require.Equal(t, "test content", string(content), "File content doesn't match expected")
	})
}

func TestCreateDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Test successful directory creation
	t.Run("SuccessfulDirCreation", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "newdir")

		success, err := CreateDir(newDir)
		require.NoError(t, err, "CreateDir failed")
		require.True(t, success, "CreateDir reported failure despite no error")

		// Verify directory exists
		info, err := os.Stat(newDir)
		require.NoError(t, err, "Failed to stat newly created directory")
		require.True(t, info.IsDir(), "Created path is not a directory")
	})

	t.Run("ExistingDirectory", func(t *testing.T) {
		existingDir := filepath.Join(tempDir, "existing")

		// Create the directory first
		err := os.Mkdir(existingDir, 0755)
		require.NoError(t, err, "Failed to create initial directory")

		// Try to create the same directory again
		_, err = CreateDir(existingDir)
		require.NoError(t, err)
	})
}

func TestDeletePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	t.Run("DeleteFile", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "delete.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Delete the file
		success, err := DeletePath(testFile)
		require.NoError(t, err, "DeletePath failed")
		require.True(t, success, "DeletePath reported failure despite no error")

		// Verify file doesn't exist
		_, err = os.Stat(testFile)
		require.True(t, os.IsNotExist(err), "File should not exist after deletion")
	})
}
