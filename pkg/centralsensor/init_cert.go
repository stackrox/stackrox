package centralsensor

import (
	"github.com/pkg/errors"
)

const (
	// UserIssuedInitCertClusterID is the cluster ID used for user-issued init certs that allow dynamic creation of clusters.
	// Use this only when you care about the difference between the user- and operator-issued init bundles.
	// Otherwise, use IsInitCertClusterID().
	UserIssuedInitCertClusterID = "00000000-0000-0000-0000-000000000000"
	// EphemeralInitCertClusterID is the cluster ID used for operator-issued init certs that allow dynamic creation of clusters.
	// Use this only when you care about the difference between the user- and operator-issued init bundles.
	// Otherwise, use IsInitCertClusterID().
	EphemeralInitCertClusterID = "00000000-0000-0000-0000-000000000001"
)

// IsInitCertClusterID returns true if the passed cluster id is for an init cert that allows dynamic creation of clusters.
func IsInitCertClusterID(clusterID string) bool {
	return clusterID == UserIssuedInitCertClusterID || clusterID == EphemeralInitCertClusterID
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
		return "", errors.Errorf("no concrete cluster ID was specified in conjunction with wildcard ID %q", id)
	}

	return id, nil
}
