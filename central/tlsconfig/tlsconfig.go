package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls"
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
	certFileInfos, err := ioutil.ReadDir(additionalCADir)
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
		content, err := ioutil.ReadFile(path.Join(additionalCADir, certFile.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "reading additional CAs cert")
		}

		certDER, err := mtls.ConvertPEMToDERs(content)
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
	content, err := ioutil.ReadFile(certFile)
	if err != nil {
		// Ignore error if default certs do not exist on filesystem
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading default cert file")
	}

	certDERsFromFile, err := mtls.ConvertPEMToDERs(content)
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

func issueInternalCertificate() (*tls.Certificate, error) {
	issuedCert, err := mtls.IssueNewCert(mtls.CentralSubject)
	if err != nil {
		return nil, errors.Wrap(err, "server keypair")
	}
	caPEM, err := mtls.CACertPEM()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert retrieval")
	}
	serverCertBundle := append(issuedCert.CertPEM, caPEM...)

	serverTLSCert, err := tls.X509KeyPair(serverCertBundle, issuedCert.KeyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "tls conversion")
	}
	return &serverTLSCert, nil
}

func getInternalCertificate() (*tls.Certificate, error) {
	// First try to load the internal certificate from files. If the files don't exist, issue
	// ourselves a cert.
	if certFromFiles, err := loadInternalCertificateFromFiles(); err != nil {
		return nil, err
	} else if certFromFiles != nil {
		return certFromFiles, nil
	}

	return issueInternalCertificate()
}
