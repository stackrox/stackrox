package zip

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/stackrox/rox/central/role/resources"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/zip"
)

const (
	additionalCAsDir       = "/usr/local/share/ca-certificates"
	additionalCAsZipSubdir = "additional-cas"
	centralCA              = "default-central-ca.crt"
)

func createIdentity(wrapper *zip.Wrapper, id string, servicePrefix string, serviceType storage.ServiceType, identityStore siDataStore.DataStore, certs *sensor.Certs) error {
	srvIDAllAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))

	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(id, serviceType))
	if err != nil {
		return err
	}
	if err := identityStore.AddServiceIdentity(srvIDAllAccessCtx, issuedCert.ID); err != nil {
		return err
	}
	wrapper.AddFiles(
		zip.NewFile(fmt.Sprintf("%s-cert.pem", servicePrefix), issuedCert.CertPEM, 0),
		zip.NewFile(fmt.Sprintf("%s-key.pem", servicePrefix), issuedCert.KeyPEM, zip.Sensitive),
	)
	certs.Files[fmt.Sprintf("secrets/%s-cert.pem", servicePrefix)] = issuedCert.CertPEM
	certs.Files[fmt.Sprintf("secrets/%s-key.pem", servicePrefix)] = issuedCert.KeyPEM
	return nil
}

func getAdditionalCAs() ([]*zip.File, error) {
	certFileInfos, err := ioutil.ReadDir(additionalCAsDir)
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
		fullPath := path.Join(additionalCAsDir, fileInfo.Name())
		contents, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, zip.NewFile(path.Join(additionalCAsZipSubdir, fileInfo.Name()), contents, 0))
	}

	if caFile, err := getDefaultCertCA(); err != nil {
		log.Errorf("Error obtaining default CA cert: %v", err)
	} else if caFile != nil {
		files = append(files, caFile)
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

// AddCertificatesToZip adds required service certificate and key files to the zip, and returns the cert and key files
func AddCertificatesToZip(wrapper *zip.Wrapper, cluster *storage.Cluster, identityStore siDataStore.DataStore) (sensor.Certs, error) {
	certs := sensor.Certs{Files: make(map[string][]byte)}
	ca, err := mtls.CACertPEM()
	if err != nil {
		return certs, err
	}
	wrapper.AddFiles(zip.NewFile("ca.pem", ca, 0))
	certs.Files["secrets/ca.pem"] = ca

	// Add MTLS files for sensor
	if err := createIdentity(wrapper, cluster.GetId(), "sensor", storage.ServiceType_SENSOR_SERVICE, identityStore, &certs); err != nil {
		return certs, err
	}

	// Add MTLS files for collector
	if err := createIdentity(wrapper, cluster.GetId(), "collector", storage.ServiceType_COLLECTOR_SERVICE, identityStore, &certs); err != nil {
		return certs, err
	}

	if features.AdmissionControlService.Enabled() && cluster.GetAdmissionController() {
		if err := createIdentity(wrapper, cluster.GetId(), "admission-control",
			storage.ServiceType_ADMISSION_CONTROL_SERVICE, identityStore, &certs); err != nil {
			return certs, err
		}
	}

	additionalCAFiles, err := getAdditionalCAs()
	if err != nil {
		return certs, err
	}
	wrapper.AddFiles(additionalCAFiles...)

	return certs, nil
}
