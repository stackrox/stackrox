//go:build !amd64

package runner

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func getCurrentSeqNumPostgres(databases *types.Databases) (int, error) {
	ver, err := version.ReadVersionGormDB(ctx, databases.GormDB)
	if err != nil {
		return 0, errors.Wrap(err, "getting current postgres sequence number")
	}

	return ver.SeqNum, nil
}
func getCurrentSeqNum(databases *types.Databases) (int, error) {
	return getCurrentSeqNumPostgres(databases)
}

func updateVersion(databases *types.Databases, newVersion *storage.Version) error {
	version.SetVersionGormDB(ctx, databases.GormDB, newVersion, false)
	return nil
}
