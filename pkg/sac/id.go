package sac

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// ResourceID is a composite ID containing SAC scope keys for the resource.
type ResourceID struct {
	ClusterID   string
	NamespaceID string
	Suffix      string
}

// NewGlobalScopeResourceID returns new ID for global scoped resource.
func NewGlobalScopeResourceID(suffix string) (ResourceID, error) {
	if suffix == "" {
		suffix = uuid.NewV4().String()
	}
	return ResourceID{
		Suffix: suffix,
	}, nil
}

// NewClusterScopeResourceID returns new ID for cluster scoped resource.
func NewClusterScopeResourceID(clusterID, suffix string) (ResourceID, error) {
	if clusterID == "" {
		return ResourceID{}, errors.New("cluster ID must be specified")
	}

	if suffix == "" {
		suffix = uuid.NewV4().String()
	}
	return ResourceID{
		ClusterID: clusterID,
		Suffix:    suffix,
	}, nil
}

// NewNamespaceScopeResourceID returns new ID for namespace scoped resource.
func NewNamespaceScopeResourceID(clusterID, namespaceID, suffix string) (ResourceID, error) {
	if clusterID == "" {
		return ResourceID{}, errors.New("cluster ID must be specified")
	}

	if namespaceID == "" {
		return ResourceID{}, errors.New("namespace ID must be specified")
	}

	if suffix == "" {
		suffix = uuid.NewV4().String()
	}
	return ResourceID{
		ClusterID:   clusterID,
		NamespaceID: namespaceID,
		Suffix:      suffix,
	}, nil
}

// ToString serializes the ResourceID to a string.
func (r ResourceID) ToString() string {
	return r.ClusterID + "/" + r.NamespaceID + "/" + r.Suffix
}

// ParseResourceID reads a ResourceID from input ID string.
func ParseResourceID(str string) (ResourceID, error) {
	if str == "" {
		return ResourceID{}, errors.New("ID string must be provided")
	}

	parts := stringutils.SplitNPadded(str, "/", 3)
	if parts[2] == "" {
		return ResourceID{}, errors.Errorf("suffix part not found in ID %q", str)
	}

	if parts[0] == "" && parts[1] != "" {
		return ResourceID{}, errors.Errorf("cluster ID not found for namespace scoped resource ID %q", str)
	}

	return ResourceID{
		ClusterID:   parts[0],
		NamespaceID: parts[1],
		Suffix:      parts[2],
	}, nil
}
