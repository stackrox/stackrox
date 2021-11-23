package postgres

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	batchInsertTemplate = "insert into networkflows (id, clusterid, value) values %s on conflict(id) do update set value = EXCLUDED.value"
)

var (
	marshaler = &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true}

	log = logging.LoggerForModule()
)

type flowStoreImpl struct {
	db        *pgxpool.Pool
	clusterID string
}

// GetAllFlows returns all the flows in the store.
func (s *flowStoreImpl) GetAllFlows(since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")

	flows, ts, err = s.readFlows(nil, since)
	return flows, ts, err
}

func (s *flowStoreImpl) GetMatchingFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlow")
	flows, ts, err = s.readFlows(pred, since)
	return flows, ts, err
}

// UpsertFlow updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(flows []*storage.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
	if len(flows) == 0 {
		return nil
	}

	defer func(now time.Time) {
		log.Infof("Upserting: %d flows in batch - %d ms", len(flows), time.Since(now).Milliseconds())
	}(time.Now())

	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.AddMany, "NetworkFlowProperties")
	numElems := 3
	batch := batcher.New(len(flows), 60000/numElems)
	for start, end, ok := batch.Next(); ok; start, end, ok = batch.Next() {
		var placeholderStr string
		data := make([]interface{}, 0, numElems*len(flows))
		for i, obj := range flows[start:end] {
			if i != 0 {
				placeholderStr += ", "
			}
			placeholderStr += postgres.GetValues(i*numElems+1, (i+1)*numElems+1)
			t := time.Now()
			value, err := marshaler.MarshalToString(obj.GetProps())
			if err != nil {
				return err
			}
			metrics.SetJSONPBOperationDurationTime(t, "Marshal", "NetworkFlowProperties")
			id := common.GetIDString(obj.GetProps())
			data = append(data, id, s.clusterID, value)
		}
		if _, err := s.db.Exec(context.Background(), fmt.Sprintf(batchInsertTemplate, placeholderStr), data...); err != nil {
			return err
		}
	}
	return nil
}

func (s *flowStoreImpl) RemoveFlowsForDeployment(id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "NetworkFlowProperties")
	_, err := s.db.Exec(context.Background(), "delete from networkflows where value->'dstEntity'->>'id' = $1 or value->'srcEntity'->>'id' = $1", id)
	return err
}

func (s *flowStoreImpl) RemoveMatchingFlows(keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	// TODO needs to be cleaned up for pruning
	return nil
}

func (s *flowStoreImpl) readFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, lastUpdateTS types.Timestamp, err error) {
	// The entry for this should be present, but make sure we have a sane default if it is not
	lastUpdateTS = *types.TimestampNow()

	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlowProperties")

	query := "select value from networkflows where clusterid = $1"
	rows, err := s.db.Query(context.Background(), query, s.clusterID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, types.Timestamp{}, nil
		}
		return nil, types.Timestamp{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, types.Timestamp{}, err
		}
		buf := bytes.NewBuffer(data)
		var flow storage.NetworkFlow
		if err := jsonpb.Unmarshal(buf, &flow); err != nil {
			return nil, types.Timestamp{}, err
		}
		if pred(flow.GetProps()) {
			flows = append(flows, &flow)
		}
	}
	return
}
