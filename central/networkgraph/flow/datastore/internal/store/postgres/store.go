package postgres

import (
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"gorm.io/gorm"
)

// This Flow is custom to match the existing interface and how the functionality works through the system.
// Basically for draft #1 we are trying to minimize the blast radius.
// There are many places to improve as we go, for instance, utilizing the DB for queries.  Currently, the service
// layer will pass in functions that loop through all the results and filter them out.  Many of these types of things
// can be handled via query and become much more efficient.  In order to really see the benefits of Postgres for
// this store, we will need to refactor how it is used.
const (
	networkFlowsTable = pkgSchema.NetworkFlowsTableName

	// The store now uses a serial primary key id so that the store can quickly insert rows.  As such, in order
	// to get the most recent row or a count of distinct rows we need to do a self join to match the fields AND
	// the largest Flow_id.  The Flow_id is not included in the object and is purely handled by postgres.  Since flows
	// have been flattened, the entire record except for the time is what makes it distinct, so we have to hit all
	// the fields in the join.
	joinStmt = " INNER JOIN (SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId, MAX(Flow_Id) AS MaxFlow FROM network_flows GROUP BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId) tmpflow on nf.Props_SrcEntity_Type = tmpflow.Props_SrcEntity_Type AND nf.Props_SrcEntity_Id = tmpflow.Props_SrcEntity_Id AND nf.Props_DstEntity_Type = tmpflow.Props_DstEntity_Type AND nf.Props_DstEntity_Id = tmpflow.Props_DstEntity_Id AND nf.Props_DstPort = tmpflow.Props_DstPort AND nf.Props_L4Protocol = tmpflow.Props_L4Protocol AND nf.ClusterId = tmpflow.ClusterId and nf.Flow_id = tmpflow.MaxFlow "

	countStmt  = "SELECT COUNT(*) FROM network_flows nf " + joinStmt
	existsStmt = "SELECT EXISTS(SELECT 1 FROM network_flows WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7)"

	getStmt = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId FROM network_flows nf " + joinStmt +
		" WHERE nf.Props_SrcEntity_Type = $1 AND nf.Props_SrcEntity_Id = $2 AND nf.Props_DstEntity_Type = $3 AND nf.Props_DstEntity_Id = $4 AND nf.Props_DstPort = $5 AND nf.Props_L4Protocol = $6 AND nf.ClusterId = $7"
	deleteStmt         = "DELETE FROM network_flows WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7"
	deleteStmtWithTime = "DELETE FROM network_flows WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7 AND LastSeenTimestamp = $8"
	walkStmt           = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId FROM network_flows nf " + joinStmt

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt         = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId FROM network_flows nf " + joinStmt + " WHERE (nf.LastSeenTimestamp >= $1 OR nf.LastSeenTimestamp IS NULL) AND nf.ClusterId = $2"
	deleteDeploymentStmt = "DELETE FROM network_flows WHERE ClusterId = $1 AND ((Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2) OR (Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2))"

	// This seemed OK in scale and long running tests because it is not executed that frequently.  A metric
	// for this was added so we can keep an eye on the time and adjust if necessary.
	getByDeploymentStmt = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId FROM network_flows nf " + joinStmt +
		" WHERE ((nf.Props_SrcEntity_Type = 1 AND nf.Props_SrcEntity_Id = $1) OR (nf.Props_DstEntity_Type = 1 AND nf.Props_DstEntity_Id = $1)) AND nf.ClusterId = $2"
)

var (
	log = logging.LoggerForModule()

	schema = walker.Walk(reflect.TypeOf((*storage.NetworkFlow)(nil)), networkFlowsTable)

	// We begin to process in batches after this number of records
	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// These were autogenerated when I ran that to get started.
	// They are not currently used within the store.
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (bool, error)
	Get(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (*storage.NetworkFlow, bool, error)
	Upsert(ctx context.Context, obj *storage.NetworkFlow) error
	UpsertMany(ctx context.Context, objs []*storage.NetworkFlow) error
	Delete(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) error
	Walk(ctx context.Context, fn func(obj *storage.NetworkFlow) error) error
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)

	// GetAllFlows The methods below are the ones that match the flow interface which is what we probably have to match.
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)
	// GetFlowsForDeployment returns all flows referencing a specific deployment id
	GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error)

	// UpsertFlows Same as other Upserts but it takes in a time
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	// RemoveFlow Same as Delete except it takes in the object vs the IDs.  Keep an eye on it.
	RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error
	// RemoveFlowsForDeployment
	RemoveFlowsForDeployment(ctx context.Context, id string) error

	// RemoveMatchingFlows We can probably phase out the functions
	// valueMatchFn checks to see if time difference vs now is greater than orphanWindow i.e. 30 minutes
	// keyMatchFn checks to see if either the source or destination are orphaned.  Orphaned means it is type deployment and the id does not exist in deployments.
	// Though that appears to be dackbox so that is gross.  May have to keep the keyMatchFn for now and replace with a join when deployments are moved to a table?
	RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error
}

type flowStoreImpl struct {
	db        *pgxpool.Pool
	clusterID string
}

func insertIntoNetworkflow(ctx context.Context, tx pgx.Tx, clusterID string, obj *storage.NetworkFlow) error {

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

func (s *flowStoreImpl) copyFromNetworkflow(ctx context.Context, tx pgx.Tx, objs ...*storage.NetworkFlow) error {

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

			_, err = tx.CopyFrom(ctx, pgx.Identifier{networkFlowsTable}, copyCols, pgx.CopyFromRows(inputRows))

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
func New(db *pgxpool.Pool, clusterID string) FlowStore {
	return &flowStoreImpl{
		db:        db,
		clusterID: clusterID,
	}
}

func (s *flowStoreImpl) copyFrom(ctx context.Context, objs ...*storage.NetworkFlow) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "NetworkFlow")
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
	conn, release, err := s.acquireConn(ctx, ops.Get, "NetworkFlow")
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

func (s *flowStoreImpl) Upsert(ctx context.Context, obj *storage.NetworkFlow) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "NetworkFlow")

	return s.upsert(ctx, obj)
}

