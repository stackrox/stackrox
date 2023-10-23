package m192tom193

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

func checkifNotifierExsists(notifiertID string, tx *gorm.DB, dbctx context.Context) (bool, error) {

	queryv1 := tx.WithContext(dbctx).Table(updatedSchema.NotifiersTableName).Select("serialized").Where(&updatedSchema.Notifiers{ID: notifiertID})
	rowsNotifier, err := queryv1.Rows()
	if err != nil {
		return false, errors.Wrapf(err, "failed to iterate table notifiers")
	}
	defer func() { _ = rowsNotifier.Close() }()
	if rowsNotifier.Next() {
		return true, nil

	}
	return false, nil
}

func getMigratedReportConfigIfExsists(reportID string, tx *gorm.DB, dbctx context.Context) (bool, *storage.ReportConfiguration, error) {
	queryv1 := tx.WithContext(dbctx).Table(updatedSchema.ReportConfigurationsTableName).Select("serialized").Where(&updatedSchema.ReportConfigurations{ID: getDeterministicID(reportID)})
	rowsV1config, err := queryv1.Rows()
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to iterate table %s", "report_configurations")
	}
	defer func() { _ = rowsV1config.Close() }()
	for rowsV1config.Next() {
		var reportConfigv2 *updatedSchema.ReportConfigurations
		if err = tx.ScanRows(rowsV1config, &reportConfigv2); err != nil {
			return false, nil, errors.Wrapf(err, "failed to scan rows")
		}
		reportv2ConfigProto, err := updatedSchema.ConvertReportConfigurationToProto(reportConfigv2)
		if err != nil {
			return false, nil, errors.Wrapf(err, "failed to convert %+v to proto", reportv2ConfigProto)
		}
		return true, reportv2ConfigProto, nil

	}
	return false, nil, nil
}

func migrate(database *types.Databases) error {

	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportConfigurationsStmt)
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportSnapshotsStmt)
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportConfigurationsStmt.Children[0])
	db = db.WithContext(database.DBCtx)
	tx := db.Begin()
	query := db.WithContext(database.DBCtx).Table(updatedSchema.ReportConfigurationsTableName).Select("serialized")
	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", "report_configurations")
	}
	defer func() { _ = rows.Close() }()
	var convertedReportConfigs []*updatedSchema.ReportConfigurations
	var reportSnapshots []*updatedSchema.ReportSnapshots
	var reportNotifiers []*updatedSchema.ReportConfigurationsNotifiers
	var errList errorhelpers.ErrorList
	for rows.Next() {
		// convert to report config proto
		var reportConfig *updatedSchema.ReportConfigurations
		if err = query.ScanRows(rows, &reportConfig); err != nil {
			return errors.Wrapf(err, "failed to scan rows")
		}
		reportConfigProto, err := updatedSchema.ConvertReportConfigurationToProto(reportConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to convert report config  %+v to proto", reportConfigProto)
		}
		// skip if version=0 and scope id is not nil since it is v2 config created in tech preview
		if reportConfigProto.Version == 2 || (reportConfigProto.Version == 0 && reportConfigProto.GetResourceScope() != nil) {
			reportConfigProto.Version = 2
			// convert report config proto back to gorm model
			convertedGormConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
			if err != nil {
				return errors.Wrapf(err, "failed to convert report config from proto %+v", reportConfigProto)
			}
			convertedReportConfigs = append(convertedReportConfigs, convertedGormConfig)

			continue
		}

		//if deterministic id exists no need to copy the config
		migrated, data, err := getMigratedReportConfigIfExsists(reportConfigProto.GetId(), tx, database.DBCtx)
		if err != nil {
			return errors.Wrapf(err, "failed to query report config with id %s", reportConfigProto.GetId())
		}
		if migrated {
			if reflect.DeepEqual(reportConfig, data) {
				log.Infof("Old v1 report config proto %+v is different from v2 copy", data)
			}
			continue
		}

		notifierFound, err := checkifNotifierExsists(reportConfigProto.GetEmailConfig().GetNotifierId(), tx, database.DBCtx)
		if err != nil {
			return errors.Wrapf(err, "failed to query notifier with id %s", reportConfigProto.GetEmailConfig().GetNotifierId())
		}

		if !notifierFound {
			log.Infof("Did not find notifier with id %s attached to report config %+v", reportConfigProto.GetEmailConfig().GetNotifierId(),
				reportConfigProto)
			log.Infof("Did not find notifier with id %s.converted configs %+v", reportConfigProto.GetEmailConfig().GetNotifierId(),
				convertedReportConfigs)
			continue
		}

		// create v2 report config from v1
		newConfig := createV2reportConfig(reportConfigProto)

		// create notifier config for report_configuration_notifier table
		notifierConfig := createNotifier(reportConfigProto)
		newConfig.Notifiers = append(newConfig.Notifiers, notifierConfig)

		// add notifier to notifier_configurations_notifiers
		reportNotifierGorm, err := updatedSchema.ConvertNotifierConfigurationFromProto(notifierConfig, 0, newConfig.GetId())
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to convert notifier config from proto %+v", notifierConfig)
			errList.AddError(errMsg)
		}
		reportNotifiers = append(reportNotifiers, reportNotifierGorm)
		// assign version to 2 to new copy and version to 1 in original so that they are not re-created during migration
		newConfig.Version = 2
		reportConfigProto.Version = 1
		// convert report config proto back to gorm model
		convertedGormNewConfig, err := updatedSchema.ConvertReportConfigurationFromProto(newConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to convert report config from proto %+v", newConfig)
		}
		convertedGormReportConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to convert from proto %+v", reportConfigProto)
			errList.AddError(errMsg)
		}
		convertedReportConfigs = append(convertedReportConfigs, convertedGormNewConfig, convertedGormReportConfig)

		// create report snapshot for last run report job
		reportSnapshot := createReportSnapshot(reportConfigProto, newConfig)
		if reportSnapshot != nil {
			// convert report snapshot to GORM
			reportSnapshotGORM, err := updatedSchema.ConvertReportSnapshotFromProto(reportSnapshot)
			if err != nil {
				errMsg := errors.Wrapf(err, "failed to convert report snapshot from proto %+v", reportSnapshot)
				errList.AddError(errMsg)
			}
			reportSnapshots = append(reportSnapshots, reportSnapshotGORM)
		}
		if len(convertedReportConfigs) == batchSize {
			if err = updateTables(tx, convertedReportConfigs, reportSnapshots, reportNotifiers); err != nil {
				result := tx.Rollback()
				if result.Error != nil {
					return errors.Wrapf(result.Error, "failed to rollback with error")
				}

			}
			convertedReportConfigs = convertedReportConfigs[:0]
			reportSnapshots = reportSnapshots[:0]
			reportNotifiers = reportNotifiers[:0]
		}
	}

	if rows.Err() != nil {
		return errors.Wrap(rows.Err(), "failed to get rows for report_configurations")
	}

	if err = updateTables(tx, convertedReportConfigs, reportSnapshots, reportNotifiers); err != nil {
		result := tx.Rollback()
		if result.Error != nil {
			return errors.Wrapf(result.Error, "failed to rollback with error")
		}

	}

	if !errList.Empty() {
		log.Error(errList)
	}
	return tx.Commit().Error

}

func updateTables(tx *gorm.DB, reportConfigs []*updatedSchema.ReportConfigurations, snapshots []*updatedSchema.ReportSnapshots, notifiers []*updatedSchema.ReportConfigurationsNotifiers) error {

	if len(reportConfigs) > 0 {

		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&reportConfigs).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report configs")
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
