package utils

import (
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
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
		MainVersion:   ver.GetVersion(),
		SeqNum:        int(ver.GetSeqNum()),
		LastPersisted: timestamp.FromProtobuf(ver.GetLastPersisted()).GoTime(),
		MinimumSeqNum: int(ver.GetMinSeqNum()),
	}, nil
}
