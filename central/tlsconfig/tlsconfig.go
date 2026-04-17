package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	// TLSCertFileName is the tls certificate filename.
	TLSCertFileName = `tls.crt`
	// TLSKeyFileName is the private key filename.
	TLSKeyFileName = `tls.key`
	// DefaultCertPath is the path where the default TLS cert is located.
	DefaultCertPath = "/run/secrets/stackrox.io/default-tls-cert"
)

// GetAdditionalCAFilePaths returns the list of file paths containing additional CAs.
func GetAdditionalCAFilePaths() ([]string, error) {
	additionalCADir := AdditionalCACertsDirPath()
	directoryEntries, err := os.ReadDir(additionalCADir)
	if err != nil {
		// Ignore error if additional CAs do not exist on filesystem
		if os.IsNotExist(err) {
			log.Debugf("Additional CA directory %q does not exist: skipping", additionalCADir)
			return nil, nil
		}
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to read additional CAs directory %q", additionalCADir))
	}

	var filePaths = set.NewStringSet()

	for _, directoryEntry := range directoryEntries {

		entryName := directoryEntry.Name()
		filePath := path.Join(additionalCADir, entryName)

		if directoryEntry.IsDir() {
			log.Debugf("Skipping additional CA directory entry %q because it is a directory", entryName)
			continue
		}

		fileInfo, err := directoryEntry.Info()
		if err != nil {
			log.Warnf("Failed to read additional CA file info for %q: %s", entryName, err)
			continue
		}

		if isSymlink(fileInfo) {
			resolvedPathForSymlink, err := filepath.EvalSymlinks(filePath)
			if err != nil {
				log.Warnf("Failed to evaluate additional CA file symlinks for file %q: %s", filePath, err)
				continue
			}
			fileInfo, err = os.Stat(resolvedPathForSymlink)
			if err != nil {
				log.Warnf("Error reading additional CA file info for symlink %q that resolved to %q: %s", filePath, resolvedPathForSymlink, err)
				continue
			}
			if fileInfo.IsDir() {
				log.Debugf("Skipping additional CA file %q because it is a symlink that resolved to a directory", filePath)
				continue
			}
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Warnf("Failed to read additional CA file %q: %s. Skipping", filePath, err)
			continue
		}

		if _, err = x509utils.ConvertPEMToDERs(content); err != nil {
			log.Warnf("Failed to convert additional CA file %q from PEM to DER format: %s. Skipping", filePath, err)
			continue
		}

		filePaths.Add(filePath)

	}

	return filePaths.AsSortedSlice(func(i, j string) bool {
		return strings.Compare(i, j) < 0
	}), nil

}

func isSymlink(fileInfo fs.FileInfo) bool {
	return fileInfo.Mode()&os.ModeSymlink != 0
}

// GetAdditionalCAs reads all additional CAs in DER format.
func GetAdditionalCAs() ([][]byte, error) {
	additionalCAFilePaths, err := GetAdditionalCAFilePaths()
	if err != nil {
		return nil, err
	}

	var certDERs [][]byte
	for _, certFilePath := range additionalCAFilePaths {
		pemBytes, err := os.ReadFile(certFilePath)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to read additional CAs cert file %q", certFilePath))
		}

		ders, err := x509utils.ConvertPEMToDERs(pemBytes)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Failed to convert additional CA cert file %q from PEM to DER format", certFilePath))
		}

		certDERs = append(certDERs, ders...)
	}

	return certDERs, nil
}

// MaybeGetDefaultCertChain reads and parses default cert chain and returns it in DER encoded format.
func MaybeGetDefaultCertChain() ([][]byte, error) {
	cert, err := MaybeGetDefaultTLSCertificateFromDirectory(DefaultCertPath)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, nil
	}
	return cert.Certificate, nil
}

// MaybeGetDefaultTLSCertificateFromDefaultDirectory loads the default TLS certificate from the default directory.
func MaybeGetDefaultTLSCertificateFromDefaultDirectory() (*tls.Certificate, error) {
	return MaybeGetDefaultTLSCertificateFromDirectory(DefaultCertPath)
}

