package legacy

import (
	"bytes"
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/common"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/tecbot/gorocksdb"
)

type flowStoreImpl struct {
	db        *rocksdb.RocksDB
	keyPrefix []byte
}

var (
	updatedTSKey = []byte("\x00")

	readOptions  = generic.DefaultReadOptions()
	writeOptions = generic.DefaultWriteOptions()
)

// GetAllFlows returns all the flows in the store.
func (s *flowStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return nil, types.Timestamp{}, err
	}
	defer s.db.DecRocksDBInProgressOps()

	flows, ts, err = s.readFlows(nil, since)
	return flows, ts, err
}

func (s *flowStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlow")
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return nil, types.Timestamp{}, err
	}
	defer s.db.DecRocksDBInProgressOps()

	flows, ts, err = s.readFlows(pred, since)

	return flows, ts, err
}

func (s *flowStoreImpl) GetFlowsForDeployment(ctx context.Context, deploymentID string) ([]*storage.NetworkFlow, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlow")
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}
	defer s.db.DecRocksDBInProgressOps()

	// Function to match flows referencing the deployment ID passed in.
	pred := func(props *storage.NetworkFlowProperties) bool {
		srcEnt := props.GetSrcEntity()
		dstEnt := props.GetDstEntity()

		// Exclude all flows having both external endpoints. Although if one endpoint is an invisible external source,
		// we still want to show the flow given that the other endpoint is visible, however, attribute it to INTERNET.
		if networkgraph.AllExternal(srcEnt, dstEnt) {
			return false
		}

		srcMatch := srcEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && srcEnt.GetId() == deploymentID
		dstMatch := dstEnt.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && dstEnt.GetId() == deploymentID

		return srcMatch || dstMatch
	}

	flows, _, err := s.readFlows(pred, nil)

	return flows, err
}

// UpsertFlows updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.UpsertAll, "NetworkFlow")
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer s.db.DecRocksDBInProgressOps()

	tsData, err := lastUpdatedTS.GogoProtobuf().Marshal()
	if err != nil {
		return err
	}

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	// Add the timestamp key
	batch.Put(s.getFullKey(updatedTSKey), tsData)
	for _, flow := range flows {
		k := s.getID(flow.GetProps())
		v, err := proto.Marshal(flow)
		if err != nil {
			return err
		}
		batch.Put(k, v)
	}
	return s.db.Write(writeOptions, batch)
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer s.db.DecRocksDBInProgressOps()

	id := s.getID(props)

	return s.db.Delete(writeOptions, id)
}

func (s *flowStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer s.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	idBytes := []byte(id)
	err := generic.DefaultForEachOverKeySet(s.db, s.keyPrefix, true, func(k []byte) error {
		if bytes.Equal(k, updatedTSKey) {
			return nil
		}
		srcID, dstID := common.GetDeploymentIDsFromKey(k)
		if bytes.Equal(idBytes, srcID) || bytes.Equal(idBytes, dstID) {
			batch.Delete(s.getFullKey(k))
			return nil
		}
		return nil
	})
	if err != nil {
		return err
	}

	return s.db.Write(writeOptions, batch)
}

func (s *flowStoreImpl) RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.RemoveMany, "NetworkFlow")

	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer s.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	err := generic.DefaultForEachOverKeySet(s.db, s.keyPrefix, true, func(k []byte) error {
		if bytes.Equal(k, updatedTSKey) {
			return nil
		}
		props, err := common.ParseID(k)
		if err != nil {
			return err
		}
		if keyMatchFn != nil && !keyMatchFn(props) {
			return nil
		}
		// No need to read the flow if valueMatchFn is nil
		if valueMatchFn != nil {
			flow, err := s.readFlow(s.getFullKey(k))
			if err != nil {
				return err
			}
			if flow == nil {
				return nil
			}
			if !valueMatchFn(flow) {
				return nil
			}
		}
		batch.Delete(s.getFullKey(k))
		return nil
	})
	if err != nil {
		return err
	}
	return s.db.Write(writeOptions, batch)
}

func (s *flowStoreImpl) getFullKey(localKey []byte) []byte {
	result := make([]byte, 0, len(s.keyPrefix)+len(localKey))
	result = append(result, s.keyPrefix...)
	result = append(result, localKey...)
	return result
}

func (s *flowStoreImpl) getID(props *storage.NetworkFlowProperties) []byte {
	return s.getFullKey(common.GetID(props))
}

func (s *flowStoreImpl) readFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, lastUpdateTS types.Timestamp, err error) {
	// The entry for this should be present, but make sure we have a sane default if it is not
	lastUpdateTS = *types.TimestampNow()

	err = generic.DefaultForEachItemWithPrefix(s.db, s.keyPrefix, true, func(k, v []byte) error {
		if bytes.Equal(k, updatedTSKey) {
			return proto.Unmarshal(v, &lastUpdateTS)
		}
		if pred != nil {
			props, err := common.ParseID(k)
			if err != nil {
				return err
			}
			if !pred(props) {
				return nil
			}
		}

		flow := new(storage.NetworkFlow)
		if err := proto.Unmarshal(v, flow); err != nil {
			return err
		}
		if since != nil && flow.LastSeenTimestamp != nil {
			if flow.LastSeenTimestamp.Compare(since) < 0 {
				return nil
			}
		}
		flows = append(flows, flow)
		return nil
	})

	return
}

// Static helper functions.
/////////////////////////
func (s *flowStoreImpl) readFlow(id []byte) (flow *storage.NetworkFlow, err error) {
	slice, err := s.db.Get(readOptions, id)
	if err != nil {
		return nil, err
	}
	if !slice.Exists() {
		return nil, nil
	}
	defer slice.Free()
	flow = new(storage.NetworkFlow)
	if err := proto.Unmarshal(slice.Data(), flow); err != nil {
		return nil, err
	}
	return flow, nil
}
