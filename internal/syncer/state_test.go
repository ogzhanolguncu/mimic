package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadState(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sync_state_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) // Clean up after test

	originalState := &SyncState{
		Version:  0,
		LastSync: time.Now().UnixMilli(),
		Entries: map[string]EntryInfo{
			"file1.txt": {
				RelativePath: "file1.txt",
				Size:         100,
				IsDir:        false,
			},
			"dir1": {
				RelativePath: "dir1",
				IsDir:        true,
			},
		},
	}

	// Test SaveState
	err = SaveState(tempDir, originalState)
	require.NoError(t, err, "SaveState should not return an error")

	// Verify file exists
	stateFilePath := filepath.Join(tempDir, stateFile)
	_, err = os.Stat(stateFilePath)
	require.NoError(t, err, "State file should exist")

	// Test LoadState
	loadedState, err := LoadState(tempDir)
	require.NoError(t, err, "LoadState should not return an error")

	// Verify content
	require.Equal(t, originalState.Version, loadedState.Version)
	require.Equal(t, originalState.LastSync, loadedState.LastSync)
	require.Equal(t, len(originalState.Entries), len(loadedState.Entries))

	// Check specific entries
	require.Contains(t, loadedState.Entries, "file1.txt")
	require.Contains(t, loadedState.Entries, "dir1")
	require.Equal(t, originalState.Entries["file1.txt"].Size, loadedState.Entries["file1.txt"].Size)
	require.Equal(t, originalState.Entries["dir1"].IsDir, loadedState.Entries["dir1"].IsDir)
}

func TestLoadStateNonExistent(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sync_state_test_nonexistent")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test LoadState on a directory with no existing state file
	state, err := LoadState(tempDir)
	require.NoError(t, err, "LoadState should create a new state file if none exists")
	require.NotNil(t, state, "LoadState should return a non-nil state")
	require.Equal(t, 1, state.Version)
	require.Empty(t, state.Entries)

	// Verify file was created
	stateFilePath := filepath.Join(tempDir, stateFile)
	_, err = os.Stat(stateFilePath)
	require.NoError(t, err, "State file should have been created")
}

func TestSaveStateErrors(t *testing.T) {
	// Test nil state
	err := SaveState("/tmp", nil)
	require.Error(t, err)
	require.Equal(t, ErrSyncStateNil, err)

	// Test empty destination
	err = SaveState("", &SyncState{})
	require.Error(t, err)
	require.Equal(t, ErrSyncStateEmptyDst, err)
}
