package m221tom222

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	postgresStore "github.com/stackrox/rox/migrator/migrations/m_221_to_m_222_remove_v1_report_configs/postgres"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	configStore := postgresStore.New(database.PostgresDB)
	ctx := sac.WithAllAccess(context.Background())
	reportConfigIDs := []string{}

	err := configStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		// if report config version is 0 or 1 then delete the report config
		// configs with version 0 are v1 report configs not migrated to v2
		// configs with version 1 are v1 report config
		if reportConfig.GetVersion() < 2 {
			reportConfigIDs = append(reportConfigIDs, reportConfig.GetId())
		}
		return nil
	})

	for _, repConfigID := range reportConfigIDs {
		err := configStore.Delete(ctx, repConfigID)
		if err != nil {
			return err
		}

	}

	return err
}
