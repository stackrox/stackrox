package cas

import (
	"context"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/pkg/mtls"
)

// NewCertsBackup returns a generator of certificate backups.
func NewCertsBackup() *CertsBackup {
	jwtKey := jwt.PrivateKeyDERPath()
	if _, err := os.Stat(jwtKey); os.IsNotExist(err) {
		jwtKey = jwt.PrivateKeyPEMPath()
	}
	return &CertsBackup{
		certFiles: []string{mtls.CAFilePath(), mtls.CAKeyFilePath(), jwtKey},
	}
}

// CertsBackup is an implementation of a PathMapGenerator which generate the layout of cert files to backup.
type CertsBackup struct {
	certFiles []string
}

// GeneratePathMap generates the map from the path within backup to its source certificate.
func (c *CertsBackup) GeneratePathMap(_ context.Context) (map[string]string, error) {
	certMap := make(map[string]string)
	// Put all the certs under the same root directory.
	for _, p := range c.certFiles {
		certMap[filepath.Base(p)] = p
	}
	return certMap, nil
}
