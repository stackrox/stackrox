package securedclustercertgen

import (
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

var scannerV2ServiceTypes = set.NewFrozenSet[storage.ServiceType](storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)
var scannerV4ServiceTypes = set.NewFrozenSet[storage.ServiceType](storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, storage.ServiceType_SCANNER_V4_DB_SERVICE)
var localScannerServiceTypes = scannerV2ServiceTypes.Union(scannerV4ServiceTypes)

var securedClusterServiceTypes = set.NewFrozenSet[storage.ServiceType](
	storage.ServiceType_SENSOR_SERVICE,
	storage.ServiceType_COLLECTOR_SERVICE,
	storage.ServiceType_ADMISSION_CONTROL_SERVICE)

var allSupportedServiceTypes = securedClusterServiceTypes.Union(localScannerServiceTypes)

type certIssuerImpl struct {
	serviceTypes             set.FrozenSet[storage.ServiceType]
	signingCA                mtls.CA
	secondaryCA              mtls.CA
	sensorSupportsCARotation bool
}

// IssueSecuredClusterCerts issues certificates for all the services of a secured cluster (including local scanner).
// It loads the CAs from disk and selects which CA to use for signing  based on Sensor capabilities
// and the optional Sensor CA fingerprint.
func IssueSecuredClusterCerts(namespace, clusterID string, sensorSupportsCARotation bool, sensorCAFingerprint string) (*storage.TypedServiceCertificateSet, error) {
	primaryCA, err := mtls.CAForSigning()
	if err != nil {
		return nil, errors.Wrap(err, "could not load CA for signing")
	}

	secondaryCA, err := mtls.SecondaryCAForSigning()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("Failed to load secondary CA for signing (certificates will still be issued): %v", err)
		}
	}

	return IssueSecuredClusterCertsWithCAs(namespace, clusterID, sensorSupportsCARotation, primaryCA, secondaryCA, sensorCAFingerprint)
}

// IssueSecuredClusterCertsWithCAs issues certificates for all the services of a secured cluster (including local scanner),
// allowing injection of a primary CA (mandatory) and secondary CA (optional). It selects which CA to use for signing
// based on Sensor capabilities and the optional Sensor CA fingerprint.
func IssueSecuredClusterCertsWithCAs(
	namespace string,
	clusterID string,
	sensorSupportsCARotation bool,
	primaryCA mtls.CA,
	secondaryCA mtls.CA,
	sensorCAFingerprint string,
) (*storage.TypedServiceCertificateSet, error) {
	if primaryCA == nil {
		return nil, errors.New("primary CA is required")
	}

	if secondaryCA != nil {
		if sensorSupportsCARotation {
			// If CA rotation is enabled, ensure the signing CA is the one that expires later.
			primaryCACert := primaryCA.Certificate()
			secondaryCACert := secondaryCA.Certificate()
			if secondaryCACert.NotAfter.After(primaryCACert.NotAfter) {
				primaryCA, secondaryCA = secondaryCA, primaryCA
			}
		} else if sensorCAFingerprint != "" && sensorCAFingerprint == cryptoutils.CertFingerprint(secondaryCA.Certificate()) {
			// If a CA fingerprint is provided, prefer the matching CA. Otherwise just use the primary CA.
			primaryCA, secondaryCA = secondaryCA, primaryCA
		}
	}

	certIssuer := certIssuerImpl{
		serviceTypes:             allSupportedServiceTypes,
		signingCA:                primaryCA,
		secondaryCA:              secondaryCA,
		sensorSupportsCARotation: sensorSupportsCARotation,
	}

	return certIssuer.issueCertificates(namespace, clusterID)
}

// IssueLocalScannerCerts issue certificates for a local scanner running in secured clusters.
func IssueLocalScannerCerts(namespace string, clusterID string) (*storage.TypedServiceCertificateSet, error) {
	// In any case, generate certificates for Scanner v2.
	serviceTypes := scannerV2ServiceTypes
	if features.ScannerV4.Enabled() {
		// Additionally, generate certificates for Scanner v4.
		serviceTypes = localScannerServiceTypes
	}

	ca, err := mtls.CAForSigning()
	if err != nil {
		return nil, errors.Wrap(err, "could not load CA for signing")
	}

	certIssuer := certIssuerImpl{
		serviceTypes:             serviceTypes,
		signingCA:                ca,
		sensorSupportsCARotation: false, // Local scanner doesn't need CA rotation support
	}

	return certIssuer.issueCertificates(namespace, clusterID)
}

