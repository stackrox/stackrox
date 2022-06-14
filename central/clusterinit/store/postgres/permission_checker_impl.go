package postgres

import (
	"context"

	"github.com/pkg/errors"
	accessPkg "github.com/stackrox/stackrox/central/clusterinit/backend/access"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
)

type permissionChecker struct{}

var (
	once     sync.Once
	instance PermissionChecker
)

func permissionCheckerSingleton() PermissionChecker {
	once.Do(func() {
		instance = permissionChecker{}
	})
	return instance
}

func checkAccess(ctx context.Context, access storage.Access) (bool, error) {
	err := accessPkg.CheckAccess(ctx, access)
	if errors.Is(err, errox.NotAuthorized) {
		return false, nil
	}
	return err == nil, err
}

func (permissionChecker) CountAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) ExistsAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) GetAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) UpsertAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionChecker) UpsertManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionChecker) DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionChecker) GetIDsAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) GetManyAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_WRITE_ACCESS)
}

func (permissionChecker) WalkAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}
