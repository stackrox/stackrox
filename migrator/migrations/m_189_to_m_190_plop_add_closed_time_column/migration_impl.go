package m189tom190

import (
	"github.com/pkg/errors"
	updatedSchema "github.com/stackrox/rox/migrator/migrations/m_189_to_m_190_plop_add_closed_time_column/schema"
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
	pgutils.CreateTableFromModel(database.DBCtx, db, updatedSchema.CreateTableListeningEndpointsStmt)
	db = db.WithContext(database.DBCtx).Table(updatedSchema.ListeningEndpointsTableName)

	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", updatedSchema.ListeningEndpointsTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedPLOPs []*updatedSchema.ListeningEndpoints
	var count int
	for rows.Next() {
		var plop *updatedSchema.ListeningEndpoints
		if err = db.ScanRows(rows, &plop); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}

		plopProto, err := updatedSchema.ConvertProcessListeningOnPortStorageToProto(plop)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", plop)
		}

		converted, err := updatedSchema.ConvertProcessListeningOnPortStorageFromProto(plopProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", plopProto)
		}
		convertedPLOPs = append(convertedPLOPs, converted)
		count++

		if len(convertedPLOPs) == batchSize {
			// Upsert converted blobs
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableListeningEndpointsStmt.GormModel).Create(&convertedPLOPs).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedPLOPs), count-len(convertedPLOPs))
			}
			convertedPLOPs = convertedPLOPs[:0]
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to get rows for %s", updatedSchema.ListeningEndpointsTableName)
	}

	if len(convertedPLOPs) > 0 {
		if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(updatedSchema.CreateTableListeningEndpointsStmt.GormModel).Create(&convertedPLOPs).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedPLOPs))
		}
	}
	log.Infof("Converted %d plop records", count)

	return nil
}
