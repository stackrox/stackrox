package m228tom229

import (
	"fmt"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// indexName is the single-column severity index on image_cves_v2. It is redundant
	// with the leading column of imagecvesv2_severity_state_isfixable_imageidv2, which
	// the background index reconciler creates after migrations complete.
	indexName = "imagecvesv2_severity"
)

func migrate(database *types.Databases) error {
	stmt := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", indexName)
	if _, err := database.PostgresDB.Exec(database.DBCtx, stmt); err != nil {
		return fmt.Errorf("dropping index %s: %w", indexName, err)
	}
	log.Infof("Dropped redundant image CVE index %s", indexName)
	return nil
}
