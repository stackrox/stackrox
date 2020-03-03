package tlsconfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"
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

	errNoTLSConfig = errors.New("no TLS config is available")
)

type providerData struct {
	provider authproviders.Provider
	certs    []*x509.Certificate
}

type configurer struct {
	rootTLSConfig *tls.Config

	serverCertSources []*[]tls.Certificate
	clientCASources   []*[]*x509.Certificate

	liveTLSConfig unsafe.Pointer
}

func (c *configurer) updateTLSConfig() {
	newTLSConfig := c.rootTLSConfig.Clone()

	newTLSConfig.Certificates = nil
	for _, certSrc := range c.serverCertSources {
		newTLSConfig.Certificates = append(newTLSConfig.Certificates, *certSrc...)
	}

	clientCAs := x509.NewCertPool()
	hasClientCAs := false
	for _, clientCASrc := range c.clientCASources {
		for _, clientCA := range *clientCASrc {
			clientCAs.AddCert(clientCA)
			hasClientCAs = true
		}
	}
	if hasClientCAs {
		newTLSConfig.ClientCAs = clientCAs
	} else {
		newTLSConfig.ClientAuth = tls.NoClientCert
	}

	newTLSConfig.BuildNameToCertificate()
	atomic.StorePointer(&c.liveTLSConfig, (unsafe.Pointer)(newTLSConfig))
}

func (c *configurer) liveConfig(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	liveCfg := (*tls.Config)(atomic.LoadPointer(&c.liveTLSConfig))
	if liveCfg == nil {
		return nil, errNoTLSConfig
	}
	return liveCfg, nil
}

func (c *configurer) TLSConfig() (*tls.Config, error) {
	rootCfg := c.rootTLSConfig.Clone()
	rootCfg.GetConfigForClient = c.liveConfig
	return rootCfg, nil
}

type managerImpl struct {
	mutex sync.RWMutex

	internalTrustRoots []*x509.Certificate
	userTrustRoots     []*x509.Certificate

	defaultCerts  []tls.Certificate
	internalCerts []tls.Certificate

	providerIDToProviderData  map[string]providerData
	certFingerprintToProvider map[string]providerData

	configurers []*configurer
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
		internalTrustRoots:       trustRoots,
		internalCerts:            []tls.Certificate{*internalCert},
	}

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

func (m *managerImpl) UpdateDefaultCert(defaultCert *tls.Certificate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if defaultCert == nil {
		m.defaultCerts = nil
	} else {
		m.defaultCerts = []tls.Certificate{*defaultCert}
	}

	m.updateConfigurersNoLock()
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
	m.userTrustRoots = nil
	for _, pd := range m.providerIDToProviderData {
		for _, cert := range pd.certs {
			fingerprint := cryptoutils.CertFingerprint(cert)
			index[fingerprint] = pd
		}
		m.userTrustRoots = append(m.userTrustRoots, pd.certs...)
	}
	m.certFingerprintToProvider = index
	m.updateConfigurersNoLock()
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

// updateTLSConfigNoLock needs to be called while holding at least RLock from m.mutex
func (m *managerImpl) updateConfigurersNoLock() {
	for _, configurer := range m.configurers {
		configurer.updateTLSConfig()
	}
}

// TLSConfigurer is called once on server startup. It has to have enough data for tls.Listen() to be happy, so
// we compute a complete one. We can't change the contents of the config afterwards, so instead we tell the tls
// package to ask us every new connection what our config really should be, and pass them the latest cached config.
func (m *managerImpl) TLSConfigurer(opts Options) (verifier.TLSConfigurer, error) {
	// certPool and certs will be filled in dynamically
	rootCfg := verifier.DefaultTLSServerConfig(nil, nil)

	rootCfg.NameToCertificate = nil
	if opts.RequireClientCert {
		rootCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	configurer := &configurer{
		rootTLSConfig: rootCfg,
	}

	for _, serverCert := range opts.ServerCerts {
		switch serverCert {
		case DefaultTLSCertSource:
			configurer.serverCertSources = append(configurer.serverCertSources, &m.defaultCerts)
		case ServiceCertSource:
			configurer.serverCertSources = append(configurer.serverCertSources, &m.internalCerts)
		default:
			return nil, errors.Errorf("invalid server cert source %v", serverCert)
		}
	}

	for _, clientCA := range opts.ClientCAs {
		switch clientCA {
		case UserCAsSource:
			configurer.clientCASources = append(configurer.clientCASources, &m.userTrustRoots)
		case ServiceCASource:
			configurer.clientCASources = append(configurer.clientCASources, &m.internalTrustRoots)
		default:
			return nil, errors.Errorf("invalid client CA source %v", clientCA)
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.configurers = append(m.configurers, configurer)
	configurer.updateTLSConfig()
	return configurer, nil
}
