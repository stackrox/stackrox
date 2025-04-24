package certgen

import (
	"bytes"
	"errors"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	ErrNoCACert = errors.New("no CA certificate in file map")
	ErrNoCAKey  = errors.New("no CA key in file map")
)

const caCertExpiry = 5 * 365 * 24 * time.Hour

// LoadCAFromFileMap loads and instantiates a StackRox service CA from the given file map. The file map
// must contain `ca-cert.pem` and `ca-key.pem` entries.
func LoadCAFromFileMap(fileMap map[string][]byte) (mtls.CA, error) {
	caCertPEM := fileMap[mtls.CACertFileName]
	if len(caCertPEM) == 0 {
		return nil, ErrNoCACert
	}
	caKeyPEM := fileMap[mtls.CAKeyFileName]
	if len(caKeyPEM) == 0 {
		return nil, ErrNoCAKey
	}
	return mtls.LoadCAForSigning(caCertPEM, caKeyPEM)
}

// LoadSecondaryCAFromFileMap loads and instantiates a StackRox service secondary CA from the given file map.
// The file map must contain `ca-secondary.pem` and `ca-secondary-key.pem` entries.
// A secondary CA is optional. Operator installations use two CA certificates in parallel
// to enable CA certificate rotation.
func LoadSecondaryCAFromFileMap(fileMap map[string][]byte) (mtls.CA, error) {
	secondaryCACertPEM := fileMap[mtls.SecondaryCACertFileName]
	if len(secondaryCACertPEM) == 0 {
		return nil, ErrNoCACert
	}
	secondaryCAKeyPEM := fileMap[mtls.SecondaryCAKeyFileName]
	if len(secondaryCAKeyPEM) == 0 {
		return nil, ErrNoCAKey
	}
	return mtls.LoadCAForSigning(secondaryCACertPEM, secondaryCAKeyPEM)
}

// AddCAToFileMap adds the CA cert and key to the given file map
func AddCAToFileMap(fileMap map[string][]byte, ca mtls.CA) {
	fileMap[mtls.CACertFileName] = ca.CertPEM()
	fileMap[mtls.CAKeyFileName] = ca.KeyPEM()
}

// AddSecondaryCAToFileMap adds the secondary CA cert and key to the given file map
func AddSecondaryCAToFileMap(fileMap map[string][]byte, ca mtls.CA) {
	fileMap[mtls.SecondaryCACertFileName] = ca.CertPEM()
	fileMap[mtls.SecondaryCAKeyFileName] = ca.KeyPEM()
}

// PromoteSecondaryCA promotes the secondary CA to primary CA, by swapping the two CA cert + key pairs
func PromoteSecondaryCA(fileMap map[string][]byte) {
	fileMap[mtls.CACertFileName], fileMap[mtls.SecondaryCACertFileName] =
		fileMap[mtls.SecondaryCACertFileName], fileMap[mtls.CACertFileName]
	fileMap[mtls.CAKeyFileName], fileMap[mtls.SecondaryCAKeyFileName] =
		fileMap[mtls.SecondaryCAKeyFileName], fileMap[mtls.CAKeyFileName]
}

// RemoveSecondaryCA removes the secondary CA from the file map
func RemoveSecondaryCA(fileMap map[string][]byte) {
	delete(fileMap, mtls.SecondaryCACertFileName)
	delete(fileMap, mtls.SecondaryCAKeyFileName)
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
		return ErrNoCACert
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
		CA: &csr.CAConfig{
			Expiry: caCertExpiry.String(),
		},
	}
	caCert, _, caKey, err := initca.New(&req)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "could not generate keypair")
	}
	return mtls.LoadCAForSigning(caCert, caKey)
}
