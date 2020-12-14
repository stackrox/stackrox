package cas

import (
	"context"
	"os"
	"path/filepath"
)

const (
	caCert      = "/run/secrets/stackrox.io/certs/ca.pem"
	caKey       = "/run/secrets/stackrox.io/certs/ca-key.pem"
	jwtKeyInDer = "/run/secrets/stackrox.io/certs/jwt-key.der"
	jwtKeyInPem = "/run/secrets/stackrox.io/certs/jwt-key.pem"
)

// NewCertsBackup returns a generator of certificate backups.
func NewCertsBackup() *CertsBackup {
	// Include jwt key in either der or in pem format, preferable in der format.
	jwtKey := jwtKeyInDer
	if _, err := os.Stat(jwtKeyInDer); os.IsNotExist(err) {
		jwtKey = jwtKeyInPem
	}
	return &CertsBackup{
		certFiles: []string{caCert, caKey, jwtKey},
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
