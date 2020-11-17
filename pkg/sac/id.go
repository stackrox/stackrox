package sac

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

const separator = "_"

// ResourceID is a composite ID containing SAC scope keys for the resource.
type ResourceID struct {
	clusterID   string
	namespaceID string
	suffix      string
}

// NewGlobalScopeResourceID returns new ID for global scoped resource.
func NewGlobalScopeResourceID(suffix string) (ResourceID, error) {
	if suffix == "" {
		suffix = uuid.NewV4().String()
	} else if err := validateSuffix(suffix); err != nil {
		return ResourceID{}, err
	}

	return ResourceID{
		suffix: suffix,
	}, nil
}

// NewClusterScopeResourceID returns new ID for cluster scoped resource.
func NewClusterScopeResourceID(clusterID, suffix string) (ResourceID, error) {
	if err := validateClusterID(clusterID); err != nil {
		return ResourceID{}, err
	}

	if suffix == "" {
		suffix = uuid.NewV4().String()
	} else if err := validateSuffix(suffix); err != nil {
		return ResourceID{}, err
	}

	return ResourceID{
		clusterID: clusterID,
		suffix:    suffix,
	}, nil
}

// NewNamespaceScopeResourceID returns new ID for namespace scoped resource.
func NewNamespaceScopeResourceID(clusterID, namespaceID, suffix string) (ResourceID, error) {
	if err := validateClusterID(clusterID); err != nil {
		return ResourceID{}, err
	}

	if err := validateNamespaceID(namespaceID); err != nil {
		return ResourceID{}, err
	}

	if suffix == "" {
		suffix = uuid.NewV4().String()
	} else if err := validateSuffix(suffix); err != nil {
		return ResourceID{}, err
	}

	return ResourceID{
		clusterID:   clusterID,
		namespaceID: namespaceID,
		suffix:      suffix,
	}, nil
}

func validateClusterID(clusterID string) error {
	if clusterID == "" {
		return errors.New("cluster ID must be specified")
	}
	if strings.Contains(clusterID, separator) {
		return errors.Errorf("cluster ID %s must not contain %q", clusterID, separator)
	}
	return nil
}

func validateNamespaceID(namespaceID string) error {
	if namespaceID == "" {
		return errors.New("namespace ID must be specified")
	}
	if strings.Contains(namespaceID, separator) {
		return errors.Errorf("namespace ID %s must not contain %q", namespaceID, separator)
	}
	return nil
}

func validateSuffix(suffix string) error {
	if strings.Contains(suffix, separator) {
		return errors.Errorf("suffix %s must not contain %q", suffix, separator)
	}
	return nil
}

// ClusterID returns the cluster ID.
func (r ResourceID) ClusterID() string {
	return r.clusterID
}

// NamespaceID returns the namespace ID.
func (r ResourceID) NamespaceID() string {
	return r.namespaceID
}

// Suffix returns the suffix of ResourceID.
func (r ResourceID) Suffix() string {
	return r.suffix
}

// String serializes the ResourceID to a string.
func (r ResourceID) String() string {
	return r.clusterID + separator + r.namespaceID + separator + r.suffix
}

// IsValid return true if the resource ID is valid, i.e. has scope info and suffix.
func (r ResourceID) IsValid() bool {
	return r.Suffix() != "" && (r.GlobalScoped() || r.ClusterScoped() || r.NamespaceScoped())
}

// GlobalScoped returns true if the resource ID is global-scoped.
func (r ResourceID) GlobalScoped() bool {
	return r.ClusterID() == "" && r.NamespaceID() == ""
}

// ClusterScoped returns true if the resource ID is cluster-scoped.
func (r ResourceID) ClusterScoped() bool {
	return r.ClusterID() != "" && r.NamespaceID() == ""
}

// NamespaceScoped returns true if the resource ID is namespace-scoped.
func (r ResourceID) NamespaceScoped() bool {
	return r.ClusterID() != "" && r.NamespaceID() != ""
}

// ParseResourceID reads a ResourceID from input ID string.
func ParseResourceID(str string) (ResourceID, error) {
	if str == "" {
		return ResourceID{}, errors.New("ID string must be provided")
	}

	parts := stringutils.SplitNPadded(str, separator, 3)
	if parts[2] == "" {
		return ResourceID{}, errors.Errorf("suffix part not found in ID %q", str)
	}

	if parts[0] == "" && parts[1] != "" {
		return ResourceID{}, errors.Errorf("cluster ID not found for namespace scoped resource ID %q", str)
	}

	return ResourceID{
		clusterID:   parts[0],
		namespaceID: parts[1],
		suffix:      parts[2],
	}, nil
}
