package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScanSource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "syncer-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	t.Run("EmptySourceDir", func(t *testing.T) {
		entries, err := ScanSource("")
		require.Error(t, err, "Expected error for empty source directory")
		require.Equal(t, ErrEmptySrcDir, err, "Expected ErrEmptySrcDir error")
		require.Nil(t, entries, "Expected nil entries for error case")
	})

	t.Run("NonExistentSourceDir", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "non-existent")
		entries, err := ScanSource(nonExistentDir)
		require.Error(t, err, "Expected error for non-existent source directory")
		require.ErrorIs(t, err, ErrSyncerSrcNotExists, "Expected ErrSyncerSrcNotExists error")
		require.Nil(t, entries, "Expected nil entries for error case")
	})

	t.Run("SourceIsFile", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "testfile.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err, "Failed to create test file")

		entries, err := ScanSource(testFile)
		require.Error(t, err, "Expected error when source is a file")
		require.Equal(t, ErrEmptySrcNotADir, err, "Expected ErrEmptySrcNotADir error")
		require.Nil(t, entries, "Expected nil entries for error case")
	})

	t.Run("ValidDirScan", func(t *testing.T) {
		// Create a valid directory structure for testing
		testDir := filepath.Join(tempDir, "valid-dir")
		require.NoError(t, os.Mkdir(testDir, 0755), "Failed to create test directory")

		// Create a subdirectory
		subDir := filepath.Join(testDir, "subdir")
		require.NoError(t, os.Mkdir(subDir, 0755), "Failed to create subdirectory")

		// Create files in root and subdirectory
		rootFile := filepath.Join(testDir, "root.txt")
		subFile := filepath.Join(subDir, "sub.txt")

		require.NoError(t, os.WriteFile(rootFile, []byte("root content"), 0644), "Failed to create root file")
		require.NoError(t, os.WriteFile(subFile, []byte("sub content"), 0644), "Failed to create sub file")

		entries, err := ScanSource(testDir)
		require.NoError(t, err, "Expected no error for valid directory scan")
		require.NotNil(t, entries, "Expected non-nil entries")

		require.Len(t, entries, 3, "Expected 3 entries")

		// Verify entries exist and have correct properties
		require.Contains(t, entries, "root.txt", "Expected root.txt entry")
		require.Contains(t, entries, "subdir", "Expected subdir entry")
		require.Contains(t, entries, filepath.Join("subdir", "sub.txt"), "Expected subdir/sub.txt entry")

		// Verify entry properties
		rootEntry := entries["root.txt"]
		require.NotNil(t, rootEntry, "Expected root.txt entry to exist")
		require.False(t, rootEntry.IsDir, "Expected root.txt to not be a directory")
		require.Equal(t, int64(len("root content")), rootEntry.Size, "Expected correct file size")
		require.NotEmpty(t, rootEntry.Checksum, "Expected non-empty checksum")

		subdirEntry := entries["subdir"]
		require.NotNil(t, subdirEntry, "Expected subdir entry to exist")
		require.True(t, subdirEntry.IsDir, "Expected subdir to be a directory")
		require.Empty(t, subdirEntry.Checksum, "Expected empty checksum for directory")
	})

	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	t.Run("PermissionDenied", func(t *testing.T) {
		// Create a directory with no read permissions
		noReadDir := filepath.Join(tempDir, "no-read")
		require.NoError(t, os.Mkdir(noReadDir, 0755), "Failed to create directory")

		// Create a subdirectory with no permissions
		nopermDir := filepath.Join(noReadDir, "noperm")
		require.NoError(t, os.Mkdir(nopermDir, 0755), "Failed to create noperm directory")

		// Set no permissions on the subdirectory
		require.NoError(t, os.Chmod(nopermDir, 0000), "Failed to change permissions")

		// The scan should succeed but skip the no-permission directory
		entries, err := ScanSource(noReadDir)
		require.NoError(t, err, "Expected no error for scan with permission denied subdirectory")
		require.NotNil(t, entries, "Expected non-nil entries")

		// Should have 1 entry: noperm (but not its contents)
		require.Contains(t, entries, "noperm", "Expected noperm entry")

		// Reset permissions to allow cleanup
		_ = os.Chmod(nopermDir, 0755)
	})
}

func TestGenerateChecksum(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "syncer-checksum-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	t.Run("NonExistentFile", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "non-existent.txt")
		checksum, err := generateChecksum(nonExistentFile)
		require.Error(t, err, "Expected error for non-existent file")
		require.ErrorIs(t, err, ErrSyncerSrcNotExists, "Expected ErrSyncerSrcNotExists error")
		require.Nil(t, checksum, "Expected nil checksum for error case")
	})

	t.Run("ValidFileChecksum", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		testContent := "test content for checksum"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err, "Failed to create test file")

		checksum1, err := generateChecksum(testFile)
		require.NoError(t, err, "Expected no error for valid file")
		require.NotNil(t, checksum1, "Expected non-nil checksum")
		require.NotEmpty(t, checksum1, "Expected non-empty checksum")

		// Generate checksum again to verify it's consistent
		checksum2, err := generateChecksum(testFile)
		require.NoError(t, err, "Expected no error for second checksum")
		require.Equal(t, checksum1, checksum2, "Expected consistent checksums")

		// Modify the file and verify checksum changes
		newContent := "modified content for checksum"
		err = os.WriteFile(testFile, []byte(newContent), 0644)
		require.NoError(t, err, "Failed to modify test file")

		checksum3, err := generateChecksum(testFile)
		require.NoError(t, err, "Expected no error for modified file")
		require.NotEqual(t, checksum1, checksum3, "Expected different checksum for modified file")
	})
}

