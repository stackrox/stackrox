package centralsensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// RegisteredInitCertClusterID is the cluster ID used for registered init certs that allow dynamic creation of clusters.
	// "Registered" refers to the fact that the metadata of init bundles which contain such certificates is
	// saved in central storage and used for revocation checks. Such init bundles are typically issued at
	// user request via the web UI or roxctl.
	// Use this only when you care about the difference between the user- and operator-issued init bundles.
	// Otherwise, use IsInitCertClusterID().
	RegisteredInitCertClusterID = "00000000-0000-0000-0000-000000000000"
	// EphemeralInitCertClusterID is the cluster ID used for ephemeral init certs that allow dynamic creation of clusters.
	// "Ephemeral" refers to the fact that the metadata of init bundles which contain such certificates is
	// not persisted anywhere, and the certificates have a short validity time. Such init bundles are
	// typically issued automatically by the k8s operator.
	// Use this only when you care about the difference between the user- and operator-issued init bundles.
	// Otherwise, use IsInitCertClusterID().
	EphemeralInitCertClusterID = "00000000-0000-0000-0000-000000000001"
)

// AllSecuredClusterServices contains service types of all services that should be included in an init bundle.
var AllSecuredClusterServices = []storage.ServiceType{
	storage.ServiceType_COLLECTOR_SERVICE,
	storage.ServiceType_SENSOR_SERVICE,
	storage.ServiceType_ADMISSION_CONTROL_SERVICE}

// IsInitCertClusterID returns true if the passed cluster id is for an init cert that allows dynamic creation of clusters.
func IsInitCertClusterID(clusterID string) bool {
	return clusterID == RegisteredInitCertClusterID || clusterID == EphemeralInitCertClusterID
}

// GetClusterID allows joining
func GetClusterID(explicitID, idFromCert string) (string, error) {
	id := explicitID
	if id == "" {
		id = idFromCert
	} else if idFromCert != id && !IsInitCertClusterID(idFromCert) {
		return "", errors.Errorf("explicit cluster ID %q does not match non-wildcard cluster ID %q from certificate", id, idFromCert)
	}

	if IsInitCertClusterID(id) {
		return "", errors.Errorf("no concrete cluster ID was specified in conjunction with wildcard ID %q. "+
			"This may be caused by Central data not being persisted between restarts; you may try deploying Central with STORAGE=pvc. "+
			"For other potential solutions reffer to https://access.redhat.com/solutions/6972449", id)
	}

	return id, nil
}
