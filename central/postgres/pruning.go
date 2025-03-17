package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
)

const (
	pruneActiveComponentsStmt = `DELETE FROM active_components child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.deploymentid = parent.id)`

	pruneClusterHealthStatusesStmt = `DELETE FROM cluster_health_statuses child WHERE NOT EXISTS
		(SELECT 1 FROM clusters parent WHERE
		child.Id = parent.Id)`

	getAllOrphanedAlerts = `SELECT id FROM alerts WHERE lifecyclestage = 0 and state = 0 and time < now() at time zone 'utc' - INTERVAL '%d MINUTES' and NOT EXISTS
		(SELECT 1 FROM deployments WHERE alerts.deployment_id = deployments.Id)`

	getAllOrphanedPods = `SELECT id FROM pods WHERE NOT EXISTS
		(SELECT 1 FROM clusters WHERE pods.clusterid = clusters.Id)`

	getAllOrphanedNodes = `SELECT id FROM nodes WHERE NOT EXISTS
		(SELECT 1 FROM clusters WHERE nodes.clusterid = clusters.Id)`

	getOrphanedProcessesByDeployment = `SELECT id FROM process_indicators pi WHERE NOT EXISTS
		(SELECT 1 FROM deployments WHERE pi.deploymentid = deployments.Id) AND
		(signal_time < now() AT time zone 'utc' - INTERVAL '%d MINUTES' OR signal_time IS NULL)`

	getOrphanedProcessesByPod = `SELECT id FROM process_indicators pi WHERE NOT EXISTS
		(SELECT 1 FROM pods WHERE pi.poduid = pods.Id) AND
		(signal_time < now() AT time zone 'utc' - INTERVAL '%d MINUTES' OR signal_time IS NULL)`

	// (snapshots.reportstatus_runstate = 2 OR snapshots.reportstatus_runstate = 3 OR snapshots.reportstatus_runstate = 4)
	// ...gives us the report jobs that are in final state.
	//
	// (SELECT MAX(latest.reportstatus_completedat) FROM ` + schema.ReportSnapshotsTableName + ` latest
	// WHERE latest.reportstatus_completedat IS NOT NULL
	// AND snapshots.reportconfigurationid = latest.reportconfigurationid
	// AND latest.reportstatus_runstate = 3
	// GROUP BY latest.reportstatus_reportnotificationmethod, latest.reportstatus_reportrequesttype)
	//	...gives us the last successful report job for each config, for each notificated method, and each request type.
	//
	// (SELECT 1 FROM ` + schema.BlobsTableName + ` blobs
	//	WHERE blobs.name not ilike '%/snapshots.reportid')
	// ...tells us if the report still exists in the blob store.
	//
	// (reportstatus_completedat < now() AT time zone 'utc' - INTERVAL '%d MINUTES')
	// ...gives us the reports that are outside the retention window.
	pruneOldReportHistory = `DELETE FROM ` + schema.ReportSnapshotsTableName + ` WHERE reportid IN
		(
			SELECT snapshots.reportid FROM ` + schema.ReportSnapshotsTableName + ` snapshots
			WHERE (snapshots.reportstatus_runstate = 2 OR snapshots.reportstatus_runstate = 3 OR snapshots.reportstatus_runstate = 4)
			AND snapshots.reportstatus_completedat NOT IN
			(
				SELECT MAX(latest.reportstatus_completedat) FROM ` + schema.ReportSnapshotsTableName + ` latest
				WHERE latest.reportstatus_completedat IS NOT NULL
				AND snapshots.reportconfigurationid = latest.reportconfigurationid
				AND latest.reportstatus_runstate = 3
				GROUP BY latest.reportstatus_reportnotificationmethod, latest.reportstatus_reportrequesttype
			)
			AND NOT EXISTS
			(
				SELECT 1 FROM ` + schema.BlobsTableName + ` blobs
				WHERE blobs.name ilike CONCAT('%%/',snapshots.reportid)
			)
			AND (snapshots.reportstatus_completedat < now() AT time zone 'utc' - INTERVAL '%d MINUTES')
		)`

	// (snapshots.reportstatus_runstate = 2 OR snapshots.reportstatus_runstate = 3 OR snapshots.reportstatus_runstate = 4)
	// ...gives us the report jobs that are in final state.
	//
	// (SELECT MAX(latest.reportstatus_completedat) FROM ` + schema.ReportSnapshotsTableName + ` latest
	// WHERE latest.reportstatus_completedat IS NOT NULL
	// AND snapshots.reportconfigurationid = latest.reportconfigurationid
	// AND latest.reportstatus_runstate = 3
	// GROUP BY latest.reportstatus_reportnotificationmethod, latest.reportstatus_reportrequesttype)
	//	...gives us the last successful report job for each config, for each notificated method, and each request type.
	//
	// (SELECT 1 FROM ` + schema.BlobsTableName + ` blobs
	//	WHERE blobs.name not ilike '%/snapshots.reportid')
	// ...tells us if the report still exists in the blob store.
	//
	// (reportstatus_completedat < now() AT time zone 'utc' - INTERVAL '%d MINUTES')
	// ...gives us the reports that are outside the retention window.
	pruneOldComplianceReportHistory = `DELETE FROM ` + schema.ComplianceOperatorReportSnapshotV2TableName + ` WHERE reportid IN
		(
			SELECT snapshots.reportid FROM ` + schema.ComplianceOperatorReportSnapshotV2TableName + ` snapshots
			WHERE (snapshots.reportstatus_runstate = 2 OR snapshots.reportstatus_runstate = 3 OR snapshots.reportstatus_runstate = 4)
			AND snapshots.reportstatus_completedat NOT IN
			(
				SELECT MAX(latest.reportstatus_completedat) FROM ` + schema.ComplianceOperatorReportSnapshotV2TableName + ` latest
				WHERE latest.reportstatus_completedat IS NOT NULL
				AND snapshots.scanconfigurationid = latest.scanconfigurationid
				AND latest.reportstatus_runstate = 3
				GROUP BY latest.reportstatus_reportnotificationmethod, latest.reportstatus_reportrequesttype
			)
			AND NOT EXISTS
			(
				SELECT 1 FROM ` + schema.BlobsTableName + ` blobs
				WHERE blobs.name ilike CONCAT('%%/',snapshots.reportid)
			)
			AND (snapshots.reportstatus_completedat < now() AT time zone 'utc' - INTERVAL '%d MINUTES')
		)`

	// Delete the log imbues with old timestamp
	pruneLogImbues = `DELETE FROM log_imbues WHERE timestamp < now() at time zone 'utc' - INTERVAL '%d MINUTES'`

	pruneAdministrationEvents = `DELETE FROM %s WHERE lastoccurredat < now() at time zone 'utc' - INTERVAL '%d MINUTES'`

	pruneDiscoveredClusters = `DELETE FROM %s WHERE lastupdatedat < now() at time zone 'utc' - INTERVAL '%d MINUTES'`

	pruneInvalidAPITokens = `DELETE FROM %s WHERE
		(
			revoked = TRUE OR
			expiration < now() at time zone 'utc' - INTERVAL '%d MINUTES'
		)` // #nosec G101
)

