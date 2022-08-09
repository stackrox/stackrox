package postgres

import (
	"context"

	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

type permissionCheckerImpl struct{}

var (
	once     sync.Once
	instance PermissionChecker
)

func permissionCheckerSingleton() PermissionChecker {
	once.Do(func() {
		instance = permissionCheckerImpl{}
	})
	return instance
}

func (permissionCheckerImpl) CountAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) ExistsAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) GetAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) WalkAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) UpsertAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) UpsertManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) DeleteAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) GetIDsAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) GetManyAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (permissionCheckerImpl) DeleteManyAllowed(ctx context.Context, keys ...sac.ScopeKey) (bool, error) {
	return true, nil
}
