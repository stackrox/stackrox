package m212tom213

import (
	"github.com/pkg/errors"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm/clause"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
	// have no need to worry about the old schema and can simply perform all our work on the new one.
	db := database.GormDB
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableProcessIndicatorsStmt)
	db = db.WithContext(database.DBCtx).Table(updatedSchema.ProcessIndicatorsTableName)

	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", updatedSchema.ProcessIndicatorsTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedIndicators []*updatedSchema.ProcessIndicators
	var count int
	for rows.Next() {
		var processIndicator *updatedSchema.ProcessIndicators
		if err = db.ScanRows(rows, &processIndicator); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		processIndicatorProto, err := updatedSchema.ConvertProcessIndicatorToProto(processIndicator)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", processIndicator)
		}

		// We only need to rewrite the ones where the time is not null
		if processIndicatorProto.GetContainerStartTime() == nil {
			continue
		}

		converted, err := updatedSchema.ConvertProcessIndicatorFromProto(processIndicatorProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", processIndicatorProto)
		}

		convertedIndicators = append(convertedIndicators, converted)
		count++

		if len(convertedIndicators) == batchSize {
			// Upsert converted blobs
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableProcessIndicatorsStmt.GormModel).Create(&convertedIndicators).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedIndicators), count-len(convertedIndicators))
			}
			convertedIndicators = convertedIndicators[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", updatedSchema.ProcessIndicatorsTableName)
	}

	if len(convertedIndicators) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableProcessIndicatorsStmt.GormModel).Create(&convertedIndicators).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedIndicators))
		}
	}
	log.Infof("Converted %d process indicators", count)

	return nil
}
