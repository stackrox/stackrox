package zip

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/stackrox/rox/central/clusters"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/zip"
)

const (
	additionalCAsZipSubdir = "additional-cas"
	centralCA              = "default-central-ca.crt"
)

func getAdditionalCAs(certs *sensor.Certs) ([]*zip.File, error) {
	certFileInfos, err := os.ReadDir(tlsconfig.AdditionalCACertsDirPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []*zip.File
	for _, fileInfo := range certFileInfos {
		if fileInfo.IsDir() || filepath.Ext(fileInfo.Name()) != ".crt" {
			continue
		}
		fullPath := path.Join(tlsconfig.AdditionalCACertsDirPath(), fileInfo.Name())
		contents, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, zip.NewFile(path.Join(additionalCAsZipSubdir, fileInfo.Name()), contents, 0))
		certs.Files[fmt.Sprintf("secrets/%s/%s", additionalCAsZipSubdir, fileInfo.Name())] = contents
	}

	if caFile, err := getDefaultCertCA(); err != nil {
		log.Errorf("Error obtaining default CA cert: %v", err)
	} else if caFile != nil {
		files = append(files, caFile)
		certs.Files[fmt.Sprintf("secrets/%s/%s", additionalCAsZipSubdir, centralCA)] = caFile.Content
	}

	return files, nil
}

func getDefaultCertCA() (*zip.File, error) {
	certFile := filepath.Join(tlsconfig.DefaultCertPath, tlsconfig.TLSCertFileName)
	keyFile := filepath.Join(tlsconfig.DefaultCertPath, tlsconfig.TLSKeyFileName)

	if filesExist, err := fileutils.AllExist(certFile, keyFile); err != nil || !filesExist {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	lastInChain, err := x509.ParseCertificate(cert.Certificate[len(cert.Certificate)-1])
	if err != nil {
		return nil, err
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
