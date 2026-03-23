package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	db postgres.DB
}

// New returns a new Store instance using the provided postgres database connection.
func New(db postgres.DB) store.Store {
	return &storeImpl{
		db: db,
	}
}

// UpsertCVE inserts a CVE row if it doesn't exist (two-phase: insert then fetch).
// Returns the UUID of the CVE row (whether newly inserted or pre-existing).
func (s *storeImpl) UpsertCVE(ctx context.Context, cveRow *store.CVERow) (string, error) {
	// Phase 1: Insert if new (ON CONFLICT DO NOTHING).
	insertSQL := `
		INSERT INTO cves (cve_name, source, severity, cvss_v2, cvss_v3, nvd_cvss_v3, summary, link, published_on, advisory_name, advisory_link, content_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (cve_name, source, content_hash) DO NOTHING
	`
	_, err := s.db.Exec(ctx, insertSQL,
		cveRow.CVEName,
		cveRow.Source,
		cveRow.Severity,
		cveRow.CvssV2,
		cveRow.CvssV3,
		cveRow.NvdCvssV3,
		cveRow.Summary,
		cveRow.Link,
		cveRow.PublishedOn,
		cveRow.AdvisoryName,
		cveRow.AdvisoryLink,
		cveRow.ContentHash,
	)
	if err != nil {
		return "", fmt.Errorf("inserting CVE row: %w", err)
	}

	// Phase 2: Fetch UUID (whether newly inserted or pre-existing).
	selectSQL := `
		SELECT id FROM cves WHERE cve_name = $1 AND source = $2 AND content_hash = $3
	`
	var id string
	err = s.db.QueryRow(ctx, selectSQL, cveRow.CVEName, cveRow.Source, cveRow.ContentHash).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("fetching CVE ID: %w", err)
	}

	return id, nil
}

// UpsertEdge inserts or updates a component_cve_edges row.
// first_system_occurrence is preserved on conflict (not updated).
// is_fixable and fixed_by are refreshed on conflict.
func (s *storeImpl) UpsertEdge(ctx context.Context, edge *store.EdgeRow) error {
	upsertSQL := `
		INSERT INTO component_cve_edges (component_id, cve_id, is_fixable, fixed_by, state, first_system_occurrence, fix_available_at)
		VALUES ($1, $2, $3, $4, $5, now(), $6)
		ON CONFLICT (component_id, cve_id) DO UPDATE
			SET is_fixable = EXCLUDED.is_fixable,
				fixed_by = EXCLUDED.fixed_by,
				fix_available_at = EXCLUDED.fix_available_at
	`
	_, err := s.db.Exec(ctx, upsertSQL,
		edge.ComponentID,
		edge.CveID,
		edge.IsFixable,
		edge.FixedBy,
		edge.State,
		edge.FixAvailableAt,
	)
	if err != nil {
		return fmt.Errorf("upserting edge row: %w", err)
	}

	return nil
}

// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
// If keepCVEIDs is empty, all edges for the component are deleted.
func (s *storeImpl) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
	deleteSQL := `
		DELETE FROM component_cve_edges
		WHERE component_id = $1
		  AND cve_id != ALL($2::uuid[])
	`
	_, err := s.db.Exec(ctx, deleteSQL, componentID, keepCVEIDs)
	if err != nil {
		return fmt.Errorf("deleting stale edges: %w", err)
	}

	return nil
}

// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
func (s *storeImpl) GetCVEsForImage(ctx context.Context, imageID string) ([]*store.CVERow, error) {
	querySQL := `
		SELECT c.id, c.cve_name, c.source, c.severity, c.cvss_v2, c.cvss_v3, c.nvd_cvss_v3,
		       c.summary, c.link, c.published_on, c.advisory_name, c.advisory_link,
		       c.content_hash, c.created_at
		FROM cves c
		JOIN component_cve_edges e ON c.id = e.cve_id
		JOIN image_component_v2 ic ON e.component_id = ic.id
		WHERE ic.imageidv2 = $1
	`

	rows, err := s.db.Query(ctx, querySQL, imageID)
	if err != nil {
		return nil, fmt.Errorf("querying CVEs for image: %w", err)
	}
	defer rows.Close()

	var cves []*store.CVERow
	for rows.Next() {
		cve, err := scanCVERow(rows)
		if err != nil {
			return nil, err
		}
		cves = append(cves, cve)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating CVE rows: %w", err)
	}

	return cves, nil
}

// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
func (s *storeImpl) GetAllReferencedCVEs(ctx context.Context) ([]*store.CVERow, error) {
	querySQL := `
		SELECT c.id, c.cve_name, c.source, c.severity, c.cvss_v2, c.cvss_v3, c.nvd_cvss_v3,
		       c.summary, c.link, c.published_on, c.advisory_name, c.advisory_link,
		       c.content_hash, c.created_at
		FROM cves c
		WHERE EXISTS (SELECT 1 FROM component_cve_edges e WHERE e.cve_id = c.id)
	`

	rows, err := s.db.Query(ctx, querySQL)
	if err != nil {
		return nil, fmt.Errorf("querying referenced CVEs: %w", err)
	}
	defer rows.Close()

	var cves []*store.CVERow
	for rows.Next() {
		cve, err := scanCVERow(rows)
		if err != nil {
			return nil, err
		}
		cves = append(cves, cve)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating CVE rows: %w", err)
	}

	return cves, nil
}

// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
// Returns number of rows deleted.
func (s *storeImpl) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
	deleteSQL := `
		DELETE FROM cves
		WHERE id IN (
			SELECT c.id FROM cves c
			WHERE NOT EXISTS (SELECT 1 FROM component_cve_edges e WHERE e.cve_id = c.id)
			LIMIT $1
		)
	`

	result, err := s.db.Exec(ctx, deleteSQL, batchSize)
	if err != nil {
		return 0, fmt.Errorf("deleting orphaned CVEs: %w", err)
	}

	return result.RowsAffected(), nil
}

// scanCVERow scans a single CVE row from pgx.Rows.
func scanCVERow(rows pgx.Rows) (*store.CVERow, error) {
	cve := &store.CVERow{}
	err := rows.Scan(
		&cve.ID,
		&cve.CVEName,
		&cve.Source,
		&cve.Severity,
		&cve.CvssV2,
		&cve.CvssV3,
		&cve.NvdCvssV3,
		&cve.Summary,
		&cve.Link,
		&cve.PublishedOn,
		&cve.AdvisoryName,
		&cve.AdvisoryLink,
		&cve.ContentHash,
		&cve.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning CVE row: %w", err)
	}
	return cve, nil
}
