package mtls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/cloudflare/cfssl/config"
	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	cflog "github.com/cloudflare/cfssl/log"
	cfsigner "github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	certsPrefix = "/run/secrets/stackrox.io/certs/"
	// defaultCACertFilePath is where the certificate is stored.
	defaultCACertFilePath = certsPrefix + "ca.pem"
	// defaultCAKeyFilePath is where the key is stored.
	defaultCAKeyFilePath = certsPrefix + "ca-key.pem"

	// defaultCertFilePath is where the certificate is stored.
	defaultCertFilePath = certsPrefix + "cert.pem"
	// defaultKeyFilePath is where the key is stored.
	defaultKeyFilePath = certsPrefix + "key.pem"

	// To account for clock skew, set certificates to be valid some time in the past.
	beforeGracePeriod = 1 * time.Hour

	certLifetime = 365 * 24 * time.Hour
)

var (
	// serialMax is the max value to be used with `rand.Int` to obtain a `*big.Int` with 64 bits of random data
	// (i.e., 1 << 64).
	serialMax = func() *big.Int {
		max := big.NewInt(1)
		max.Lsh(max, 64)
		return max
	}()
)

func init() {
	// The cfssl library prints logs at Info level when it processes a
	// Certificate Signing Request (CSR) or issues a new certificate.
	// These logs do not help the user understand anything, so here
	// we adjust the log level to exclude them.
	cflog.Level = cflog.LevelWarning
}

var (
	// CentralSubject is the identity used in certificates for Central.
	CentralSubject = Subject{ServiceType: storage.ServiceType_CENTRAL_SERVICE, Identifier: "Central"}

	// SensorSubject is the identity used in certificates for Sensor.
	SensorSubject = Subject{ServiceType: storage.ServiceType_SENSOR_SERVICE, Identifier: "Sensor"}

	// ScannerSubject is the identity used in certificates for Scanner.
	ScannerSubject = Subject{ServiceType: storage.ServiceType_SCANNER_SERVICE, Identifier: "Scanner"}

	readCAOnce sync.Once
	caCert     *x509.Certificate
	caCertDER  []byte
	caCertErr  error
)

// IssuedCert is a representation of an issued certificate
type IssuedCert struct {
	CertPEM []byte
	KeyPEM  []byte
	ID      *storage.ServiceIdentity
}

// LeafCertificateFromFile reads a tls.Certificate (including private key and cert)
// from the canonical locations on non-central services.
func LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePathSetting.Setting(), keyFilePathSetting.Setting())
}

// loadCACertDER reads the PEM-decoded bytes of the cert from the local file system.
func loadCACertDER() ([]byte, error) {
	b, err := ioutil.ReadFile(caFilePathSetting.Setting())
	if err != nil {
		return nil, errors.Wrap(err, "file access")
	}
	decoded, _ := pem.Decode(b)
	if decoded == nil {
		return nil, fmt.Errorf("invalid PEM")
	}
	return decoded.Bytes, nil
}

// CACertPEM returns the PEM-encoded CA certificate.
func CACertPEM() ([]byte, error) {
	_, caDER, err := CACert()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert loading")
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caDER,
	}), nil
}

// CACert reads the cert from the local file system and returns the cert and the DER encoding.
func CACert() (*x509.Certificate, []byte, error) {
	readCAOnce.Do(func() {
		der, err := loadCACertDER()
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be decoded")
			return
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be parsed")
			return
		}
		caCert = cert
		caCertDER = der
	})
	return caCert, caCertDER, caCertErr
}

func signerFromCABytes(caCert, caKey []byte) (cfsigner.Signer, error) {
	parsedCa, err := helpers.ParseCertificatePEM(caCert)
	if err != nil {
		return nil, err
	}

	priv, err := helpers.ParsePrivateKeyPEMWithPassword(caKey, nil)
	if err != nil {
		return nil, err
	}

	return local.NewSigner(priv, parsedCa, cfsigner.DefaultSigAlgo(priv), signingPolicy())
}

func signer() (cfsigner.Signer, error) {
	return local.NewSignerFromFile(caFilePathSetting.Setting(), caKeyFilePathSetting.Setting(), signingPolicy())
}