// MaybeGetDefaultTLSCertificateFromDirectory loads the default TLS certificate from the given directory.
func MaybeGetDefaultTLSCertificateFromDirectory(dir string) (*tls.Certificate, error) {
	certFile := filepath.Join(dir, TLSCertFileName)
	keyFile := filepath.Join(dir, TLSKeyFileName)

	if exists, err := fileutils.Exists(certFile); err != nil || !exists {
		if err != nil {
			log.Warnw("Error checking if default TLS certificate file exists", logging.Err(err))
			return nil, err
		}
		log.Debugf("Default TLS certificate file %q does not exist. Skipping", certFile)
		return nil, nil
	}

	if exists, err := fileutils.Exists(keyFile); err != nil || !exists {
		if err != nil {
			log.Warnw("Error checking if default TLS key file exists", logging.Err(err))
			return nil, err
		}
		log.Debugf("Default TLS key file %q does not exist. Skipping", keyFile)
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		if strings.Contains(err.Error(), "private key does not match public key") {
			return nil, errors.Wrap(err, "loading default certificate; if the certificate file contains a certificate chain, ensure that the certificate chain is in the correct order (the first certificate should be the leaf certificate, any following certificates should form the certificate chain)")
		}
		return nil, errors.Wrap(err, "loading default certificate failed")
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing leaf certificate failed")
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

// LoadInternalCertificateFromDirectory loads the internal service leaf certificate
// (cert.pem + key.pem) from the given directory and verifies it against the
// internal CA trust roots.
func LoadInternalCertificateFromDirectory(dir string) (*tls.Certificate, error) {
	certFile := filepath.Join(dir, mtls.ServiceCertFileName)
	keyFile := filepath.Join(dir, mtls.ServiceKeyFileName)

	if filesExist, err := fileutils.AllExist(certFile, keyFile); err != nil || !filesExist {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading internal certificate")
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing internal leaf certificate")
	}

	trustPool, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "building trust pool for internal certificate verification")
	}
	if _, err := cert.Leaf.Verify(x509.VerifyOptions{Roots: trustPool}); err != nil {
		return nil, errors.Wrap(err, "verifying internal certificate against trusted CAs")
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
	certFromFiles, err := loadInternalCertificateFromFiles()
	if err != nil {
		return nil, err
	}
	if certFromFiles != nil {
		if certFromFiles.Leaf == nil {
			certFromFiles.Leaf, err = x509.ParseCertificate(certFromFiles.Certificate[0])
			if err != nil {
				return nil, errors.Wrap(err, "loaded internal certificate is invalid")
			}
		}
		return buildInternalCerts(certFromFiles, namespace)
	}

	// No cert files on disk — issue an ephemeral cert from scratch.
	ephemeralCert, err := issueInternalCertificate(namespace)
	if err != nil {
		return nil, err
	}
	return []tls.Certificate{*ephemeralCert}, nil
}

// buildInternalCerts returns a cert slice containing the given cert. If the cert
// is not valid for all DNS names in the given namespace, an additional ephemeral
// cert with the correct SANs is appended.
func buildInternalCerts(cert *tls.Certificate, namespace string) ([]tls.Certificate, error) {
	if cert.Leaf != nil && validForAllDNSNames(cert.Leaf, mtls.CentralSubject.AllHostnamesForNamespace(namespace)...) {
		return []tls.Certificate{*cert}, nil
	}

	log.Warnw("Internal TLS certificate is not valid for all cluster-internal DNS names, "+
		"issuing ephemeral certificate with adequate DNS names",
		logging.String("namespace", namespace),
		logging.Strings("internalDNSNames", mtls.CentralSubject.AllHostnamesForNamespace(namespace)))
	ephemeralCert, err := issueInternalCertificate(namespace)
	if err != nil {
		return []tls.Certificate{*cert}, err
	}
	return []tls.Certificate{*cert, *ephemeralCert}, nil
}

func validForAllDNSNames(cert *x509.Certificate, dnsNames ...string) bool {
	for _, dnsName := range dnsNames {
		if err := cert.VerifyHostname(dnsName); err != nil {
			return false
		}
	}
	return true
}
