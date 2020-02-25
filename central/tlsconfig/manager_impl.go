package tlsconfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	watchInterval = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

type providerData struct {
	provider authproviders.Provider
	certs    []*x509.Certificate
}

func newManager() (*managerImpl, error) {
	ca, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	trustRoots := []*x509.Certificate{ca}

	internalCert, err := getInternalCertificate()
	if err != nil {
		return nil, err
	}

	mgr := &managerImpl{
		providerIDToProviderData: make(map[string]providerData),
		trustRoots:               trustRoots,
		internalCerts:            []tls.Certificate{*internalCert},
	}

	mgr.updateTLSConfigNoLock()

	wh := &defaultCertWatchHandler{
		mgr: mgr,
	}

	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), DefaultCertPath, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)

	return mgr, nil
}

type managerImpl struct {
	mutex sync.RWMutex

	cachedCfg  unsafe.Pointer
	trustRoots []*x509.Certificate

	defaultCerts  []tls.Certificate
	internalCerts []tls.Certificate

	providerIDToProviderData  map[string]providerData
	certFingerprintToProvider map[string]providerData
}

func (m *managerImpl) UpdateDefaultCert(defaultCert *tls.Certificate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if defaultCert == nil {
		m.defaultCerts = nil
	} else {
		m.defaultCerts = []tls.Certificate{*defaultCert}
	}

	m.updateTLSConfigNoLock()
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
	m.updateTLSConfigNoLock()
	log.Debugf("%d fingerprints registered", len(index))
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

func (m *managerImpl) getTLSConfig() (*tls.Config, error) {
	cfg := (*tls.Config)(atomic.LoadPointer(&m.cachedCfg))
	return cfg, nil
}

// updateTLSConfigNoLock needs to be called while holding at least RLock from m.mutex
func (m *managerImpl) updateTLSConfigNoLock() {
	cfg := m.computeConfigNoLock()
	atomic.StorePointer(&m.cachedCfg, unsafe.Pointer(cfg))
}

// computeConfigNoLock must be called while holding at least RLock from m.mutex
func (m *managerImpl) computeConfigNoLock() *tls.Config {
	clientCAs := x509.NewCertPool()
	for _, c := range m.trustRoots {
		clientCAs.AddCert(c)
	}

	for _, pd := range m.providerIDToProviderData {
		for _, cert := range pd.certs {
			log.Debugf("Adding client CA cert to the TLS trust pool: %q", cryptoutils.CertFingerprint(cert))
			clientCAs.AddCert(cert)
		}
	}

	serverCerts := make([]tls.Certificate, 0, len(m.defaultCerts)+len(m.internalCerts))
	serverCerts = append(serverCerts, m.defaultCerts...)
	serverCerts = append(serverCerts, m.internalCerts...)
	cfg := verifier.DefaultTLSServerConfig(clientCAs, serverCerts)
	return cfg
}

// TLSConfigurer is called once on server startup. It has to have enough data for tls.Listen() to be happy, so
// we compute a complete one. We can't change the contents of the config afterwards, so instead we tell the tls
// package to ask us every new connection what our config really should be, and pass them the latest cached config.
func (m *managerImpl) TLSConfigurer() verifier.TLSConfigurer {
	return verifier.TLSConfigurerFunc(func() (*tls.Config, error) {
		m.mutex.RLock()
		defer m.mutex.RUnlock()

		cfg := m.computeConfigNoLock()
		cfg.GetConfigForClient = func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			return m.getTLSConfig()
		}
		return cfg, nil
	})
}
