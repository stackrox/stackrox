package m200tom201

import (
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/schema/new"
	rulesStore "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/stores/rules"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm/clause"
)

const (
	controlAnnotationBase = "control.compliance.openshift.io/"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()

	scanTypeToString = map[storage.ScanType]string{
		storage.ScanType_NODE_SCAN:     "Node",
		storage.ScanType_PLATFORM_SCAN: "Platform",
	}
)

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
	pgutils.CreateTableFromModel(database.DBCtx, db, newSchema.CreateTableComplianceOperatorProfileV2Stmt)
	db = db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorProfileV2TableName)
	query := db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorProfileV2TableName).Select("serialized")

	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", newSchema.ComplianceOperatorProfileV2TableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedProfiles []*newSchema.ComplianceOperatorProfileV2
	var count int
	for rows.Next() {
		var profile *newSchema.ComplianceOperatorProfileV2
		if err = query.ScanRows(rows, &profile); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		profileProto, err := newSchema.ConvertComplianceOperatorProfileV2ToProto(profile)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", profile)
		}

		// Add the profile ref id
		profileProto.ProfileRefId = createProfileRefID(profileProto.GetClusterId(), profileProto.GetProfileId(), profileProto.GetProductType())

		converted, err := newSchema.ConvertComplianceOperatorProfileV2FromProto(profileProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", profileProto)
		}
		convertedProfiles = append(convertedProfiles, converted)
		count++

		if len(convertedProfiles) == batchSize {
			// Upsert converted profiles
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorProfileV2Stmt.GormModel).Create(&convertedProfiles).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d profiles after %d upserted", len(convertedProfiles), count-len(convertedProfiles))
			}
			convertedProfiles = convertedProfiles[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", newSchema.ComplianceOperatorProfileV2TableName)
	}

	if len(convertedProfiles) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorProfileV2Stmt.GormModel).Create(&convertedProfiles).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d profiles", len(convertedProfiles))
		}
	}
	log.Infof("Converted %d profile records", count)

	return nil
}

func migrateRules(database *types.Databases) error {
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, newSchema.CreateTableComplianceOperatorRuleV2Stmt)
	db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorRuleV2TableName)

	// Need to use store because of `RuleControls`
	// Since we are only using the walk to retrieve data, we can used the updated store to retrieve the data and update it.
	ruleStore := rulesStore.New(database.PostgresDB)

	ruleCount := 0
	rules := make([]*storage.ComplianceOperatorRuleV2, 0)
	err := ruleStore.Walk(database.DBCtx, func(obj *storage.ComplianceOperatorRuleV2) error {
		obj.ParentRule = obj.GetAnnotations()[v1alpha1.RuleIDAnnotationKey]
		obj.RuleRefId = buildDeterministicID(obj.GetClusterId(), obj.GetParentRule())

		var newControls []*storage.RuleControls
		for _, control := range obj.GetControls() {
			controlAnnotationValues := strings.Split(obj.GetAnnotations()[controlAnnotationBase+control.GetStandard()], ";")

			// Add a control entry for each Control + Standard. This data is intentionally denormalized for easier querying.
			for _, controlValue := range controlAnnotationValues {
				newControls = append(newControls, &storage.RuleControls{
					Standard: control.GetStandard(),
					Control:  controlValue,
				})
			}
		}
		obj.Controls = newControls

		rules = append(rules, obj)
		if len(rules) >= batchSize {
			err := ruleStore.UpsertMany(database.DBCtx, rules)
			if err != nil {
				return errors.Wrapf(err, "failed to convert %+v to proto", rules)
			}
			ruleCount = ruleCount + len(rules)
			rules = rules[:0]
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(rules) > 0 {
		ruleCount = ruleCount + len(rules)
		if err := ruleStore.UpsertMany(database.DBCtx, rules); err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", rules)
		}
	}

	log.Infof("Converted %d rule records", ruleCount)
	return nil
}

func migrateScans(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, newSchema.CreateTableComplianceOperatorScanV2Stmt)
	db = db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorScanV2TableName)
	query := db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorScanV2TableName).Select("serialized")

	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", newSchema.ComplianceOperatorScanV2TableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedScans []*newSchema.ComplianceOperatorScanV2
	var count int
	for rows.Next() {
		var scan *newSchema.ComplianceOperatorScanV2
		if err = query.ScanRows(rows, &scan); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		scanProto, err := newSchema.ConvertComplianceOperatorScanV2ToProto(scan)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", scan)
		}

		// Add the profile ref id and scan ref id
		scanProto.ProductType = scanTypeToString[scanProto.GetScanType()]
		scanProto.ScanRefId = buildDeterministicID(scanProto.GetClusterId(), scanProto.GetScanName())
		if scanProto.Profile == nil {
			return errors.Wrapf(err, "failed to set profile %+v to proto", scan)
		}
		scanProto.Profile.ProfileRefId = createProfileRefID(scanProto.GetClusterId(), scanProto.Profile.ProfileId, scanProto.GetProductType())

		converted, err := newSchema.ConvertComplianceOperatorScanV2FromProto(scanProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", scanProto)
		}
		convertedScans = append(convertedScans, converted)
		count++

		if len(convertedScans) == batchSize {
			// Upsert converted scans
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorScanV2Stmt.GormModel).Create(&convertedScans).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d scans after %d upserted", len(convertedScans), count-len(convertedScans))
			}
			convertedScans = convertedScans[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", newSchema.ComplianceOperatorScanV2TableName)
	}

	if len(convertedScans) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorScanV2Stmt.GormModel).Create(&convertedScans).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d scans", len(convertedScans))
		}
	}
	log.Infof("Converted %d scan records", count)

	return nil
}

