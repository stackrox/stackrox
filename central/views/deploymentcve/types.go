package deploymentcve

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// VulnFinding represents a vulnerability finding in a deployment's image component.
// The db tags correspond to the search field aliases used in the select query.
type VulnFinding struct {
	DeploymentID     string                        `db:"deployment_id"`
	DeploymentName   string                        `db:"deployment"`
	ClusterID        string                        `db:"cluster_id"`
	ClusterName      string                        `db:"cluster"`
	Namespace        string                        `db:"namespace"`
	ImageID          string                        `db:"image_sha"`
	ImageRegistry    string                        `db:"image_registry"`
	ImageRemote      string                        `db:"image_remote"`
	ImageTag         string                        `db:"image_tag"`
	OperatingSystem  string                        `db:"image_os"`
	ComponentName    string                        `db:"component"`
	ComponentVersion string                        `db:"component_version"`
	CVE              string                        `db:"cve"`
	CVSS             float32                       `db:"cvss"`
	Severity         storage.VulnerabilitySeverity `db:"severity"`
	EPSSProbability  float32                       `db:"epss_probability"`
	FixedBy          string                        `db:"fixed_by"`
}

// CveView provides functionality to query deployment CVE data.
//
//go:generate mockgen-wrapper
type CveView interface {
	// WalkVulnFindings calls fn on each vulnerability in every deployment
	// image component. Results are filtered based on the access scope in the context.
	WalkVulnFindings(ctx context.Context, fn func(*VulnFinding) error) error
}
