package satoken

import (
	"bytes"
	"os"
)

// LoadTokenFromFile loads the Kubernetes service account JWT token from the canonical file location and returns the
// token or an error.
func LoadTokenFromFile() (string, error) {
	contents, err := os.ReadFile(ServiceAccountTokenJWTPath)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(contents)), nil
}
