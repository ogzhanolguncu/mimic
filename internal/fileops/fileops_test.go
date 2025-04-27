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

	t.Run("SuccessfulCopy", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Copy the file
		success, err := CopyFile(testFile)
		require.NoError(t, err, "CopyFile failed")
		require.True(t, success, "CopyFile reported failure despite no error")

		// Verify the content
		content, err := os.ReadFile(testFile)
		require.NoError(t, err, "Failed to read copied file")
		require.Equal(t, "test content", string(content), "File content doesn't match expected")
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

		success, err := CopyFile(nonExistentFile)
		require.Error(t, err, "Expected error when copying non-existent file")
		require.False(t, success, "CopyFile reported success when it should have failed")

		// Check if error is wrapped correctly
		require.ErrorIs(t, err, ErrRead, "Error should wrap ErrRead")
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
		success, err := CreateDir(existingDir)
		require.Error(t, err, "Expected error when creating existing directory")
		require.False(t, success, "CreateDir reported success when it should have failed")
		require.ErrorIs(t, err, ErrMkDir, "Error should wrap ErrMkDir")
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

	t.Run("DeleteNonEmptyDir", func(t *testing.T) {
		// Create a test directory with a file in it
		testDir := filepath.Join(tempDir, "nonemptydir")
		err := os.Mkdir(testDir, 0755)
		require.NoError(t, err, "Failed to create test directory")

		// Create a file inside the directory
		testFile := filepath.Join(testDir, "file.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Try to delete the directory
		success, err := DeletePath(testDir)
		require.Error(t, err, "Expected error when deleting non-empty directory")
		require.False(t, success, "DeletePath reported success when it should have failed")
		require.ErrorIs(t, err, ErrRemoveDir, "Error should wrap ErrRemoveDir")
	})
}
