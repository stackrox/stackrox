package m188tom189

import (
	"github.com/pkg/errors"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_188_to_m_189_logimbues_add_time_column/schema"
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
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableLogImbuesStmt)
	db = db.WithContext(database.DBCtx).Table(updatedSchema.LogImbuesTableName)

	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", updatedSchema.LogImbuesTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedLogImbues []*updatedSchema.LogImbues
	var count int
	for rows.Next() {
		var logImbue *updatedSchema.LogImbues
		if err = db.ScanRows(rows, &logImbue); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		logImbueProto, err := updatedSchema.ConvertLogImbueToProto(logImbue)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", logImbue)
		}

		converted, err := updatedSchema.ConvertLogImbueFromProto(logImbueProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", logImbueProto)
		}
		convertedLogImbues = append(convertedLogImbues, converted)
		count++

		if len(convertedLogImbues) == batchSize {
			// Upsert converted blobs
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableLogImbuesStmt.GormModel).Create(&convertedLogImbues).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedLogImbues), count-len(convertedLogImbues))
			}
			convertedLogImbues = convertedLogImbues[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", updatedSchema.LogImbuesTableName)
	}

	if len(convertedLogImbues) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableLogImbuesStmt.GormModel).Create(&convertedLogImbues).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedLogImbues))
		}
	}
	log.Infof("Converted %d log imbues", count)

	return nil
}
