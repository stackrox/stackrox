package postgres

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	pruneActiveComponentsStmt = `DELETE FROM active_components child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.deploymentid = parent.id)`

	pruneClusterHealthStatusesStmt = `DELETE FROM cluster_health_statuses child WHERE NOT EXISTS
		(SELECT 1 FROM clusters parent WHERE
		child.Id = parent.Id)`
)

var (
	log = logging.LoggerForModule()
)

// PruneActiveComponents - prunes active components
// TODO (ROX-12710):  This will no longer be necessary when the foreign keys are added back
func PruneActiveComponents(ctx context.Context, pool *postgres.DB) {
	if _, err := pool.Exec(ctx, pruneActiveComponentsStmt); err != nil {
		log.Errorf("failed to prune active components: %v", err)
	}
}

// PruneClusterHealthStatuses - prunes cluster health statuses
// TODO (ROX-12711):  This will no longer be necessary when the foreign keys are added back
func PruneClusterHealthStatuses(ctx context.Context, pool *postgres.DB) {
	if _, err := pool.Exec(ctx, pruneClusterHealthStatusesStmt); err != nil {
		log.Errorf("failed to prune cluster health statuses: %v", err)
	}
}
