package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
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
	joinStmt = ` INNER JOIN 
	(SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, 
	Props_L4Protocol, ClusterId, MAX(Flow_Id) AS MaxFlow 
	FROM %s 
	GROUP BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId) tmpflow 
	on nf.Props_SrcEntity_Type = tmpflow.Props_SrcEntity_Type AND nf.Props_SrcEntity_Id = tmpflow.Props_SrcEntity_Id AND 
	nf.Props_DstEntity_Type = tmpflow.Props_DstEntity_Type AND nf.Props_DstEntity_Id = tmpflow.Props_DstEntity_Id AND 
	nf.Props_DstPort = tmpflow.Props_DstPort AND nf.Props_L4Protocol = tmpflow.Props_L4Protocol AND 
	nf.ClusterId = tmpflow.ClusterId and nf.Flow_id = tmpflow.MaxFlow `

	deleteStmt         = "DELETE FROM network_flows_v2 WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7"
	deleteStmtWithTime = "DELETE FROM network_flows_v2 WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7 AND LastSeenTimestamp = $8"
	walkStmt           = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text FROM %s nf " + joinStmt

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, 
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text 
	FROM %s nf ` + joinStmt +
		` WHERE (nf.LastSeenTimestamp >= $1 OR nf.LastSeenTimestamp IS NULL)`
	deleteSrcDeploymentStmt = "DELETE FROM network_flows_v2 WHERE ClusterId = $1 AND Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2"
	deleteDstDeploymentStmt = "DELETE FROM network_flows_v2 WHERE ClusterId = $1 AND Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2"

	getByDeploymentStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, 
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_SrcEntity_Type = 1 AND nf.Props_SrcEntity_Id = $1
	UNION ALL
	SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_DstEntity_Type = 1 AND nf.Props_DstEntity_Id = $1`

	pruneStaleNetworkFlowsStmt = `DELETE FROM %s a USING (
      SELECT MAX(flow_id) as Max_Flow, Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId
        FROM %s
        GROUP BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId
		HAVING COUNT(*) > 1
      ) b
      WHERE a.Props_SrcEntity_Type = b.Props_SrcEntity_Type
 	AND a.Props_SrcEntity_Id = b.Props_SrcEntity_Id
 	AND a.Props_DstEntity_Type = b.Props_DstEntity_Type
 	AND a.Props_DstEntity_Id = b.Props_DstEntity_Id
	AND a.Props_DstPort = b.Props_DstPort
	AND a.Props_L4Protocol = b.Props_L4Protocol
	AND a.ClusterId = b.ClusterId
      AND a.Flow_Id <> b.Max_Flow;
	`

	pruneNetworkFlowsSrcStmt = `DELETE FROM %s child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.Props_SrcEntity_Id = parent.id::text AND parent.clusterid = $1) 
		AND Props_SrcEntity_Type = 1
		AND LastSeenTimestamp < $2`

	pruneNetworkFlowsDestStmt = `DELETE FROM %s child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.Props_DstEntity_Id = parent.id::text AND parent.clusterid = $1) 
		AND Props_DstEntity_Type = 1
		AND LastSeenTimestamp < $2`
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

	deleteTimeout = env.PostgresDefaultNetworkFlowDeleteTimeout.DurationSetting()

	queryTimeout = env.PostgresDefaultNetworkFlowQueryTimeout.DurationSetting()
)

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// GetAllFlows The methods below are the ones that match the flow interface which is what we probably have to match.
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	// GetFlowsForDeployment returns all flows referencing a specific deployment id
	GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error)

	// UpsertFlows Same as other Upserts but it takes in a time
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	// RemoveFlow Same as Delete except it takes in the object vs the IDs.  Keep an eye on it.
	RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error
	// RemoveFlowsForDeployment remove flows associated with a deployment
	RemoveFlowsForDeployment(ctx context.Context, id string) error

	// RemoveStaleFlows remove stale duplicate network flows
	RemoveStaleFlows(ctx context.Context) error

	// RemoveOrphanedFlows remove flows that have been orphaned by deployments
	RemoveOrphanedFlows(ctx context.Context, orphanWindow *time.Time) error
}

