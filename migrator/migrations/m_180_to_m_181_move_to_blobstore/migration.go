package m179tom180

import (
	"context"
	"database/sql"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm"
)

const (
	scannerDefBlobName = "/offline/scanner/scanner-defs.zip"
	scannerDefPath     = "/var/lib/stackrox/scannerdefinitions/scanner-defs.zip"
)

var (
	migration = types.Migration{
		StartingSeqNum: 180,
		VersionAfter:   &storage.Version{SeqNum: 181},
		Run: func(databases *types.Databases) error {
			err := moveToBlobs(databases.GormDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}
	log          = logging.LoggerForModule()
	toBeMigrated = map[string]string{
		scannerDefPath: scannerDefBlobName,
	}
)

func moveToBlobs(db *gorm.DB) (err error) {
	ctx := sac.WithAllAccess(context.Background())
	db = db.WithContext(ctx).Table(schema.BlobsTableName)
	if err := db.WithContext(ctx).AutoMigrate(schema.CreateTableBlobsStmt.GormModel); err != nil {
		return err
	}
	tx := db.Model(schema.CreateTableBlobsStmt.GormModel).Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for p, blobName := range toBeMigrated {
		f, err := os.Open(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		target := &schema.Blobs{Name: blobName}
		result := tx.Take(target)
		if result.Error != nil {
			return result.Error
		}
		var blob *storage.Blob
		if result.RowsAffected == 0 {
			// Create
			// err := o.tx.QueryRow(ctx, "select lo_create($1)", oid).Scan(&oid)
			var oid int
			tx.Select("lo_create(0)").Find()
			tx.Exec("SELECT lo_create(0)").Find(&oid)
			blob = &storage.Blob{
				Name:         blobName,
				Oid:          0,
				Length:       0,
				ModifiedTime: nil,
			}
		} else {
			// Update
			existingBlob, err := schema.ConvertBlobToProto(target)
			if err != nil {
				return err
			}
			blob = &storage.Blob{
				Name:         blobName,
				Oid:          existingBlob.Oid,
				Length:       0,
				ModifiedTime: nil,
			}
		}
		blobModel, err := schema.ConvertBlobFromProto(blob)
		if err != nil {
			return err
		}
		tx.Exec("")
		tx = tx.FirstOrCreate(blobModel)

	}
	return tx.Commit().Error
}

func init() {
	migrations.MustRegisterMigration(migration)
}
