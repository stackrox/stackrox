package export

import (
	"archive/zip"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/cas"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/utils"
)

// BackupCerts backs up the certs and writes a ZIP archive to the given writer.
func BackupCerts(ctx context.Context, backupListener listener.BackupListener, out io.Writer) error {
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

	if err := generators.PutPathMapInZip(cas.NewCertsBackup(), backup.KeysBaseFolder).WriteTo(ctx, zipWriter); err != nil {
		return listen(errors.Wrap(err, "backing up certificates"))
	}

	return listen(zipWriter.Close())
}