var (
	orphanedQueryTimeout = env.PruneOrphanedQueryTimeout.DurationSetting()

	pruningTimeout = env.PostgresDefaultPruningStatementTimeout.DurationSetting()

	log = logging.LoggerForModule()
)

// PruneActiveComponents - prunes active components.
// TODO (ROX-12710):  This will no longer be necessary when the foreign keys are added back
func PruneActiveComponents(ctx context.Context, pool postgres.DB) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	if _, err := pool.Exec(pruneCtx, pruneActiveComponentsStmt); err != nil {
		log.Errorf("failed to prune active components: %v", err)
	}
}

// PruneClusterHealthStatuses - prunes cluster health statuses.
// TODO (ROX-12711):  This will no longer be necessary when the foreign keys are added back
func PruneClusterHealthStatuses(ctx context.Context, pool postgres.DB) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	if _, err := pool.Exec(pruneCtx, pruneClusterHealthStatusesStmt); err != nil {
		log.Errorf("failed to prune cluster health statuses: %v", err)
	}
}

func getOrphanedIDs(ctx context.Context, pool postgres.DB, query string) ([]string, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get orphaned alerts")
	}
	return pgutils.ScanStrings(rows)
}

// GetOrphanedAlertIDs returns the alert IDs for alerts that are orphaned, so they can be resolved.
func GetOrphanedAlertIDs(ctx context.Context, pool postgres.DB, orphanWindow time.Duration) ([]string, error) {
	return pgutils.Retry2(ctx, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(ctx, orphanedQueryTimeout)
		defer cancel()

		query := fmt.Sprintf(getAllOrphanedAlerts, int(orphanWindow.Minutes()))
		return getOrphanedIDs(ctx, pool, query)
	})
}

