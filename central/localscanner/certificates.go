package localscanner

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"golang.org/x/exp/slices"
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

var (
	supportedServices = func() []storage.ServiceType {
		svcs := []storage.ServiceType{
			storage.ServiceType_SCANNER_SERVICE,
			storage.ServiceType_SCANNER_DB_SERVICE,
		}
		if features.ScannerV4.Enabled() {
			svcs = append(svcs,
				storage.ServiceType_SCANNER_V4_INDEXER_SERVICE,
				storage.ServiceType_SCANNER_V4_MATCHER_SERVICE,
				storage.ServiceType_SCANNER_V4_DB_SERVICE,
			)
		}
		return svcs
	}()
)

// IssueLocalScannerCerts issue certificates for a local scanner running in secured clusters.
func IssueLocalScannerCerts(namespace string, clusterID string) (*storage.TypedServiceCertificateSet, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}

	var certIssueError error
	caPem, scannerCertificate, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_SERVICE, namespace, clusterID)
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	_, scannerDBCertificate, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_DB_SERVICE, namespace, clusterID)
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}

	certificates := []*storage.TypedServiceCertificate{
		scannerCertificate,
		scannerDBCertificate,
	}

	// TODO: Under what circumstances exactly do we issue certs for Scanner V4 here?
	// * Central could have support for Scanner V4 enabled in one way or another.
	// * Sensor could have support or Scanner V4 enabled in one way or another.
	// * Sensor could inform Central when requesting localscanner certs for which services new certs are required.
	if features.ScannerV4.Enabled() {
		_, scannerV4IndexerCertificate, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, namespace, clusterID)
		if err != nil {
			certIssueError = multierror.Append(certIssueError, err)
		}
		_, scannerV4DBCertificate, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_V4_DB_SERVICE, namespace, clusterID)
		if err != nil {
			certIssueError = multierror.Append(certIssueError, err)
		}
		certificates = append(certificates, scannerV4IndexerCertificate, scannerV4DBCertificate)
	}

	if certIssueError != nil {
		return nil, certIssueError
	}

	return &storage.TypedServiceCertificateSet{
		CaPem:        caPem,
		ServiceCerts: certificates,
	}, nil
}

func localScannerCertificatesFor(serviceType storage.ServiceType, namespace string, clusterID string) (caPem []byte, cert *storage.TypedServiceCertificate, err error) {
	certificates, err := generateServiceCertMap(serviceType, namespace, clusterID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generating certificate for service %s", serviceType)
	}
	caPem = certificates[mtls.CACertFileName]
	cert = &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: certificates[mtls.ServiceCertFileName],
			KeyPem:  certificates[mtls.ServiceKeyFileName],
		},
	}
	return caPem, cert, err
}

func generateServiceCertMap(serviceType storage.ServiceType, namespace string, clusterID string) (secretDataMap, error) {
	var err error
	if !slices.Contains(supportedServices, serviceType) {
		return nil, errors.Errorf("can only generate certificates for Scanner services, service type %s is not supported",
			serviceType)
	}

	ca, err := mtls.CAForSigning()
	if err != nil {
		return nil, errors.Wrap(err, "could not load CA for signing")
	}

	numServiceCertDataEntries := 3 // cert pem + key pem + ca pem
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	subject := mtls.NewSubject(clusterID, serviceType)
	issueOpts := []mtls.IssueCertOption{
		// TODO(ROX-9128): restore after we make sure clients can reliably reconnect
		// after certificate rotation, as part of ROX-8577.
		// mtls.WithValidityExpiringInDays(),
		mtls.WithNamespace(namespace),
	}
	if err := certgen.IssueServiceCert(fileMap, ca, subject, "", issueOpts...); err != nil {
		return nil, errors.Wrap(err, "error generating service certificate")
	}
	certgen.AddCACertToFileMap(fileMap, ca)

	return fileMap, nil
}
