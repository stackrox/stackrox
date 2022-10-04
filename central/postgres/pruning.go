package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	pruneActiveComponentsStmt = `DELETE FROM active_components child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.deploymentid = parent.id)`

	pruneClusterHealthStatusesStmt = `DELETE FROM cluster_health_statuses child WHERE NOT EXISTS
		(SELECT 1 FROM clusters parent WHERE
		child.Id = parent.Id)`

	pruneStaleNetworkFlowsStmt = `DELETE FROM network_flows a USING (
      SELECT MAX(flow_id) as max_flow, props_srcentity_type, props_srcentity_id, props_dstentity_type, props_dstentity_id, props_dstport, props_l4protocol, clusterid
        FROM network_flows
        GROUP BY props_srcentity_type, props_srcentity_id, props_dstentity_type, props_dstentity_id, props_dstport, props_l4protocol, clusterid
		HAVING COUNT(*) > 1
      ) b
      WHERE a.props_srcentity_type = b.props_srcentity_type
 	AND a.props_srcentity_id = b.props_srcentity_id
 	AND a.props_dstentity_type = b.props_dstentity_type
 	AND a.props_dstentity_id = b.props_dstentity_id
	AND a.props_dstport = b.props_dstport
	AND a.props_l4protocol = b.props_l4protocol
	AND a.clusterid = b.clusterid
      AND a.flow_id <> b.max_flow;
	`
)

var (
	log = logging.LoggerForModule()
)

// PruneActiveComponents - prunes active components
// TODO (ROX-12710):  This will no longer be necessary when the foreign keys are added back
func PruneActiveComponents(ctx context.Context, pool *pgxpool.Pool) {
	if _, err := pool.Exec(ctx, pruneActiveComponentsStmt); err != nil {
		log.Errorf("failed to prune active components: %v", err)
	}
}

// PruneClusterHealthStatuses - prunes cluster health statuses
// TODO (ROX-12711):  This will no longer be necessary when the foreign keys are added back
func PruneClusterHealthStatuses(ctx context.Context, pool *pgxpool.Pool) {
	if _, err := pool.Exec(ctx, pruneClusterHealthStatusesStmt); err != nil {
		log.Errorf("failed to prune cluster health statuses: %v", err)
	}
}

// PruneStaleNetworkFlows - prunes duplicate network flows
func PruneStaleNetworkFlows(ctx context.Context, pool *pgxpool.Pool) {
	if _, err := pool.Exec(ctx, pruneStaleNetworkFlowsStmt); err != nil {
		log.Errorf("failed to prune stale network flows: %v", err)
	}
}
