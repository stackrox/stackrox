package mtls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/cloudflare/cfssl/config"
	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	cfsigner "github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	certsPrefix = "/run/secrets/stackrox.io/certs/"
	// caCertFilePath is where the certificate is stored.
	caCertFilePath = certsPrefix + "ca.pem"
	// caKeyFilePath is where the key is stored.
	caKeyFilePath = certsPrefix + "ca-key.pem"

	// CertFilePath is where the certificate is stored.
	CertFilePath = certsPrefix + "cert.pem"
	// KeyFilePath is where the key is stored.
	KeyFilePath = certsPrefix + "key.pem"

	// To account for clock skew, set certificates to be valid some time in the past.
	beforeGracePeriod = 1 * time.Hour

	certLifetime = 365 * 24 * time.Hour
)

var (
	// CentralSubject is the identity used in certificates for Central.
	CentralSubject = Subject{ServiceType: storage.ServiceType_CENTRAL_SERVICE, Identifier: "Central"}

	// SensorSubject is the identity used in certificates for Sensor.
	SensorSubject = Subject{ServiceType: storage.ServiceType_SENSOR_SERVICE, Identifier: "Sensor"}

	// BenchmarkSubject is the identity used in certificates for Benchmark
	BenchmarkSubject = Subject{ServiceType: storage.ServiceType_BENCHMARK_SERVICE, Identifier: "Benchmark"}

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
	return tls.LoadX509KeyPair(CertFilePath, KeyFilePath)
}

// CACertDER reads the PEM-decoded bytes of the cert from the local file system.
func CACertDER() ([]byte, error) {
	b, err := ioutil.ReadFile(caCertFilePath)
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
		der, err := CACertDER()
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
	return local.NewSignerFromFile(caCertFilePath, caKeyFilePath, signingPolicy())
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

	serial, err := randomSerial()
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
			SerialNumber: strconv.FormatInt(serial, 10),
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

	serial, err := randomSerial()
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
			SerialNumber: strconv.FormatInt(serial, 10),
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

func randomSerial() (int64, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0, errors.Wrap(err, "serial number generation")
	}
	return serial.Int64(), nil
}

func generateIdentity(subj Subject, serial int64) *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Id:     subj.Identifier,
		Type:   subj.ServiceType,
		Serial: serial,
	}
}
