package transitional

import (
	"context"

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

	var globalAccessModes []storage.Access
	switch id.Role().GlobalAccess {
	case storage.Access_READ_WRITE_ACCESS:
		globalAccessModes = append(globalAccessModes, storage.Access_READ_WRITE_ACCESS)
		fallthrough
	case storage.Access_READ_ACCESS:
		globalAccessModes = append(globalAccessModes, storage.Access_READ_ACCESS)
	}
	if len(globalAccessModes) > 0 {
		return sac.AllowFixedScopes(sac.AccessModeScopeKeys(globalAccessModes...))
	}

	var readResources []permissions.ResourceHandle
	var writeResources []permissions.ResourceHandle

	for resourceName, access := range id.Role().GetResourceToAccess() {
		resource := permissions.Resource(resourceName)
		switch access {
		case storage.Access_READ_WRITE_ACCESS:
			writeResources = append(writeResources, resource)
			fallthrough
		case storage.Access_READ_ACCESS:
			readResources = append(readResources, resource)
		}
	}

	return sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS):       sac.AllowFixedScopes(sac.ResourceScopeKeys(readResources...)),
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(sac.ResourceScopeKeys(writeResources...)),
	}
}

// LegacyAccessScopesContextEnricher enriches the given context with a global access scope checker
// that enforces access controls based on legacy roles and permissions.
func LegacyAccessScopesContextEnricher(ctx context.Context) (context.Context, error) {
	scc := scopeCheckerForIdentity(authn.IdentityFromContext(ctx))
	return sac.WithGlobalAccessScopeChecker(ctx, scc), nil
}
