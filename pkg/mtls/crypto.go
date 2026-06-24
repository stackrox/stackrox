package mtls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	// CACertFileName is the canonical file name (basename) of the file storing the CA certificate.
	CACertFileName = "ca.pem"
	// CAKeyFileName is the canonical file name (basename) of the file storing the CA certificate private key.
	CAKeyFileName = "ca-key.pem"

	// SecondaryCACertFileName is the file name of the secondary CA certificate.
	// Operator installations use two CA certificates in parallel to enable CA certificate rotation.
	SecondaryCACertFileName = "ca-secondary.pem"

	// SecondaryCAKeyFileName is the file name of the secondary CA private key.
	SecondaryCAKeyFileName = "ca-secondary-key.pem"

	// ServiceCertFileName is the canonical file name (basename) of the file storing the public part of
	// an internal service certificate. Note that if files for several services are stored in the same
	// location (directory or file map), it is common to prefix the file name with the service name in
	// slug-case (e.g., `scanner-db-cert.pem`).
	ServiceCertFileName = "cert.pem"
	// ServiceKeyFileName is the canonical file name (basename) of the file storing the private key for
	// an internal service certificate. The same remark as above regarding prefixes applies.
	ServiceKeyFileName = "key.pem"

	// CertsPrefix is the filesystem prefix under which service certificates and keys are stored.
	CertsPrefix = "/run/secrets/stackrox.io/certs/"
	// defaultCACertFilePath is where the certificate is stored.
	defaultCACertFilePath = CertsPrefix + CACertFileName
	// defaultCAKeyFilePath is where the key is stored.
	defaultCAKeyFilePath = CertsPrefix + CAKeyFileName
	// defaultSecondaryCACertFilePath is where the secondary CA certificate is stored.
	defaultSecondaryCACertFilePath = CertsPrefix + SecondaryCACertFileName
	// defaultSecondaryCAKeyFilePath is where the key of the secondary CA certificate is stored.
	defaultSecondaryCAKeyFilePath = CertsPrefix + SecondaryCAKeyFileName

	// defaultCertFilePath is where the certificate is stored.
	defaultCertFilePath = CertsPrefix + ServiceCertFileName
	// defaultKeyFilePath is where the key is stored.
	defaultKeyFilePath = CertsPrefix + ServiceKeyFileName

	// To account for clock skew, set certificates to be valid some time in the past.
	beforeGracePeriod = 1 * time.Hour

	certLifetime = 365 * 24 * time.Hour

	ephemeralProfileWithExpirationInHours             = "ephemeralWithExpirationInHours"
	ephemeralProfileWithExpirationInHoursCertLifetime = 3 * time.Hour // NB: keep in sync with operator's InitBundleReconcilePeriod

	ephemeralProfileWithExpirationInDays             = "ephemeralWithExpirationInDays"
	ephemeralProfileWithExpirationInDaysCertLifetime = 2 * 24 * time.Hour
)

var (
	log = logging.LoggerForModule()

	// serialMax is the max value to be used with `rand.Int` to obtain a `*big.Int` with 64 bits of random data
	// (i.e., 1 << 64).
	serialMax = func() *big.Int {
		max := big.NewInt(1)
		max.Lsh(max, 64)
		return max
	}()
)

var (
	// CentralSubject is the identity used in certificates for Central.
	CentralSubject = Subject{ServiceType: storage.ServiceType_CENTRAL_SERVICE, Identifier: "Central"}

	// CentralDBSubject is the identity used in certificates for Central DB.
	CentralDBSubject = Subject{ServiceType: storage.ServiceType_CENTRAL_DB_SERVICE, Identifier: "Central DB"}

	// SensorSubject is the identity used in certificates for Sensor.
	SensorSubject = Subject{ServiceType: storage.ServiceType_SENSOR_SERVICE, Identifier: "Sensor"}

	// AdmissionControlSubject is the identity used in certificates for Admission Control.
	AdmissionControlSubject = Subject{ServiceType: storage.ServiceType_ADMISSION_CONTROL_SERVICE, Identifier: "Admission Control"}

	// ScannerSubject is the identity used in certificates for Scanner.
	ScannerSubject = Subject{ServiceType: storage.ServiceType_SCANNER_SERVICE, Identifier: "Scanner"}

	// ScannerDBSubject is the identity used in certificates for Scanners Postgres DB
	ScannerDBSubject = Subject{ServiceType: storage.ServiceType_SCANNER_DB_SERVICE, Identifier: "Scanner DB"}

	// ScannerV4IndexerSubject is the identity used in certificates for Scanner V4 Indexer.
	ScannerV4IndexerSubject = Subject{ServiceType: storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, Identifier: "Scanner V4 Indexer"}

	// ScannerV4MatcherSubject is the identity used in certificates for Scanner V4 Matcher.
	ScannerV4MatcherSubject = Subject{ServiceType: storage.ServiceType_SCANNER_V4_MATCHER_SERVICE, Identifier: "Scanner V4 Matcher"}

	// ScannerV4DBSubject is the identity used in certificates for Scanner V4 DB.
	ScannerV4DBSubject = Subject{ServiceType: storage.ServiceType_SCANNER_V4_DB_SERVICE, Identifier: "Scanner V4 DB"}

	// ScannerV4Subject is the identity used in certificates for Scanner V4 running in combo-mode (testing, only).
	ScannerV4Subject = Subject{ServiceType: storage.ServiceType_SCANNER_V4_SERVICE, Identifier: "Scanner V4"}

	readCACertOnce     sync.Once
	caCert             *x509.Certificate
	caCertDER          []byte
	caCertFileContents []byte
	caCertErr          error

	readSecondaryCACertOnce     sync.Once
	secondaryCACert             *x509.Certificate
	secondaryCACertDER          []byte
	secondaryCACertFileContents []byte
	secondaryCACertErr          error

	readCAKeyOnce     sync.Once
	caKeyFileContents []byte
	caKeyErr          error

	readSecondaryCAKeyOnce     sync.Once
	secondaryCAKeyFileContents []byte
	secondaryCAKeyErr          error

	caForSigningOnce sync.Once
	caForSigning     CA
	caForSigningErr  error

	secondaryCAForSigningOnce sync.Once
	secondaryCAForSigning     CA
	secondaryCAForSigningErr  error
)

