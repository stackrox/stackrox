package certwatch

import (
	"crypto/tls"
	"crypto/x509"
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/tlscheck"
)

var errNoTLSConfig = errors.New("no TLS config is available")

// TLSConfigHolder holds a pointer to the tls.Config instance and provides an ability to update it in runtime.
type TLSConfigHolder struct {
	rootTLSConfig *tls.Config
	// fallbackClientAuth overrides rootTLSConfig.ClientAuth if clientCASources is empty.
	fallbackClientAuth tls.ClientAuthType

	serverCertSources []*[]tls.Certificate
	clientCASources   []*[]*x509.Certificate

	customTLSCertVerifier tlscheck.TLSCertVerifier

	liveTLSConfig unsafe.Pointer
}

// NewTLSConfigHolder instantiates a new instance of TLSConfigHolder
func NewTLSConfigHolder(rootCfg *tls.Config, fallbackClientAuth tls.ClientAuthType) *TLSConfigHolder {
	return &TLSConfigHolder{
		rootTLSConfig:      rootCfg,
		fallbackClientAuth: fallbackClientAuth,
	}
}

// UpdateTLSConfig updates live tls.Config based on the recent certificates state.
func (c *TLSConfigHolder) UpdateTLSConfig() {
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
		newTLSConfig.ClientAuth = c.fallbackClientAuth
	}

	if c.customTLSCertVerifier != nil {
		newTLSConfig.InsecureSkipVerify = true
		newTLSConfig.VerifyPeerCertificate = tlscheck.VerifyPeerCertFunc(newTLSConfig, c.customTLSCertVerifier)
	}

	atomic.StorePointer(&c.liveTLSConfig, (unsafe.Pointer)(newTLSConfig))
}

func (c *TLSConfigHolder) liveConfig(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	liveCfg := (*tls.Config)(atomic.LoadPointer(&c.liveTLSConfig))
	if liveCfg == nil {
		return nil, errNoTLSConfig
	}
	return liveCfg, nil
}

// TLSConfig returns the latest version of tls.Config stored in memory.
func (c *TLSConfigHolder) TLSConfig() (*tls.Config, error) {
	rootCfg := c.rootTLSConfig.Clone()
	rootCfg.GetConfigForClient = c.liveConfig
	return rootCfg, nil
}

// AddServerCertSource adds server cert source.
func (c *TLSConfigHolder) AddServerCertSource(serverCertSource *[]tls.Certificate) {
	c.serverCertSources = append(c.serverCertSources, serverCertSource)
}

// AddClientCertSource adds client cert source.
func (c *TLSConfigHolder) AddClientCertSource(clientCertSource *[]*x509.Certificate) {
	c.clientCASources = append(c.clientCASources, clientCertSource)
}

// SetCustomCertVerifier adds a custom TLS certificate verifier.
func (c *TLSConfigHolder) SetCustomCertVerifier(customVerifier tlscheck.TLSCertVerifier) {
	c.customTLSCertVerifier = customVerifier
}
