package zip

import (
	"fmt"
	"path"
	"strings"

	"github.com/stackrox/rox/central/clusters"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
)

// GenerateCertsAndAddToZip generates all the required certs for the cluster, and returns them.
func GenerateCertsAndAddToZip(cluster *storage.Cluster, identityStore siDataStore.DataStore) (sensor.Certs, error) {
	certs := sensor.Certs{Files: make(map[string][]byte)}
	ca, err := mtls.CACertPEM()
	if err != nil {
		return certs, err
	}
	certs.Files["secrets/"+mtls.CACertFileName] = ca

	identities, err := clusters.IssueSecuredClusterCertificates(cluster, namespaces.StackRox, identityStore)
	if err != nil {
		return certs, err
	}
	for serviceType, issuedCert := range identities {
		addCerts(&certs, serviceType, issuedCert)
	}

	return certs, nil
}

func addCerts(certs *sensor.Certs, serviceType storage.ServiceType, issuedCert *mtls.IssuedCert) {
	components := strings.Split(serviceType.String(), "_")
	components = components[:len(components)-1] // last component is "SERVICE"
	servicePrefix := strings.ToLower(strings.Join(components, "-"))

	certFileName := fmt.Sprintf("%s-cert.pem", servicePrefix)
	keyFileName := fmt.Sprintf("%s-key.pem", servicePrefix)

	certs.Files[path.Join("secrets", certFileName)] = issuedCert.CertPEM
	certs.Files[path.Join("secrets", keyFileName)] = issuedCert.KeyPEM
}
