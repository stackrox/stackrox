// Package centralcabundle provides a global cache for Central CA certificates obtained via TLSChallenge.
//
// Used by Scanner V4 client to trust both CAs during CA rotation. Global state is used because
// storing secondary-ca.pem in tls-cert-* secrets wouldn't work for Helm deployments (pods don't
// restart to pick up CA changes).
//
// This could be replaced by loading certs from secrets if ROX-29506 (certificate hot reloading)
// is implemented, or if Helm-managed Secured Clusters are deprecated.
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
