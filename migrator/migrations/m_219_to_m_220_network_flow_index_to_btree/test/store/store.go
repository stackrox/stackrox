package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/sync"
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
	walkStmt           = "SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text FROM %s nf " + joinStmt

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type,
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		` WHERE (nf.LastSeenTimestamp >= $1 OR nf.LastSeenTimestamp IS NULL)`
	deleteSrcDeploymentStmt = "DELETE FROM network_flows_v2 WHERE ClusterId = $1 AND Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2"
	deleteDstDeploymentStmt = "DELETE FROM network_flows_v2 WHERE ClusterId = $1 AND Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2"

	getByDeploymentStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type,
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_SrcEntity_Type = 1 AND nf.Props_SrcEntity_Id = $1
	UNION ALL
	SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_DstEntity_Type = 1 AND nf.Props_DstEntity_Id = $1`

	getExternalByDeploymentStmt = `SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type,
	nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_SrcEntity_Type = 1 AND nf.Props_SrcEntity_Id = $1 AND nf.Props_DstEntity_Type = 4
	UNION ALL
	SELECT nf.Props_SrcEntity_Type, nf.Props_SrcEntity_Id, nf.Props_DstEntity_Type, nf.Props_DstEntity_Id, nf.Props_DstPort, nf.Props_L4Protocol, nf.LastSeenTimestamp, nf.UpdatedAt, nf.ClusterId::text
	FROM %s nf ` + joinStmt +
		`WHERE nf.Props_DstEntity_Type = 1 AND nf.Props_DstEntity_Id = $1 AND nf.Props_SrcEntity_Type = 4`

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
		AND UpdatedAt < $2`

	pruneNetworkFlowsDestStmt = `DELETE FROM %s child WHERE NOT EXISTS
		(SELECT 1 from deployments parent WHERE child.Props_DstEntity_Id = parent.id::text AND parent.clusterid = $1)
		AND Props_DstEntity_Type = 1
		AND UpdatedAt < $2`

	pruneNetworkFlowsReturnStmt = ` RETURNING Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type,
		Props_DstEntity_Id, Props_DstPort, Props_L4Protocol,
		LastSeenTimestamp, UpdatedAt, ClusterId::text;`

	// The idea behind this statement is to prune orphan external (discovered)
	// entities from the entities table. When flows are pruned using the above
	// statements, the returned deleted flows are used to construct a list of
	// deletion candidates for the external entities table, and then if any of
	// those are no longer referenced by a network flow, they are deleted.
	pruneOrphanExternalNetworkEntitiesStmt = `DELETE FROM network_entities entity
		WHERE entity.Info_Id = ANY($1) AND entity.Info_ExternalSource_Discovered = true
		AND
		NOT EXISTS
		(SELECT 1 FROM %s flow WHERE
			flow.Props_SrcEntity_Type = 4 AND flow.Props_SrcEntity_Id = entity.Info_Id)
		AND
		NOT EXISTS
		(SELECT 1 FROM %s flow WHERE
			flow.Props_DstEntity_Type = 4 AND flow.Props_DstEntity_Id = entity.Info_Id)
		RETURNING entity.Info_Id;`

	orphanedEntitiesPruningBatchSize = 100
)

var (
	log = logging.LoggerForModule()

	// We begin to process in batches after this number of records
	batchAfter = 100

	deleteTimeout = env.PostgresDefaultNetworkFlowDeleteTimeout.DurationSetting()

	queryTimeout = env.PostgresDefaultNetworkFlowQueryTimeout.DurationSetting()
)

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// UpsertFlows Same as other Upserts but it takes in a time
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
}

type flowStoreImpl struct {
	db             postgres.DB
	mutex          sync.Mutex
	clusterID      uuid.UUID
	partitionName  string
	networktreeMgr networktree.Manager
}

