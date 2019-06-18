package manager

import (
	"crypto/x509"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
)

// ClientCAManager is the manager interface for client CA certificates
type ClientCAManager interface {
	RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider authproviders.Provider)
	GetProviderForFingerprint(fingerprint string) authproviders.Provider
	TLSConfigurer() verifier.TLSConfigurer
}

var (
	instance     *managerImpl
	instanceOnce sync.Once
)

// Instance returns the ClientCAManager.
func Instance() ClientCAManager {
	instanceOnce.Do(func() {
		instance = newManager()
	})
	return instance
}

func newManager() *managerImpl {
	return &managerImpl{
		providerIDToProviderData: make(map[string]providerData),
	}
}
