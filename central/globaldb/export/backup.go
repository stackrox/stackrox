package export

import (
	"archive/zip"
	"context"
	"io"
	"path/filepath"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/cas"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/dbs"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

// Backup backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func Backup(ctx context.Context, boltDB *bolt.DB, rocksDB *rocksdb.RocksDB, includeCerts bool, out io.Writer) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)

	if err := generators.PutStreamInZip(dbs.NewBoltBackup(boltDB), backup.BoltFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up bolt")
	}

	if err := generators.PutTarInZip(generators.PutDirectoryInTar(dbs.NewRocksBackup(rocksDB)), backup.RocksFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up rocksdb")
	}

	if includeCerts {
		if err := generators.PutPathMapInZip(cas.NewCertsBackup(), backup.KeysBaseFolder).WriteTo(ctx, zipWriter); err != nil {
			return errors.Wrap(err, "backing up certificates")
		}
	}

	if err := generators.PutStreamInZip(generators.PutFileInStream(filepath.Join(migrations.CurrentPath(), backup.MigrationVersion)), backup.MigrationVersion).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up migration version")
	}

	return zipWriter.Close()
}

// BackupPostgres backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func BackupPostgres(ctx context.Context, postgresDB *pgxpool.Pool, includeCerts bool, out io.Writer) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)

	// TODO:  Fix it so it can work with a file.
	if err := generators.PutTarInZip(generators.PutDirectoryInTar(dbs.NewPostgresBackup(postgresDB)), backup.PostgresFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up postgres")
	}

	return zipWriter.Close()
}
