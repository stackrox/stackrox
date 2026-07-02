package updater

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnpackBundle opens a vulnerabilities.zip and returns the contained bundles.
// Each file in the ZIP (e.g., "alpine.json.zst") becomes a BundleData entry.
func UnpackBundle(zipPath string) ([]*BundleData, error) {
	f, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat ZIP file: %w", err)
	}

	return UnpackBundleFromReader(f, stat.Size())
}

// UnpackBundleFromReader reads a ZIP from an io.ReaderAt.
func UnpackBundleFromReader(r io.ReaderAt, size int64) ([]*BundleData, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZIP reader: %w", err)
	}

	var bundles []*BundleData

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Open file
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s in ZIP: %w", file.Name, err)
		}

		// Read content
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s in ZIP: %w", file.Name, err)
		}

		// Compute SHA256 fingerprint
		hash := sha256.Sum256(data)
		fingerprint := hex.EncodeToString(hash[:])

		// Extract name (remove .zst extension)
		name := file.Name
		name = strings.TrimSuffix(name, ".zst")
		name = filepath.Base(name) // Get just the filename, not directory

		bundles = append(bundles, &BundleData{
			Name:        name,
			Data:        data,
			Fingerprint: fingerprint,
		})
	}

	return bundles, nil
}
