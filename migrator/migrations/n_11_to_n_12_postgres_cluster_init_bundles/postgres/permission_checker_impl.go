package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
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

func checkAccess(_ context.Context, _ storage.Access) (bool, error) {
	return true, nil
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
