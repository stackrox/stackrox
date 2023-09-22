package m192tom193

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_192_to_m_193_report_config_migrations/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm/clause"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 3
)

func create_v2_report_config(reportConfigProto *storage.ReportConfiguration) *storage.ReportConfiguration {
	// clone v1 report config
	newConfig := reportConfigProto.Clone()
	// populate id
	newConfig.Id = uuid.NewV4().String()
	// assign collection id in resource scope
	newConfig.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: reportConfigProto.GetScopeId()},
	}
	//set scopeid to empty string so that v2 api does filter out configs as v1
	newConfig.ScopeId = ""
	// add vuln report filter to v2 copy of report config
	vulnFilter := &storage.VulnerabilityReportFilters{
		Severities: reportConfigProto.GetVulnReportFilters().GetSeverities(),
		Fixability: reportConfigProto.GetVulnReportFilters().GetFixability(),
		ImageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED,
			storage.VulnerabilityReportFilters_WATCHED},
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

func create_notifier(reportConfigProto *storage.ReportConfiguration) *storage.NotifierConfiguration {
	notifierConfig := &storage.NotifierConfiguration{
		Ref: &storage.NotifierConfiguration_Id{
			Id: reportConfigProto.GetEmailConfig().GetNotifierId()},
		NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				MailingLists:  reportConfigProto.GetEmailConfig().GetMailingLists(),
				CustomSubject: reportConfigProto.GetEmailConfig().GetCustomSubject(),
				CustomBody:    reportConfigProto.GetEmailConfig().GetCustomBody(),
			},
		}}
	return notifierConfig
}

func create_report_snapshot(v1Config *storage.ReportConfiguration, v2Config *storage.ReportConfiguration) *storage.ReportSnapshot {

	// create report snapshot for last run report job
	var runState storage.ReportStatus_RunState
	if v1Config.GetLastRunStatus().GetReportStatus() == 1 {
		runState = storage.ReportStatus_FAILURE
	} else {
		runState = storage.ReportStatus_DELIVERED
	}
	reportSnapshot := &storage.ReportSnapshot{
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
			RunState:          runState,
			ReportRequestType: storage.ReportStatus_SCHEDULED,
			CompletedAt:       v1Config.LastSuccessfulRunTime,
		},
	}
	if v1Config.GetLastRunStatus() != nil {
		reportSnapshot.ReportStatus = &storage.ReportStatus{
			RunState:          runState,
			ReportRequestType: storage.ReportStatus_SCHEDULED,
			CompletedAt:       v1Config.LastSuccessfulRunTime,
		}
	}

	return reportSnapshot

}

func migrate(database *types.Databases) error {
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableReportConfigurationsStmt)
	db = db.WithContext(database.DBCtx).Table("report_configurations")

	dbSnapshot := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, dbSnapshot, updatedSchema.CreateTableReportSnapshotsStmt)
	dbSnapshot = dbSnapshot.WithContext(database.DBCtx).Table("report_snapshots")

	dbNotifier := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, dbNotifier, updatedSchema.CreateTableReportConfigurationsStmt.Children[0])
	dbNotifier = dbNotifier.WithContext(database.DBCtx).Table("report_configurations_notifiers")
	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", "report_configurations")
	}
	if rows.Err() != nil {
		utils.Should(rows.Err())
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", "report_configurations")
	}
	defer func() { _ = rows.Close() }()
	var convertedReportConfigs []*updatedSchema.ReportConfigurations
	var reportSnapshots []*updatedSchema.ReportSnapshots
	var reportNotifiers []*updatedSchema.ReportConfigurationsNotifiers
	for rows.Next() {
		var reportConfig *updatedSchema.ReportConfigurations
		if err = db.ScanRows(rows, &reportConfig); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}
		// convert to report config proto
		reportConfigProto, err := updatedSchema.ConvertReportConfigurationToProto(reportConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", reportConfigProto)
		}
		// skip if version=1 since config is migrated
		// skip if version=2
		if reportConfigProto.Version != 0 {
			continue
		}
		if reportConfigProto.Version == 0 && reportConfigProto.GetResourceScope() != nil {
			reportConfigProto.Version = 2
			// convert report config proto back to gorm model
			convertedGormConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
			if err != nil {
				return errors.Wrapf(err, "failed to convert from proto %+v", reportConfigProto)
			}
			convertedReportConfigs = append(convertedReportConfigs, convertedGormConfig)

			continue
		}
		// create v2 report config from v1
		newConfig := create_v2_report_config(reportConfigProto)

		// create notifier config for report_configuration_notifier table
		notifierConfig := create_notifier(reportConfigProto)
		newConfig.Notifiers = append(newConfig.Notifiers, notifierConfig)

		// add notifier to notifier_configurations_notifiers
		reportNotifierGorm, err := updatedSchema.ConvertNotifierConfigurationFromProto(notifierConfig, 0, newConfig.GetId())
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", notifierConfig)
		}
		reportNotifiers = append(reportNotifiers, reportNotifierGorm)

		// assign version to 2 to new copy and version to 1 in original so that they are not re-created during migration
		newConfig.Version = 2
		reportConfigProto.Version = 1
		// convert report config proto back to gorm model
		convertedGormNewConfig, err := updatedSchema.ConvertReportConfigurationFromProto(newConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", newConfig)
		}
		convertedGormReportConfig, err := updatedSchema.ConvertReportConfigurationFromProto(reportConfigProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", reportConfigProto)
		}
		convertedReportConfigs = append(convertedReportConfigs, convertedGormNewConfig, convertedGormReportConfig)

		// create report snapshot for last run report job
		reportSnapshot := create_report_snapshot(reportConfigProto, newConfig)
		if reportConfigProto.GetLastRunStatus() != nil {
			// convert report snapshot to GORM
			reportSnapshotGORM, err := updatedSchema.ConvertReportSnapshotFromProto(reportSnapshot)
			if err != nil {
				return errors.Wrapf(err, "failed to convert from proto %+v", reportSnapshot)
			}
			reportSnapshots = append(reportSnapshots, reportSnapshotGORM)
		}
		if len(convertedReportConfigs) == batchSize {
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&convertedReportConfigs).Error; err != nil {
				return errors.Wrap(err, "failed to upsert converted report configs")
			}
			if err = dbSnapshot.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&reportSnapshots).Error; err != nil {
				return errors.Wrap(err, "failed to upsert converted report snapshots")
			}

			if err = dbNotifier.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableNotifiersStmt.GormModel).Create(&reportNotifiers).Error; err != nil {
				return errors.Wrap(err, "failed to upsert converted report notifier configurations")
			}
			convertedReportConfigs = convertedReportConfigs[:0]
			reportSnapshots = reportSnapshots[:0]
			reportNotifiers = reportNotifiers[:0]
		}
	}
	if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&convertedReportConfigs).Error; err != nil {
		return errors.Wrap(err, "failed to upsert converted report configs")
	}

	if err = dbSnapshot.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&reportSnapshots).Error; err != nil {
		return errors.Wrap(err, "failed to upsert converted report snapshots")
	}

	if err = dbNotifier.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableNotifiersStmt.GormModel).Create(&reportNotifiers).Error; err != nil {
		return errors.Wrap(err, "failed to upsert converted report notifier configurations")
	}

	return nil
}
