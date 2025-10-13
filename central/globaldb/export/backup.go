package export

import (
	"archive/zip"
	"context"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/cas"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/dbs"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/utils"
)

// BackupPostgres backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func BackupPostgres(ctx context.Context, postgresDB postgres.DB, backupListener listener.BackupListener, includeCerts bool, out io.Writer) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)

	listen := func(err error) error {
		if err == nil {
			backupListener.OnBackupSuccess(ctx)
			return nil
		}
		backupListener.OnBackupFail(ctx)
		return err
	}

	if err := generators.PutStreamInZip(dbs.NewPostgresSize(postgresDB), backup.PostgresSizeFileName).WriteTo(ctx, zipWriter); err != nil {
		return listen(errors.Wrap(err, "unable to get postgres size"))
	}

	if err := generators.PutStreamInZip(dbs.NewPostgresBackup(postgresDB), backup.PostgresFileName).WriteTo(ctx, zipWriter); err != nil {
		return listen(errors.Wrap(err, "backing up postgres"))
	}

	if includeCerts {
		if err := generators.PutPathMapInZip(cas.NewCertsBackup(), backup.KeysBaseFolder).WriteTo(ctx, zipWriter); err != nil {
			return listen(errors.Wrap(err, "backing up certificates"))
		}

		if err := generators.PutStreamInZip(generators.PutFileInStream(pgconfig.DBPasswordFile), filepath.Join(backup.DatabaseBaseFolder, backup.DatabasePassword)).WriteTo(ctx, zipWriter); err != nil {
			return listen(errors.Wrap(err, "backing up postgres password"))
		}
	}

	if err := generators.PutStreamInZip(dbs.NewPostgresVersion(postgresDB), backup.MigrationVersion).WriteTo(ctx, zipWriter); err != nil {
		return listen(errors.Wrap(err, "unable to get postgres version"))
	}
	return listen(zipWriter.Close())
}
