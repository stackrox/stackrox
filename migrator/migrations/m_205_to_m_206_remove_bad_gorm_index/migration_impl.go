package m205tom206

import (
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	gormDB := database.GormDB
	tx := gormDB.Exec("ALTER TABLE compliance_integrations DROP CONSTRAINT IF EXISTS idx_compliance_integrations_clusterid")
	if tx.Error != nil {
		log.Infof("Failed to drop compliance integrations table bogus unique constraint it likely does not exist and that is OK: %v", tx.Error)
	}

	return nil
}
