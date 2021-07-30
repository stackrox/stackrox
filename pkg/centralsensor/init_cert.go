package centralsensor

import (
	"github.com/pkg/errors"
)

const (
	// InitCertClusterID is the cluster ID used for init certs that allow dynamic creation of clusters.
	InitCertClusterID = "00000000-0000-0000-0000-000000000000"
)

// GetClusterID allows joining
func GetClusterID(explicitID, idFromCert string) (string, error) {
	id := explicitID
	if id == "" {
		id = idFromCert
	} else if idFromCert != id && idFromCert != InitCertClusterID {
		return "", errors.Errorf("explicit cluster ID %q does not match non-wildcard cluster ID %q from certificate", id, idFromCert)
	}

	if id == InitCertClusterID {
		return "", errors.Errorf("no concrete cluster ID was specified in conjunction with wildcard ID %q", InitCertClusterID)
	}

	return id, nil
}