func (s *flowStoreImpl) UpsertMany(ctx context.Context, objs []*storage.NetworkFlow) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

	// for small batches, simply write them 1 at a time.
	if len(objs) < batchAfter {
		return s.upsert(ctx, objs...)
	}

	return s.copyFrom(ctx, objs...)
}

func (s *flowStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

	// RocksDB implementation was adding the lastUpdatedTS to a key.  That is not necessary in PG world so that
	// parameter is not being passed forward and should be removed from the interface once RocksDB is removed.
	if len(flows) < batchAfter {
		return s.upsert(ctx, flows...)
	}

	return s.copyFrom(ctx, flows...)
}

// Count returns the number of objects in the store
func (s *flowStoreImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "NetworkFlow")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *flowStoreImpl) Exists(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "NetworkFlow")

	row := s.db.QueryRow(ctx, existsStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store
func (s *flowStoreImpl) Get(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (*storage.NetworkFlow, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.Get, "NetworkFlow")
	if err != nil {
		return nil, false, err
	}
	defer release()

	// We can discuss this a bit, but this statement should only ever return 1 row.  Doing it this way allows
	// us to use the readRows function
	rows, err := conn.Query(ctx, getStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil || flows == nil {
		return nil, false, err
	}

	return flows[0], true, nil
}

func (s *flowStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
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
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
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
	rows, err := s.db.Query(ctx, walkStmt)
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
	return nil
}

// RemoveFlowsForDeployment removes all flows where the source OR destination match the deployment id
func (s *flowStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveFlowsByDeployment, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.RemoveFlowsByDeployment, "NetworkFlow")
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
func (s *flowStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	var rows pgx.Rows
	var err error
	// Default to Now as that is when we are reading them
	lastUpdateTS := *types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}
	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}

	return flows, lastUpdateTS, nil
}

// GetMatchingFlows iterates over all of the objects in the store and applies the closure
func (s *flowStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	var rows pgx.Rows
	var err error

	// Default to Now as that is when we are reading them
	lastUpdateTS := *types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}

	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, pred)

	return flows, lastUpdateTS, err
}

// GetFlowsForDeployment returns the flows matching the deployment ID
func (s *flowStoreImpl) GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetFlowsForDeployment, "NetworkFlow")
	var rows pgx.Rows
	var err error

	rows, err = s.db.Query(ctx, getByDeploymentStmt, deploymentID, s.clusterID)

	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)

	return flows, err
}

func (s *flowStoreImpl) delete(ctx context.Context, objs ...*storage.NetworkFlowProperties) error {
	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
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
		_, err := tx.Exec(ctx, deleteStmt, obj.GetSrcEntity().GetType(), obj.GetSrcEntity().GetId(), obj.GetDstEntity().GetType(), obj.GetDstEntity().GetId(), obj.GetDstPort(), obj.GetL4Protocol(), s.clusterID)

		if err != nil {
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

// RemoveFlow removes the specified flow from the store
func (s *flowStoreImpl) RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	if err := s.delete(ctx, props); err != nil {
		return err
	}
	return nil
}

// RemoveMatchingFlows removes flows from the store that fit the criteria specified in both keyMatchFn AND valueMatchFN
// keyMatchFn will return true if a flow references a source OR destination deployment that has been deleted
// valueMatchFn will return true if the lastSeenTimestamp of a flow is more than 30 minutes ago.
// TODO(ROX-9921) Figure out what to do with the functions.
func (s *flowStoreImpl) RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	if err != nil {
		return err
	}
	defer release()

	// TODO(ROX-9921) Look at refactoring how these predicates work as an overall refactor of flows.
	// This operation matches if the either the dest or src deployment no longer exists AND then
	// if the last seen time is outside a time window.  Since we do not yet know what deployments exist
	// in Postgres we cannot fully do this work in SQL.  Additionally, there may be issues with the synchronization
	// of when flow is created vs a deployment deleted that may also make that problematic.
	if keyMatchFn != nil {
		rows, err := conn.Query(ctx, walkStmt)
		if err != nil {
			return err
		}
		defer rows.Close()

		// keyMatchFn is passed in to the readRows method in order to filter down to rows referencing
		// deleted deployments.
		deleteFlows, err := s.readRows(rows, keyMatchFn)

		if err != nil {
			return nil
		}

		for _, flow := range deleteFlows {
			if valueMatchFn != nil && !valueMatchFn(flow) {
				continue
			}
			// This is a cleanup operation so we can make it slow for now
			tx, err := conn.Begin(ctx)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, deleteStmtWithTime, flow.GetProps().GetSrcEntity().GetType(), flow.GetProps().GetSrcEntity().GetId(), flow.GetProps().GetDstEntity().GetType(), flow.GetProps().GetDstEntity().GetId(), flow.GetProps().GetDstPort(), flow.GetProps().GetL4Protocol(), s.clusterID, pgutils.NilOrTime(flow.GetLastSeenTimestamp()))

			if err != nil {
				if err := tx.Rollback(ctx); err != nil {
					return err
				}
				return err
			}

			if err := tx.Commit(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

//// Used for testing

func dropTableNetworkflow(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS network_flows CASCADE")
}

// Destroy destroys the tables
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableNetworkflow(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB, clusterID string) FlowStore {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, networkFlowsTable)
	return New(db, clusterID)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *flowStoreImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *flowStoreImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
