package zip

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusters"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/zip"
)

const (
	additionalCAsZipSubdir = "additional-cas"
	centralCA              = "default-central-ca.crt"
)

func getAdditionalCAs(certs *sensor.Certs) ([]*zip.File, error) {

	additionalCAFilePaths, err := tlsconfig.GetAdditionalCAFilePaths()
	if err != nil {
		return nil, err
	}

	var files []*zip.File
	for _, additionalCAFilePath := range additionalCAFilePaths {
		contents, err := os.ReadFile(additionalCAFilePath)
		if err != nil {
			return nil, err
		}
		fileName := filepath.Base(additionalCAFilePath)
		files = append(files, zip.NewFile(path.Join(additionalCAsZipSubdir, fileName), contents, 0))
		certs.Files[fmt.Sprintf("secrets/%s/%s", additionalCAsZipSubdir, fileName)] = contents
	}

	if zipForDefaultTLSCertCA, err := maybeCreateZipFileForDefaultTLSCertCA(); err != nil {
		log.Errorf("Error obtaining default TLS Certificate: %v", err)
	} else if zipForDefaultTLSCertCA != nil {
		files = append(files, zipForDefaultTLSCertCA)
		certs.Files[fmt.Sprintf("secrets/%s/%s", additionalCAsZipSubdir, centralCA)] = zipForDefaultTLSCertCA.Content
	}

	return files, nil
}

// maybeCreateZipFileForDefaultTLSCertCA returns a zip file containing the default CA cert if it is not trusted by the system roots.
// If there is no default CA cert, or if it is already trusted by the system roots, it returns nil.
func maybeCreateZipFileForDefaultTLSCertCA() (*zip.File, error) {
	defaultTLSCer, err := tlsconfig.MaybeGetDefaultTLSCertificateFromDefaultDirectory()
	if err != nil {
		return nil, errors.Wrap(err, "error getting default TLS certificate from default directory")
	}
	if defaultTLSCer == nil || len(defaultTLSCer.Certificate) == 0 {
		return nil, nil
	}

	lastInChain, err := x509.ParseCertificate(defaultTLSCer.Certificate[len(defaultTLSCer.Certificate)-1])
	if err != nil {
		return nil, errors.Wrap(err, "error parsing default TLS certificate")
	}

	// Only add cert to bundle if it is not trusted by system roots.
	if _, err := lastInChain.Verify(x509.VerifyOptions{}); err != nil {
		pemEncodedCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: lastInChain.Raw})
		return zip.NewFile(path.Join(additionalCAsZipSubdir, centralCA), pemEncodedCert, 0), nil
	}

	return nil, nil
}

// GenerateCertsAndAddToZip generates all the required certs for the cluster, and returns them.
// If the passed wrapper is not-nil, the certificates are added to the wrapper.
func GenerateCertsAndAddToZip(wrapper *zip.Wrapper, cluster *storage.Cluster, identityStore siDataStore.DataStore) (sensor.Certs, error) {
	certs := sensor.Certs{Files: make(map[string][]byte)}
	ca, err := mtls.CACertPEM()
	if err != nil {
		return certs, err
	}
	if wrapper != nil {
		wrapper.AddFiles(zip.NewFile(mtls.CACertFileName, ca, 0))
	}
	certs.Files["secrets/"+mtls.CACertFileName] = ca

	identities, err := clusters.IssueSecuredClusterCertificates(cluster, namespaces.StackRox, identityStore)
	if err != nil {
		return certs, err
	}
	for serviceType, issuedCert := range identities {
		addCerts(wrapper, &certs, serviceType, issuedCert)
	}

	if wrapper != nil {
		additionalCAFiles, err := getAdditionalCAs(&certs)
		if err != nil {
			return certs, err
		}
		wrapper.AddFiles(additionalCAFiles...)
	}

	return certs, nil
}

func addCerts(wrapper *zip.Wrapper, certs *sensor.Certs, serviceType storage.ServiceType, issuedCert *mtls.IssuedCert) {
	components := strings.Split(serviceType.String(), "_")
	components = components[:len(components)-1] // last component is "SERVICE"
	servicePrefix := strings.ToLower(strings.Join(components, "-"))

	certFileName := fmt.Sprintf("%s-cert.pem", servicePrefix)
	keyFileName := fmt.Sprintf("%s-key.pem", servicePrefix)

	if wrapper != nil {
		wrapper.AddFiles(
			zip.NewFile(certFileName, issuedCert.CertPEM, 0),
			zip.NewFile(keyFileName, issuedCert.KeyPEM, zip.Sensitive),
		)
	}
	certs.Files[path.Join("secrets", certFileName)] = issuedCert.CertPEM
	certs.Files[path.Join("secrets", keyFileName)] = issuedCert.KeyPEM
}