// IssuedCert is a representation of an issued certificate
type IssuedCert struct {
	CertPEM  []byte
	KeyPEM   []byte
	X509Cert *x509.Certificate
	ID       *storage.ServiceIdentity
}

// LeafCertificateFromFile reads a tls.Certificate (including private key and cert).
func LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePathSetting.Setting(), keyFilePathSetting.Setting())
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

func readCAKey() ([]byte, error) {
	readCAKeyOnce.Do(func() {
		caKeyBytes, err := os.ReadFile(caKeyFilePathSetting.Setting())
		if err != nil {
			caKeyErr = errors.Wrap(err, "reading CA key")
			return
		}
		caKeyFileContents = caKeyBytes
	})
	return caKeyFileContents, caKeyErr
}

func readCA() (*x509.Certificate, []byte, []byte, error) {
	readCACertOnce.Do(func() {
		caCert, caCertFileContents, caCertDER, caCertErr = readCAFromFile(caFilePathSetting.Setting())
	})
	return caCert, caCertFileContents, caCertDER, caCertErr
}

func readSecondaryCAKey() ([]byte, error) {
	readSecondaryCAKeyOnce.Do(func() {
		caKeyBytes, err := os.ReadFile(secondaryCAKeyFilePathSetting.Setting())
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Warnf("Failed to read secondary CA key, some Sensors may not be able to connect to Central: %v", err)
			}

			secondaryCAKeyErr = errors.Wrap(err, "reading secondary CA key")
			return
		}
		secondaryCAKeyFileContents = caKeyBytes
	})
	return secondaryCAKeyFileContents, secondaryCAKeyErr
}

func readSecondaryCA() (*x509.Certificate, []byte, []byte, error) {
	readSecondaryCACertOnce.Do(func() {
		secondaryCACert, secondaryCACertFileContents, secondaryCACertDER, secondaryCACertErr = readCAFromFile(
			secondaryCAFilePathSetting.Setting())

		if secondaryCACertErr != nil && !errors.Is(secondaryCACertErr, os.ErrNotExist) {
			log.Warnf("Failed to read secondary CA cert, some Sensors may not be able to connect to Central: %v", secondaryCACertErr)
		}
	})
	return secondaryCACert, secondaryCACertFileContents, secondaryCACertDER, secondaryCACertErr
}

func readCAFromFile(filePath string) (*x509.Certificate, []byte, []byte, error) {
	caBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "reading CA file")
	}

	der, err := x509utils.ConvertPEMToDERs(caBytes)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "CA cert could not be decoded")
	}
	if len(der) == 0 {
		return nil, nil, nil, errors.New("reading CA file failed")
	}

	cert, err := x509.ParseCertificate(der[0])
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "CA cert could not be parsed")

	}
	return cert, caBytes, der[0], nil
}

// CACert reads the cert from the local file system and returns the cert and the DER encoding.
func CACert() (*x509.Certificate, []byte, error) {
	caCert, _, caCertDER, caCertErr := readCA()
	return caCert, caCertDER, caCertErr
}

// SecondaryCACert reads the secondary CA cert from the local file system and returns the cert and the DER encoding.
// Note that the secondary CA cert is optional, and may only be present in Operator-based installations.
func SecondaryCACert() (*x509.Certificate, []byte, error) {
	caCert, _, caCertDER, caCertErr := readSecondaryCA()
	return caCert, caCertDER, caCertErr
}