func (c *certIssuerImpl) issueCertificates(namespace string, clusterID string) (*storage.TypedServiceCertificateSet, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the secured cluster")
	}

	var certIssueError error
	var caPem []byte

	serviceCerts := make([]*storage.TypedServiceCertificate, 0, c.serviceTypes.Cardinality())
	for _, serviceType := range c.serviceTypes.AsSlice() {
		ca, cert, err := c.certificateFor(serviceType, namespace, clusterID)
		if err != nil {
			certIssueError = multierror.Append(certIssueError, err)
			continue
		}
		serviceCerts = append(serviceCerts, cert)
		if caPem == nil {
			caPem = ca
		}
	}

	if certIssueError != nil {
		return nil, certIssueError
	}

	certsSet := &storage.TypedServiceCertificateSet{}
	if caPem != nil {
		certsSet.SetCaPem(caPem)
	}
	certsSet.SetServiceCerts(serviceCerts)

	// Populate CA bundle for rotation-capable Sensors
	if c.sensorSupportsCARotation {
		caBundlePem, err := c.buildCABundle()
		if err != nil {
			log.Warnf("Failed to build CA bundle for rotation-capable Sensor (certificates will still be issued): %v", err)
		} else {
			if caBundlePem != nil {
				certsSet.SetCaBundlePem(caBundlePem)
			} else {
				certsSet.ClearCaBundlePem()
			}
			log.Debug("Populated CA bundle for rotation-capable Sensor")
		}
	}

	return &certsSet, nil
}

func (c *certIssuerImpl) certificateFor(serviceType storage.ServiceType, namespace string, clusterID string) (caPem []byte, cert *storage.TypedServiceCertificate, err error) {
	certificates, err := c.generateServiceCertMap(serviceType, namespace, clusterID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generating certificate for service %s", serviceType)
	}
	caPem = certificates[mtls.CACertFileName]
	sc := &storage.ServiceCertificate{}
	if x := certificates[mtls.ServiceCertFileName]; x != nil {
		sc.SetCertPem(x)
	}
	if x := certificates[mtls.ServiceKeyFileName]; x != nil {
		sc.SetKeyPem(x)
	}
	cert = &storage.TypedServiceCertificate{}
	cert.SetServiceType(serviceType)
	cert.SetCert(sc)
	return caPem, cert, err
}

func (c *certIssuerImpl) generateServiceCertMap(serviceType storage.ServiceType, namespace string, clusterID string) (secretDataMap, error) {
	if !c.serviceTypes.Contains(serviceType) {
		return nil, errors.Errorf("service type %s is not supported",
			serviceType)
	}

	numServiceCertDataEntries := 3 // cert pem + key pem + ca pem
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	subject := mtls.NewSubject(clusterID, serviceType)
	issueOpts := []mtls.IssueCertOption{
		mtls.WithNamespace(namespace),
	}
	if err := certgen.IssueServiceCert(fileMap, c.signingCA, subject, "", issueOpts...); err != nil {
		return nil, errors.Wrap(err, "error generating service certificate")
	}
	certgen.AddCACertToFileMap(fileMap, c.signingCA)

	return fileMap, nil
}

// buildCABundle creates a PEM-concatenated CA bundle from available CA certificates.
// This bundle contains all CA certificates that Central trusts for CA rotation.
func (c *certIssuerImpl) buildCABundle() ([]byte, error) {
	var allCertsPEM []byte

	// Always include the primary CA
	allCertsPEM = append(allCertsPEM, c.signingCA.CertPEM()...)

	// Include secondary CA if it exists
	if c.secondaryCA != nil {
		allCertsPEM = append(allCertsPEM, c.secondaryCA.CertPEM()...)
	}

	return allCertsPEM, nil
}
