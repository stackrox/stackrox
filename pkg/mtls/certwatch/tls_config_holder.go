package certwatch

import (
	"crypto/tls"
	"crypto/x509"
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	errNoTLSConfig = errors.New("no TLS config is available")
)

// TLSConfigHolder holds a pointer to the tls.Config instance and provides an ability to update it in runtime.
type TLSConfigHolder struct {
	rootTLSConfig *tls.Config

	serverCertSources []*[]tls.Certificate
	clientCASources   []*[]*x509.Certificate

	liveTLSConfig unsafe.Pointer
}

// NewTLSConfigHolder instantiates a new instance of TLSConfigHolder
func NewTLSConfigHolder(rootCfg *tls.Config) *TLSConfigHolder {
	return &TLSConfigHolder{
		rootTLSConfig: rootCfg,
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
		newTLSConfig.ClientAuth = tls.NoClientCert
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
