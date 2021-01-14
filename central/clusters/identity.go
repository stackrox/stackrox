package clusters

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

var allSecuredClusterServices = []storage.ServiceType{storage.ServiceType_COLLECTOR_SERVICE, storage.ServiceType_SENSOR_SERVICE, storage.ServiceType_ADMISSION_CONTROL_SERVICE}

// CertBundle contains a bundle of generated certificates for each service type
type CertBundle map[storage.ServiceType]*mtls.IssuedCert

// CreateIdentity creates a new cluster identity for a service
func CreateIdentity(clusterID string, serviceType storage.ServiceType, identityStore siDataStore.DataStore) (*mtls.IssuedCert, error) {
	srvIDAllAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))

	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(clusterID, serviceType))
	if err != nil {
		return nil, err
	}
	if err := identityStore.AddServiceIdentity(srvIDAllAccessCtx, issuedCert.ID); err != nil {
		return nil, err
	}

	return issuedCert, nil
}

// IssueSecuredClusterCertificates creates a bundle which contains all service certificates for a given cluster
func IssueSecuredClusterCertificates(cluster *storage.Cluster, identityStore siDataStore.DataStore) (CertBundle, error) {
	certs := make(CertBundle)
	for _, serviceType := range getEnabledServices(cluster) {
		issuedCert, err := CreateIdentity(cluster.GetId(), serviceType, identityStore)
		if err != nil {
			return certs, err
		}
		certs[serviceType] = issuedCert
	}
	return certs, nil
}

// IssueSecuredClusterInitCertificates creates a cert bundle which holds init certificates which have a UUID nil cluster id
// These certificates are used to register new clusters at central.
// All certificates share the same init bundle UUID written in the OU subject field.
func IssueSecuredClusterInitCertificates() (CertBundle, uuid.UUID, error) {
	initID := centralsensor.InitCertClusterID
	certs := make(CertBundle)
	bundleID := uuid.NewV4()
	for _, serviceType := range allSecuredClusterServices {
		issuedCert, err := mtls.IssueNewCert(mtls.NewInitSubject(initID, serviceType, bundleID))

		if err != nil {
			return certs, bundleID, err
		}
		certs[serviceType] = issuedCert
	}
	return certs, bundleID, nil
}

func getEnabledServices(cluster *storage.Cluster) []storage.ServiceType {
	serviceTypes := []storage.ServiceType{storage.ServiceType_COLLECTOR_SERVICE, storage.ServiceType_SENSOR_SERVICE}
	if features.AdmissionControlService.Enabled() && cluster.GetAdmissionController() {
		serviceTypes = append(serviceTypes, storage.ServiceType_ADMISSION_CONTROL_SERVICE)
	}
	return serviceTypes
}
