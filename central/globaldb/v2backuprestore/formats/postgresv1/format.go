package postgresv1

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/rox/pkg/backup"
)

func init() {
	formats.MustRegisterNewFormat(
		"postgresv1",
		common.NewFileHandler(backup.PostgresFileName, false, restorePostgresDB),
		// TODO: ROX-10087 deal with certs
	)
}
