package m001tom002

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

const sleepDuration = 3 * time.Minute

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     1,
		VersionAfterSeqNum: 2,
		Description:        "Slow test migration that sleeps for 3 minutes to simulate processing",
		Run: func(ctx context.Context, db postgres.DB) error {
			log.Infof("Background migration m_001_to_m_002: sleeping for %s to simulate processing", sleepDuration)

			select {
			case <-time.After(sleepDuration):
				log.Info("Background migration m_001_to_m_002: completed successfully")
				return nil
			case <-ctx.Done():
				log.Info("Background migration m_001_to_m_002: interrupted by context cancellation")
				return ctx.Err()
			}
		},
	})
}
