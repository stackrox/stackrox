package x509utils

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
)

// ConvertPEMToDERs converts the given certBytes to DER.
// Returns multiple DERs if multiple PEMs were passed.
func ConvertPEMToDERs(certBytes []byte) ([][]byte, error) {
	var result [][]byte

	restBytes := certBytes
	for {
		var decoded *pem.Block
		decoded, restBytes = pem.Decode(restBytes)

		if decoded == nil && len(result) == 0 {
			return nil, errors.New("invalid PEM")
		} else if decoded == nil {
			return result, nil
		}

		result = append(result, decoded.Bytes)
		if len(restBytes) == 0 {
			return result, nil
		}
	}
}

// ConvertPEMTox509Certs convert a PEM encoded certificate chain to a slice of x509 certificates.
func ConvertPEMTox509Certs(certBytes []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	derCerts, err := ConvertPEMToDERs(certBytes)
	if err != nil {
		return nil, err
	}

	for _, derCert := range derCerts {
		x509Cert, err := x509.ParseCertificate(derCert)
		if err != nil {
			return nil, errors.Wrap(err, "could not convert cert")
		}
		certs = append(certs, x509Cert)
	}
	return certs, nil
}
