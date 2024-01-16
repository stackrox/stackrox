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

var v2ServiceTypes = set.NewFrozenSet[storage.ServiceType](storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)
var v4ServiceTypes = set.NewFrozenSet[storage.ServiceType](storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, storage.ServiceType_SCANNER_V4_DB_SERVICE)
var allSupportedServiceTypes = func() set.FrozenSet[storage.ServiceType] {
	types := v2ServiceTypes.Unfreeze()
	if features.ScannerV4Support.Enabled() {
		types = types.Union(v4ServiceTypes.Unfreeze())
	}
	return types.Freeze()
}()

// IssueLocalScannerCerts issue certificates for a local scanner running in secured clusters.
func IssueLocalScannerCerts(namespace string, clusterID string) (*storage.TypedServiceCertificateSet, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}

	var certIssueError error
	var caPem []byte

	// In any case, generate certificates for Scanner v2.
	serviceTypes := v2ServiceTypes
	if features.ScannerV4.Enabled() {
		// Additionally, generate certificates for Scanner v4.
		serviceTypes = allSupportedServiceTypes
	}

	serviceCerts := make([]*storage.TypedServiceCertificate, 0, serviceTypes.Cardinality())
	for _, serviceType := range serviceTypes.AsSlice() {
		ca, cert, err := localScannerCertificatesFor(serviceType, namespace, clusterID)
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

	certsSet := storage.TypedServiceCertificateSet{
		CaPem:        caPem,
		ServiceCerts: serviceCerts,
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
	if !allSupportedServiceTypes.Contains(serviceType) {
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
