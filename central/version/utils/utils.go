package utils

import (
	"github.com/pkg/errors"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// ReadVersionPostgres - reads the version from the postgres database.
func ReadVersionPostgres(pool postgres.DB) (*migrations.MigrationVersion, error) {
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
		MinimumSeqNum: int(ver.MinSeqNum),
	}, nil
}

// ReadPreviousVersionPostgres - reads the version from the postgres database.
// TODO(ROX-18005) -- remove this.  During transition away from serialized version, UpgradeStatus will make this call against
// the older database.  In that case we will need to process the serialized data.
func ReadPreviousVersionPostgres(pool postgres.DB) (*migrations.MigrationVersion, error) {
	store := vStore.NewPostgres(pool)

	ver, err := store.GetPreviousVersion()
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
func SetCurrentVersionPostgres(pool postgres.DB) {
	if curr, err := ReadVersionPostgres(pool); err != nil ||
		curr.MainVersion != version.GetMainVersion() ||
		curr.SeqNum != migrations.CurrentDBVersionSeqNum() ||
		curr.MinimumSeqNum != migrations.MinimumSupportedDBVersionSeqNum() {
		newVersion := &storage.Version{
			SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
			Version:       version.GetMainVersion(),
			MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum()),
			LastPersisted: timestamp.Now().GogoProtobuf(),
		}
		setVersionPostgres(pool, newVersion)
	}
}

func setVersionPostgres(pool postgres.DB, updatedVersion *storage.Version) {
	store := vStore.NewPostgres(pool)

	err := store.UpdateVersion(updatedVersion)
	utils.CrashOnError(errors.Wrapf(err, "failed to write migration version to %s", pool.Config().ConnConfig.Database))
}