type flowStoreImpl struct {
	db            postgres.DB
	mutex         sync.Mutex
	clusterID     uuid.UUID
	partitionName string
}

func (s *flowStoreImpl) insertIntoNetworkflow(ctx context.Context, tx *postgres.Tx, clusterID uuid.UUID, obj *storage.NetworkFlow) error {

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

	finalStr := fmt.Sprintf("INSERT INTO %s (Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId) VALUES($1, $2, $3, $4, $5, $6, $7, $8)", s.partitionName)
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

			_, err = tx.CopyFrom(ctx, pgx.Identifier{s.partitionName}, copyCols, pgx.CopyFromRows(inputRows))

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
func New(db postgres.DB, clusterID string) FlowStore {
	clusterUUID, err := uuid.FromString(clusterID)
	if err != nil {
		log.Errorf("cluster ID is not valid.  %v", err)
		return nil
	}

	partitionName := fmt.Sprintf("network_flows_v2_%s", strings.ReplaceAll(clusterID, "-", "_"))
	partitionCreate := `create table if not exists %s partition of network_flows_v2 
		for values in ('%s')`

	err = pgutils.Retry(func() error {
		_, err := db.Exec(context.Background(), fmt.Sprintf(partitionCreate, partitionName, clusterID))
		return err
	})
	if err != nil {
		log.Errorf("unable to create partition %q.  %v", partitionName, err)
		return nil
	}

	return &flowStoreImpl{
		db:            db,
		clusterID:     clusterUUID,
		partitionName: partitionName,
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
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
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

		if err := s.insertIntoNetworkflow(ctx, tx, s.clusterID, obj); err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
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
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

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

func (s *flowStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
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
	return flows, rows.Err()
}

// RemoveFlowsForDeployment removes all flows where the source OR destination match the deployment id
func (s *flowStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveFlowsByDeployment, "NetworkFlow")

	return pgutils.Retry(func() error {
		return s.retryableRemoveFlowsForDeployment(ctx, id)
	})
}

func (s *flowStoreImpl) retryableRemoveFlowsForDeployment(ctx context.Context, id string) error {
	// These remove operations can overlap.  Using a lock to avoid deadlocks in the database.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// To avoid a full scan with an OR delete source and destination flows separately
	err := s.removeDeploymentFlows(ctx, deleteSrcDeploymentStmt, id)
	if err != nil {
		return err
	}

	return s.removeDeploymentFlows(ctx, deleteDstDeploymentStmt, id)
}

func (s *flowStoreImpl) removeDeploymentFlows(ctx context.Context, deleteStmt string, id string) error {
	conn, release, err := s.acquireConn(ctx, ops.RemoveFlowsByDeployment, "NetworkFlow")
	if err != nil {
		return err
	}
	defer release()

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	// To avoid a full scan with an OR delete source and destination flows separately
	if _, err := tx.Exec(ctx, deleteStmt, s.clusterID, id); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
		}
		return err
	}

	return tx.Commit(ctx)
}

// GetAllFlows returns the object, if it exists from the store, timestamp and error
func (s *flowStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")

	return pgutils.Retry3(func() ([]*storage.NetworkFlow, *types.Timestamp, error) {
		return s.retryableGetAllFlows(ctx, since)
	})
}

func (s *flowStoreImpl) retryableGetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	var rows pgx.Rows
	var err error
	// Default to Now as that is when we are reading them
	lastUpdateTS := types.TimestampNow()

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		partitionWalkStmt := fmt.Sprintf(walkStmt, s.partitionName, s.partitionName)
		rows, err = s.db.Query(ctx, partitionWalkStmt)
	} else {
		partitionSinceStmt := fmt.Sprintf(getSinceStmt, s.partitionName, s.partitionName)
		rows, err = s.db.Query(ctx, partitionSinceStmt, pgutils.NilOrTime(since))
	}
	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}

	return flows, lastUpdateTS, nil
}

