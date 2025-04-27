package config

// Default configuration constants
const (
	DefaultChunkSize      = 32 << 20 // 32MB in bytes
	DefaultVerbose        = false
	DefaultDryRun         = false
	DefaultChecksum       = false
	DefaultBandwidthLimit = 0 // No limit
)

// Default empty slice for exclude patterns
var DefaultExcludePatterns = []string{".DS_Store"}

// Config holds all user-configurable settings for the sync operation.
// These parameters control the behavior, performance and safety of the sync process.
type Config struct {
	// Verbose enables detailed logging of operations (Debug level).
	Verbose bool
	// DryRun simulates all operations without making actual filesystem changes.
	DryRun bool
	// Checksum enables comparing file content hashes instead of just mtime/size.
	// More accurate but potentially slower as it requires reading files.
	Checksum bool
	// ChunkSize defines the buffer size in bytes for file copying
	ChunkSize int64
	// ExcludePatterns contains glob patterns for files/directories to skip
	ExcludePatterns []string
	// BandwidthLimit restricts transfer speed in KB/s
	BandwidthLimit int
}

// NewDefaultConfig creates a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Verbose:         DefaultVerbose,
		DryRun:          DefaultDryRun,
		Checksum:        DefaultChecksum,
		ChunkSize:       DefaultChunkSize,
		ExcludePatterns: DefaultExcludePatterns,
		BandwidthLimit:  DefaultBandwidthLimit,
	}
}
