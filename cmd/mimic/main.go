package main

import (
	"flag"
	"log/slog"
	"os"
	"slices"

	"github.com/ogzhanolguncu/mimic/internal/config"
	dryrun "github.com/ogzhanolguncu/mimic/internal/dry_run"
	"github.com/ogzhanolguncu/mimic/internal/flags"
	"github.com/ogzhanolguncu/mimic/internal/logger"
	"github.com/ogzhanolguncu/mimic/internal/syncer"
)

func main() {
	cfg := flags.Parse()
	setupLogging(cfg.Verbose)

	args := flag.Args()
	srcDir, dstDir := args[0], args[1]

	logger.Info("Starting sync process",
		"source", srcDir,
		"destination", dstDir,
		"config", cfg)

	if err := runSync(srcDir, dstDir, cfg); err != nil {
		logger.Fatal("Sync process failed", "error", err)
	}

	logger.Info("Sync process completed successfully")
}

// setupLogging configures the logger based on verbose setting
func setupLogging(verbose bool) {
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	logger.Initialize(logger.Config{
		Level:  logLevel,
		Output: os.Stderr,
	})
}

// runSync performs the actual synchronization process
func runSync(srcDir string, dstDir string, cfg *config.Config) error {
	// Load or create state
	state, err := syncer.LoadState(dstDir)
	if err != nil {
		return err
	}

	// Scan source directory
	sourceEntries, err := syncer.ScanSource(srcDir)
	if err != nil {
		return err
	}

	// Compare states and determine actions
	logger.Info("Comparing states")
	actions := syncer.CompareStates(sourceEntries, state.Entries)

	if cfg.DryRun {
		dryrun.PrintFullReport(actions)
		return nil
	}

	// Filter out "none" actions for reporting
	actionCount := len(slices.DeleteFunc(slices.Clone(actions), func(a syncer.SyncAction) bool {
		return a.Type == syncer.ActionNone
	}))
	logger.Info("Found actions to perform", "count", actionCount)

	// Execute actions
	logger.Info("Executing sync actions")
	if err := syncer.ExecuteActions(srcDir, dstDir, actions, cfg); err != nil {
		return err
	}

	// Update and save state
	state.Entries = sourceEntries
	return syncer.SaveState(dstDir, state)
}
