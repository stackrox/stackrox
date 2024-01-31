package collector

import (
	"crypto/tls"
	"crypto/x509"
)

type insecureVerifier struct{}

func (v *insecureVerifier) VerifyPeerCertificate(_ *x509.Certificate, _ []*x509.Certificate, _ *tls.Config) error {
	return nil
}
