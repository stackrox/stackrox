package certgen

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
)

// AddCertToFileMap adds `cert.pem` and `key.pem` entries for the given certificate (prefixed with
// fileNamePrefix, if any) to the given file map.
func AddCertToFileMap(fileMap map[string][]byte, cert *mtls.IssuedCert, fileNamePrefix string) {
	fileMap[fileNamePrefix+mtls.ServiceCertFileName] = cert.CertPEM
	fileMap[fileNamePrefix+mtls.ServiceKeyFileName] = cert.KeyPEM
}

// IssueServiceCert issues a certificate for the given service, and stores it in the given fileMap
// (keys prefixed with fileNamePrefix, if any).
func IssueServiceCert(fileMap map[string][]byte, ca mtls.CA, subject mtls.Subject, fileNamePrefix string, opts ...mtls.IssueCertOption) error {
	cert, err := ca.IssueCertForSubject(subject, opts...)
	if err != nil {
		return errors.Wrapf(err, "could not issue cert for %s", subject.Identifier)
	}
	AddCertToFileMap(fileMap, cert, fileNamePrefix)
	return nil
}

// IssueOtherServiceCerts issues certificates for the given subjects, and stores them in the given file
// map. The file name prefix is chosen as the slug-case of the service type plus a trailing hyphen.
func IssueOtherServiceCerts(fileMap map[string][]byte, ca mtls.CA, subjs []mtls.Subject, opts ...mtls.IssueCertOption) error {
	for _, subj := range subjs {
		if err := IssueServiceCert(fileMap, ca, subj, services.ServiceTypeToSlugName(subj.ServiceType)+"-", opts...); err != nil {
			return err
		}
	}
	return nil
}

// VerifyServiceCert verifies that the service certificate (stored with the given fileNamePrefix in the file
// map) is a valid service certificate for the given serviceType, relative to the given CA.
func VerifyServiceCert(fileMap map[string][]byte, ca mtls.CA, serviceType storage.ServiceType, fileNamePrefix string) error {
	return VerifyCert(fileMap, fileNamePrefix, GetValidateServiceCertFunc(ca, serviceType))
}

// ValidateCertFunc is a function which validates the passed certificate and returns error, if any.
type ValidateCertFunc func(certificate *x509.Certificate) error

// VerifyCert verifies that the certificate (stored with the given fileNamePrefix in the file
// map) is a valid certificate by using a given validate function.
func VerifyCert(fileMap map[string][]byte, fileNamePrefix string, validate ValidateCertFunc) error {
	certPEM := fileMap[fileNamePrefix+mtls.ServiceCertFileName]
	if len(certPEM) == 0 {
		return errors.New("no service certificate in file map")
	}
	cert, err := helpers.ParseCertificatePEM(certPEM)
	if err != nil {
		return errors.New("unparseable certificate in file map")
	}

	if err := validate(cert); err != nil {
		return errors.Wrap(err, "failed to validate certificate")
	}

	keyPEM := fileMap[fileNamePrefix+mtls.ServiceKeyFileName]
	if len(keyPEM) == 0 {
		return errors.New("no service private key in file map")
	}

	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return errors.Wrap(err, "mismatched certificate and private key, or invalid private key")
	}

	return nil
}

// GetValidateServiceCertFunc returns a function which checks whether the service certificate for the given serviceType is valid.
func GetValidateServiceCertFunc(ca mtls.CA, serviceType storage.ServiceType) ValidateCertFunc {
	return func(cert *x509.Certificate) error {
		subjFromCert, err := ca.ValidateAndExtractSubject(cert)
		if err != nil {
			return errors.Wrap(err, "failed to validate certificate and extract subject")
		}
		if subjFromCert.ServiceType != serviceType {
			return errors.Errorf("unexpected certificate service type: got %s, expected %s", subjFromCert.ServiceType, serviceType)
		}
		return nil
	}
}