func signingPolicy() *config.Signing {
	return &config.Signing{
		Default: &config.SigningProfile{
			Usage:    []string{"signing", "key encipherment", "server auth", "client auth"},
			Expiry:   certLifetime + beforeGracePeriod,
			Backdate: beforeGracePeriod,
			CSRWhitelist: &config.CSRWhitelist{
				PublicKey:          true,
				PublicKeyAlgorithm: true,
				SignatureAlgorithm: true,
			},
		},
	}
}

// IssueNewCertFromCA issues a certificate from the CA that is passed in
func IssueNewCertFromCA(subj Subject, caCert, caKey []byte) (cert *IssuedCert, err error) {
	returnErr := func(err error, prefix string) (*IssuedCert, error) {
		return nil, errors.Wrapf(err, "%s", prefix)
	}

	s, err := signerFromCABytes(caCert, caKey)
	if err != nil {
		return returnErr(err, "signer creation")
	}

	serial, err := RandomSerial()
	if err != nil {
		return returnErr(err, "serial generation")
	}
	csr := &cfcsr.CertificateRequest{
		KeyRequest: cfcsr.NewBasicKeyRequest(),
	}
	csrBytes, keyBytes, err := cfcsr.ParseRequest(csr)
	if err != nil {
		return returnErr(err, "request parsing")
	}

	req := cfsigner.SignRequest{
		Hosts:   subj.AllHostnames(),
		Request: string(csrBytes),
		Subject: &cfsigner.Subject{
			CN:           subj.CN(),
			Names:        []cfcsr.Name{subj.Name()},
			SerialNumber: serial.String(),
		},
	}
	certBytes, err := s.Sign(req)
	if err != nil {
		return returnErr(err, "signing")
	}

	return &IssuedCert{
		CertPEM: certBytes,
		KeyPEM:  keyBytes,
	}, nil
}

func validateSubject(subj Subject) error {
	errorList := errorhelpers.NewErrorList("")
	if subj.ServiceType == storage.ServiceType_UNKNOWN_SERVICE {
		errorList.AddString("Subject service type must be known")
	}
	if subj.Identifier == "" {
		errorList.AddString("Subject Identifier must be non-empty")
	}
	return errorList.ToError()
}

// IssueNewCert generates a new key and certificate chain for a sensor.
func IssueNewCert(subj Subject) (cert *IssuedCert, err error) {
	returnErr := func(err error, prefix string) (*IssuedCert, error) {
		return nil, errors.Wrapf(err, "%s", prefix)
	}

	if err := validateSubject(subj); err != nil {
		// Purposefully didn't use returnErr because errorList.ToError() returned from validateSubject is already prefixed
		return nil, err
	}

	s, err := signer()
	if err != nil {
		return returnErr(err, "signer creation")
	}

	serial, err := RandomSerial()
	if err != nil {
		return returnErr(err, "serial generation")
	}
	csr := &cfcsr.CertificateRequest{
		KeyRequest: cfcsr.NewBasicKeyRequest(),
	}
	csrBytes, keyBytes, err := cfcsr.ParseRequest(csr)
	if err != nil {
		return returnErr(err, "request parsing")
	}

	req := cfsigner.SignRequest{
		Hosts:   subj.AllHostnames(),
		Request: string(csrBytes),
		Subject: &cfsigner.Subject{
			CN:           subj.CN(),
			Names:        []cfcsr.Name{subj.Name()},
			SerialNumber: serial.String(),
		},
	}
	certBytes, err := s.Sign(req)
	if err != nil {
		return returnErr(err, "signing")
	}

	id := generateIdentity(subj, serial)

	return &IssuedCert{
		CertPEM: certBytes,
		KeyPEM:  keyBytes,
		ID:      id,
	}, nil
}

// RandomSerial returns a new integer that can be used as a certificate serial number (i.e., it is positive and contains
// 64 bits of random data).
func RandomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return nil, errors.Wrap(err, "serial number generation")
	}
	serial.Add(serial, big.NewInt(1)) // Serial numbers must be positive.
	return serial, nil
}

func generateIdentity(subj Subject, serial *big.Int) *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Id:   subj.Identifier,
		Type: subj.ServiceType,
		Srl: &storage.ServiceIdentity_SerialStr{
			SerialStr: serial.String(),
		},
	}
}
