package localscanner

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

// IssueLocalScannerCerts issue certificates for a local scanner running in secured clusters.
func IssueLocalScannerCerts(namespace string, clusterID string) (*storage.TypedServiceCertificateSet, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}
	var certs = map[storage.ServiceType]*storage.TypedServiceCertificate{
		storage.ServiceType_SCANNER_SERVICE:    nil,
		storage.ServiceType_SCANNER_DB_SERVICE: nil,
	}
	var certIssueError error
	var caPem []byte

	if features.ScannerV4Support.Enabled() && features.ScannerV4.Enabled() {
		certs[storage.ServiceType_SCANNER_V4_INDEXER_SERVICE] = nil
		certs[storage.ServiceType_SCANNER_V4_MATCHER_SERVICE] = nil
		certs[storage.ServiceType_SCANNER_V4_DB_SERVICE] = nil
	}

	for serviceID := range certs {
		ca, cert, err := localScannerCertificatesFor(serviceID, namespace, clusterID)
		if err != nil {
			certIssueError = multierror.Append(certIssueError, err)
		}
		certs[serviceID] = cert
		if caPem == nil {
			caPem = ca
		}
	}

	if certIssueError != nil {
		return nil, certIssueError
	}

	certsList := make([]*storage.TypedServiceCertificate, 0, len(certs))
	for _, cert := range certs {
		certsList = append(certsList, cert)
	}

	certsSet := storage.TypedServiceCertificateSet{
		CaPem:        caPem,
		ServiceCerts: certsList,
	}

	return &certsSet, nil
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
	supportedServices := set.NewFrozenSet(
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
		storage.ServiceType_SCANNER_V4_INDEXER_SERVICE,
		storage.ServiceType_SCANNER_V4_MATCHER_SERVICE,
		storage.ServiceType_SCANNER_V4_DB_SERVICE,
	)

	if !supportedServices.Contains(serviceType) {
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
