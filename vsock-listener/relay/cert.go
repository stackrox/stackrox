package relay

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	alternativeCAPath = `/run/secrets/stackrox.io/ca/ca.pem`
)

func configureCA() error {
	// Check for existence of CA cert in default location
	if exists, err := fileutils.Exists(mtls.CAFilePath()); err != nil {
		log.Errorf("Failed to stat CA certificate in default location: %v. Assuming it doesn't exist...", err)
	} else if exists {
		return nil // CA cert found in default location
	}

	// Check for existence of CA cert in fallback location
	if exists, err := fileutils.Exists(alternativeCAPath); err != nil {
		return errors.Wrap(err, "failed to check for existence of alternate CA certificate")
	} else if !exists {
		return errors.New("did not find CA certificate in primary nor alternate location")
	}

	// Found fallback CA
	log.Info("Switching to fallback CA file location")
	if err := utils.ShouldErr(os.Setenv(mtls.CAFileEnvName, alternativeCAPath)); err != nil {
		return errors.Wrap(err, "failed to update environment for alternative CA location")
	}
	log.Info("Successfully configured CA to be read from fallback location")
	return nil
}

func isUsableServiceCert(certFilePath, namespace string) bool {
	certFromFile, err := x509utils.LoadCertificatePEMFile(certFilePath)
	if err != nil {
		log.Errorf("Failed to load service certificate: %v", err)
		return false
	}
	desiredDNS := mtls.CollectorSubject.HostnameForNamespace(namespace) + ".svc"
	if err := certFromFile.VerifyHostname(desiredDNS); err != nil {
		log.Infof("mTLS certificate with common name %q is not valid for DNS name %s: %v", certFromFile.Subject.CommonName, desiredDNS, err)
		return false
	}
	return true
}

func configureCerts(namespace string) error {
	if allExist, err := fileutils.AllExist(mtls.CertFilePath(), mtls.KeyFilePath()); err != nil {
		log.Infof("Could not stat certificate and key in default location: %v. Assuming they don't exist...", err)
	} else if allExist && isUsableServiceCert(mtls.CertFilePath(), namespace) {
		// Found usable cert in default location
		return nil
	}

	return errors.New("Failed to find all required certificate files")
}
