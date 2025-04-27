package flags

import (
	"flag"
	"os"

	"github.com/ogzhanolguncu/mimic/internal/config"
	"github.com/ogzhanolguncu/mimic/internal/logger"
)

func Parse() *config.Config {
	cfg := config.NewDefaultConfig()

	flag.BoolVar(&cfg.Verbose, "verbose", config.DefaultVerbose, "Enable detailed debug logging")
	flag.BoolVar(&cfg.DryRun, "dry-run", config.DefaultDryRun, "Simulate operations without making changes")
	flag.BoolVar(&cfg.Checksum, "checksum", config.DefaultChecksum, "Use checksum comparison instead of mtime/size")
	flag.Int64Var(&cfg.ChunkSize, "chunk-size", config.DefaultChunkSize, "Buffer size in bytes for file copying")
	flag.IntVar(&cfg.BandwidthLimit, "bandwidth-limit", config.DefaultBandwidthLimit, "Bandwidth limit in KB/s (0 for unlimited)")

	flag.Parse()

	if flag.NArg() != 2 {
		logger.Error("Usage: mimic [options] <source_directory> <destination_directory>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	return cfg
}