// GetMatchingFlows iterates over all of the objects in the store and applies the closure
func (s *flowStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlow")

	return pgutils.Retry3(func() ([]*storage.NetworkFlow, *types.Timestamp, error) {
		return s.retryableGetMatchingFlows(ctx, pred, since)
	})
}

func (s *flowStoreImpl) retryableGetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error) {
	var rows pgx.Rows
	var err error

	// Default to Now as that is when we are reading them
	lastUpdateTS := types.TimestampNow()

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		partitionWalkStmt := fmt.Sprintf(walkStmt, s.partitionName, s.partitionName)
		rows, err = s.db.Query(ctx, partitionWalkStmt)
	} else {
		partitionSinceStmt := fmt.Sprintf(getSinceStmt, s.partitionName, s.partitionName)
		rows, err = s.db.Query(ctx, partitionSinceStmt, pgutils.NilOrTime(since))
	}

	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, pred)
	if err != nil {
		return nil, nil, err
	}

	return flows, lastUpdateTS, nil
}

// GetFlowsForDeployment returns the flows matching the deployment ID
func (s *flowStoreImpl) GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetFlowsForDeployment, "NetworkFlow")

	return pgutils.Retry2(func() ([]*storage.NetworkFlow, error) {
		return s.retryableGetFlowsForDeployment(ctx, deploymentID)
	})
}
func (s *flowStoreImpl) retryableGetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	var rows pgx.Rows
	var err error

	partitionGetByDeploymentStmt := fmt.Sprintf(getByDeploymentStmt, s.partitionName, s.partitionName, s.partitionName, s.partitionName)
	rows, err = s.db.Query(ctx, partitionGetByDeploymentStmt, deploymentID)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil {
		return nil, err
	}

	return flows, nil
}

func (s *flowStoreImpl) delete(ctx context.Context, objs ...*storage.NetworkFlowProperties) error {
	// These remove operations can overlap.  Using a lock to avoid deadlocks in the database.
	s.mutex.Lock()
	defer s.mutex.Unlock()

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
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
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

	return pgutils.Retry(func() error {
		return s.delete(ctx, props)
	})
}

// RemoveOrphanedFlows removes flows that have been orphaned
func (s *flowStoreImpl) RemoveOrphanedFlows(ctx context.Context, orphanWindow *time.Time) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "NetworkFlow")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// To avoid a full scan with an OR delete source and destination flows separately
	pruneStmt := fmt.Sprintf(pruneNetworkFlowsSrcStmt, s.partitionName)
	err := s.pruneFlows(ctx, pruneStmt, orphanWindow)
	if err != nil {
		return err
	}

	pruneStmt = fmt.Sprintf(pruneNetworkFlowsDestStmt, s.partitionName)
	return s.pruneFlows(ctx, pruneStmt, orphanWindow)
}

func (s *flowStoreImpl) pruneFlows(ctx context.Context, deleteStmt string, orphanWindow *time.Time) error {
	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	if err != nil {
		return err
	}
	defer release()

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if _, err := conn.Exec(ctx, deleteStmt, s.clusterID, orphanWindow); err != nil {
		return err
	}

	return nil
}

// RemoveStaleFlows - remove stale duplicate network flows
func (s *flowStoreImpl) RemoveStaleFlows(ctx context.Context) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	// These remove operations can overlap.  Using a lock to avoid deadlocks in the database.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	conn, release, err := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	if err != nil {
		return err
	}
	defer release()

	// This is purposefully not retried as this is an optimization and not a requirement
	// It is also currently prone to statement timeouts
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()
	prune := fmt.Sprintf(pruneStaleNetworkFlowsStmt, s.partitionName, s.partitionName)
	_, err = conn.Exec(ctx, prune)

	return err
}

//// Used for testing

func dropTableNetworkflow(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS network_flows_v2 CASCADE")
}

// Destroy destroys the tables
func Destroy(ctx context.Context, db postgres.DB) {
	dropTableNetworkflow(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db postgres.DB, gormDB *gorm.DB, clusterID string) FlowStore {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, networkFlowsTable)
	return New(db, clusterID)
}
