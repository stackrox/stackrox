package certwatch

import (
	"crypto/tls"
	"crypto/x509"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/tlscheck"
)

var log = logging.LoggerForModule()

var errNoTLSConfig = errors.New("no TLS config is available")

// TLSConfigHolder holds a pointer to the tls.Config instance and provides an ability to update it in runtime.
type TLSConfigHolder struct {
	rootTLSConfig *tls.Config
	// fallbackClientAuth overrides rootTLSConfig.ClientAuth if clientCASources is empty.
	fallbackClientAuth tls.ClientAuthType

	serverCertSources []*[]tls.Certificate
	clientCASources   []*[]*x509.Certificate

	customTLSCertVerifier tlscheck.TLSCertVerifier

	liveTLSConfig atomic.Pointer[tls.Config]
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

	// Add custom GetCertificate to log certificate selection
	newTLSConfig.GetCertificate = c.logCertSelection

	c.liveTLSConfig.Store(newTLSConfig)
}

func (c *TLSConfigHolder) logCertSelection(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	liveCfg := c.liveTLSConfig.Load()
	if liveCfg == nil {
		return nil, errNoTLSConfig
	}

	sni := clientHello.ServerName
	if sni == "" {
		sni = "<empty>"
	}

	log.Infof("TLS connection: SNI=%s, available certificates=%d", sni, len(liveCfg.Certificates))

	// Log details about each available certificate
	for i, cert := range liveCfg.Certificates {
		if cert.Leaf != nil {
			log.Infof("  Cert[%d]: CN=%s, DNSNames=%v", i, cert.Leaf.Subject.CommonName, cert.Leaf.DNSNames)
		} else if len(cert.Certificate) > 0 {
			// Parse the certificate if Leaf is not populated
			parsed, err := x509.ParseCertificate(cert.Certificate[0])
			if err == nil {
				log.Infof("  Cert[%d]: CN=%s, DNSNames=%v", i, parsed.Subject.CommonName, parsed.DNSNames)
			} else {
				log.Infof("  Cert[%d]: <unable to parse>", i)
			}
		}
	}

	// Let Go's default certificate selection do its thing, but log the result
	selectedCert, err := tlsConfigGetCertificate(liveCfg, clientHello)
	if err != nil {
		log.Warnf("Certificate selection failed for SNI=%s: %v", sni, err)
		return nil, err
	}

	if selectedCert != nil && selectedCert.Leaf != nil {
		log.Infof("Selected certificate: CN=%s, DNSNames=%v", selectedCert.Leaf.Subject.CommonName, selectedCert.Leaf.DNSNames)
	} else if selectedCert != nil && len(selectedCert.Certificate) > 0 {
		parsed, err := x509.ParseCertificate(selectedCert.Certificate[0])
		if err == nil {
			log.Infof("Selected certificate: CN=%s, DNSNames=%v", parsed.Subject.CommonName, parsed.DNSNames)
		}
	}

	return selectedCert, nil
}

// tlsConfigGetCertificate implements Go's default certificate selection logic
func tlsConfigGetCertificate(config *tls.Config, clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// This mimics what crypto/tls does internally
	// If there are no certificates, return error
	if len(config.Certificates) == 0 {
		return nil, errors.New("no certificates configured")
	}

	// Try to find a certificate that matches the SNI
	if clientHello.ServerName != "" {
		for i := range config.Certificates {
			cert := &config.Certificates[i]
			if err := matchCertificate(cert, clientHello.ServerName); err == nil {
				return cert, nil
			}
		}
	}

	// Fallback to the first certificate
	return &config.Certificates[0], nil
}

// matchCertificate checks if a certificate matches the given server name
func matchCertificate(cert *tls.Certificate, serverName string) error {
	var leaf *x509.Certificate
	if cert.Leaf != nil {
		leaf = cert.Leaf
	} else if len(cert.Certificate) > 0 {
		var err error
		leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return err
		}
	}

	if leaf == nil {
		return errors.New("no leaf certificate")
	}

	return leaf.VerifyHostname(serverName)
}

func (c *TLSConfigHolder) liveConfig(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	liveCfg := c.liveTLSConfig.Load()
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
