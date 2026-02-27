package m220tom221

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	postgresStore "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_remove_v1_report_configs/postgres"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	configStore := postgresStore.New(database.PostgresDB)
	ctx := sac.WithAllAccess(context.Background())
	reportConfigs := []*storage.ReportConfiguration{}

	err := configStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		// if report config version is 0 or 1 then delete the report config
		// configs with version 0 are v1 report configs not migrated to v2
		if reportConfig.GetVersion() < 2 {
			reportConfigs = append(reportConfigs, reportConfig)
		}
		return nil
	})

	for _, repConfig := range reportConfigs {
		err := configStore.Delete(ctx, repConfig.GetId())
		if err != nil {
			return err
		}

	}

	return err
}
