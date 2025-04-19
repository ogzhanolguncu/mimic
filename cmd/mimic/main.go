package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/ogzhanolguncu/mimic/internal/syncer"
)

func main() {
	// Define command-line flags
	sourceDir := flag.String("source", "", "Source directory to copy from (required)")
	destDir := flag.String("dest", "", "Destination directory to copy to (required)")

	// Parse the flags
	flag.Parse()

	// Check for required flags
	if *sourceDir == "" || *destDir == "" {
		flag.Usage()
		log.Fatalf("Error: source and destination directories are required")
	}

	entries, err := syncer.ScanSource(*sourceDir)
	log.Printf("Errors %+v", err)
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		log.Printf("Error marshaling entries: %v", err)
	} else {
		log.Printf("Entries:\n%s", jsonData)
	}

	if err != nil {
		log.Fatalf("Error scanning source: %v", err)
	}
}
