package transitional

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac"
)

func scopeCheckerForIdentity(id authn.Identity) sac.ScopeCheckerCore {
	if id == nil {
		return sac.DenyAllAccessScopeChecker()
	}
	if id.Service() != nil {
		return sac.AllowAllAccessScopeChecker()
	}

	minimumAccess := computeMinimumAccess(id.Permissions())
	if minimumAccess == storage.Access_READ_WRITE_ACCESS {
		return sac.AllowAllAccessScopeChecker()
	}

	var readResources []permissions.ResourceHandle
	var writeResources []permissions.ResourceHandle

	for resourceName, access := range id.Permissions().GetResourceToAccess() {
		resource := permissions.Resource(resourceName)
		switch access {
		case storage.Access_READ_WRITE_ACCESS:
			writeResources = append(writeResources, resource)
			fallthrough
		case storage.Access_READ_ACCESS:
			readResources = append(readResources, resource)
		}
	}

	var readAccessSCC sac.ScopeCheckerCore
	if minimumAccess == storage.Access_READ_ACCESS {
		readAccessSCC = sac.AllowAllAccessScopeChecker()
	} else {
		readAccessSCC = sac.AllowFixedScopes(sac.ResourceScopeKeys(readResources...))
	}

	return sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS):       readAccessSCC,
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(sac.ResourceScopeKeys(writeResources...)),
	}
}

// LegacyAccessScopesContextEnricher enriches the given context with a global access scope checker
// that enforces access controls based on legacy roles and permissions.
func LegacyAccessScopesContextEnricher(ctx context.Context) (context.Context, error) {
	scc := scopeCheckerForIdentity(authn.IdentityFromContext(ctx))
	return sac.WithGlobalAccessScopeChecker(ctx, scc), nil
}

func computeMinimumAccess(r *storage.Role) storage.Access {
	minimumAccess := storage.Access_READ_WRITE_ACCESS
	for _, resource := range resources.ListAll() {
		if r.ResourceToAccess[string(resource)] < minimumAccess {
			minimumAccess = r.ResourceToAccess[string(resource)]
			if minimumAccess == storage.Access_NO_ACCESS {
				return minimumAccess
			}
		}
	}
	return minimumAccess
}