// GetOrphanedPodIDs returns the pod IDs for pods that are orphaned, so they can be removed.
func GetOrphanedPodIDs(ctx context.Context, pool postgres.DB) ([]string, error) {
	return pgutils.Retry2(ctx, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(ctx, orphanedQueryTimeout)
		defer cancel()

		return getOrphanedIDs(ctx, pool, getAllOrphanedPods)
	})
}

// GetOrphanedNodeIDs returns the node ids that have a cluster that has been removed.
func GetOrphanedNodeIDs(ctx context.Context, pool postgres.DB) ([]string, error) {
	return pgutils.Retry2(ctx, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(ctx, orphanedQueryTimeout)
		defer cancel()

		return getOrphanedIDs(ctx, pool, getAllOrphanedNodes)
	})
}

// GetOrphanedProcessIDsByDeployment returns the process ids that have a deployment that has been removed.
func GetOrphanedProcessIDsByDeployment(ctx context.Context, pool postgres.DB, orphanWindow time.Duration) ([]string, error) {
	return pgutils.Retry2(ctx, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(ctx, orphanedQueryTimeout)
		defer cancel()

		query := fmt.Sprintf(getOrphanedProcessesByDeployment, int(orphanWindow.Minutes()))
		return getOrphanedIDs(ctx, pool, query)
	})
}

// GetOrphanedProcessIDsByPod returns the process ids that have a pod that has been removed.
func GetOrphanedProcessIDsByPod(ctx context.Context, pool postgres.DB, orphanWindow time.Duration) ([]string, error) {
	return pgutils.Retry2(ctx, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(ctx, orphanedQueryTimeout)
		defer cancel()

		query := fmt.Sprintf(getOrphanedProcessesByPod, int(orphanWindow.Minutes()))
		return getOrphanedIDs(ctx, pool, query)
	})
}

// PruneReportHistory prunes report history as per specified retentionDuration and a few static criteria.
func PruneReportHistory(ctx context.Context, pool postgres.DB, retentionDuration time.Duration) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	query := fmt.Sprintf(pruneOldReportHistory, int(retentionDuration.Minutes()))
	if _, err := pool.Exec(pruneCtx, query); err != nil {
		log.Errorf("failed to prune report history: %v", err)
	}
}

// PruneComplianceReportHistory prunes compliance report history as per specified retentionDuration and a few static criteria.
func PruneComplianceReportHistory(ctx context.Context, pool postgres.DB, retentionDuration time.Duration) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	query := fmt.Sprintf(pruneOldComplianceReportHistory, int(retentionDuration.Minutes()))
	if _, err := pool.Exec(pruneCtx, query); err != nil {
		log.Errorf("failed to prune compliance report history: %v", err)
	}
}

// PruneLogImbues prunes old log imbues.
func PruneLogImbues(ctx context.Context, pool postgres.DB, orphanWindow time.Duration) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	query := fmt.Sprintf(pruneLogImbues, int(orphanWindow.Minutes()))
	if _, err := pool.Exec(pruneCtx, query); err != nil {
		log.Errorf("failed to prune log imbues: %v", err)
	}
}

// PruneAdministrationEvents prunes administration events that have occurred before the specified retention duration.
func PruneAdministrationEvents(ctx context.Context, pool postgres.DB, retentionDuration time.Duration) {
	query := fmt.Sprintf(pruneAdministrationEvents, schema.AdministrationEventsTableName,
		int(retentionDuration.Minutes()))
	if _, err := pool.Exec(ctx, query); err != nil {
		log.Errorf("failed to prune administration events: %v", err)
	}
}

// PruneDiscoveredClusters prunes discovered clusters that haven't been updated since the retention duration.
func PruneDiscoveredClusters(ctx context.Context, pool postgres.DB, retentionDuration time.Duration) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	query := fmt.Sprintf(pruneDiscoveredClusters, schema.DiscoveredClustersTableName,
		int(retentionDuration.Minutes()))
	if _, err := pool.Exec(pruneCtx, query); err != nil {
		log.Errorf("failed to prune discovered clusters: %v", err)
	}
}

// PruneInvalidAPITokens prunes expired or revoked API tokens.
func PruneInvalidAPITokens(ctx context.Context, pool postgres.DB, retentionDuration time.Duration) {
	pruneCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, pruningTimeout)
	defer cancel()

	query := fmt.Sprintf(pruneInvalidAPITokens,
		schema.APITokensTableName,
		int(retentionDuration.Minutes()),
	)
	if _, err := pool.Exec(pruneCtx, query); err != nil {
		log.Errorf("failed to prune invalid api tokens: %v", err)
	}
}
