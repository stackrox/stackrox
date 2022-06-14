package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/fileutils"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/x509utils"
	"go.uber.org/zap"
)

const (
	// TLSCertFileName is the tls certificate filename.
	TLSCertFileName = `tls.crt`
	// TLSKeyFileName is the private key filename.
	TLSKeyFileName = `tls.key`
	// DefaultCertPath is the path where the default TLS cert is located.
	DefaultCertPath = "/run/secrets/stackrox.io/default-tls-cert"
)

// GetAdditionalCAs reads all additional CAs in DER format.
func GetAdditionalCAs() ([][]byte, error) {
	additionalCADir := AdditionalCACertsDirPath()
	certFileInfos, err := os.ReadDir(additionalCADir)
	if err != nil {
		// Ignore error if additional CAs do not exist on filesystem
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading additional CAs directory")
	}

	var certDERs [][]byte
	for _, certFile := range certFileInfos {
		if filepath.Ext(certFile.Name()) != ".crt" {
			continue
		}
		content, err := os.ReadFile(path.Join(additionalCADir, certFile.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "reading additional CAs cert")
		}

		certDER, err := x509utils.ConvertPEMToDERs(content)
		if err != nil {
			return nil, errors.Wrap(err, "converting additional CA cert to DER")
		}
		certDERs = append(certDERs, certDER...)
	}

	return certDERs, nil
}

// GetDefaultCertChain reads and parses default cert chain and returns it in DER encoded format
func GetDefaultCertChain() ([][]byte, error) {
	certFile := filepath.Join(DefaultCertPath, TLSCertFileName)
	content, err := os.ReadFile(certFile)
	if err != nil {
		// Ignore error if default certs do not exist on filesystem
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading default cert file")
	}

	certDERsFromFile, err := x509utils.ConvertPEMToDERs(content)
	if err != nil {
		return nil, errors.Wrap(err, "converting additional CA cert to DER")
	}

	return certDERsFromFile, nil
}

// loadDefaultCertificate load the default tls certificate
func loadDefaultCertificate(dir string) (*tls.Certificate, error) {
	certFile := filepath.Join(dir, TLSCertFileName)
	keyFile := filepath.Join(dir, TLSKeyFileName)

	if filesExist, err := fileutils.AllExist(certFile, keyFile); err != nil || !filesExist {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing leaf certificate")
	}

	return &cert, nil
}

func loadInternalCertificateFromFiles() (*tls.Certificate, error) {
	if filesExist, err := fileutils.AllExist(mtls.CertFilePath(), mtls.KeyFilePath()); err != nil || !filesExist {
		return nil, err
	}

	cert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func issueInternalCertificate(namespace string) (*tls.Certificate, error) {
	issuedCert, err := mtls.IssueNewCert(mtls.CentralSubject, mtls.WithNamespace(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "server keypair")
	}
	caPEM, err := mtls.CACertPEM()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert retrieval")
	}
	serverCertBundle := append(issuedCert.CertPEM, []byte("\n")...)
	serverCertBundle = append(serverCertBundle, caPEM...)

	serverTLSCert, err := tls.X509KeyPair(serverCertBundle, issuedCert.KeyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "tls conversion")
	}
	return &serverTLSCert, nil
}

func getInternalCertificates(namespace string) ([]tls.Certificate, error) {
	var internalCerts []tls.Certificate
	// First try to load the internal certificate from files. If the files don't exist, issue
	// ourselves a cert.
	if certFromFiles, err := loadInternalCertificateFromFiles(); err != nil {
		return nil, err
	} else if certFromFiles != nil {
		internalCerts = append(internalCerts, *certFromFiles)
	}

	if len(internalCerts) > 0 {
		serviceCert, err := x509.ParseCertificate(internalCerts[0].Certificate[0])
		if err != nil {
			return nil, errors.Wrap(err, "loaded internal certificate is invalid")
		}
		if validForAllDNSNames(serviceCert, mtls.CentralSubject.AllHostnamesForNamespace(namespace)...) {
			return internalCerts, nil // cert loaded from secret is sufficient
		}
	}

	log.Warnw("Internal TLS certificates are not valid for all cluster-internal DNS names due to deployment in "+
		"alternative namespace, issuing ephemeral certificate with adequate DNS names",
		zap.String("namespace", namespace), zap.Strings("internalDNSNames", mtls.CentralSubject.AllHostnamesForNamespace(namespace)))
	newInternalCert, err := issueInternalCertificate(namespace)
	if err != nil {
		return internalCerts, err
	}
	internalCerts = append(internalCerts, *newInternalCert)
	return internalCerts, nil
}

func validForAllDNSNames(cert *x509.Certificate, dnsNames ...string) bool {
	for _, dnsName := range dnsNames {
		if err := cert.VerifyHostname(dnsName); err != nil {
			return false
		}
	}
	return true
}
