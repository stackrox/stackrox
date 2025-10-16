package m192tom193

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 20
)

func createV2reportConfig(reportConfigProto *storage.ReportConfiguration) *storage.ReportConfiguration {
	// clone v1 report config
	newConfig := reportConfigProto.CloneVT()
	// populate id
	id := getDeterministicID(reportConfigProto.GetId())
	newConfig.SetId(id)
	// assign collection id in resource scope
	rs := &storage.ResourceScope{}
	rs.SetCollectionId(reportConfigProto.GetScopeId())
	newConfig.SetResourceScope(rs)
	// set scope id to empty string so that v2 api does filter out v2 configs
	newConfig.SetScopeId("")
	// add vuln report filter to v2 copy of report config
	vulnFilter := &storage.VulnerabilityReportFilters{}
	vulnFilter.SetSeverities(reportConfigProto.GetVulnReportFilters().GetSeverities())
	vulnFilter.SetFixability(reportConfigProto.GetVulnReportFilters().GetFixability())
	vulnFilter.SetImageTypes([]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED})
	if reportConfigProto.GetVulnReportFilters().GetSinceLastReport() {
		vulnFilter.SetSinceLastSentScheduledReport(true)
	} else {
		vulnFilter.SetAllVuln(true)
	}
	newConfig.SetVulnReportFilters(proto.ValueOrDefault(vulnFilter))
	return newConfig
}

func createNotifier(reportConfigProto *storage.ReportConfiguration) *storage.NotifierConfiguration {
	enc := &storage.EmailNotifierConfiguration{}
	enc.SetMailingLists(reportConfigProto.GetEmailConfig().GetMailingLists())
	notifierConfig := &storage.NotifierConfiguration{}
	notifierConfig.SetId(reportConfigProto.GetEmailConfig().GetNotifierId())
	notifierConfig.SetEmailConfig(proto.ValueOrDefault(enc))
	return notifierConfig
}

func createReportSnapshot(v1Config *storage.ReportConfiguration, v2Config *storage.ReportConfiguration) *storage.ReportSnapshot {

	// create report snapshot for last successful scheduled report job
	if v1Config.GetLastSuccessfulRunTime() == nil {
		return nil
	}
	if v1Config.GetLastRunStatus() != nil {
		cs := &storage.CollectionSnapshot{}
		cs.SetId(v2Config.GetResourceScope().GetCollectionId())
		rs := &storage.ReportStatus{}
		rs.SetRunState(storage.ReportStatus_DELIVERED)
		rs.SetReportRequestType(storage.ReportStatus_SCHEDULED)
		rs.SetCompletedAt(v1Config.GetLastSuccessfulRunTime())
		rs2 := &storage.ReportSnapshot{}
		rs2.SetReportConfigurationId(v2Config.GetId())
		rs2.SetName(v2Config.GetName())
		rs2.SetDescription(v2Config.GetDescription())
		rs2.SetType(storage.ReportSnapshot_VULNERABILITY)
		rs2.SetReportId(uuid.NewV4().String())
		rs2.SetCollection(cs)
		rs2.SetSchedule(v2Config.GetSchedule())
		rs2.SetReportStatus(rs)
		rs2.SetVulnReportFilters(proto.ValueOrDefault(v2Config.GetVulnReportFilters().CloneVT()))
		return rs2
	}
	return nil
}

func checkifNotifierExists(dbctx context.Context, notifierID string, db *gorm.DB) (bool, error) {
	var id string
	row := db.WithContext(dbctx).Table(updatedSchema.NotifiersTableName).Select("id").Where(&updatedSchema.Notifiers{ID: notifierID}).Limit(1).Find(&id)
	if row.Error != nil {
		return false, row.Error
	}
	return id != "", nil
}

func getMigratedReportConfigIfExists(dbctx context.Context, reportID string, db *gorm.DB) (bool, *storage.ReportConfiguration, error) {
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
			if reportConfigProto.GetVersion() == 2 {
				continue
			}
			// if version=0 and scope id is not nil it is v2 config created in tech preview. just set version = 2
			if reportConfigProto.GetVersion() == 0 && reportConfigProto.GetResourceScope() != nil {
				reportConfigProto.SetVersion(2)
				// convert report config proto back to gorm model
				convertedGormConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
				if err == nil {
					convertedReportConfigs = append(convertedReportConfigs, convertedGormConfig)
				} else {

					log.Errorf("failed to convert report config from proto %+v with error %+v", reportConfigProto, err)
				}

				continue
			}

			// since checkifNotifierExsists only reads data from older migration, no need to write a new tx
			notifierFound, err := checkifNotifierExists(database.DBCtx, reportConfigProto.GetEmailConfig().GetNotifierId(), db)
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
			newConfig.SetNotifiers(append(newConfig.GetNotifiers(), notifierConfig))
			// assign version to 2 to new copy and version to 1 in original so that they are not re-created during migration
			newConfig.SetVersion(2)
			reportConfigProto.SetVersion(1)

			// if deterministic id exists no need to copy the config
			// since getMigratedReportConfigIfExsists only reads data from older migration, no need to write a new tx
			migrated, data, err := getMigratedReportConfigIfExists(database.DBCtx, newConfig.GetId(), db)
			if err != nil {
				return errors.Wrapf(err, "failed to query v2 report config with id %s", newConfig.GetId())
			}
			if migrated {
				if !newConfig.EqualVT(data) {
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
	if len(reportConfigs) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&reportConfigs).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
	}

	for len(snapshots) >= batchSize {
		updateSnapshot := snapshots[0:batchSize]
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&updateSnapshot).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
		snapshots = snapshots[batchSize:]
	}
	if len(snapshots) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&snapshots).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
	}

	for len(notifiers) > batchSize {
		updateNotifiers := notifiers[0:batchSize]
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.Children[0].GormModel).Create(&updateNotifiers).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report notifier configurations")
		}
		notifiers = notifiers[batchSize:]
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
