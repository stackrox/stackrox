package tlsconfig

import (
	"crypto/x509"

	"github.com/stackrox/stackrox/pkg/auth/authproviders"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/mtls/verifier"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

// ServerCertSource is an enum type that determines the source for obtaining the TLS server certificate.
type ServerCertSource int

const (
	// DefaultTLSCertSource is the source for the user-configured default TLS certificate.
	DefaultTLSCertSource ServerCertSource = iota
	// ServiceCertSource is the source for the StackRox internal service TLS certificate.
	ServiceCertSource
)

// ClientCASource is an enum type that determines the source for obtaining TLS client certificate authorities.
type ClientCASource int

const (
	// UserCAsSource is the source for the user-configured (via PKI auth providers) CAs
	UserCAsSource ClientCASource = iota
	// ServiceCASource is the source for the StackRox internal service CA.
	ServiceCASource
)

// Options specifies details for a TLS configuration.
type Options struct {
	ServerCerts       []ServerCertSource
	ClientCAs         []ClientCASource
	RequireClientCert bool
}

// Manager is the manager interface for client CA certificates
type Manager interface {
	RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider authproviders.Provider)
	GetProviderForFingerprint(fingerprint string) authproviders.Provider
	TLSConfigurer(opts Options) (verifier.TLSConfigurer, error)
}

var (
	instance     *managerImpl
	instanceOnce sync.Once
)

// ManagerInstance returns the Manager.
func ManagerInstance() Manager {
	instanceOnce.Do(func() {
		i, err := newManager(env.Namespace.Setting())
		utils.CrashOnError(err)
		instance = i
	})
	return instance
}
