package roxdbv1

import (
	"io"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/rox/pkg/backup"
)

func init() {
	formats.MustRegisterNewFormat(
		"roxdbv1",
		common.NewFileHandler(backup.BoltFileName, false, restoreBoltDB),
		common.NewFileHandler(backup.RocksFileName, true, restoreRocksDB),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaCertPem), true, discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaKeyPem), true, discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInDer), true, discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInPem), true, discard),
		common.NewFileHandler(backup.MigrationVersion, true, restoreMigrationVersion),
	)
}

func discard(_ common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	if _, err := io.Copy(io.Discard, fileReader); err != nil {
		return errors.Wrap(err, "could not discard data file")
	}
	return nil
}