func TestShouldExclude(t *testing.T) {
	testCases := []struct {
		name          string
		relPath       string
		matchers      []string
		shouldExclude bool
	}{
		{
			name:          "Exact file match",
			relPath:       "file.txt",
			matchers:      []string{"file.txt"},
			shouldExclude: true,
		},
		{
			name:          "File with glob pattern",
			relPath:       "temp.log",
			matchers:      []string{"*.log"},
			shouldExclude: true,
		},
		{
			name:          "File not matching pattern",
			relPath:       "data.csv",
			matchers:      []string{"*.log", "*.tmp"},
			shouldExclude: false,
		},
		{
			name:          "Directory exact match",
			relPath:       "node_modules/package/file.js",
			matchers:      []string{"node_modules/"},
			shouldExclude: true,
		},
		{
			name:          "Directory not matching",
			relPath:       "src/components/file.js",
			matchers:      []string{"node_modules/"},
			shouldExclude: false,
		},
		{
			name:          "Multiple patterns - one match",
			relPath:       ".DS_Store",
			matchers:      []string{"*.log", ".DS_Store"},
			shouldExclude: true,
		},
		{
			name:          "Directory as exact path",
			relPath:       "node_modules",
			matchers:      []string{"node_modules/"},
			shouldExclude: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := shouldExclude(tc.relPath, tc.matchers)
			require.Equal(t, tc.shouldExclude, result,
				"Expected shouldExclude(%q, %v) to be %v, got %v",
				tc.relPath, tc.matchers, tc.shouldExclude, result)
		})
	}
}

func TestShouldCompareStates(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name          string
		sourceScan    map[string]EntryInfo
		loadedEntries map[string]EntryInfo
		expected      []SyncAction
	}{
		{
			name: "CreateNewFile",
			sourceScan: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime,
					Size:         100,
					IsDir:        false,
				},
			},
			loadedEntries: map[string]EntryInfo{},
			expected: []SyncAction{
				{
					Type:         ActionCreate,
					RelativePath: "file1.txt",
					SourceInfo: EntryInfo{
						RelativePath: "file1.txt",
						Mtime:        fixedTime,
						Size:         100,
						IsDir:        false,
					},
				},
			},
		},
		{
			name: "UpdateExistingFile",
			sourceScan: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime,
					Size:         200, // Changed size
					IsDir:        false,
				},
			},
			loadedEntries: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime.Add(-10 * time.Minute), // Older time
					Size:         100,                              // Original size
					IsDir:        false,
				},
			},
			expected: []SyncAction{
				{
					Type:         ActionUpdate,
					RelativePath: "file1.txt",
					SourceInfo: EntryInfo{
						RelativePath: "file1.txt",
						Mtime:        fixedTime,
						Size:         200,
						IsDir:        false,
					},
				},
			},
		},
		{
			name:       "DeleteRemovedFile",
			sourceScan: map[string]EntryInfo{},
			loadedEntries: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime.Add(-10 * time.Minute),
					Size:         100,
					IsDir:        false,
				},
			},
			expected: []SyncAction{
				{
					Type:         ActionDelete,
					RelativePath: "file1.txt",
					SourceInfo:   EntryInfo{},
				},
			},
		},
		{
			name: "NoChangeNeeded",
			sourceScan: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime,
					Size:         100,
					IsDir:        false,
				},
			},
			loadedEntries: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        fixedTime,
					Size:         100,
					IsDir:        false,
				},
			},
			expected: []SyncAction{
				{
					Type:         ActionNone,
					RelativePath: "file1.txt",
					SourceInfo:   EntryInfo{},
				},
			},
		},
		{
			name: "MixedOperations",
			sourceScan: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Size:         100,
					IsDir:        false,
				},
				"file2.txt": {
					RelativePath: "file2.txt",
					Mtime:        fixedTime,
					Size:         200,
					IsDir:        false,
				},
				"dir1": {
					RelativePath: "dir1",
					Mtime:        fixedTime,
					Size:         0,
					IsDir:        true,
				},
			},
			loadedEntries: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Size:         100,
					IsDir:        false,
				},
				"oldfile.txt": {
					RelativePath: "oldfile.txt",
					Mtime:        fixedTime.Add(-24 * time.Hour),
					Size:         50,
					IsDir:        false,
				},
			},
			expected: []SyncAction{
				{
					Type:         ActionNone,
					RelativePath: "file1.txt",
					SourceInfo:   EntryInfo{},
				},
				{
					Type:         ActionCreate,
					RelativePath: "file2.txt",
					SourceInfo: EntryInfo{
						RelativePath: "file2.txt",
						Mtime:        fixedTime,
						Size:         200,
						IsDir:        false,
					},
				},
				{
					Type:         ActionCreate,
					RelativePath: "dir1",
					SourceInfo: EntryInfo{
						RelativePath: "dir1",
						Mtime:        fixedTime,
						Size:         0,
						IsDir:        true,
					},
				},
				{
					Type:         ActionDelete,
					RelativePath: "oldfile.txt",
					SourceInfo:   EntryInfo{},
				},
			},
		},
		{
			name: "WithinTimeThreshold",
			sourceScan: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        time.Date(2023, 1, 1, 12, 0, 0, 500*1000*1000, time.UTC), // 500ms difference
					Size:         100,
					IsDir:        false,
				},
			},
			loadedEntries: map[string]EntryInfo{
				"file1.txt": {
					RelativePath: "file1.txt",
					Mtime:        time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					Size:         100,
					IsDir:        false,
				},
			},
			expected: []SyncAction{
				{
					Type:         ActionNone, // Should be considered the same due to threshold
					RelativePath: "file1.txt",
					SourceInfo:   EntryInfo{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareStates(tc.sourceScan, tc.loadedEntries)
			require.Equal(t, tc.expected, result)
		})
	}
}
