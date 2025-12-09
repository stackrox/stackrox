package m214tom215

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	indexes = []string{"processindicators_poduid", "processindicators_deploymentid"}
	log     = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply changing the index type from hash to btree if the process_indicators
	// indexes for deployment id and poduid are still hash.  There is at least one instance
	// in the field where the indexes have already been moved to btree, we do not want to
	// force that instance to remigrate.  So we will not simply drop and re-add, but
	// verify index is hash, drop old index, re-add it as btree.
	db := database.PostgresDB
	ctx, cancel := context.WithTimeout(database.DBCtx, types.DefaultMigrationTimeout)
	defer cancel()

	dropStatements := make([]string, 0, 2)
	createStatements := make([]string, 0, 2)

	// processindicators_poduid
	// processindicators_deploymentid

	//SELECT tab.relname, cls.relname, am.amname
	//FROM pg_index idx
	//JOIN pg_class cls ON cls.oid=idx.indexrelid
	//JOIN pg_class tab ON tab.oid=idx.indrelid
	//JOIN pg_am am ON am.oid=cls.relam
	//where cls.relname = 'processindicators_deploymentid';

	for _, dropStatement := range dropStatements {
		_, err := db.Exec(ctx, dropStatement)
		if err != nil {
			log.Error(errors.Wrapf(err, "unable to execute %s", dropStatement))
		}
	}

	for _, createStatement := range createStatements {
		_, err := db.Exec(ctx, createStatement)
		if err != nil {
			log.Error(errors.Wrapf(err, "unable to execute %s", createStatement))
		}
	}

	log.Infof("Process indicator index migration complete")

	return nil
}
