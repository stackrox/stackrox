package updated

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
)

// This Flow is custom to match the existing interface and how the functionality works through the system.
// Basically for draft #1 we are trying to minimize the blast radius.
// There are many places to improve as we go, for instance, utilizing the DB for queries.  Currently, the service
// layer will pass in functions that loop through all the results and filter them out.  Many of these types of things
// can be handled via query and become much more efficient.  In order to really see the benefits of Postgres for
// this store, we will need to refactor how it is used.
const (
	countStmt = `SELECT COUNT(*) FROM network_flows_v2 WHERE ClusterId = $1`

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

	walkStmt = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text FROM network_flows_v2 nf " + joinStmt

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, 
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text 
	FROM network_flows_v2 nf ` + joinStmt +
		` WHERE (nf.LastSeenTimestamp >= $1 OR nf.LastSeenTimestamp IS NULL)`

	getByDeploymentStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, 
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text
	FROM network_flows_v2 nf ` + joinStmt +
		`WHERE nf.Props_SrcEntity_Type = 1 AND nf.Props_SrcEntity_Id = $1 AND nf.ClusterId = $2
	UNION ALL
	SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.ClusterId::text
	FROM network_flows_v2 nf ` + joinStmt +
		`WHERE nf.Props_DstEntity_Type = 1 AND nf.Props_DstEntity_Id = $1 AND nf.ClusterId = $2`

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
)

var (
	log = logging.LoggerForModule()
)

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// Count gets the total count of records for this cluster to support migration
	Count(ctx context.Context) (int, error)
	// GetAllFlows The methods below are the ones that match the flow interface which is what we probably have to match.
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, *types.Timestamp, error)
	// GetFlowsForDeployment returns all flows referencing a specific deployment id
	GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error)

	// RemoveStaleFlows - remove stale duplicate network flows
	RemoveStaleFlows(ctx context.Context) error

	// GetPartitionName - returns the partition name
	GetPartitionName() string
}

type flowStoreImpl struct {
	db            postgres.DB
	clusterID     uuid.UUID
	partitionName string
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

	_, err = db.Exec(context.Background(), fmt.Sprintf(partitionCreate, partitionName, clusterID))
	if err != nil {
		log.Info(err)
		panic("error creating table: " + partitionCreate)
	}

	return &flowStoreImpl{
		db:            db,
		clusterID:     clusterUUID,
		partitionName: partitionName,
	}
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
	return flows, rows.Err()
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
		partitionWalkStmt := fmt.Sprintf(walkStmt, s.partitionName)
		rows, err = s.db.Query(ctx, partitionWalkStmt)
	} else {
		partitionSinceStmt := fmt.Sprintf(getSinceStmt, s.partitionName)
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
		partitionWalkStmt := fmt.Sprintf(walkStmt, s.partitionName)
		rows, err = s.db.Query(ctx, partitionWalkStmt)
	} else {
		partitionSinceStmt := fmt.Sprintf(getSinceStmt, s.partitionName)
		rows, err = s.db.Query(ctx, partitionSinceStmt, pgutils.NilOrTime(since))
	}

	if err != nil {
		return nil, nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, pred)

	return flows, lastUpdateTS, err
}

// GetFlowsForDeployment returns the flows matching the deployment ID
func (s *flowStoreImpl) GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	return pgutils.Retry2(func() ([]*storage.NetworkFlow, error) {
		return s.retryableGetFlowsForDeployment(ctx, deploymentID)
	})
}
func (s *flowStoreImpl) retryableGetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	var rows pgx.Rows
	var err error

	partitionDeploymentDeleteStmt := fmt.Sprintf(getByDeploymentStmt, s.partitionName, s.partitionName)
	rows, err = s.db.Query(ctx, partitionDeploymentDeleteStmt, deploymentID, s.clusterID)

	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)

	return flows, err
}

// Count gets the total count of records for this cluster to support migration
func (s *flowStoreImpl) Count(ctx context.Context) (int, error) {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	row := conn.QueryRow(ctx, countStmt, s.clusterID)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Errorf("error fetching database version: %v", err)
		return 0, err
	}

	return count, nil
}

// RemoveStaleFlows - remove stale duplicate network flows
func (s *flowStoreImpl) RemoveStaleFlows(ctx context.Context) error {
	conn, release, err := s.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer release()

	// This is purposefully not retried as this is an optimization and not a requirement
	// It is also currently prone to statement timeouts
	prune := fmt.Sprintf(pruneStaleNetworkFlowsStmt, s.partitionName, s.partitionName)
	_, err = conn.Exec(ctx, prune)
	return err
}

// GetPartitionName - returns the partition name
func (s *flowStoreImpl) GetPartitionName() string {
	return s.partitionName
}

//// Used for testing

func dropTableNetworkflow(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS network_flows_v2 CASCADE")
}

// Destroy destroys the tables
func Destroy(ctx context.Context, db postgres.DB) {
	dropTableNetworkflow(ctx, db)
}
