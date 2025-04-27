package syncer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ogzhanolguncu/mimic/internal/logger"
)

type SyncState struct {
	Version  int                  `json:"v"`  // Schema version of the state file.
	LastSync int64                `json:"ls"` // When the previous sync completed.
	Entries  map[string]EntryInfo `json:"e"`  // Maps relative paths to their metadata.
}

var (
	ErrSyncStateMarshal       = errors.New("sync_state: failed to marshal SyncState")
	ErrSyncStateNil           = errors.New("sync_state: nil state provided")
	ErrSyncStateDstDir        = errors.New("sync_state: failed to create dst dir")
	ErrSyncStateEmptyDst      = errors.New("sync_state: dst is empty")
	ErrSyncStateWrite         = errors.New("sync_state: failed to write a file")
	ErrSyncStateReplace       = errors.New("sync_state: failed to replace the state file")
	ErrSyncStateRead          = errors.New("sync_state: failed to read a file")
	ErrSyncStateJSONParse     = errors.New("sync_state: failed to parse JSON")
	ErrSyncStateJSONSerialize = errors.New("sync_state: failed to serialize JSON")
)

const stateFile = ".sync_state"

func LoadState(dstDir string) (*SyncState, error) {
	if dstDir == "" {
		return nil, ErrSyncStateEmptyDst
	}

	op := "LoadState"
	logger.Debug("loading state", "operation", op, "dir", dstDir)

	stateFileLocation := filepath.Join(dstDir, stateFile)

	_, err := os.Stat(stateFileLocation)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.Info("state file does not exist, creating new one", "operation", op, "path", stateFileLocation)

			data := &SyncState{
				Version:  1,
				LastSync: time.Now().UnixMilli(),
				Entries:  make(map[string]EntryInfo),
			}

			return data, SaveState(dstDir, data)
		}
		return nil, fmt.Errorf("%w: %v", ErrSyncStateRead, err)
	}

	data, err := os.ReadFile(stateFileLocation)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncStateRead, err)
	}

	synState := &SyncState{}
	if err := json.Unmarshal(data, synState); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncStateJSONParse, err)
	}

	return synState, nil
}

func SaveState(dstDir string, state *SyncState) error {
	if state == nil {
		return ErrSyncStateNil
	}
	if dstDir == "" {
		return ErrSyncStateEmptyDst
	}

	op := "SaveState"
	logger.Debug("saving state", "operation", op, "dir", dstDir)

	stateFileLocation := filepath.Join(dstDir, stateFile)

	state.LastSync = time.Now().UnixMilli()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSyncStateJSONSerialize, err)
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrSyncStateDstDir, err)
	}

	tempFile := stateFileLocation + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("%w: %v", ErrSyncStateWrite, err)
	}

	if err := os.Rename(tempFile, stateFileLocation); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("%w: %v", ErrSyncStateReplace, err)
	}

	logger.Info("state saved successfully", "operation", op)
	return nil
}
