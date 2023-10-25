package m192tom193

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 20
)

func createV2reportConfig(reportConfigProto *storage.ReportConfiguration) *storage.ReportConfiguration {
	// clone v1 report config
	newConfig := reportConfigProto.Clone()
	// populate id
	id := getDeterministicID(reportConfigProto.GetId())
	newConfig.Id = id
	// assign collection id in resource scope
	newConfig.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: reportConfigProto.GetScopeId()},
	}
	// set scope id to empty string so that v2 api does filter out v2 configs
	newConfig.ScopeId = ""
	// add vuln report filter to v2 copy of report config
	vulnFilter := &storage.VulnerabilityReportFilters{
		Severities: reportConfigProto.GetVulnReportFilters().GetSeverities(),
		Fixability: reportConfigProto.GetVulnReportFilters().GetFixability(),
		ImageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
	}
	if reportConfigProto.GetVulnReportFilters().GetSinceLastReport() {
		vulnFilter.CvesSince = &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
			SinceLastSentScheduledReport: true,
		}
	} else {
		vulnFilter.CvesSince = &storage.VulnerabilityReportFilters_AllVuln{
			AllVuln: true,
		}
	}
	newConfig.Filter = &storage.ReportConfiguration_VulnReportFilters{
		VulnReportFilters: vulnFilter,
	}
	return newConfig
}

func createNotifier(reportConfigProto *storage.ReportConfiguration) *storage.NotifierConfiguration {
	notifierConfig := &storage.NotifierConfiguration{
		Ref: &storage.NotifierConfiguration_Id{
			Id: reportConfigProto.GetEmailConfig().GetNotifierId()},
		NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				MailingLists: reportConfigProto.GetEmailConfig().GetMailingLists(),
			},
		}}
	return notifierConfig
}

func createReportSnapshot(v1Config *storage.ReportConfiguration, v2Config *storage.ReportConfiguration) *storage.ReportSnapshot {

	// create report snapshot for last successful scheduled report job
	if v1Config.GetLastSuccessfulRunTime() == nil {
		return nil
	}
	if v1Config.GetLastRunStatus() != nil {
		return &storage.ReportSnapshot{
			ReportConfigurationId: v2Config.GetId(),
			Name:                  v2Config.GetName(),
			Description:           v2Config.GetDescription(),
			Type:                  storage.ReportSnapshot_VULNERABILITY,
			ReportId:              uuid.NewV4().String(),
			Collection: &storage.CollectionSnapshot{
				Id: v2Config.GetResourceScope().GetCollectionId(),
			},
			Schedule: v2Config.GetSchedule(),
			ReportStatus: &storage.ReportStatus{
				RunState:          storage.ReportStatus_DELIVERED,
				ReportRequestType: storage.ReportStatus_SCHEDULED,
				CompletedAt:       v1Config.LastSuccessfulRunTime,
			},
		}
	}
	return nil
}

func checkifNotifierExists(notifierID string, db *gorm.DB, dbctx context.Context) (bool, error) {
	var id string
	row := db.WithContext(dbctx).Table(updatedSchema.NotifiersTableName).Select("id").Where(&updatedSchema.Notifiers{ID: notifierID}).Limit(1).Find(&id)
	if row.Error != nil {
		return false, row.Error
	}
	return id != "", nil
}

func getMigratedReportConfigIfExists(reportID string, db *gorm.DB, dbctx context.Context) (bool, *storage.ReportConfiguration, error) {
	var reportConfig updatedSchema.ReportConfigurations
	row := db.WithContext(dbctx).Table(updatedSchema.ReportConfigurationsTableName).Select("serialized").Where(&updatedSchema.ReportConfigurations{ID: reportID}).Limit(1).Find(&reportConfig)
	if row.Error != nil {
		return false, nil, row.Error
	}
	reportv2ConfigProto, err := updatedSchema.ConvertReportConfigurationToProto(&reportConfig)
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to convert %+v to proto", reportConfig)
	}
	if reportv2ConfigProto.GetId() != "" {
		return true, reportv2ConfigProto, err
	}
	return false, nil, err
}

