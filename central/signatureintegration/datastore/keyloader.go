package datastore

import (
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
)

// loadKeysFromDir reads all files from the given directory and returns those that contain valid
// PEM-encoded public keys. Files that are not valid public keys are skipped with a warning.
// If the directory does not exist or is empty, an empty slice is returned without error.
// Duplicate keys (same PEM content) are deduplicated — the first file encountered is kept.
// This function is safe to call concurrently.
func loadKeysFromDir(dir string) ([]*storage.CosignPublicKeyVerification_PublicKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "reading Red Hat signing keys directory %q", dir)
	}

	seen := make(map[string]struct{})
	var keys []*storage.CosignPublicKeyVerification_PublicKey

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		fullPath := filepath.Join(dir, name)

		contents, err := os.ReadFile(fullPath)
		if err != nil {
			log.Warnf("Skipping Red Hat signing key file %q: %v", fullPath, err)
			continue
		}

		block, rest := pem.Decode(contents)
		if !signatures.IsValidPublicKeyPEMBlock(block, rest) {
			log.Warnf("Skipping Red Hat signing key file %q: not a valid PEM-encoded public key", fullPath)
			continue
		}

		pemStr := string(contents)
		if _, dup := seen[pemStr]; dup {
			log.Debugf("Skipping duplicate Red Hat signing key in file %q", fullPath)
			continue
		}
		seen[pemStr] = struct{}{}

		keys = append(keys, &storage.CosignPublicKeyVerification_PublicKey{
			Name:            name,
			PublicKeyPemEnc: pemStr,
		})
	}

	return keys, nil
}
