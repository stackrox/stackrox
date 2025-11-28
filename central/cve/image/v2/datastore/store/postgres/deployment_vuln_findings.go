package postgres

import (
	"context"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

const deploymentVulnFindingsQuery = `
		SELECT
			d.id AS deployment_id,
			d.name AS deployment_name,
			d.clusterid AS cluster_id,
			d.clustername AS cluster_name,
			d.namespace AS namespace,
			i.id AS image_id,
			i.name_registry AS image_registry,
			i.name_remote AS image_remote,
			i.name_tag AS image_tag,
			i.scan_operatingsystem AS operating_system,
			c.name AS component_name,
			c.version AS component_version,
			v.cvebaseinfo_cve AS cve,
			v.cvss AS cvss,
			v.severity AS severity,
			v.cvebaseinfo_epss_epssprobability AS epss_probability,
			v.fixedby AS fixed_by
		FROM deployments d
		INNER JOIN deployments_containers dc ON d.id = dc.deployments_id
		INNER JOIN images_v2 i ON dc.image_idv2 = i.id
		INNER JOIN image_component_v2 c ON i.id = c.imageidv2
		INNER JOIN image_cves_v2 v ON c.id = v.componentid
		ORDER BY d.id, i.id, c.id, v.id
	`

// WalkDeploymentVulnFindings executes a single SQL JOIN query to retrieve all
// vulnerability findings with deployment context. This eliminates N+1 queries
// by joining:
// - deployments
// - deployments_containers
// - images_v2
// - image_component_v2
// - image_cves_v2
//
// The query streams results and calls the callback function for each finding.
func WalkDeploymentVulnFindings(ctx context.Context, db postgres.DB, fn func(*store.DeploymentVulnFinding) error) error {
	roChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS)
	deploymentChecker := roChecker.Resource(resources.Deployment)
	imageChecker := roChecker.Resource(resources.Image)

	if !deploymentChecker.IsAllowed() || !imageChecker.IsAllowed() {
		// No access - return early
		return nil
	}

	rows, err := db.Query(ctx, deploymentVulnFindingsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var finding store.DeploymentVulnFinding
		err := rows.Scan(
			&finding.DeploymentID,
			&finding.DeploymentName,
			&finding.ClusterID,
			&finding.ClusterName,
			&finding.Namespace,
			&finding.ImageID,
			&finding.ImageRegistry,
			&finding.ImageRemote,
			&finding.ImageTag,
			&finding.OperatingSystem,
			&finding.ComponentName,
			&finding.ComponentVersion,
			&finding.CVE,
			&finding.CVSS,
			&finding.Severity,
			&finding.EPSSProbability,
			&finding.FixedBy,
		)
		if err != nil {
			return err
		}

		if err := fn(&finding); err != nil {
			return err
		}
	}

	return rows.Err()
}
