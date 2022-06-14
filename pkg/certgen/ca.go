package certgen

import (
	"bytes"
	"errors"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	errNoCACert = errors.New("no CA certificate in file map")
	errNoCAKey  = errors.New("no CA key in file map")
)

// LoadCAFromFileMap loads and instantiates a StackRox service CA from the given file map. The file map
// must contain `ca-cert.pem` and `ca-key.pem` entries.
func LoadCAFromFileMap(fileMap map[string][]byte) (mtls.CA, error) {
	caCertPEM := fileMap[mtls.CACertFileName]
	if len(caCertPEM) == 0 {
		return nil, errNoCACert
	}
	caKeyPEM := fileMap[mtls.CAKeyFileName]
	if len(caKeyPEM) == 0 {
		return nil, errNoCAKey
	}
	return mtls.LoadCAForSigning(caCertPEM, caKeyPEM)
}

// AddCAToFileMap adds the CA cert and key to the given file map
func AddCAToFileMap(fileMap map[string][]byte, ca mtls.CA) {
	fileMap[mtls.CACertFileName] = ca.CertPEM()
	fileMap[mtls.CAKeyFileName] = ca.KeyPEM()
}

// AddCACertToFileMap adds the public CA certificate only to the file map.
func AddCACertToFileMap(fileMap map[string][]byte, ca mtls.CA) {
	fileMap[mtls.CACertFileName] = ca.CertPEM()
}

// VerifyCACertInFileMap verifies that the public CA certificate stored in the given file
// map is the same as the given one.
func VerifyCACertInFileMap(fileMap map[string][]byte, ca mtls.CA) error {
	caCertPEM := fileMap[mtls.CACertFileName]
	if len(caCertPEM) == 0 {
		return errNoCACert
	}
	caCert, err := helpers.ParseCertificatePEM(caCertPEM)
	if err != nil {
		return pkgErrors.Wrap(err, "unparseable CA certificate in file map")
	}
	if !bytes.Equal(caCert.Raw, ca.Certificate().Raw) {
		return errors.New("mismatching CA certificate in file map")
	}
	return nil
}

// GenerateCA generates a new StackRox service CA.
func GenerateCA() (mtls.CA, error) {
	serial, err := mtls.RandomSerial()
	if err != nil {
		return nil, pkgErrors.Wrap(err, "could not generate a serial number")
	}
	req := csr.CertificateRequest{
		CN:           mtls.ServiceCACommonName,
		KeyRequest:   csr.NewKeyRequest(),
		SerialNumber: serial.String(),
	}
	caCert, _, caKey, err := initca.New(&req)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "could not generate keypair")
	}
	return mtls.LoadCAForSigning(caCert, caKey)
}
