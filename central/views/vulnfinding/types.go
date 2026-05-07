package vulnfinding

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Finding represents a single vulnerability occurrence: one CVE in one component
// in one image in one deployment.
type Finding interface {
	GetDeploymentID() string
	GetImageID() string
	GetCVE() string
	GetComponentName() string
	GetComponentVersion() string
	GetIsFixable() bool
	GetFixedBy() string
	GetState() storage.VulnerabilityState
	GetSeverity() storage.VulnerabilitySeverity
	GetCVSS() float32
	GetRepositoryCPE() string
}

// FindingView provides SQL-based vulnerability findings for the export API.
type FindingView interface {
	Get(ctx context.Context, q *v1.Query) ([]Finding, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}
