package x509utils

import (
	"crypto/x509"

	"github.com/pkg/errors"
)

// ParseCertificateChain parses the given chain of DER-encoded certificates.
// If there is an error parsing any single cert in the chain, this error is returned along with the partial
// result of successfully parsed certs. This allows determining which cert in the encoded chain failed to parse.
func ParseCertificateChain(derChain [][]byte) ([]*x509.Certificate, error) {
	chain := make([]*x509.Certificate, 0, len(derChain))
	for _, der := range derChain {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return chain, err
		}
		chain = append(chain, cert)
	}
	return chain, nil
}

// VerifyCertificateChain verifies the leaf (first element) in a certificate chain, treating the remaining certificates
// in the chain as intermediate certificates.
// Verification is performed relative to the given VerifyOptions, with the exception of `Intermediates` - this field is
// not taken into consideration at all.
func VerifyCertificateChain(chain []*x509.Certificate, verifyOpts x509.VerifyOptions) error {
	if len(chain) == 0 {
		return errors.New("empty certificate chain")
	}

	leaf := chain[0]
	verifyOpts.Intermediates = x509.NewCertPool()
	for _, intermediate := range chain[1:] {
		verifyOpts.Intermediates.AddCert(intermediate)
	}

	_, err := leaf.Verify(verifyOpts)
	return err
}
