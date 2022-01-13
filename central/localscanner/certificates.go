package localscanner

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	featureFlag = features.LocalImageScanning
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

// IssueLocalScannerCerts issue certificates for a local scanner running in secured clusters.
func IssueLocalScannerCerts(namespace string, clusterID string) (*central.IssueLocalScannerCertsResponse, error) {
	if !featureFlag.Enabled() {
		return nil, errors.Errorf("feature '%s' is disabled", featureFlag.Name())
	}
	if namespace == "" {
		return nil, errors.New("namespace is required to issue the certificates for the local scanner")
	}

	var certIssueError error
	scannerCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_SERVICE, namespace, clusterID)
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	scannerDBCertificates, err := localScannerCertificatesFor(storage.ServiceType_SCANNER_DB_SERVICE, namespace, clusterID)
	if err != nil {
		certIssueError = multierror.Append(certIssueError, err)
	}
	if certIssueError != nil {
		return nil, certIssueError
	}

	return &central.IssueLocalScannerCertsResponse{
		ScannerCerts:   scannerCertificates,
		ScannerDbCerts: scannerDBCertificates,
	}, nil
}

func localScannerCertificatesFor(serviceType storage.ServiceType, namespace string, clusterID string) (*central.LocalScannerCertificates, error) {
	certificates, err := generateServiceCertMap(serviceType, namespace, clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "generating certificate for service %s", serviceType)
	}

	return &central.LocalScannerCertificates{
		Ca:   certificates[mtls.CACertFileName],
		Cert: certificates[mtls.ServiceCertFileName],
		Key:  certificates[mtls.ServiceKeyFileName],
	}, nil
}

func generateServiceCertMap(serviceType storage.ServiceType, namespace string, clusterID string) (secretDataMap, error) {
	if serviceType != storage.ServiceType_SCANNER_SERVICE && serviceType != storage.ServiceType_SCANNER_DB_SERVICE {
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
		mtls.WithValidityExpiringInDays(),
		mtls.WithNamespace(namespace),
	}
	if err := certgen.IssueServiceCert(fileMap, ca, subject, "", issueOpts...); err != nil {
		return nil, errors.Wrap(err, "error generating service certificate")
	}
	certgen.AddCACertToFileMap(fileMap, ca)

	return fileMap, nil
}
