package m180tom181

import (
	"context"
	"database/sql"
	"os"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/gorm/largeobject"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

const (
	scannerDefBlobName = "/offline/scanner/scanner-defs.zip"
)

var (
	scannerDefPath = "/var/lib/stackrox/scannerdefinitions/scanner-defs.zip"
)

var (
	migration = types.Migration{
		StartingSeqNum: 180,
		VersionAfter:   &storage.Version{SeqNum: 181},
		Run: func(databases *types.Databases) error {
			err := moveToBlobs(databases.GormDB)
			if err != nil {
				return errors.Wrap(err, "moving persistent files to blobs")
			}
			return nil
		},
	}
	log = logging.LoggerForModule()
)

func moveToBlobs(db *gorm.DB) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	db = db.WithContext(ctx).Table(schema.BlobsTableName)
	pgutils.CreateTableFromModel(context.Background(), db, schema.CreateTableBlobsStmt)

	tx := db.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err = moveScannerDefinitions(tx); err != nil {
		result := tx.Rollback()
		if result.Error != nil {
			log.Warnf("failed to rollback with error %v", result.Error)
		}
		return errors.Wrap(err, "failed to move scanner definition to blob store.")
	}

	return tx.Commit().Error
}

func moveScannerDefinitions(tx *gorm.DB) error {
	fd, err := os.Open(scannerDefPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", scannerDefPath)
	}
	defer utils.IgnoreError(fd.Close)
	stat, err := fd.Stat()
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return nil
	}
	modTime, err := timestamp.TimestampProto(stat.ModTime())
	if err != nil {
		return errors.Wrapf(err, "invalid timestamp %v", stat.ModTime())
	}

	// Prepare blob
	blob := &storage.Blob{
		Name:         scannerDefBlobName,
		Length:       stat.Size(),
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: modTime,
	}
	los := largeobject.LargeObjects{DB: tx}

	// Find the blob if it exists
	var targets []schema.Blobs
	result := tx.Limit(1).Where(&schema.Blobs{Name: scannerDefBlobName}).Find(&targets)
	if result.Error != nil {
		return result.Error
	}

	if len(targets) == 0 {
		blob.Oid, err = los.Create()
		if err != nil {
			return errors.Wrap(err, "failed to create large object")
		}
	} else {
		// Update
		existingBlob, err := schema.ConvertBlobToProto(&targets[0])
		if err != nil {
			return errors.Wrapf(err, "existing blob is not valid %+v", targets[0])
		}
		blob.Oid = existingBlob.Oid
	}
	blobModel, err := schema.ConvertBlobFromProto(blob)
	if err != nil {
		return errors.Wrapf(err, "failed to convert blob to blob model %+v", blob)
	}
	tx = tx.FirstOrCreate(blobModel)
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to create blob metadata")
	}
	return los.Upsert(blob.Oid, fd)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
