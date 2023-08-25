package postgresv1

import (
	"path"

	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/rox/pkg/backup"
)

func init() {
	formats.MustRegisterNewFormat(
		"postgresv1",
		common.NewFileHandler(backup.PostgresSizeFileName, false, checkPostgresSize),
		common.NewFileHandler(backup.PostgresFileName, false, restorePostgresDB),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaCertPem), true, formats.Discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaKeyPem), true, formats.Discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInDer), true, formats.Discard),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInPem), true, formats.Discard),
		common.NewFileHandler(path.Join(backup.DatabaseBaseFolder, backup.DatabasePassword), true, formats.Discard),
		common.NewFileHandler(backup.MigrationVersion, true, formats.Discard),
	)
}
