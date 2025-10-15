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

// NodeCVEStore provides functionality for node CVE operations
type NodeCVEStore interface {
	GetCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.NodeCVE, error)
	CopyFromNodeCves(ctx context.Context, tx *postgres.Tx, objs ...*storage.NodeCVE) error
	RemoveOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error
	MarkOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error
}

type nodeCVEStoreImpl struct {
	cache *nodeCVECache
}

// NewNodeCVEStore creates a new NodeCVEStore instance
func NewNodeCVEStore() NodeCVEStore {
	return &nodeCVEStoreImpl{
		cache: newNodeCVECache(),
	}
}

func (s *nodeCVEStoreImpl) GetCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.NodeCVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeCVEs")

	// Check cache first
	cachedCVEs, missingIDs := s.cache.GetMany(cveIDs)
	idToCVEMap := make(map[string]*storage.NodeCVE, len(cveIDs))

	// Add cached CVEs to result
	for id, cve := range cachedCVEs {
		idToCVEMap[id] = cve
	}

	// If all CVEs were found in cache, return early
	if len(missingIDs) == 0 {
		return idToCVEMap, nil
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
		idToCVEMap[msg.GetId()] = msg
	}

	// Cache the newly fetched CVEs
	if len(dbCVEs) > 0 {
		s.cache.SetMany(dbCVEs)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating over CVE rows")
	}

	return idToCVEMap, nil
}

func (s *nodeCVEStoreImpl) CopyFromNodeCves(ctx context.Context, tx *postgres.Tx, objs ...*storage.NodeCVE) error {
	copyCols := []string{
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

	// Process CVEs in batches using slices.Chunk
	for batch := range slices.Chunk(objs, batchSize) {
		// Prepare data for this batch
		inputRows := make([][]interface{}, 0, len(batch))
		deletes := make([]string, 0, len(batch))

		for _, obj := range batch {
			serialized, marshalErr := obj.MarshalVT()
			if marshalErr != nil {
				return errors.Wrapf(marshalErr, "marshaling CVE %s", obj.GetId())
			}

			inputRows = append(inputRows, []interface{}{
				obj.GetId(),
				obj.GetCveBaseInfo().GetCve(),
				protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
				protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
				obj.GetOperatingSystem(),
				obj.GetCvss(),
				obj.GetSeverity(),
				obj.GetImpactScore(),
				obj.GetSnoozed(),
				protocompat.NilOrTime(obj.GetSnoozeExpiry()),
				obj.GetOrphaned(),
				protocompat.NilOrTime(obj.GetOrphanedTime()),
				serialized,
			})

			deletes = append(deletes, obj.GetId())
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
	for _, obj := range objs {
		cveMap[obj.GetId()] = obj
	}
	s.cache.SetMany(cveMap)

	return nil
}

func (s *nodeCVEStoreImpl) RemoveOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error {
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

func (s *nodeCVEStoreImpl) MarkOrphanedNodeCVEs(ctx context.Context, tx *postgres.Tx) error {
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
