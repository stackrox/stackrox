package postgres

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
)

// nodeCVEStore provides functionality for node CVE operations
type nodeCVEStore struct {
	cache *nodeCVECache
}

// newNodeCVEStore creates a new nodeCVEStore instance
func newNodeCVEStore() *nodeCVEStore {
	return &nodeCVEStore{
		cache: newNodeCVECache(),
	}
}

func (s *nodeCVEStore) GetCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.NodeCVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeCVEs")

	// Check cache first
	cachedCVEs, missingIDs := s.cache.GetMany(cveIDs)

	// If all CVEs were found in cache, return early
	if len(missingIDs) == 0 {
		return cachedCVEs, nil
	}

	// Fetch missing CVEs from database
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeCVEsTable+" WHERE id = ANY($1::text[])", missingIDs)
	if err != nil {
		return nil, errors.Wrap(err, "querying CVEs from database")
	}
	defer rows.Close()

	dbCVEs := make(map[string]*storage.NodeCVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, errors.Wrap(err, "scanning CVE data")
		}
		msg := &storage.NodeCVE{}
		if err := msg.UnmarshalVTUnsafe(data); err != nil {
			return nil, errors.Wrap(err, "unmarshaling CVE data")
		}
		dbCVEs[msg.GetId()] = msg
		cachedCVEs[msg.GetId()] = msg
	}

	// Cache the newly fetched CVEs
	if len(dbCVEs) > 0 {
		s.cache.SetMany(dbCVEs)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over CVE rows")
	}

	return cachedCVEs, nil
}

func (s *nodeCVEStore) CopyFromNodeCves(ctx context.Context, tx *postgres.Tx, nodeCVEs ...*storage.NodeCVE) error {
	// Process CVEs in batches using slices.Chunk
	for batch := range slices.Chunk(nodeCVEs, batchSize) {
		// Prepare data for this batch
		inputRows := make([][]interface{}, 0, len(batch))
		deletes := make([]string, 0, len(batch))

		for _, nodeCVE := range batch {
			inputRow, err := prepareCVEInputRow(nodeCVE)
			if err != nil {
				return err
			}

			inputRows = append(inputRows, inputRow)
			deletes = append(deletes, nodeCVE.GetId())
		}

		// Copy does not upsert so have to delete first.
		_, err := tx.Exec(ctx, "DELETE FROM "+nodeCVEsTable+" WHERE id = ANY($1::text[])", deletes)
		if err != nil {
			return errors.Wrap(err, "deleting CVEs before copy")
		}

		// Invalidate cache for deleted CVEs
		if len(deletes) > 0 {
			s.cache.DeleteMany(deletes)
		}

		// Insert the batch
		_, err = tx.CopyFrom(ctx, pgx.Identifier{nodeCVEsTable}, copyCols, pgx.CopyFromRows(inputRows))
		if err != nil {
			return errors.Wrap(err, "copying CVE batch to database")
		}
	}

	// Update cache with all successfully inserted/updated CVEs
	cveMap := make(map[string]*storage.NodeCVE)
	for _, nodeCVE := range nodeCVEs {
		cveMap[nodeCVE.GetId()] = nodeCVE
	}
	s.cache.SetMany(cveMap)

	return nil
}

var copyCols = []string{
	"id",
	"cvebaseinfo_cve",
	"cvebaseinfo_publishedon",
	"cvebaseinfo_createdat",
	"operatingsystem",
	"cvss",
	"severity",
	"impactscore",
	"snoozed",
	"snoozeexpiry",
	"orphaned",
	"orphanedtime",
	"serialized",
}

// prepareCVEInputRow converts a NodeCVE object into a database input row
// IMPORTANT: The order of values must exactly match the order of copyCols
func prepareCVEInputRow(nodeCVE *storage.NodeCVE) ([]interface{}, error) {
	serialized, err := nodeCVE.MarshalVT()
	if err != nil {
		return nil, errors.Wrapf(err, "marshaling CVE %s", nodeCVE.GetId())
	}

	return []interface{}{
		nodeCVE.GetId(),
		nodeCVE.GetCveBaseInfo().GetCve(),
		protocompat.NilOrTime(nodeCVE.GetCveBaseInfo().GetPublishedOn()),
		protocompat.NilOrTime(nodeCVE.GetCveBaseInfo().GetCreatedAt()),
		nodeCVE.GetOperatingSystem(),
		nodeCVE.GetCvss(),
		nodeCVE.GetSeverity(),
		nodeCVE.GetImpactScore(),
		nodeCVE.GetSnoozed(),
		protocompat.NilOrTime(nodeCVE.GetSnoozeExpiry()),
		nodeCVE.GetOrphaned(),
		protocompat.NilOrTime(nodeCVE.GetOrphanedTime()),
		serialized,
	}, nil
}

func (s *nodeCVEStore) RemoveOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error {
	// Delete orphaned CVEs and return their IDs for cache invalidation
	rows, err := tx.Query(ctx, "DELETE FROM "+nodeCVEsTable+" WHERE NOT EXISTS (SELECT "+componentCVEEdgesTable+".nodecveid FROM "+componentCVEEdgesTable+" WHERE "+nodeCVEsTable+".id = "+componentCVEEdgesTable+".nodecveid) RETURNING id")
	if err != nil {
		return errors.Wrap(err, "deleting orphaned CVEs")
	}
	defer rows.Close()

	var deletedIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return errors.Wrap(err, "scanning deleted CVE ID")
		}
		deletedIDs = append(deletedIDs, id)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating over deleted CVE rows")
	}

	// Invalidate cache for deleted CVEs
	if len(deletedIDs) > 0 {
		s.cache.DeleteMany(deletedIDs)
	}

	return nil
}

func (s *nodeCVEStore) MarkOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeCVEs")

	iTime := time.Now()
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeCVEsTable+" WHERE orphaned = 'false' AND not exists (select "+componentCVEEdgesTable+".nodecveid from "+componentCVEEdgesTable+" where "+nodeCVEsTable+".id = "+componentCVEEdgesTable+".nodecveid)")
	if err != nil {
		return errors.Wrap(err, "querying orphaned CVEs")
	}
	defer rows.Close()
	orphanedNodeCVEs := make([]*storage.NodeCVE, 0)
	ids := set.NewStringSet()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return errors.Wrap(err, "scanning orphaned CVE data")
		}
		msg := &storage.NodeCVE{}
		if err := msg.UnmarshalVTUnsafe(data); err != nil {
			return errors.Wrap(err, "unmarshaling orphaned CVE data")
		}
		if ids.Add(msg.GetId()) {
			msg.Orphaned = true
			msg.OrphanedTime = protocompat.ConvertTimeToTimestampOrNil(&iTime)
			orphanedNodeCVEs = append(orphanedNodeCVEs, msg)
		}
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating over orphaned CVE rows")
	}

	if err := s.CopyFromNodeCves(ctx, tx, orphanedNodeCVEs...); err != nil {
		return errors.Wrap(err, "marking CVEs as orphaned")
	}

	return nil
}
