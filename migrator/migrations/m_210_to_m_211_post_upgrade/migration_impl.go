package m210tom211

import (
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Perform post-upgrade steps: upgrade extensions, and do manual analyze
func migrate(database *types.Databases) error {
	gormDB := database.GormDB
	tx := gormDB.Exec("ALTER EXTENSION \"pg_stat_statements\" UPDATE")
	if tx.Error != nil {
		log.Infof("Failed to upgrade \"pg_stat_statements\" extension: %v", tx.Error)
	}

	tx = gormDB.Exec("ANALYZE")
	if tx.Error != nil {
		log.Infof("Failed to analyze: %v", tx.Error)
	}

	return nil
}
