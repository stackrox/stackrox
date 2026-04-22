package example

import (
	"context"

	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

// You must import a migration in ../../runner/all.go for this to be exececuted
func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     0,
		VersionAfterSeqNum: 1,
		Description:        "Test migration that logs a message",
		Run: func(ctx context.Context, db postgres.DB) error {
			log.Info("Background migration m_000_to_m_001: starting")
			log.Info("Background migration m_000_to_m_001: this is an example migration that simply logs")
			log.Info("Background migration m_000_to_m_001: completed successfully")
			return nil
		},
	})
}
