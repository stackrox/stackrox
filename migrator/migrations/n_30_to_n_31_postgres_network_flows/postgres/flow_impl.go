// This file was originally generated with
// //go:generate cp ../../../../central/networkgraph/flow/datastore/internal/store/postgres/store.go flow_impl.go

package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
)

// This Flow is custom to match the existing interface and how the functionality works through the system.
// Basically for draft #1 we are trying to minimize the blast radius.
// There are many places to improve as we go, for instance, utilizing the DB for queries.  Currently, the service
// layer will pass in functions that loop through all the results and filter them out.  Many of these types of things
// can be handled via query and become much more efficient.  In order to really see the benefits of Postgres for
// this store, we will need to refactor how it is used.
const (
	baseTable = "network_flows"

	// The store now uses a serial primary key id so that the store can quickly insert rows.  As such, in order
	// to get the most recent row or a count of distinct rows we need to do a self join to match the fields AND
	// the largest Flow_id.  The Flow_id is not included in the object and is purely handled by postgres.  Since flows
	// have been flattened, the entire record except for the time is what makes it distinct, so we have to hit all
	// the fields in the join.
	joinStmt = ` INNER JOIN 
	(SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, 
	Props_L4Protocol, ClusterId, MAX(Flow_Id) AS MaxFlow 
	FROM network_flows 
	GROUP BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId) tmpflow 
	on nf.Props_SrcEntity_Type = tmpflow.Props_SrcEntity_Type AND nf.Props_SrcEntity_Id = tmpflow.Props_SrcEntity_Id AND 
	nf.Props_DstEntity_Type = tmpflow.Props_DstEntity_Type AND nf.Props_DstEntity_Id = tmpflow.Props_DstEntity_Id AND 
	nf.Props_DstPort = tmpflow.Props_DstPort AND nf.Props_L4Protocol = tmpflow.Props_L4Protocol AND 
	nf.ClusterId = tmpflow.ClusterId and nf.Flow_id = tmpflow.MaxFlow `

	deleteStmt = "DELETE FROM network_flows WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7"
	walkStmt   = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text FROM network_flows nf " + joinStmt + " WHERE nf.ClusterId = $1"

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt         = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text FROM network_flows nf " + joinStmt + " WHERE (nf.LastSeenTimestamp >= $1 OR nf.LastSeenTimestamp IS NULL) AND nf.ClusterId = $2"
	deleteDeploymentStmt = "DELETE FROM network_flows WHERE ClusterId = $1 AND ((Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2) OR (Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2))"
)

var (
	log = logging.LoggerForModule()

	// We begin to process in batches after this number of records
	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

type flowStoreImpl struct {
	db        postgres.DB
	clusterID uuid.UUID
}

func insertIntoNetworkflow(ctx context.Context, tx *postgres.Tx, clusterID uuid.UUID, obj *storage.NetworkFlow) error {

	values := []interface{}{
		// parent primary keys start
		obj.GetProps().GetSrcEntity().GetType(),
		obj.GetProps().GetSrcEntity().GetId(),
		obj.GetProps().GetDstEntity().GetType(),
		obj.GetProps().GetDstEntity().GetId(),
		obj.GetProps().GetDstPort(),
		obj.GetProps().GetL4Protocol(),
		pgutils.NilOrTime(obj.GetLastSeenTimestamp()),
		clusterID,
	}

	finalStr := "INSERT INTO network_flows (Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId) VALUES($1, $2, $3, $4, $5, $6, $7, $8)"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

func (s *flowStoreImpl) copyFromNetworkflow(ctx context.Context, tx *postgres.Tx, objs ...*storage.NetworkFlow) error {

	inputRows := [][]interface{}{}
	var err error

	copyCols := []string{
		"props_srcentity_type",
		"props_srcentity_id",
		"props_dstentity_type",
		"props_dstentity_id",
		"props_dstport",
		"props_l4protocol",
		"lastseentimestamp",
		"clusterid",
	}

	for idx, obj := range objs {
		inputRows = append(inputRows, []interface{}{
			obj.GetProps().GetSrcEntity().GetType(),
			obj.GetProps().GetSrcEntity().GetId(),
			obj.GetProps().GetDstEntity().GetType(),
			obj.GetProps().GetDstEntity().GetId(),
			obj.GetProps().GetDstPort(),
			obj.GetProps().GetL4Protocol(),
			pgutils.NilOrTime(obj.GetLastSeenTimestamp()),
			s.clusterID,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			_, err = tx.CopyFrom(ctx, pgx.Identifier{baseTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB, clusterID string) store.FlowStore {
	clusterUUID, err := uuid.FromString(clusterID)
	if err != nil {
		log.Errorf("cluster ID is not valid.  %v", err)
		return nil
	}

	return &flowStoreImpl{
		db:        db,
		clusterID: clusterUUID,
	}
}

func (s *flowStoreImpl) copyFrom(ctx context.Context, objs ...*storage.NetworkFlow) error {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromNetworkflow(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *flowStoreImpl) upsert(ctx context.Context, objs ...*storage.NetworkFlow) error {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer release()

	// Moved the transaction outside the loop which greatly improved the performance of these individual inserts.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objs {

		if err := insertIntoNetworkflow(ctx, tx, s.clusterID, obj); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *flowStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	return pgutils.Retry(func() error {
		return s.retryableUpsertFlows(ctx, flows, lastUpdateTS)
	})
}

func (s *flowStoreImpl) retryableUpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, _ timestamp.MicroTS) error {
	// RocksDB implementation was adding the lastUpdatedTS to a key.  That is not necessary in PG world so that
	// parameter is not being passed forward and should be removed from the interface once RocksDB is removed.
	if len(flows) < batchAfter {
		return s.upsert(ctx, flows...)
	}

	return s.copyFrom(ctx, flows...)
}

func (s *flowStoreImpl) acquireConn(ctx context.Context) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *flowStoreImpl) readRows(rows pgx.Rows, pred func(*storage.NetworkFlowProperties) bool) ([]*storage.NetworkFlow, error) {
	var flows []*storage.NetworkFlow

	for rows.Next() {
		var srcType storage.NetworkEntityInfo_Type
		var srcID string
		var destType storage.NetworkEntityInfo_Type
		var destID string
		var port uint32
		var protocol storage.L4Protocol
		var lastTime *time.Time
		var clusterID string

		if err := rows.Scan(&srcType, &srcID, &destType, &destID, &port, &protocol, &lastTime, &clusterID); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		var ts *types.Timestamp
		if lastTime != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*lastTime)
		}

		flow := &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: srcType,
					Id:   srcID,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: destType,
					Id:   destID,
				},
				DstPort:    port,
				L4Protocol: protocol,
			},
			LastSeenTimestamp: ts,
			ClusterId:         clusterID,
		}

		// Apply the predicate function.  Will phase out as we move away form Rocks to where clause
		if pred == nil || pred(flow.Props) {
			flows = append(flows, flow)
		}
	}

	log.Debugf("Read returned %d flows", len(flows))
	return flows, nil
}

// Delete removes the specified ID from the store
func (s *flowStoreImpl) Delete(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) error {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID); err != nil {
		return err
	}
	return nil
}

