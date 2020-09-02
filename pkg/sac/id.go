package sac

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/stringutils"
)

// clusterScopedResourceID is a ID generated for a cluster scoped resource.
type clusterScopedResourceID struct {
	ClusterID  string
	ResourceID string
}

// NewClusterScopeResourceID returns new instance of clusterScopedResourceID.
func NewClusterScopeResourceID(clusterID, resourceID string) (clusterScopedResourceID, error) {
	if clusterID == "" {
		return clusterScopedResourceID{}, errors.New("cluster ID must be specified")
	}

	if resourceID == "" {
		return clusterScopedResourceID{}, errors.New("resource ID must be specified")
	}
	return clusterScopedResourceID{ClusterID: clusterID, ResourceID: resourceID}, nil
}

// GetClusterScopedResourceID reads a clusterScopedResourceID from string form.
func GetClusterScopedResourceID(str string) (clusterScopedResourceID, error) {
	clusterIDEncoded, resourceIDEncoded := stringutils.Split2(str, ":")
	if resourceIDEncoded == "" {
		return clusterScopedResourceID{}, errors.Errorf("invalid ID: %s", str)
	}

	clusterID, err := base64.RawURLEncoding.DecodeString(clusterIDEncoded)
	if err != nil {
		return clusterScopedResourceID{}, err
	}

	resourceID, err := base64.RawURLEncoding.DecodeString(resourceIDEncoded)
	if err != nil {
		return clusterScopedResourceID{}, err
	}
	return clusterScopedResourceID{ClusterID: string(clusterID), ResourceID: string(resourceID)}, nil
}

// ToString serializes the clusterScopedResourceID to a string.
func (cID clusterScopedResourceID) ToString() string {
	clusterEncoded := base64.RawURLEncoding.EncodeToString([]byte(cID.ClusterID))
	resourceEncoded := base64.RawURLEncoding.EncodeToString([]byte(cID.ResourceID))
	return clusterEncoded + ":" + resourceEncoded
}
