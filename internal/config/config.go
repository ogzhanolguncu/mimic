package config

// Config holds all user-configurable settings for the sync operation.
// These parameters control the behavior, performance and safety of the sync process.
type Config struct {
	// Verbose enables detailed logging of operations (Debug level).
	Verbose bool

	// DryRun simulates all operations without making actual filesystem changes.
	DryRun bool

	// Checksum enables comparing file content hashes instead of just mtime/size.
	// More accurate but potentially slower as it requires reading files.
	Checksum bool // Added this common flag

	// ChunkSize defines the buffer size in bytes for file copying (Advanced).
	ChunkSize int64 // TODO: Implement later

	// ExcludePatterns contains glob patterns for files/directories to skip (Advanced).
	ExcludePatterns []string // TODO: Implement later (needs parsing/matching)

	// BandwidthLimit restricts transfer speed in KB/s (Advanced).
	BandwidthLimit int // TODO: Implement later
}

// NewDefaultConfig creates a Config struct with sensible default values.
func NewDefaultConfig() *Config {
	return &Config{
		Verbose:         false,
		DryRun:          false,
		Checksum:        false,            // Default to faster mtime/size comparison
		ChunkSize:       32 * 1024 * 1024, // Default 32MB copy buffer
		ExcludePatterns: []string{},
		BandwidthLimit:  0, // No limit
	}
}
