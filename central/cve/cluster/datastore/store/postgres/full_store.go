package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/set"
)

// NewFullStore augments the generated store with upsert and delete cluster cves functions.
func NewFullStore(db *pgxpool.Pool) store.Store {
	return &fullStoreImpl{
		db:    db,
		Store: New(db),
	}
}

type fullStoreImpl struct {
	db *pgxpool.Pool

	Store
}

func (s *fullStoreImpl) DeleteClusterCVEsForCluster(ctx context.Context, clusterID string) error {
	conn, release, err := s.acquireConn(ctx, ops.RemoveMany, "ClusterCVE")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM "+pkgSchema.ClusterCveEdgesTableName+" WHERE clusterid == $1", clusterID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM "+pkgSchema.ClusterCvesTableName+" WHERE not exists (select "+pkgSchema.ClusterCveEdgesTableName+".cveid from "+pkgSchema.ClusterCveEdgesTableName+")", clusterID)
	if err != nil {
		return err
	}
	return nil
}

func (s *fullStoreImpl) UpsertClusterCVEParts(ctx context.Context, cveType storage.CVE_CVEType, cvePartsArr ...converter.ClusterCVEParts) error {
	iTime := protoTypes.TimestampNow()
	conn, release, err := s.acquireConn(ctx, ops.UpdateMany, "ClusterCVE")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

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

	if err := removeEdgesAndCVEsForClusters(ctx, tx, cveType, impactedClusterIDs); err != nil {
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

	if err := copyFromClusterCVEEdges(ctx, tx, edges...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}

	return tx.Commit(ctx)
}

func copyFromCVEs(ctx context.Context, tx pgx.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ClusterCVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy, so first we must delete the rows, and re-add them.
	var deletes []string
	copyCols := []string{
		"id",
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

	for idx, obj := range objs {
		if storedCVE := existingCVEs[obj.GetId()]; storedCVE != nil {
			obj.Snoozed = storedCVE.GetSuppressed()
			obj.CveBaseInfo.CreatedAt = storedCVE.GetCreatedAt()
			obj.SnoozeStart = storedCVE.GetSuppressActivation()
			obj.SnoozeExpiry = storedCVE.GetSuppressExpiry()
		} else {
			obj.CveBaseInfo.CreatedAt = iTime
		}

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetCveBaseInfo().GetCve(),
			obj.GetCveBaseInfo().GetPublishedOn(),
			obj.GetCveBaseInfo().GetCreatedAt(),
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
	return err
}

func copyFromClusterCVEEdges(ctx context.Context, tx pgx.Tx, objs ...*storage.ClusterCVEEdge) error {
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

	if len(objs) == 0 {
		return nil
	}

	deletes := set.NewStringSet()

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			obj.GetClusterId(),
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
	return err
}

func getCVEs(ctx context.Context, tx pgx.Tx, cveIDs []string) (map[string]*storage.CVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "CVE")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+pkgSchema.ClusterCvesTableName+" WHERE id = ANY($1::text[])", cveIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToCVEMap := make(map[string]*storage.CVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.CVE{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, nil
}

func removeEdgesAndCVEsForClusters(ctx context.Context, tx pgx.Tx, cveType storage.CVE_CVEType, clusterIDs []string) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+pkgSchema.ClusterCveEdgesTableName+" WHERE clusterid == ANY($1::text[]) and cveid in (select id from "+pkgSchema.ClusterCvesTableName+" where cvetype = $2)", clusterIDs, cveType)
	if err != nil {
		return err
	}
	return nil
}

func (s *fullStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}
