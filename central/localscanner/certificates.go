package localscanner

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

func generateServiceCertificate(serviceType storage.ServiceType, namespace string, clusterID string) (*mtls.IssuedCert, error) {
	if serviceType != storage.ServiceType_SCANNER_SERVICE && serviceType != storage.ServiceType_SCANNER_DB_SERVICE {
		return nil, errors.Errorf("can only generate certificates for Scanner services, service type %s is not supported",
			serviceType)
	}
	subject := mtls.NewSubject(clusterID, serviceType)
	issueOpts := []mtls.IssueCertOption{
		mtls.WithLocalScannerProfile(),
		mtls.WithNamespace(namespace),
	}
	scannerCert, err := mtls.IssueNewCert(subject, issueOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "error issuing certificate")
	}
	return scannerCert, nil
}

func generateServiceCertMap(serviceType storage.ServiceType, namespace string, clusterID string) (secretDataMap, error) {
	numServiceCertDataEntries := 3 // cert pem + key pem + ca pem
	fileMap := make(secretDataMap, numServiceCertDataEntries)

	cert, err := generateServiceCertificate(serviceType, namespace, clusterID)
	if err != nil {
		return fileMap, errors.Wrap(err, "error generating service certificate")
	}
	certgen.AddCertToFileMap(fileMap, cert, "")

	centralCA, err := mtls.CACertPEM()
	if err != nil {
		return fileMap, errors.Wrap(err, "could not load central CA")
	}
	fileMap[mtls.CACertFileName] = centralCA

	return fileMap, nil
}
