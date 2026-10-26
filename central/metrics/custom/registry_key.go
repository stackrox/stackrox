package custom

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// registryKeyForIdentity keeps a Sensor's registry stable across short-lived
// token renewals. Include the effective permissions so a changed metrics role
// never reuses series gathered with an older grant. The request identity itself
// is not changed: data sources still apply its own access scope and permissions.
func registryKeyForIdentity(id authn.Identity) string {
	roles := id.Roles()
	if len(roles) != 1 {
		return id.UID()
	}
	role, ok := roles[0].(*tokens.InternalRole)
	if !ok || len(role.Clusters) != 1 {
		return id.UID()
	}
	var clusterID string
	for cluster, namespaces := range role.Clusters {
		if len(namespaces) != 1 || namespaces[0] != "*" {
			return id.UID()
		}
		clusterID = cluster
	}
	if clusterID == "" {
		return id.UID()
	}

	permissions := role.GetPermissions()
	names := make([]string, 0, len(permissions))
	for name := range permissions {
		names = append(names, name)
	}
	slices.Sort(names)
	var snapshot strings.Builder
	for _, name := range names {
		fmt.Fprintf(&snapshot, "%s=%d;", name, permissions[name])
	}
	digest := sha256.Sum256([]byte(snapshot.String()))
	return fmt.Sprintf("sensor-metrics:%s:%x", clusterID, digest)
}
