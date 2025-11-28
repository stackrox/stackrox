package store

import "github.com/stackrox/rox/generated/storage"

// DeploymentVulnFinding represents a single vulnerability finding in the context of a deployment.
// This is a flattened view joining deployments, images, components, and CVEs optimized for metrics collection.
type DeploymentVulnFinding struct {
	// Deployment context
	DeploymentID   string
	DeploymentName string
	ClusterID      string
	ClusterName    string
	Namespace      string

	// Image context
	ImageID         string
	ImageRegistry   string
	ImageRemote     string
	ImageTag        string
	OperatingSystem string

	// Component context
	ComponentName    string
	ComponentVersion string

	// Vulnerability data
	CVE             string
	CVSS            float32
	Severity        storage.VulnerabilitySeverity
	EPSSProbability float32
	// Note: EPSSPercentile is not available in the denormalized image_cves_v2 table,
	// it's only in the embedded protobuf. If needed, it could be added to the schema.
	FixedBy string
}
