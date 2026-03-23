package store

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// CVERow represents a row in the cves table.
type CVERow struct {
	ID           string // UUID; populated after UpsertCVE.
	CVEName      string
	Source       string
	Severity     string
	CvssV2       *float32
	CvssV3       *float32
	NvdCvssV3    *float32
	Summary      *string
	Link         *string
	PublishedOn  *time.Time
	AdvisoryName *string
	AdvisoryLink *string
	ContentHash  string
	CreatedAt    time.Time
}

// EdgeRow represents a row in the component_cve_edges table.
type EdgeRow struct {
	ID                    string // UUID; may be empty (DB assigns).
	ComponentID           string // TEXT FK to image_component_v2.id.
	CveID                 string // UUID FK to cves.id.
	IsFixable             bool
	FixedBy               *string
	State                 string // OBSERVED | DEFERRED | FALSE_POSITIVE.
	FirstSystemOccurrence time.Time
	FixAvailableAt        *time.Time
}

// Store provides custom SQL operations for the normalized CVE tables.
//
//go:generate mockgen-wrapper
type Store interface {
	// UpsertCVE inserts a CVE row if it doesn't exist (two-phase: insert then fetch).
	// Returns the UUID of the CVE row (whether newly inserted or pre-existing).
	UpsertCVE(ctx context.Context, cveRow *CVERow) (string, error)

	// UpsertEdge inserts or updates a component_cve_edges row.
	// first_system_occurrence is preserved on conflict (not updated).
	// is_fixable and fixed_by are refreshed on conflict.
	UpsertEdge(ctx context.Context, edge *EdgeRow) error

	// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
	// If keepCVEIDs is empty, all edges for the component are deleted.
	DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error

	// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
	GetCVEsForImage(ctx context.Context, imageID string) ([]*CVERow, error)

	// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
	GetAllReferencedCVEs(ctx context.Context) ([]*CVERow, error)

	// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
	// Returns number of rows deleted.
	DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error)

	// Count returns the number of rows in the cves table.
	// The query parameter is accepted for interface compatibility but is currently ignored;
	// all CVEs are counted regardless of filter criteria.
	Count(ctx context.Context, q *v1.Query) (int, error)

	// Exists returns true if a CVE row with the given UUID exists in the cves table.
	Exists(ctx context.Context, id string) (bool, error)

	// GetIDs returns the UUIDs of all rows in the cves table.
	GetIDs(ctx context.Context) ([]string, error)
}
