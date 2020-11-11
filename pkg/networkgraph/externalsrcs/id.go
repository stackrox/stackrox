package externalsrcs

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sac"
)

// NewClusterScopedID returns a cluster scope type resource ID for external network resources.
func NewClusterScopedID(cluster, cidr string) (sac.ResourceID, error) {
	if cidr == "" {
		return sac.ResourceID{}, errors.New("CIDR must be provided")
	}
	return sac.NewClusterScopeResourceID(cluster, base64.RawURLEncoding.EncodeToString([]byte(cidr)))
}

// NewGlobalScopedScopedID returns a global scope type resource ID for external network resources.
func NewGlobalScopedScopedID(cidr string) (sac.ResourceID, error) {
	if cidr == "" {
		return sac.ResourceID{}, errors.New("CIDR must be provided")
	}
	return sac.NewGlobalScopeResourceID(base64.RawURLEncoding.EncodeToString([]byte(cidr)))
}

// CIDRFromID returns CIDR from external source ID.
func CIDRFromID(id string) (string, error) {
	resID, err := sac.ParseResourceID(id)
	if err != nil {
		return "", err
	}
	return CIDRFromResourceID(resID)
}

// CIDRFromResourceID returns CIDR from external source resource typed ID.
func CIDRFromResourceID(id sac.ResourceID) (string, error) {
	cidr, err := base64.RawURLEncoding.DecodeString(id.Suffix())
	if err != nil {
		return "", errors.Wrapf(err, "decoding suffix %s to CIDR", id.Suffix())
	}
	return string(cidr), nil
}
