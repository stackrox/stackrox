package roxdbv1

import (
	"path"

	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/rox/pkg/backup"
)

func init() {
	formats.MustRegisterNewFormat(
		"roxdbv1",
		common.NewFileHandler(backup.BoltFileName, false, restoreBoltDB),
		common.NewFileHandler(backup.BadgerFileName, true, restoreBadger),
		common.NewFileHandler(backup.RocksFileName, true, restoreRocksDB),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaCertPem), true, nil),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.CaKeyPem), true, nil),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInDer), true, nil),
		common.NewFileHandler(path.Join(backup.KeysBaseFolder, backup.JwtKeyInPem), true, nil),
	)
}
