package localscanner

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
)

// secretDataMap represents data stored as part of a secret.
type secretDataMap = map[string][]byte

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
