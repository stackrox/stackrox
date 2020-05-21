package roxdbv1

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
)

func init() {
	formats.MustRegisterNewFormat(
		"roxdbv1",
		common.NewFileHandler("bolt.db", false, restoreBoltDB),
		common.NewFileHandler("badger.db", true, restoreBadger),
		common.NewFileHandler("rocks.db", true, restoreRocksDB),
	)
}
