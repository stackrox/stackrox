package utils

import (
	"os"

	"github.com/pkg/errors"
)

const (
	encryptionKeyFile = "/run/secrets/stackrox.io/central-encryption-key/encryption-key"
)

// GetNotifierSecretEncryptionKey returns the key for encrypting/decrypting notifier secrets
func GetNotifierSecretEncryptionKey() (string, error) {
	key, err := os.ReadFile(encryptionKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "Could not load notifier encryption key")
	}
	return string(key), nil
}
