package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()

	// We begin to process in batches after this number of records
	batchAfter = 100
)

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// UpsertFlows Same as other Upserts but it takes in a time
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
}

type flowStoreImpl struct {
	db            postgres.DB
	clusterID     uuid.UUID
	partitionName string
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
func New(db postgres.DB, clusterID string) FlowStore {
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
		db:            db,
		clusterID:     clusterUUID,
		partitionName: partitionName,
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
