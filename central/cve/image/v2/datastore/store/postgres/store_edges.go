package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
	parentStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
)

// EdgeStore provides custom SQL operations for the component_cve_edges table.
type EdgeStore interface {
	// UpsertEdge inserts or updates a component_cve_edges row.
	// first_system_occurrence is preserved on conflict (not updated).
	// is_fixable, fixed_by, and fix_available_at are refreshed on conflict.
	UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error

	// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
	// If keepCVEIDs is empty, all edges for the component are deleted.
	DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error

	// GetCVEsForImage returns all CVEs for a given image (joined through component_cve_edges and image_component_v2).
	GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error)

	// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
	GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error)

	// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
	// Returns the number of rows deleted.
	DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error)

	// GetCVEsWithEdges retrieves CVEs and their component edges for an image.
	GetCVEsWithEdges(ctx context.Context, imageID string) ([]parentStore.CVEEdgePair, error)

	// GetCVEWithEdge retrieves a single CVE by ID along with its edge for a component.
	GetCVEWithEdge(ctx context.Context, cveID string, componentID string) (*parentStore.CVEEdgePair, bool, error)
}

type edgeStoreImpl struct {
	db postgres.DB
}

// NewEdgeStore returns a new EdgeStore backed by the provided postgres database.
func NewEdgeStore(db postgres.DB) EdgeStore {
	return &edgeStoreImpl{db: db}
}

// UpsertEdge inserts or updates a component_cve_edges row.
// first_system_occurrence is preserved on conflict (not updated).
// is_fixable, fixed_by, and fix_available_at are refreshed on conflict.
func (s *edgeStoreImpl) UpsertEdge(ctx context.Context, edge *storage.NormalizedComponentCVEEdge) error {
	upsertSQL := `
		INSERT INTO component_cve_edges (component_id, cve_id, is_fixable, fixed_by, state, first_system_occurrence, fix_available_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (component_id, cve_id) DO UPDATE
			SET is_fixable       = EXCLUDED.is_fixable,
				fixed_by         = EXCLUDED.fixed_by,
				fix_available_at = EXCLUDED.fix_available_at
	`
	_, err := s.db.Exec(ctx, upsertSQL,
		edge.GetComponentId(),
		edge.GetCveId(),
		edge.GetIsFixable(),
		edge.GetFixedBy(),
		edge.GetState(),
		protocompat.NilOrTime(edge.GetFirstSystemOccurrence()),
		protocompat.NilOrTime(edge.GetFixAvailableAt()),
	)
	if err != nil {
		return fmt.Errorf("upserting edge row: %w", err)
	}

	return nil
}