// Walk iterates over all of the objects in the store and applies the closure
// TODO(ROX-9921) Investigate this method to see if it is doing what it should
func (s *flowStoreImpl) Walk(ctx context.Context, fn func(obj *storage.NetworkFlow) error) error {
	rows, err := s.db.Query(ctx, walkStmt, s.clusterID)
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var srcType storage.NetworkEntityInfo_Type
		var srcID string
		var destType storage.NetworkEntityInfo_Type
		var destID string
		var port uint32
		var protocol storage.L4Protocol
		var lastTime *time.Time
		var clusterID string

		if err := rows.Scan(&srcType, &srcID, &destType, &destID, &port, &protocol, &lastTime, &clusterID); err != nil {
			return err
		}

		var ts *types.Timestamp
		if lastTime != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*lastTime)
		}

		flow := &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: srcType,
					Id:   srcID,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: destType,
					Id:   destID,
				},
				DstPort:    port,
				L4Protocol: protocol,
			},
			LastSeenTimestamp: ts,
			ClusterId:         clusterID,
		}

		if err := fn(flow); err != nil {
			return err
		}
	}
	return rows.Err()
}

// RemoveFlowsForDeployment removes all flows where the source OR destination match the deployment id
func (s *flowStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	return pgutils.Retry(func() error {
		return s.retryableRemoveFlowsForDeployment(ctx, id)
	})
}

func (s *flowStoreImpl) retryableRemoveFlowsForDeployment(ctx context.Context, id string) error {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteDeploymentStmt, s.clusterID, id); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetAllFlows returns the object, if it exists from the store, timestamp and error
func (s *flowStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	return pgutils.Retry3(func() ([]*storage.NetworkFlow, *types.Timestamp, error) {
		return s.retryableGetAllFlows(ctx, since)
	})
}

func (s *flowStoreImpl) retryableGetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	var rows pgx.Rows
	var err error
	// Default to Now as that is when we are reading them
	lastUpdateTS := types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt, s.clusterID)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}
	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}

	return flows, lastUpdateTS, rows.Err()
}

// GetMatchingFlows iterates over all of the objects in the store and applies the closure
func (s *flowStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	return pgutils.Retry3(func() ([]*storage.NetworkFlow, *types.Timestamp, error) {
		return s.retryableGetMatchingFlows(ctx, pred, since)
	})
}

func (s *flowStoreImpl) retryableGetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	var rows pgx.Rows
	var err error

	// Default to Now as that is when we are reading them
	lastUpdateTS := types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt, s.clusterID)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}

	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, pred)
	if err != nil {
		return nil, nil, err
	}

	return flows, lastUpdateTS, rows.Err()
}
