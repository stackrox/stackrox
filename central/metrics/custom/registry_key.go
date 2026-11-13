package custom

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

// registryKeyForIdentity keeps a Sensor's registry stable across short-lived
// token renewals. Include the effective permissions so a changed metrics role
// never reuses series gathered with an older grant. The request identity itself
// is not changed: data sources still apply its own access scope and permissions.
func registryKeyForIdentity(ctx context.Context, id authn.Identity) string {
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

	grants := role.GetPermissions()
	// A removed cluster is absent from the request's effective access scope.
	// Do not reuse its cached registry while the old series are still present.
	clusterAllowed := false
	for name, access := range grants {
		if access != storage.Access_READ_ACCESS {
			continue
		}
		resource, known := resources.MetadataForResource(permissions.Resource(name))
		if known && (resource.GetScope() == permissions.ClusterScope || resource.GetScope() == permissions.NamespaceScope) &&
			sac.GlobalAccessScopeChecker(ctx).AccessMode(access).Resource(resource).ClusterID(clusterID).IsAllowed() {
			clusterAllowed = true
			break
		}
	}
	if !clusterAllowed {
		return id.UID()
	}
	names := make([]string, 0, len(grants))
	for name := range grants {
		names = append(names, name)
	}
	slices.Sort(names)
	var snapshot strings.Builder
	for _, name := range names {
		fmt.Fprintf(&snapshot, "%s=%d;", name, grants[name])
	}
	digest := sha256.Sum256([]byte(snapshot.String()))
	return fmt.Sprintf("sensor-metrics:%s:%x", clusterID, digest)
}
