package version

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	migGorm "github.com/stackrox/rox/migrator/postgres/gorm"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"gorm.io/gorm"
)

// ReadVersionPostgres - reads the version from the postgres database.
func ReadVersionPostgres(t context.Context, dbName string) (*migrations.MigrationVersion, error) {
	gc := migGorm.GetConfig()

	ver := migrations.MigrationVersion{MainVersion: "0", SeqNum: 0}
	db, err := gc.ConnectWithRetries(dbName)
	if err != nil {
		return &ver, nil
	}
	defer migGorm.Close(db)
	return ReadVersionGormDB(t, db)
}

// ReadVersionGormDB - reads the version from the postgres database with a gorm instance.
func ReadVersionGormDB(ctx context.Context, db *gorm.DB) (*migrations.MigrationVersion, error) {
	pkgSchema.ApplySchemaForTable(ctx, db, pkgSchema.VersionsSchema.Table)
	var modelVersion pkgSchema.Versions
	ver := migrations.MigrationVersion{MainVersion: "0", SeqNum: 0}
	result := db.WithContext(ctx).Table(pkgSchema.VersionsSchema.Table).First(&modelVersion)
	if result.Error != nil {
		return &ver, nil
	}

	protoVersion, err := ConvertVersionToProto(&modelVersion)
	if err != nil {
		return &ver, nil
	}

	log.WriteToStderrf("Migration version from DB = %s.", protoVersion)

	ver.MainVersion = protoVersion.GetVersion()
	ver.SeqNum = int(protoVersion.GetSeqNum())
	ver.LastPersisted = timestamp.FromProtobuf(protoVersion.GetLastPersisted()).GoTime()
	return &ver, nil
}

// SetVersionPostgres - sets the version in the named postgres database
func SetVersionPostgres(ctx context.Context, dbName string, updatedVersion *storage.Version) {
	db, err := migGorm.GetConfig().ConnectWithRetries(dbName)
	if err != nil {
		utils.Must(errors.Wrapf(err, "failed to connect to database %s", dbName))
	}
	defer migGorm.Close(db)
	SetVersionGormDB(ctx, db, updatedVersion, true)
}

// SetVersionGormDB - sets the version in the postgres database specified with the Gorm instance
func SetVersionGormDB(ctx context.Context, db *gorm.DB, updatedVersion *storage.Version, ensureSchema bool) {
	if ensureSchema {
		pkgSchema.ApplySchemaForTable(ctx, db, pkgSchema.VersionsSchema.Table)
	}
	modelVersion, err := ConvertVersionFromProto(updatedVersion)
	if err != nil {
		utils.Must(errors.Wrapf(err, "failed to write migration version to %s", "name"))
	}
	err = pgutils.Retry(func() error {
		result := db.Table(pkgSchema.VersionsSchema.Table).WithContext(ctx).Save(modelVersion)
		return result.Error
	})
	if err != nil {
		utils.Must(errors.Wrapf(err, "failed to write migration version to %s", "name"))
	}
}

// SetCurrentVersionPostgres - sets the current version in the postgres database
func SetCurrentVersionPostgres(ctx context.Context) {
	newVersion := &storage.Version{
		SeqNum:        int32(migrations.CurrentDBVersionSeqNum()),
		Version:       version.GetMainVersion(),
		LastPersisted: timestamp.Now().GogoProtobuf(),
	}
	SetVersionPostgres(ctx, migrations.GetCurrentClone(), newVersion)
}
