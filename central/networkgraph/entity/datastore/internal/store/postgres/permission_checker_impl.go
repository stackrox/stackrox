package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

type permissionCheckerImpl struct{}

var (
	once     sync.Once
	instance pgSearch.PermissionChecker

	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

func permissionCheckerSingleton() pgSearch.PermissionChecker {
	once.Do(func() {
		instance = permissionCheckerImpl{}
	})
	return instance
}

func genericPassThrough(ctx context.Context, access storage.Access) (bool, error) {
	scopeChecker := networkGraphSAC.ScopeChecker(ctx, access)
	eas, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: resources.NetworkGraph,
		Access:   access,
	})
	if err != nil {
		return false, err
	}
	if eas == nil {
		return false, nil
	}
	if eas.State == effectiveaccessscope.Excluded {
		return false, nil
	}
	return true, nil
}

func (permissionCheckerImpl) ReadAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) WriteAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_WRITE_ACCESS)
}
