package utils

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	hostPrefix = "/host"
)

// HostPathToLocal converts a path on the host to one on the local container's filesystem.
func HostPathToLocal(hostPath string) (string, error) {
	if !filepath.IsAbs(hostPath) {
		return "", errors.Errorf("host path %q is not an absolute path", hostPath)
	}
	return hostPrefix + hostPath, nil
}

// ReadHostFile attempts to read the contents of a file on the host.
func ReadHostFile(hostPath string) ([]byte, error) {
	localPath, err := HostPathToLocal(hostPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(localPath)
	return data, errors.Wrapf(err, "trying to read host file %s", hostPath)
}