func (s *flowStoreImpl) insertIntoNetworkflow(ctx context.Context, tx *postgres.Tx, clusterID uuid.UUID, obj *storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {

	values := []interface{}{
		// parent primary keys start
		obj.GetProps().GetSrcEntity().GetType(),
		obj.GetProps().GetSrcEntity().GetId(),
		obj.GetProps().GetDstEntity().GetType(),
		obj.GetProps().GetDstEntity().GetId(),
		obj.GetProps().GetDstPort(),
		obj.GetProps().GetL4Protocol(),
		protocompat.NilOrTime(obj.GetLastSeenTimestamp()),
		clusterID,
		protocompat.NilOrNow(protoconv.ConvertMicroTSToProtobufTS(lastUpdateTS)),
	}

	finalStr := fmt.Sprintf("INSERT INTO %s (Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId, UpdatedAt) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)", s.partitionName)
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

func (s *flowStoreImpl) copyFromNetworkflow(ctx context.Context, tx *postgres.Tx, lastUpdateTS timestamp.MicroTS, objs ...*storage.NetworkFlow) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)
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
		"updatedat",
	}

	for idx, obj := range objs {
		inputRows = append(inputRows, []interface{}{
			obj.GetProps().GetSrcEntity().GetType(),
			obj.GetProps().GetSrcEntity().GetId(),
			obj.GetProps().GetDstEntity().GetType(),
			obj.GetProps().GetDstEntity().GetId(),
			obj.GetProps().GetDstPort(),
			obj.GetProps().GetL4Protocol(),
			protocompat.NilOrTime(obj.GetLastSeenTimestamp()),
			s.clusterID,
			protocompat.NilOrNow(protoconv.ConvertMicroTSToProtobufTS(lastUpdateTS)),
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
func New(db postgres.DB, clusterID string, networktreeMgr networktree.Manager) FlowStore {
	clusterUUID, err := uuid.FromString(clusterID)
	if err != nil {
		log.Errorf("cluster ID is not valid.  %v", err)
		return nil
	}

	partitionName := fmt.Sprintf("network_flows_v2_%s", strings.ReplaceAll(clusterID, "-", "_"))
	partitionCreate := `create table if not exists %s partition of network_flows_v2
		for values in ('%s')`

	ctx := context.Background()
	err = pgutils.Retry(ctx, func() error {
		_, err := db.Exec(ctx, fmt.Sprintf(partitionCreate, partitionName, clusterID))
		return err
	})
	if err != nil {
		log.Errorf("unable to create partition %q.  %v", partitionName, err)
		return nil
	}

	return &flowStoreImpl{
		db:             db,
		clusterID:      clusterUUID,
		partitionName:  partitionName,
		networktreeMgr: networktreeMgr,
	}
}

func (s *flowStoreImpl) copyFrom(ctx context.Context, lastUpdateTS timestamp.MicroTS, objs ...*storage.NetworkFlow) error {
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromNetworkflow(ctx, tx, lastUpdateTS, objs...); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
		}
		return err
	}
	return tx.Commit(ctx)
}

func (s *flowStoreImpl) upsert(ctx context.Context, lastUpdateTS timestamp.MicroTS, objs ...*storage.NetworkFlow) error {
	// Moved the transaction outside the loop which greatly improved the performance of these individual inserts.
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err := s.insertIntoNetworkflow(ctx, tx, s.clusterID, obj, lastUpdateTS); err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
			}
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *flowStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

	return pgutils.Retry(ctx, func() error {
		return s.retryableUpsertFlows(ctx, flows, lastUpdateTS)
	})
}

func (s *flowStoreImpl) retryableUpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	if lastUpdateTS <= 0 {
		lastUpdateTS = timestamp.Now()
	}
	// RocksDB implementation was adding the lastUpdatedTS to a key.  That is not necessary in PG world so that
	// parameter is not being passed forward and should be removed from the interface once RocksDB is removed.
	if len(flows) < batchAfter {
		return s.upsert(ctx, lastUpdateTS, flows...)
	}

	return s.copyFrom(ctx, lastUpdateTS, flows...)
}

func (s *flowStoreImpl) begin(ctx context.Context) (*postgres.Tx, context.Context, error) {
	return postgres.GetTransaction(ctx, s.db)
}
