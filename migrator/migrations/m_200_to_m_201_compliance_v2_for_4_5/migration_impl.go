package m200tom201

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm/clause"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

// TODO(dont-merge): generate/write and import any store required for the migration (skip any unnecessary step):
//  - create a schema subdirectory
//  - create a schema/old subdirectory
//  - create a schema/new subdirectory
//  - create a stores subdirectory
//  - create a stores/previous subdirectory
//  - create a stores/updated subdirectory
//  - copy the old schemas from pkg/postgres/schema to schema/old
//  - copy the old stores from their location in central to appropriate subdirectories in stores/previous
//  - generate the new schemas in pkg/postgres/schema and the new stores where they belong
//  - copy the newly generated schemas from pkg/postgres/schema to schema/new
//  - remove the calls to GetSchemaForTable and to RegisterTable from the copied schema files
//  - remove the xxxTableName constant from the copied schema files
//  - copy the newly generated stores from their location in central to appropriate subdirectories in stores/updated
//  - remove any unused function from the copied store files (the minimum for the public API should contain Walk, UpsertMany, DeleteMany)
//  - remove the scoped access control code from the copied store files
//  - remove the metrics collection code from the copied store files

// TODO(dont-merge): Determine if this change breaks a previous releases database.
// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

func migrate(database *types.Databases) error {
	if err := migrateProfiles(database); err != nil {
		return err
	}

	if err := migrateRules(database); err != nil {
		return err
	}

	if err := migrateScans(database); err != nil {
		return err
	}

	return migrateResults(database)
}

func migrateProfiles(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, schema.CreateTableComplianceOperatorProfileV2Stmt)
	db = db.WithContext(database.DBCtx).Table(schema.ComplianceOperatorProfileV2TableName)
	query := db.WithContext(database.DBCtx).Table(schema.ComplianceOperatorProfileV2TableName).Select("serialized")

	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", schema.ComplianceOperatorProfileV2TableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedProfiles []*schema.ComplianceOperatorProfileV2
	var count int
	for rows.Next() {
		var profile *schema.ComplianceOperatorProfileV2
		if err = query.ScanRows(rows, &profile); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		profileProto, err := schema.ConvertComplianceOperatorProfileV2ToProto(profile)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", profile)
		}

		// Add the profile ref id
		profileProto.ProfileRefId = createProfileRefID(profileProto)

		converted, err := schema.ConvertComplianceOperatorProfileV2FromProto(profileProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", profileProto)
		}
		convertedProfiles = append(convertedProfiles, converted)
		count++

		if len(convertedProfiles) == batchSize {
			// Upsert converted blobs
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(schema.CreateTableComplianceOperatorProfileV2Stmt.GormModel).Create(&convertedProfiles).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedProfiles), count-len(convertedProfiles))
			}
			convertedProfiles = convertedProfiles[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", schema.ComplianceOperatorProfileV2TableName)
	}

	if len(convertedProfiles) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(schema.CreateTableComplianceOperatorProfileV2Stmt.GormModel).Create(&convertedProfiles).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedProfiles))
		}
	}
	log.Infof("Converted %d profile records", count)

	return nil
}

func migrateRules(database *types.Databases) error {
	// Need to use store because of `RuleControls`

	return nil
}

func migrateScans(database *types.Databases) error {
	return nil
}

func migrateResults(database *types.Databases) error {
	return nil
}

func createProfileRefID(profile *storage.ComplianceOperatorProfileV2) string {
	interimUUID := buildDeterministicID(profile.GetClusterId(), profile.GetProfileId())

	return buildDeterministicID(interimUUID, profile.GetProductType())
}

func buildDeterministicID(part1 string, part2 string) string {
	baseUUID, err := uuid.FromString(part1)
	if err != nil {
		log.Error(err)
		return ""
	}
	return uuid.NewV5(baseUUID, part2).String()
}
