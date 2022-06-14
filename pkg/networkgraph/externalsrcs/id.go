package externalsrcs

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/sac"
)

// Zero out the host address from CIDR string since two distinct CIDRs could represent same network.
// For example, 10.10.10.10/16 and 10.10.10.20/16 represent the same network.

// NewClusterScopedID returns a cluster scope type resource ID for external network resources.
func NewClusterScopedID(cluster, cidr string) (sac.ResourceID, error) {
	ipNet, err := validateAndGetNetwork(cidr)
	if err != nil {
		return sac.ResourceID{}, err
	}
	return sac.NewClusterScopeResourceID(cluster, encode(ipNet))
}

// NewGlobalScopedScopedID returns a global scope type resource ID for external network resources.
func NewGlobalScopedScopedID(cidr string) (sac.ResourceID, error) {
	ipNet, err := validateAndGetNetwork(cidr)
	if err != nil {
		return sac.ResourceID{}, err
	}
	return sac.NewGlobalScopeResourceID(encode(ipNet))
}

func validateAndGetNetwork(cidr string) (string, error) {
	if cidr == "" {
		return "", errors.New("CIDR must be provided")
	}

	ipNet := net.IPNetworkFromCIDR(cidr).String()
	if ipNet == "" {
		return "", errors.Errorf("CIDR %s is invalid", cidr)
	}

	return ipNet, nil
}

// NetworkFromID returns CIDR from external source ID.
func NetworkFromID(id string) (string, error) {
	resID, err := sac.ParseResourceID(id)
	if err != nil {
		return "", err
	}
	return NetworkFromResourceID(resID)
}

// NetworkFromResourceID returns CIDR from external source resource typed ID.
func NetworkFromResourceID(id sac.ResourceID) (string, error) {
	cidr, err := base64.RawURLEncoding.DecodeString(id.Suffix())
	if err != nil {
		return "", errors.Wrapf(err, "decoding suffix %s to CIDR", id.Suffix())
	}
	return string(cidr), nil
}

func encode(ipNet string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ipNet))
}
