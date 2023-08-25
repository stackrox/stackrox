package postgres

import (
	"context"

	"github.com/pkg/errors"
	accessPkg "github.com/stackrox/rox/central/clusterinit/backend/access"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

type permissionChecker struct{}

var (
	once     sync.Once
	instance pgSearch.PermissionChecker
)

func permissionCheckerSingleton() pgSearch.PermissionChecker {
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

func (permissionChecker) ReadAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_ACCESS)
}

func (permissionChecker) WriteAllowed(ctx context.Context) (bool, error) {
	return checkAccess(ctx, storage.Access_READ_WRITE_ACCESS)
}
