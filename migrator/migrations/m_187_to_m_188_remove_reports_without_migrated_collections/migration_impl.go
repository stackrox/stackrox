package m187tom188

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/m_187_to_m_188_remove_reports_without_migrated_collections/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

func migrate(db *gorm.DB) error {
	ctx := sac.WithAllAccess(context.Background())

	rows, err := db.WithContext(ctx).Table(schema.ReportConfigurationsTableName).Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", schema.ReportConfigurationsTableName)
	}
	if rows.Err() != nil {
		utils.Should(rows.Err())
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.ReportConfigurationsTableName)
	}
	defer func() { _ = rows.Close() }()

	collectionsTable := db.WithContext(ctx).Table(schema.CollectionsTableName).Session(&gorm.Session{})

	//var count int
	var toDelete []schema.ReportConfigurations
	for rows.Next() {
		var config schema.ReportConfigurations
		if err = db.ScanRows(rows, &config); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}
		configProto, err := schema.ConvertReportConfigurationToProto(&config)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", config)
		}

		//scope_id is not null then it is a V1 reportconfig
		reportID := configProto.GetId()
		scopeID := configProto.GetScopeId()
		collectionID := configProto.GetResourceScope().GetCollectionId()

		//scope_id is null/empty and collection_id is null/empty -> undefined, so delete
		if strings.TrimSpace(scopeID) == "" && strings.TrimSpace(collectionID) == "" {
			toDelete = append(toDelete, schema.ReportConfigurations{ID: reportID})
			continue
		}

		// If scope_id is empty but collection_id is not, then it's a V2, so skip
		// NOTE: This will skip even if the config has a scope_id (valid or invalid)
		if strings.TrimSpace(collectionID) != "" {
			continue
		}

		// Otherwise it's a V1 config, and scope_id is not empty, so check if it's pointing to a valid collection
		var count int64
		result := collectionsTable.Where(schema.Collections{ID: scopeID}).Count(&count)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// This looks like a valid error, but I don't think it should fail the entire migration, so log and move on
			log.Errorf("error while trying to fetch collection with id %s", scopeID)
			continue
		}

		// If no collection with that id could be found, it's an invalid config, so delete
		if count == 0 || errors.Is(result.Error, gorm.ErrRecordNotFound) {
			toDelete = append(toDelete, schema.ReportConfigurations{ID: reportID})
			continue
		}
	}

	if len(toDelete) != 0 {
		log.Infof("Deleting %d report configurations because they have invalid collections", len(toDelete))
		// Txn is probably overkill given that it's one where in delete query...
		return db.WithContext(ctx).Table(schema.ReportConfigurationsTableName).Transaction(func(tx *gorm.DB) error {
			result := tx.Delete(&toDelete)
			if result.Error != nil || result.RowsAffected != int64(len(toDelete)) {
				log.Errorf("failed to delete %d reports configurations with invalid collections", len(toDelete))
				return errors.Wrapf(result.Error, "failed to delete report configurations in batch")
			}
			return nil
		})
	}

	log.Debug("No invalid report configurations found")
	return nil
}
