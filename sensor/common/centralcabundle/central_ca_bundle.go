package centralcabundle

import (
	"crypto/x509"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	caCerts      []*x509.Certificate
	caCertsMutex sync.RWMutex
)

// Set stores the Central CA certificates.
func Set(cas []*x509.Certificate) {
	caCertsMutex.Lock()
	defer caCertsMutex.Unlock()
	caCerts = append([]*x509.Certificate(nil), cas...)
	if len(cas) > 0 {
		log.Infof("Stored %d Central CA certificate(s)", len(cas))
	}
}

// Get returns a copy of the stored Central CA certificates.
// Returns nil if no CAs have been stored.
func Get() []*x509.Certificate {
	caCertsMutex.RLock()
	defer caCertsMutex.RUnlock()
	return append([]*x509.Certificate(nil), caCerts...)
}
