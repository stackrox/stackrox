package m192tom193

import (
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
	newConfig.Id = uuid.NewV5(uuid.FromStringOrPanic(reportConfigProto.GetId()), "report config").String()
	// assign collection id in resource scope
	newConfig.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: reportConfigProto.GetScopeId()},
	}
	//set scope id to empty string so that v2 api does filter out v2 configs
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
		if err = db.ScanRows(rows, &reportConfig); err != nil {
			return errors.Wrapf(err, "failed to scan rows")
		}
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
		newConfig := createV2reportConfig(reportConfigProto)

		// create notifier config for report_configuration_notifier table
		notifierConfig := createNotifier(reportConfigProto)
		newConfig.Notifiers = append(newConfig.Notifiers, notifierConfig)

		// add notifier to notifier_configurations_notifiers
		reportNotifierGorm, err := updatedSchema.ConvertNotifierConfigurationFromProto(notifierConfig, 0, newConfig.GetId())
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to convert from proto %+v", notifierConfig)
			errList.AddError(errMsg)
		}
		reportNotifiers = append(reportNotifiers, reportNotifierGorm)
		// assign version to 2 to new copy and version to 1 in original so that they are not re-created during migration
		newConfig.Version = 2
		reportConfigProto.Version = 1
		// convert report config proto back to gorm model
		convertedGormNewConfig, err := updatedSchema.ConvertReportConfigurationFromProto(newConfig)
		if err != nil {
			errMsg := errors.Wrapf(err, "failed to convert from proto %+v", newConfig)
			errList.AddError(errMsg)
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
				errMsg := errors.Wrapf(err, "failed to convert from proto %+v", reportSnapshot)
				errList.AddError(errMsg)
			}
			reportSnapshots = append(reportSnapshots, reportSnapshotGORM)
		}
		if len(convertedReportConfigs) == batchSize {
			err = updateTables(db, dbNotifier, dbSnapshot, convertedReportConfigs, reportSnapshots, reportNotifiers)
			convertedReportConfigs = convertedReportConfigs[:0]
			reportSnapshots = reportSnapshots[:0]
			reportNotifiers = reportNotifiers[:0]
			if err != nil {
				return err
			}
		}
	}

	if rows.Err() != nil {
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", "report_configurations")
	}

	err = updateTables(db, dbNotifier, dbSnapshot, convertedReportConfigs, reportSnapshots, reportNotifiers)
	if err != nil {
		return err
	}
	if !errList.Empty() {
		return errList.ToError()
	}
	return nil
}

//func updateTablestransaction(db *gorm.DB, dbNotifier *gorm.DB, dbSnapshot *gorm.DB, reportConfigs []*updatedSchema.ReportConfigurations, snapshots []*updatedSchema.ReportSnapshots, notifiers []*updatedSchema.ReportConfigurationsNotifiers) error {
//	tx := db.Session(&gorm.Session{SkipDefaultTransaction: true})
//
//	return db.Transaction(func(tx *gorm.DB) error {
//		if len(reportConfigs) > 0 {
//			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&reportConfigs).Error; err != nil {
//				return errors.Wrap(err, "failed to upsert converted report configs")
//			}
//		}
//
//		// return nil will commit the whole transaction
//		return nil
//	})
//}

func updateTables(db *gorm.DB, dbNotifier *gorm.DB, dbSnapshot *gorm.DB, reportConfigs []*updatedSchema.ReportConfigurations, snapshots []*updatedSchema.ReportSnapshots, notifiers []*updatedSchema.ReportConfigurationsNotifiers) error {

	if len(reportConfigs) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.GormModel).Create(&reportConfigs).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report configs")
		}
	}
	if len(snapshots) > 0 {
		if err := dbSnapshot.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportSnapshotsStmt.GormModel).Create(&snapshots).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report snapshots")
		}
	}

	if len(notifiers) > 0 {

		if err := dbNotifier.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableReportConfigurationsStmt.Children[0].GormModel).Create(&notifiers).Error; err != nil {
			return errors.Wrap(err, "failed to upsert converted report notifier configurations")
		}
	}
	return nil
}
