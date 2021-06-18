package x509utils

import (
	"crypto/x509"
	"io/ioutil"

	"github.com/cloudflare/cfssl/helpers"
)

// LoadCertificatePEMFile loads a PEM-encoded certificate from a file.
func LoadCertificatePEMFile(filename string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return helpers.ParseCertificatePEM(data)
}
