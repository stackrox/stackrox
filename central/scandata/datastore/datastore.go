package datastore

import (
	"context"

	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides access to scan data (image_scan_v2 + scan_components + scan_findings)
type DataStore interface {
	// UpsertScanData atomically replaces all scan data for an image
	UpsertScanData(ctx context.Context, data *types.ScanData) error

	// GetScanDataByImageID returns complete scan data for an image
	GetScanDataByImageID(ctx context.Context, imageID string) (*types.ScanData, error)

	// DeleteByImageID removes all scan data for an image
	DeleteByImageID(ctx context.Context, imageID string) error

	// ListCVEs returns the CVE list page data with GROUP BY aggregation
	ListCVEs(ctx context.Context, limit, offset int) ([]*types.CVEListRow, int, error)

	// GetFindingsByCVE returns all findings for a specific CVE name
	GetFindingsByCVE(ctx context.Context, cveName string) ([]*storage.ScanFinding, error)

	// GetFindingsWithComponentsByCVE returns findings joined with component metadata for a CVE.
	GetFindingsWithComponentsByCVE(ctx context.Context, cveName string) ([]*types.FindingWithComponent, error)

	// GetFindingsWithComponentsByImageID returns findings joined with component metadata for an image.
	GetFindingsWithComponentsByImageID(ctx context.Context, imageID string) ([]*types.FindingWithComponent, error)

	// GetFindingsByImageID returns all findings for an image
	GetFindingsByImageID(ctx context.Context, imageID string) ([]*storage.ScanFinding, error)

	// GetImageInfoByDigests returns image UUID and full name for the given SHA digests.
	GetImageInfoByDigests(ctx context.Context, digests []string) (map[string]types.ImageBasicInfo, error)

	// ListDeployments returns deployments with their CVE counts and severity
	ListDeployments(ctx context.Context, limit, offset int) ([]*types.DeploymentListRow, int, error)

	// GetDeploymentImages returns images for a deployment with CVE summary
	GetDeploymentImages(ctx context.Context, deploymentID string) ([]*types.DeploymentImageRow, error)

	// GetDeploymentByID returns basic deployment info
	GetDeploymentByID(ctx context.Context, deploymentID string) (*types.DeploymentListRow, error)

	// ListAdvisories returns distinct advisories with image counts
	ListAdvisories(ctx context.Context, limit, offset int) ([]*types.AdvisoryListRow, int, error)

	// ListComponents returns distinct components with CVE severity breakdown
	ListComponents(ctx context.Context, limit, offset int) ([]*types.ComponentListRow, int, error)

	// GetComponentVersions returns all versions of a component with CVE data
	GetComponentVersions(ctx context.Context, componentName string) ([]*types.ComponentVersionInfo, error)

	// GetImageOS returns the operating system for an image
	GetImageOS(ctx context.Context, imageID string) (string, error)
}