// DeleteStaleEdges removes edges for a component whose cve_id is NOT in keepCVEIDs.
// If keepCVEIDs is empty, all edges for the component are deleted.
func (s *edgeStoreImpl) DeleteStaleEdges(ctx context.Context, componentID string, keepCVEIDs []string) error {
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

// GetCVEsForImage returns all CVEs for a given image via a 3-table JOIN.
func (s *edgeStoreImpl) GetCVEsForImage(ctx context.Context, imageID string) ([]*storage.NormalizedCVE, error) {
	querySQL := `
		SELECT c.serialized
		FROM cves c
		JOIN component_cve_edges e ON c.id = e.cve_id
		JOIN ` + pkgSchema.ImageComponentV2TableName + ` ic ON e.component_id = ic.id
		WHERE ic.imageidv2 = $1
	`
	rows, err := s.db.Query(ctx, querySQL, imageID)
	if err != nil {
		return nil, fmt.Errorf("querying CVEs for image: %w", err)
	}

	cves, err := pgutils.ScanRows[storage.NormalizedCVE, *storage.NormalizedCVE](rows)
	if err != nil {
		return nil, fmt.Errorf("scanning CVE rows for image: %w", err)
	}

	return cves, nil
}

// GetAllReferencedCVEs returns all CVEs referenced by at least one component_cve_edges row.
func (s *edgeStoreImpl) GetAllReferencedCVEs(ctx context.Context) ([]*storage.NormalizedCVE, error) {
	querySQL := `
		SELECT c.serialized
		FROM cves c
		WHERE EXISTS (SELECT 1 FROM component_cve_edges e WHERE e.cve_id = c.id)
	`
	rows, err := s.db.Query(ctx, querySQL)
	if err != nil {
		return nil, fmt.Errorf("querying referenced CVEs: %w", err)
	}

	cves, err := pgutils.ScanRows[storage.NormalizedCVE, *storage.NormalizedCVE](rows)
	if err != nil {
		return nil, fmt.Errorf("scanning referenced CVE rows: %w", err)
	}

	return cves, nil
}

// DeleteOrphanedCVEsBatch deletes up to batchSize CVEs with no referencing edges.
// Returns the number of rows deleted.
func (s *edgeStoreImpl) DeleteOrphanedCVEsBatch(ctx context.Context, batchSize int) (int64, error) {
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

// GetCVEsWithEdges retrieves CVEs and their component edges for an image.
func (s *edgeStoreImpl) GetCVEsWithEdges(ctx context.Context, imageID string) ([]parentStore.CVEEdgePair, error) {
	querySQL := `
		SELECT
			c.serialized as cve_data,
			e.component_id,
			e.cve_id,
			e.is_fixable,
			e.fixed_by,
			e.state,
			e.first_system_occurrence,
			e.fix_available_at
		FROM cves c
		JOIN component_cve_edges e ON c.id = e.cve_id
		JOIN ` + pkgSchema.ImageComponentV2TableName + ` ic ON e.component_id = ic.id
		WHERE ic.imageidv2 = $1
		ORDER BY c.cve_name, e.component_id
	`

	rows, err := s.db.Query(ctx, querySQL, imageID)
	if err != nil {
		return nil, fmt.Errorf("querying CVE+edge pairs for image: %w", err)
	}
	defer rows.Close()

	var pairs []parentStore.CVEEdgePair
	for rows.Next() {
		var cve storage.NormalizedCVE
		var edge storage.NormalizedComponentCVEEdge
		var cveData []byte
		var firstSysOcc, fixAvailAt *time.Time

		err := rows.Scan(
			&cveData,
			&edge.ComponentId,
			&edge.CveId,
			&edge.IsFixable,
			&edge.FixedBy,
			&edge.State,
			&firstSysOcc,
			&fixAvailAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning CVE+edge row: %w", err)
		}

		if err := cve.UnmarshalVT(cveData); err != nil {
			return nil, fmt.Errorf("unmarshaling CVE: %w", err)
		}

		if firstSysOcc != nil {
			edge.FirstSystemOccurrence = timestamppb.New(*firstSysOcc)
		}
		if fixAvailAt != nil {
			edge.FixAvailableAt = timestamppb.New(*fixAvailAt)
		}

		pairs = append(pairs, parentStore.CVEEdgePair{CVE: &cve, Edge: &edge})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating CVE+edge rows: %w", err)
	}

	return pairs, nil
}

// GetCVEWithEdge retrieves a single CVE by ID along with its edge for a component.
func (s *edgeStoreImpl) GetCVEWithEdge(ctx context.Context, cveID string, componentID string) (*parentStore.CVEEdgePair, bool, error) {
	querySQL := `
		SELECT
			c.serialized as cve_data,
			e.component_id,
			e.cve_id,
			e.is_fixable,
			e.fixed_by,
			e.state,
			e.first_system_occurrence,
			e.fix_available_at
		FROM cves c
		JOIN component_cve_edges e ON c.id = e.cve_id
		WHERE c.id = $1 AND e.component_id = $2
	`

	row := s.db.QueryRow(ctx, querySQL, cveID, componentID)

	var cve storage.NormalizedCVE
	var edge storage.NormalizedComponentCVEEdge
	var cveData []byte
	var firstSysOcc, fixAvailAt *time.Time

	err := row.Scan(
		&cveData,
		&edge.ComponentId,
		&edge.CveId,
		&edge.IsFixable,
		&edge.FixedBy,
		&edge.State,
		&firstSysOcc,
		&fixAvailAt,
	)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("querying CVE+edge: %w", err)
	}

	if err := cve.UnmarshalVT(cveData); err != nil {
		return nil, false, fmt.Errorf("unmarshaling CVE: %w", err)
	}

	if firstSysOcc != nil {
		edge.FirstSystemOccurrence = timestamppb.New(*firstSysOcc)
	}
	if fixAvailAt != nil {
		edge.FixAvailableAt = timestamppb.New(*fixAvailAt)
	}

	return &parentStore.CVEEdgePair{CVE: &cve, Edge: &edge}, true, nil
}
