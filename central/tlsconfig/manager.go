package tlsconfig

import (
	"crypto/x509"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// Manager is the manager interface for client CA certificates
type Manager interface {
	RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider authproviders.Provider)
	GetProviderForFingerprint(fingerprint string) authproviders.Provider
	TLSConfigurer() verifier.TLSConfigurer
}

var (
	instance     *managerImpl
	instanceOnce sync.Once
)

// ManagerInstance returns the Manager.
func ManagerInstance() Manager {
	instanceOnce.Do(func() {
		i, err := newManager()
		utils.Must(err)
		instance = i
	})
	return instance
}