// CAForSigning reads the cert and key from the local file system and returns
// a corresponding CA instance that can be used for signing.
func CAForSigning() (CA, error) {
	caForSigningOnce.Do(func() {
		_, certPEM, _, err := readCA()
		if err != nil {
			caForSigningErr = errors.Wrap(err, "could not read CA cert file")
			return
		}
		keyPEM, err := readCAKey()
		if err != nil {
			caForSigningErr = errors.Wrap(err, "could not read CA key file")
			return
		}

		caForSigning, caForSigningErr = LoadCAForSigning(certPEM, keyPEM)
	})

	return caForSigning, caForSigningErr
}

func SecondaryCAForSigning() (CA, error) {
	secondaryCAForSigningOnce.Do(func() {
		_, certPEM, _, err := readSecondaryCA()
		if err != nil {
			secondaryCAForSigningErr = errors.Wrap(err, "could not read secondary CA certificate PEM")
			return
		}

		keyPEM, err := readSecondaryCAKey()
		if err != nil {
			secondaryCAForSigningErr = errors.Wrap(err, "could not read secondary CA key PEM")
			return
		}

		secondaryCAForSigning, secondaryCAForSigningErr = LoadCAForSigning(certPEM, keyPEM)
	})

	return secondaryCAForSigning, secondaryCAForSigningErr
}

type certProfile struct {
	lifetime    time.Duration
	gracePeriod time.Duration
}

var certProfiles = map[string]certProfile{
	"":                                    {lifetime: certLifetime, gracePeriod: beforeGracePeriod},
	ephemeralProfileWithExpirationInHours: {lifetime: ephemeralProfileWithExpirationInHoursCertLifetime},
	ephemeralProfileWithExpirationInDays:  {lifetime: ephemeralProfileWithExpirationInDaysCertLifetime},
}

func loadCAFromFiles() (*x509.Certificate, crypto.Signer, error) {
	caCert, _, _, err := readCA()
	if err != nil {
		return nil, nil, errors.Wrap(err, "reading CA cert")
	}
	keyPEM, err := readCAKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "reading CA key")
	}
	caKey, err := x509utils.ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing CA key")
	}
	return caCert, caKey, nil
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

func issueCert(subj Subject, caCert *x509.Certificate, caKey crypto.Signer, opts []IssueCertOption) (*IssuedCert, error) {
	if err := validateSubject(subj); err != nil {
		return nil, err
	}

	serial, err := RandomSerial()
	if err != nil {
		return nil, errors.Wrap(err, "serial generation")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "key generation")
	}

	var issueOpts issueOptions
	issueOpts.apply(opts)

	profile, ok := certProfiles[issueOpts.signerProfile]
	if !ok {
		return nil, errors.Errorf("unknown signer profile %q", issueOpts.signerProfile)
	}

	now := time.Now()
	notBefore := issueOpts.notBefore
	if notBefore.IsZero() {
		notBefore = now.Add(-profile.gracePeriod)
	}
	notAfter := issueOpts.expiresAt
	if notAfter.IsZero() {
		notAfter = notBefore.Add(profile.lifetime + profile.gracePeriod)
	}

	var hosts []string
	hosts = append(hosts, subj.AllHostnames()...)
	if ns := issueOpts.namespace; ns != "" && ns != namespaces.StackRox {
		hosts = append(hosts, subj.AllHostnamesForNamespace(ns)...)
	}

	name := subj.Name()
	name.CommonName = subj.CN()
	name.SerialNumber = serial.String()

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               name,
		DNSNames:              hosts,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "certificate creation")
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "key marshaling")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	x509Cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, errors.Wrap(err, "parsing generated certificate")
	}

	return &IssuedCert{
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
		X509Cert: x509Cert,
		ID:       generateIdentity(subj, serial),
	}, nil
}

// IssueNewCert generates a new key and certificate chain for a sensor.
func IssueNewCert(subj Subject, opts ...IssueCertOption) (cert *IssuedCert, err error) {
	caCert, caKey, err := loadCAFromFiles()
	if err != nil {
		return nil, errors.Wrap(err, "loading CA")
	}
	return issueCert(subj, caCert, caKey, opts)
}

// IssueNewCrsCert generates a new key and certificate chain for a CRS.
func IssueNewCrsCert(crsId uuid.UUID, validUntil time.Time) (cert *IssuedCert, err error) {
	subj := NewInitSubject(centralsensor.RegisteredInitCertClusterID, storage.ServiceType_REGISTRANT_SERVICE, crsId)
	caCert, caKey, err := loadCAFromFiles()
	if err != nil {
		return nil, errors.Wrap(err, "loading CA")
	}
	return issueCert(subj, caCert, caKey, []IssueCertOption{
		WithValidityNotAfter(validUntil),
	})
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
		Id:        subj.Identifier,
		Type:      subj.ServiceType,
		SerialStr: serial.String(),
	}
}
