package postgres

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sync"
)

type permissionCheckerImpl struct{}

var (
	once     sync.Once
	instance PermissionChecker

	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

func permissionCheckerSingleton() PermissionChecker {
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

func (permissionCheckerImpl) CountAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) ExistsAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) GetAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) WalkAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) UpsertAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionCheckerImpl) UpsertManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionCheckerImpl) DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionCheckerImpl) GetIDsAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) GetManyAllowed(ctx context.Context) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_ACCESS)
}

func (permissionCheckerImpl) DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return genericPassThrough(ctx, storage.Access_READ_WRITE_ACCESS)
}
