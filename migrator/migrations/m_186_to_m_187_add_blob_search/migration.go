package m186tom187

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	afterSchema "github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_add_blob_search/schema/after"
	beforeSchema "github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_add_blob_search/schema/before"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	migration = types.Migration{
		StartingSeqNum: 186,
		VersionAfter:   &storage.Version{SeqNum: 187},
		Run: func(databases *types.Databases) error {
			err := convert(databases.GormDB)
			if err != nil {
				return errors.Wrap(err, "moving persistent files to blobs")
			}
			return nil
		},
	}
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func convert(db *gorm.DB) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, db, afterSchema.CreateTableBlobsStmt)
	db = db.WithContext(ctx).Table(afterSchema.BlobsTableName)

	rows, err := db.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to iterate table %s", afterSchema.BlobsTableName)
	}
	if rows.Err() != nil {
		utils.Should(rows.Err())
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", afterSchema.BlobsTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedBlobs []*afterSchema.Blobs
	var count int
	for rows.Next() {
		var blob beforeSchema.Blobs
		if err = db.ScanRows(rows, &blob); err != nil {
			return errors.Wrap(err, "failed to scan rows")
		}
		blobProto, err := beforeSchema.ConvertBlobToProto(&blob)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", blob)
		}
		converted, err := afterSchema.ConvertBlobFromProto(blobProto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", blobProto)
		}
		convertedBlobs = append(convertedBlobs, converted)
		count++
		if len(convertedBlobs) == batchSize {
			// Upsert converted blobs
			if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(afterSchema.CreateTableBlobsStmt.GormModel).Create(&convertedBlobs).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedBlobs), count-len(convertedBlobs))
			}
			convertedBlobs = convertedBlobs[:0]
		}
	}
	if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(afterSchema.CreateTableBlobsStmt.GormModel).Create(&convertedBlobs).Error; err != nil {
		return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedBlobs))
	}
	log.Infof("Converted %d blobs", count)
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
