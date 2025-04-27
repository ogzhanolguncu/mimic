package main

import (
	"encoding/json"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/ogzhanolguncu/mimic/internal/config"
	"github.com/ogzhanolguncu/mimic/internal/syncer"
)

func main() {
	verbose := flag.Bool("verbose", false, "Enable detailed debug logging")
	dryRun := flag.Bool("dry-run", false, "Simulate operations without making changes")
	useChecksum := flag.Bool("checksum", false, "Use checksum comparison instead of mtime/size")
	// TODO: Add flags for ChunkSize, ExcludePatterns, BandwidthLimit later

	flag.Parse()

	if flag.NArg() != 2 {
		slog.Error("Usage: go_sync [options] <source_directory> <destination_directory>")
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
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	syncer.SetLogger(logger)

	logger.Info("Starting sync process", "source", srcDir, "destination", dstDir, "config", cfg) // Log config

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

	logger.Info("Comparing states.")
	actions := syncer.CompareStates(sourceEntries, state.Entries)
	log.Printf("\nFound %d actions to perform. Actions: %+v\n", len(actions), actions)

	state.Entries = sourceEntries
	syncer.SaveState(dstDir, state)

	logger.Info("Sync process completed successfully.")

	jsonData, err := json.MarshalIndent(sourceEntries, "", "  ")
	if err != nil {
		log.Printf("Error marshaling entries: %v", err)
	} else {
		log.Printf("Entries:\n%s", jsonData)
	}

	if err != nil {
		log.Fatalf("Error scanning source: %v", err)
	}
}