func migrateResults(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, newSchema.CreateTableComplianceOperatorCheckResultV2Stmt)
	db = db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorCheckResultV2TableName)
	query := db.WithContext(database.DBCtx).Table(newSchema.ComplianceOperatorCheckResultV2TableName).Select("serialized")

	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", newSchema.ComplianceOperatorCheckResultV2TableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedResults []*newSchema.ComplianceOperatorCheckResultV2
	var count int
	for rows.Next() {
		var result *newSchema.ComplianceOperatorCheckResultV2
		if err = query.ScanRows(rows, &result); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		resultProto, err := newSchema.ConvertComplianceOperatorCheckResultV2ToProto(result)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", result)
		}

		// Add the scan_ref_id and rule_ref_id
		resultProto.ScanRefId = buildDeterministicID(resultProto.GetClusterId(), resultProto.GetScanName())
		resultProto.RuleRefId = buildDeterministicID(resultProto.GetClusterId(), resultProto.GetAnnotations()[v1alpha1.RuleIDAnnotationKey])

		converted, err := newSchema.ConvertComplianceOperatorCheckResultV2FromProto(resultProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", resultProto)
		}
		convertedResults = append(convertedResults, converted)
		count++

		if len(convertedResults) == batchSize {
			// Upsert converted check results
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorCheckResultV2Stmt.GormModel).Create(&convertedResults).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d check results after %d upserted", len(convertedResults), count-len(convertedResults))
			}
			convertedResults = convertedResults[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", newSchema.ComplianceOperatorCheckResultV2TableName)
	}

	if len(convertedResults) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(newSchema.CreateTableComplianceOperatorCheckResultV2Stmt.GormModel).Create(&convertedResults).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d check results", len(convertedResults))
		}
	}
	log.Infof("Converted %d check result records", count)

	return nil
}

func createProfileRefID(clusterID, profileID, productType string) string {
	interimUUID := buildDeterministicID(clusterID, profileID)

	return buildDeterministicID(interimUUID, productType)
}

func buildDeterministicID(part1 string, part2 string) string {
	baseUUID, err := uuid.FromString(part1)
	if err != nil {
		log.Error(err)
		return ""
	}
	return uuid.NewV5(baseUUID, part2).String()
}
