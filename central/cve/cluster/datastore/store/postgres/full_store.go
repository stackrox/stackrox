package postgres

import (
	"context"
	"testing"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	clusterCVEsTable    = pkgSchema.ClusterCvesTableName
	clusterCVEEdgeTable = pkgSchema.ClusterCveEdgesTableName
)

// NewFullStore augments the generated store with upsert and delete cluster cves functions.
func NewFullStore(db postgres.DB) store.Store {
	return &fullStoreImpl{
		db:    db,
		Store: New(db),
	}
}

// NewFullTestStore is used for testing.
func NewFullTestStore(_ testing.TB, db postgres.DB, store Store) store.Store {
	return &fullStoreImpl{
		db:    db,
		Store: store,
	}
}

type fullStoreImpl struct {
	db postgres.DB

	Store
}

func (s *fullStoreImpl) DeleteClusterCVEsForCluster(ctx context.Context, clusterID string) error {
	conn, release, err := s.acquireConn(ctx, ops.RemoveMany, "ClusterCVEs")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM "+clusterCVEEdgeTable+" WHERE clusterid = $1", uuid.FromStringOrNil(clusterID))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "DELETE FROM "+clusterCVEsTable+" WHERE not exists (select "+clusterCVEEdgeTable+".cveid from "+clusterCVEEdgeTable+" where "+clusterCVEEdgeTable+".cveid = "+clusterCVEsTable+".id)")
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *fullStoreImpl) ReconcileClusterCVEParts(ctx context.Context, cveType storage.CVE_CVEType, cvePartsArr ...converter.ClusterCVEParts) error {
	iTime := protoTypes.TimestampNow()

	cves := make([]*storage.ClusterCVE, 0, len(cvePartsArr))
	var edges []*storage.ClusterCVEEdge
	var impactedClusterIDs []string
	for _, parts := range cvePartsArr {
		cves = append(cves, parts.CVE)
		for _, child := range parts.Children {
			edges = append(edges, child.Edge)
			impactedClusterIDs = append(impactedClusterIDs, child.ClusterID)
		}
	}

	conn, release, err := s.acquireConn(ctx, ops.UpdateMany, "ClusterCVE")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := copyFromClusterCVEEdges(ctx, tx, cveType, impactedClusterIDs, edges...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := copyFromCVEs(ctx, tx, iTime, cves...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	return tx.Commit(ctx)
}

func copyFromCVEs(ctx context.Context, tx *postgres.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ClusterCVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy, so first we must delete the rows, and re-add them.
	var deletes []string
	copyCols := []string{
		"id",
		"type",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"cvss",
		"severity",
		"impactscore",
		"snoozed",
		"snoozeexpiry",
		"serialized",
	}

	ids := set.NewStringSet()
	for _, obj := range objs {
		ids.Add(obj.GetId())
	}
	existingCVEs, err := getCVEs(ctx, tx, ids.AsSlice())
	if err != nil {
		return err
	}

	for idx, obj := range objs {
		if storedCVE := existingCVEs[obj.GetId()]; storedCVE != nil {
			obj.Snoozed = storedCVE.GetSnoozed()
			obj.CveBaseInfo.CreatedAt = storedCVE.GetCveBaseInfo().GetCreatedAt()
			obj.SnoozeStart = storedCVE.GetSnoozeStart()
			obj.SnoozeExpiry = storedCVE.GetSnoozeExpiry()
		} else {
			obj.CveBaseInfo.CreatedAt = iTime
		}

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetType(),
			obj.GetCveBaseInfo().GetCve(),
			pgutils.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			pgutils.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
			obj.GetCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetSnoozed(),
			pgutils.NilOrTime(obj.GetSnoozeExpiry()),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+pkgSchema.ClusterCvesTableName+" WHERE id = ANY($1::text[])", deletes)
			if err != nil {
				return err
			}
			// Clear the inserts for the next batch.
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{pkgSchema.ClusterCvesTableName}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return removeOrphanedClusterCVEs(ctx, tx)
}

func copyFromClusterCVEEdges(ctx context.Context, tx *postgres.Tx, cveType storage.CVE_CVEType, clusters []string, objs ...*storage.ClusterCVEEdge) error {
	inputRows := [][]interface{}{}

	var err error
	copyCols := []string{
		"id",
		"isfixable",
		"fixedby",
		"clusterid",
		"cveid",
		"serialized",
	}

	oldEdges, err := getClusterCVEEdgeIDs(ctx, tx, cveType, clusters)
	if err != nil {
		return err
	}

	deletes := set.NewStringSet()

	for idx, obj := range objs {
		oldEdges.Remove(obj.GetId())

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			uuid.FromStringOrNil(obj.GetClusterId()),
			obj.GetCveId(),
			serialized,
		})

		// Add the id to be deleted.
		deletes.Add(obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+pkgSchema.ClusterCveEdgesTableName+" WHERE id = ANY($1::text[])", deletes.AsSlice())
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{pkgSchema.ClusterCveEdgesTableName}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return removeOrphanedImageCVEEdges(ctx, tx, oldEdges.AsSlice())
}

func getCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.ClusterCVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "CVE")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+pkgSchema.ClusterCvesTableName+" WHERE id = ANY($1::text[])", cveIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idToCVEMap := make(map[string]*storage.ClusterCVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ClusterCVE{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, rows.Err()
}

func getClusterCVEEdgeIDs(ctx context.Context, tx *postgres.Tx, cveType storage.CVE_CVEType, clusterIDs []string) (set.StringSet, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ClusterCVEEdgeIDs")

	rows, err := tx.Query(ctx, "select id FROM "+clusterCVEEdgeTable+" WHERE clusterid = ANY($1::uuid[]) and cveid in (select id from "+clusterCVEsTable+" where type = $2)", clusterIDs, cveType)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return set.NewStringSet(ids...), rows.Err()
}

func removeOrphanedImageCVEEdges(ctx context.Context, tx *postgres.Tx, orphanedEdgeIDs []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "ClusterCVEEdges")

	_, err := tx.Exec(ctx, "DELETE FROM "+clusterCVEEdgeTable+" WHERE id = ANY($1::text[])", orphanedEdgeIDs)
	return err
}

func removeOrphanedClusterCVEs(ctx context.Context, tx *postgres.Tx) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "ClusterCVEs")

	_, err := tx.Exec(ctx, "DELETE FROM "+clusterCVEsTable+" WHERE not exists (select "+clusterCVEEdgeTable+".cveid from "+clusterCVEEdgeTable+" where "+clusterCVEsTable+".id = "+clusterCVEEdgeTable+".cveid)")
	return err
}

func (s *fullStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}
