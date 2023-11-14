package m180tom181

import (
	"context"
	"database/sql"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/gorm/largeobject"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

const (
	scannerDefBlobName  = "/offline/scanner/scanner-defs.zip"
	uploadProbeBlobRoot = "/offline/probe-uploads"
	dataFileName        = "data"
	crc32FileName       = "crc32"
)

var (
	scannerDefPath  = "/var/lib/stackrox/scannerdefinitions/scanner-defs.zip"
	uploadProbeRoot = "/var/lib/stackrox/probe-uploads"
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

	moveScannerDefinitions(db)

	moveProbesToBlob(db)

	return nil
}

func moveScannerDefinitions(db *gorm.DB) {
	tx := db.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	var err error
	if err = moveFileToBlob(tx, scannerDefBlobName, scannerDefPath, nil); err != nil {
		result := tx.Rollback()
		if result.Error != nil {
			log.Warnf("failed to rollback with error %v", result.Error)
		}
	} else {
		err = tx.Commit().Error
	}

	if err != nil {
		log.Errorf("Failed to move scanner definition to blob store with error \n%v \nPlease upload the scanner definition again to continue working with offline mode", err)
	} else {
		log.Info("Migrate scanner definitions successfully")
	}
}

func moveFileToBlob(tx *gorm.DB, blobName string, file string, crc32Data []byte) error {
	fd, err := os.Open(file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", file)
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
		Name:         blobName,
		Length:       stat.Size(),
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: modTime,
	}
	var dataReader io.ReadCloser = fd
	if crc32Data != nil {
		dataReader = ioutils.NewCRC32ChecksumReader(fd, crc32.IEEETable, binary.BigEndian.Uint32(crc32Data))
		blob.Checksum = string(crc32Data)
	}
	los := largeobject.LargeObjects{DB: tx}

	// Find the blob if it exists
	var targets []schema.Blobs
	result := tx.Limit(1).Where(&schema.Blobs{Name: blobName}).Find(&targets)
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
	return los.Upsert(blob.Oid, dataReader)
}

func moveProbesToBlob(db *gorm.DB) {
	// Go through all the subdir in upload root and find all probes.
	entries, err := os.ReadDir(uploadProbeRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("could not read probe upload root directory %v", err)
		}
		return
	}

	for _, ent := range entries {
		if ent.Name() == "." || ent.Name() == ".." {
			continue
		}
		if !ent.IsDir() {
			log.Warnf("Unexpected non-directory entry %q in probe upload root directory", ent.Name())
			continue
		}
		if !probeupload.IsValidModuleVersion(ent.Name()) {
			log.Warnf("Unexpected non-module-version directory entry %q in probe upload root directory", ent.Name())
			continue
		}

		// Each bundle are uploaded separately, so they are migrated separately.
		tx := db.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		err = moveModVersion(tx, ent.Name())
		if err != nil {
			result := tx.Rollback()
			if result.Error != nil {
				log.Warnf("failed to rollback with error %v", result.Error)
			}
		} else {
			err = tx.Commit().Error
		}
		if err != nil {
			log.Errorf("failed to move module probe version %s to blob store with error \n%v \nPlease upload it again to continue working with offline mode", ent.Name(), err)
		} else {
			log.Infof("Migrate module version %s successfully", ent.Name())
		}
	}
}

func moveModVersion(tx *gorm.DB, modVer string) error {
	subDir := filepath.Join(uploadProbeRoot, modVer)
	subDirEntries, err := os.ReadDir(subDir)
	if err != nil {
		return errors.Wrap(err, "could not read module version subdirectory")
	}

	for _, subDirEnt := range subDirEntries {
		if subDirEnt.Name() == "." || subDirEnt.Name() == ".." {
			continue
		}

		if !subDirEnt.IsDir() {
			log.Warnf("Unexpected non-directory entry %q in probe upload directory for module version %s", subDirEnt.Name(), modVer)
			continue
		}
		if probeupload.IsValidProbeName(subDirEnt.Name()) {
			// Read CRC file
			modPath := filepath.Join(subDir, subDirEnt.Name())
			crc32FilePath := filepath.Join(modPath, crc32FileName)
			crc32Data, err := os.ReadFile(crc32FilePath)
			if err != nil {
				return err
			}
			if len(crc32Data) != 4 {
				return errors.Errorf("crc32 file %s does not contain a valid CRC-32 checksum (%d bytes)", crc32FilePath, len(crc32Data))
			}

			if err = moveFileToBlob(tx, path.Join(uploadProbeBlobRoot, modVer, subDirEnt.Name()), filepath.Join(modPath, dataFileName), crc32Data); err != nil {
				return err
			}
		}
	}

	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
