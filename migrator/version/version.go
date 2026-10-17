package version

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"gorm.io/gorm"
)

// ReadVersionPostgres - reads the version from the postgres database.
func ReadVersionPostgres(t context.Context, dbName string) (*migrations.MigrationVersion, error) {
	gc := migGorm.GetConfig()

	db, err := gc.ConnectWithRetries(dbName)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to read database version")
	}
	defer migGorm.Close(db)
	return ReadVersionGormDB(t, db)
}

// ReadVersionGormDB reads column-based version metadata without changing the schema.
// A missing table or version record is treated as a fresh installation.
func ReadVersionGormDB(ctx context.Context, db *gorm.DB) (*migrations.MigrationVersion, error) {
	var exists bool
	if err := db.WithContext(ctx).Raw("SELECT to_regclass('versions') IS NOT NULL").Scan(&exists).Error; err != nil {
		return nil, errors.Wrap(err, "checking version metadata table")
	}
	var records []pkgSchema.Versions
	if exists {
		if err := db.WithContext(ctx).Raw("SELECT seqnum, version, minseqnum, lastpersisted FROM versions LIMIT 2").Scan(&records).Error; err != nil {
			return nil, errors.Wrap(err, "reading version metadata; manual recovery required")
		}
	}
	if len(records) == 0 {
		return &migrations.MigrationVersion{MainVersion: "0"}, nil
	}
	if len(records) != 1 {
		return nil, errors.New("multiple database version records; contact support before repairing the database")
	}
	record := records[0]
	if record.SeqNum <= 0 || record.MinSeqNum < 0 {
		return nil, errors.New("invalid database sequence metadata; only a database without a version record can be a fresh installation")
	}
	ver := &migrations.MigrationVersion{
		MainVersion:   record.Version,
		SeqNum:        int(record.SeqNum),
		MinimumSeqNum: int(record.MinSeqNum),
	}
	if record.LastPersisted != nil {
		ver.LastPersisted = *record.LastPersisted
	}
	log.WriteToStderrf("Migration version from DB = %+v.", ver)
	return ver, nil
}

// UpdateVersionPostgres updates the existing version record, respecting an outer transaction.
// A missing record is an error, not an invitation to initialize or repair metadata.
func UpdateVersionPostgres(ctx context.Context, db postgres.DB, updatedVersion *storage.Version) error {
	result, err := db.Exec(ctx, "UPDATE versions SET seqnum = $1, version = $2, minseqnum = $3, lastpersisted = $4",
		updatedVersion.GetSeqNum(), updatedVersion.GetVersion(), updatedVersion.GetMinSeqNum(),
		protocompat.NilOrTime(updatedVersion.GetLastPersisted()))
	if err != nil {
		return errors.Wrap(err, "updating migration version")
	}
	if result.RowsAffected() != 1 {
		return errors.Errorf("updating migration version: expected one version record, updated %d; contact support before repairing the database", result.RowsAffected())
	}
	return nil
}

// SetVersionPostgres - sets the version in the named postgres database
func SetVersionPostgres(ctx context.Context, dbName string, updatedVersion *storage.Version) {
	db, err := migGorm.GetConfig().ConnectWithRetries(dbName)
	if err != nil {
		utils.Must(errors.Wrapf(err, "failed to connect to database %s", dbName))
	}
	defer migGorm.Close(db)
	SetVersion(ctx, db, updatedVersion, true)
}

// SetVersion - sets the version in the postgres database specified with the Gorm instance
func SetVersion(ctx context.Context, db *gorm.DB, updatedVersion *storage.Version, ensureSchema bool) {
	if ensureSchema {
		pkgSchema.ApplySchemaForTable(ctx, db, pkgSchema.VersionsSchema.Table)
	}

	err := pgutils.Retry(ctx, func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			// Gorm broke Save, so we have to do delete/insert:  https://github.com/go-gorm/gorm/pull/6149/files
			result := tx.Exec("DELETE FROM versions")
			if err := result.Error; err != nil {
				return err
			}

			result = tx.Exec("INSERT INTO versions (seqnum, version, minseqnum, lastpersisted) VALUES($1, $2, $3, $4)",
				updatedVersion.GetSeqNum(), updatedVersion.GetVersion(), updatedVersion.GetMinSeqNum(),
				protocompat.NilOrTime(updatedVersion.GetLastPersisted()))
			return result.Error
		})
	})
	if err != nil {
		utils.Must(errors.Wrapf(err, "failed to write migration version to %s", "name"))
	}
}

// SetCurrentVersion - sets the current version via gormDB database
func SetCurrentVersion(ctx context.Context, gormDB *gorm.DB) error {
	minimum, err := migrations.MinimumSupportedDBVersionSeqNum()
	if err != nil {
		return errors.Wrap(err, "determining minimum supported database sequence")
	}
	newVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       version.GetMainVersion(),
		MinSeqNum:     int32(minimum),
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	SetVersion(ctx, gormDB, newVersion, false)
	return nil
}
