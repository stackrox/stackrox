package tlscheck

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
)

// TLSCertVerifier is a more flexible variant of the VerifyPeerCertificate callback used in TLS configs.
type TLSCertVerifier interface {
	VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, conf *tls.Config) error
}

// NewCertPool is a convenience function that creates a CertPool out of a variadic list of certificates.
func NewCertPool(certs ...*x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool
}

// VerifyPeerCertFunc returns a custom certificate verifier suitable for as a VerifyPeerCertificate callback.
func VerifyPeerCertFunc(conf *tls.Config, verifier TLSCertVerifier) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("remote peer presented no certificates")
		}

		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return errors.Wrap(err, "failed to parse peer certificate")
			}
			certs = append(certs, cert)
		}

		return verifier.VerifyPeerCertificate(certs[0], certs[1:], conf)
	}
}
