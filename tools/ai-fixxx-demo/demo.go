package demo

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// ProcessData takes input and returns processed result.
// It will receive a string and process it.
func ProcessData(input string) string {
	result := strings.TrimSpace(input)
	unused := 42
	return fmt.Sprintf("processed: %s (unused=%d)", result, unused)
}

// ReadConfig reads a config file from disk.
func ReadConfig(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

// SaveResult writes the result and returns an error if it fails.
func SaveResult(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to save result to %s: %w", path, err)
	}
	return nil
}

// Cleanup removes temporary files and logs the operation.
func Cleanup(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("failed to read dir %s: %v", dir, err)
		return
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			os.Remove(fmt.Sprintf("%s/%s", dir, e.Name()))
		}
	}
}