func migrate(database *types.Databases) error {

	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportConfigurationsStmt)
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportSnapshotsStmt)
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportConfigurationsStmt.Children[0])
	db = db.WithContext(database.DBCtx)
	return db.Transaction(func(tx *gorm.DB) error {
		query := tx.WithContext(database.DBCtx).Table(updatedSchema.ReportConfigurationsTableName).Select("serialized")
		rows, err := query.Rows()
		if err != nil {
			return errors.Wrap(err, "failed to iterate table report configurations")
		}
		defer func() { _ = rows.Close() }()
		var convertedReportConfigs []*updatedSchema.ReportConfigurations
		var reportSnapshots []*updatedSchema.ReportSnapshots
		var reportNotifiers []*updatedSchema.ReportConfigurationsNotifiers
		for rows.Next() {
			// convert to report config proto
			var reportConfig *updatedSchema.ReportConfigurations
			if err = query.ScanRows(rows, &reportConfig); err != nil {
				return errors.Wrap(err, "failed to scan rows")
			}
			reportConfigProto, err := updatedSchema.ConvertReportConfigurationToProto(reportConfig)
			if err != nil {
				return errors.Wrapf(err, "failed to convert report config from gorm to proto %+v ", reportConfig)
			}
			if reportConfigProto.Version == 2 {
				continue
			}
			// if version=0 and scope id is not nil it is v2 config created in tech preview. just set version = 2
			if reportConfigProto.Version == 0 && reportConfigProto.GetResourceScope() != nil {
				reportConfigProto.Version = 2
				// convert report config proto back to gorm model
				convertedGormConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
				if err == nil {
					convertedReportConfigs = append(convertedReportConfigs, convertedGormConfig)
				} else {

					log.Errorf("failed to convert report config from proto %+v with error %+v", reportConfigProto, err)
				}

				continue
			}

			//since checkifNotifierExsists only reads data from older migration, no need to write a new tx
			notifierFound, err := checkifNotifierExists(reportConfigProto.GetEmailConfig().GetNotifierId(), db, database.DBCtx)
			if err != nil {
				return errors.Wrapf(err, "failed to query notifier with id %s", reportConfigProto.GetEmailConfig().GetNotifierId())
			}

			if !notifierFound {
				log.Errorf("Did not find notifier with id %s attached to report config %+v", reportConfigProto.GetEmailConfig().GetNotifierId(),
					reportConfigProto)
				continue
			}

			// create v2 report config from v1
			newConfig := createV2reportConfig(reportConfigProto)

			// create notifier config for report_configuration_notifier table
			notifierConfig := createNotifier(reportConfigProto)
			newConfig.Notifiers = append(newConfig.Notifiers, notifierConfig)
			// assign version to 2 to new copy and version to 1 in original so that they are not re-created during migration
			newConfig.Version = 2
			reportConfigProto.Version = 1

			//if deterministic id exists no need to copy the config
			//since getMigratedReportConfigIfExsists only reads data from older migration, no need to write a new tx
			migrated, data, err := getMigratedReportConfigIfExists(newConfig.GetId(), db, database.DBCtx)
			if err != nil {
				return errors.Wrapf(err, "failed to query v2 report config with id %s", newConfig.GetId())
			}
			if migrated {
				if !reflect.DeepEqual(newConfig, data) {
					log.Errorf("v1 report config %+v was migrated to create v2 report config.The v1 config has changed since last migration. Skipping re-migration...", reportConfigProto)
				}
				continue
			}

			// add notifier to notifier_configurations_notifiers
			reportNotifierGorm, err := updatedSchema.ConvertNotifierConfigurationFromProto(notifierConfig, 0, newConfig.GetId())
			if err != nil {
				log.Errorf("failed to convert notifier config from proto to gorm %+v with error %+v", notifierConfig, err)
				continue
			}

			// convert report config proto back to gorm model
			convertedGormNewConfig, err := updatedSchema.ConvertReportConfigurationFromProto(newConfig)
			if err != nil {
				log.Errorf("failed to convert report config from proto to gorm %+v with error %+v", newConfig, err)
				continue
			}
			convertedGormReportConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
			if err != nil {
				log.Errorf("failed to convert report config from proto to gorm %+v with error %+v", reportConfigProto, err)
				continue
			}

			// create report snapshot for last run report job
			reportSnapshot := createReportSnapshot(reportConfigProto, newConfig)
			if reportSnapshot != nil {
				// convert report snapshot to GORM
				reportSnapshotGORM, err := updatedSchema.ConvertReportSnapshotFromProto(reportSnapshot)
				if err != nil {
					log.Errorf("failed to convert snapshot from proto to gorm  %+v with error %+v", reportSnapshot, err)
					continue
				}
				reportSnapshots = append(reportSnapshots, reportSnapshotGORM)

			}
			reportNotifiers = append(reportNotifiers, reportNotifierGorm)
			convertedReportConfigs = append(convertedReportConfigs, convertedGormNewConfig, convertedGormReportConfig)

		}
		if rows.Err() != nil {
			return errors.Wrap(rows.Err(), "failed to get rows for report_configurations")
		}

		if err = updateTables(tx, convertedReportConfigs, reportSnapshots, reportNotifiers); err != nil {
			return errors.Wrap(err, "failed to upsert report configs")
		}

		return nil
	})
}

func updateTables(tx *gorm.DB, reportConfigs []*updatedSchema.ReportConfigurations, snapshots []*updatedSchema.ReportSnapshots, notifiers []*updatedSchema.ReportConfigurationsNotifiers) error {

	for len(reportConfigs) >= batchSize {
		updateConfig := reportConfigs[0:batchSize]
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&updateConfig).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report configs")
		}
		reportConfigs = reportConfigs[batchSize:]
	}
	for len(snapshots) >= batchSize {
		updateSnapshot := snapshots[0:batchSize]
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&updateSnapshot).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
		snapshots = snapshots[batchSize:]
	}

	for len(notifiers) > batchSize {
		updateNotifiers := notifiers[0:batchSize]
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.Children[0].GormModel).Create(&updateNotifiers).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report notifier configurations")
		}
		notifiers = notifiers[batchSize:]
	}

	if len(reportConfigs) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&reportConfigs).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
	}

	if len(snapshots) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&snapshots).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
	}

	if len(notifiers) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.Children[0].GormModel).Create(&notifiers).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report notifier configurations")
		}
	}

	return nil

}

func getDeterministicID(configID string) string {
	return uuid.NewV5(uuid.FromStringOrPanic(configID), "report config").String()
}
