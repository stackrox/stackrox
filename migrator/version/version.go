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

// ReadVersionGormDB - reads the version from the postgres database with a gorm instance.
func ReadVersionGormDB(ctx context.Context, db *gorm.DB) (*migrations.MigrationVersion, error) {
	var exists bool
	if err := db.WithContext(ctx).Raw("SELECT to_regclass('versions') IS NOT NULL").Scan(&exists).Error; err != nil {
		return nil, errors.Wrap(err, "checking version metadata table")
	}
	ver := migrations.MigrationVersion{MainVersion: "0"}
	var records []pkgSchema.Versions
	if exists {
		// SELECT * also reads historical tables containing only serialized metadata.
		if err := db.WithContext(ctx).Raw("SELECT * FROM versions LIMIT 2").Scan(&records).Error; err != nil {
			return nil, errors.Wrap(err, "reading version metadata")
		}
	}
	if len(records) == 0 {
		var populated bool
		err := db.WithContext(ctx).Raw(`SELECT EXISTS (
			SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = ANY(current_schemas(false))
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
			AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
			AND c.relname <> 'versions'
			AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = 'pg_class'::regclass
				AND d.objid = c.oid AND d.deptype = 'e'))`).Scan(&populated).Error
		if err != nil {
			return nil, errors.Wrap(err, "checking for existing application data")
		}
		if populated {
			return nil, errors.New("missing version metadata in a populated database; contact support before repairing the database")
		}
		return &ver, nil
	}
	if len(records) != 1 {
		return nil, errors.New("multiple database version records; contact support before repairing the database")
	}

	protoVersion, err := ConvertVersionToProto(&records[0])
	if err != nil {
		return nil, errors.Wrap(err, "decoding version metadata")
	}

	log.WriteToStderrf("Migration version from DB = %s.", protoVersion)

	ver.MainVersion = protoVersion.GetVersion()
	ver.SeqNum = int(protoVersion.GetSeqNum())
	ver.MinimumSeqNum = int(protoVersion.GetMinSeqNum())
	ver.LastPersisted = timestamp.FromProtobuf(protoVersion.GetLastPersisted()).GoTime()
	if ver.SeqNum <= 0 || ver.MinimumSeqNum < 0 {
		return nil, errors.New("invalid database sequence metadata; only a database without a version record can be a fresh installation")
	}
	return &ver, nil
}

// UpdateVersionPostgres - updates the version allowing for outer transaction.
func UpdateVersionPostgres(ctx context.Context, db postgres.DB, updatedVersion *storage.Version) error {
	err := pgutils.Retry(ctx, func() error {
		_, err := db.Exec(ctx, "WITH cleared AS (DELETE FROM versions) INSERT INTO versions (seqnum, version, minseqnum, lastpersisted) VALUES($1, $2, $3, $4)",
			updatedVersion.GetSeqNum(), updatedVersion.GetVersion(), updatedVersion.GetMinSeqNum(),
			protocompat.NilOrTime(updatedVersion.GetLastPersisted()))
		return err
	})
	return errors.Wrap(err, "failed to write migration version")
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
func SetCurrentVersion(ctx context.Context, gormDB *gorm.DB) {
	newVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       version.GetMainVersion(),
		MinSeqNum:     int32(migrations.MinimumSupportedDBVersionSeqNum()),
		LastPersisted: protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()),
	}
	SetVersion(ctx, gormDB, newVersion, false)
}
