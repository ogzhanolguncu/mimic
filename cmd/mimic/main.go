package main

import (
	"flag"
	"log/slog"
	"os"
	"slices"

	"github.com/ogzhanolguncu/mimic/internal/config"
	"github.com/ogzhanolguncu/mimic/internal/logger"
	"github.com/ogzhanolguncu/mimic/internal/syncer"
)

func main() {
	verbose := flag.Bool("verbose", false, "Enable detailed debug logging")
	dryRun := flag.Bool("dry-run", false, "Simulate operations without making changes")
	useChecksum := flag.Bool("checksum", false, "Use checksum comparison instead of mtime/size")
	flag.Parse()

	if flag.NArg() != 2 {
		logger.Error("Usage: go_sync [options] <source_directory> <destination_directory>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	srcDir := flag.Arg(0)
	dstDir := flag.Arg(1)

	cfg := config.NewDefaultConfig()
	cfg.Verbose = *verbose
	cfg.DryRun = *dryRun
	cfg.Checksum = *useChecksum

	logLevel := slog.LevelInfo
	if cfg.Verbose {
		logLevel = slog.LevelDebug
	}

	logger.Initialize(logger.Config{
		Level:  logLevel,
		Output: os.Stderr,
	})

	logger.Info("Starting sync process",
		"source", srcDir,
		"destination", dstDir,
		"config", cfg)

	state, err := syncer.LoadState(dstDir)
	if err != nil {
		logger.Error("Failed to scan load or create state", "error", err)
		os.Exit(1)
	}

	sourceEntries, err := syncer.ScanSource(srcDir)
	if err != nil {
		logger.Error("Failed to scan source directory", "error", err)
		os.Exit(1)
	}

	logger.Info("Comparing states")
	actions := syncer.CompareStates(sourceEntries, state.Entries)
	logger.Info("Found actions to perform", "count", len(actions))

	logger.Info("Executing states")
	err = syncer.ExecuteActions(srcDir, dstDir, actions)
	if err != nil {
		logger.Fatal("Failed to execute actions", "error", err)
	}

	state.Entries = sourceEntries
	syncer.SaveState(dstDir, state)

	actions = slices.DeleteFunc(actions, func(a syncer.SyncAction) bool {
		return a.Type == syncer.ActionNone
	})

	logger.Info("Sync process completed successfully", "actions", len(actions))
}
