package postgres

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// TODO(ROX-9887): Implement SAC logic from datastore
type permissionChecker struct{}

var (
	once     sync.Once
	instance PermissionChecker

	multiResourceSAC = sac.ForResources(
		sac.ForResource(permissions.ResourceMetadata{}),
	)
)

func permissionCheckerSingleton() PermissionChecker {
	once.Do(func() {
		instance = permissionChecker{}
	})
	return instance
}

func (permissionChecker) CountAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}

func (permissionChecker) ExistsAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}

func (permissionChecker) GetAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}

func (permissionChecker) UpsertAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return multiResourceSAC.WriteAllowedToAny(ctx)
}

func (permissionChecker) UpsertManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return multiResourceSAC.WriteAllowedToAny(ctx)
}

func (permissionChecker) DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return multiResourceSAC.WriteAllowedToAny(ctx)
}

func (permissionChecker) GetIDsAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}

func (permissionChecker) GetManyAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}

func (permissionChecker) DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return multiResourceSAC.WriteAllowedToAny(ctx)
}

func (permissionChecker) WalkAllowed(ctx context.Context) (bool, error) {
	return multiResourceSAC.ReadAllowedToAny(ctx)
}
