package manager

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type providerData struct {
	provider authproviders.Provider
	certs    []*x509.Certificate
}

type managerImpl struct {
	mutex sync.RWMutex

	providerIDToProviderData  map[string]providerData
	certFingerprintToProvider map[string]providerData
}

func (m *managerImpl) GetProviderForFingerprint(fingerprint string) authproviders.Provider {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	pd, ok := m.certFingerprintToProvider[fingerprint]
	if ok {
		return pd.provider
	}
	return nil
}

func (m *managerImpl) reindexNoLock() {
	index := make(map[string]providerData)
	for _, pd := range m.providerIDToProviderData {
		for _, cert := range pd.certs {
			fingerprint := cryptoutils.CertFingerprint(cert)
			index[fingerprint] = pd
		}
	}
	m.certFingerprintToProvider = index
	log.Debugf("%d fingerprints registered: %+v", len(index), index)
}

func (m *managerImpl) RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate) {
	id := provider.ID()
	if id == "" {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Debugf("Provider %q registered with %d certificates", id, len(certs))
	m.providerIDToProviderData[id] = providerData{
		provider: provider,
		certs:    certs,
	}
	m.reindexNoLock()
}

func (m *managerImpl) UnregisterAuthProvider(provider authproviders.Provider) {
	id := provider.ID()
	if id == "" {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Debugf("Provider %q unregistered", id)
	delete(m.providerIDToProviderData, id)
	m.reindexNoLock()
}

func (m *managerImpl) TLSConfigurer() verifier.TLSConfigurer {
	return verifier.TLSConfigurerFunc(func() (*tls.Config, error) {
		cfg, err := tlsconfig.NewCentralTLSConfigurer().TLSConfig()
		if err != nil {
			return nil, err
		}
		m.mutex.RLock()
		defer m.mutex.RUnlock()
		for _, pd := range m.providerIDToProviderData {
			for _, cert := range pd.certs {
				log.Debugf("Adding client CA cert to the TLS trust pool: %q", (cryptoutils.CertFingerprint(cert)))
				cfg.ClientCAs.AddCert(cert)
			}
		}
		return cfg, nil
	})
}
