package utils

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// ReadVersionPostgres - reads the version from the postgres database.
func ReadVersionPostgres(pool *pgxpool.Pool) (*migrations.MigrationVersion, error) {
	store := vStore.NewPostgres(pool)

	ver, err := store.GetVersion()
	if err != nil {
		utils.Should(err)
		return nil, err
	}

	return &migrations.MigrationVersion{
		MainVersion:   ver.Version,
		SeqNum:        int(ver.SeqNum),
		LastPersisted: timestamp.FromProtobuf(ver.GetLastPersisted()).GoTime(),
	}, nil
}

// SetCurrentVersionPostgres - sets the current version in the postgres database
func SetCurrentVersionPostgres(pool *pgxpool.Pool) {
	if curr, err := ReadVersionPostgres(pool); err != nil || curr.MainVersion != version.GetMainVersion() || curr.SeqNum != migrations.CurrentDBVersionSeqNum() {
		newVersion := &storage.Version{
			SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
			Version:       version.GetMainVersion(),
			LastPersisted: timestamp.Now().GogoProtobuf(),
		}
		setVersionPostgres(pool, newVersion)
	}
}

func setVersionPostgres(pool *pgxpool.Pool, updatedVersion *storage.Version) {
	store := vStore.NewPostgres(pool)

	err := store.UpdateVersion(updatedVersion)
	utils.CrashOnError(errors.Wrapf(err, "failed to write migration version to %s", pool.Config().ConnConfig.Database))
}
