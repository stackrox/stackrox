// This file was originally generated with
// //go:generate cp ../../../../central/networkgraph/flow/datastore/internal/store/rocksdb/flow_impl.go .

package legacy

import (
	"bytes"
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/common"
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
func (s *flowStoreImpl) GetAllFlows(_ context.Context, since *types.Timestamp) (flows []*storage.NetworkFlow, ts *types.Timestamp, err error) {
	if err := s.db.IncRocksDBInProgressOps(); err != nil {
		return nil, nil, err
	}
	defer s.db.DecRocksDBInProgressOps()

	flows, ts, err = s.readFlows(nil, since)
	return flows, ts, err
}

// UpsertFlows updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(_ context.Context, flows []*storage.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
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

func (s *flowStoreImpl) getFullKey(localKey []byte) []byte {
	result := make([]byte, 0, len(s.keyPrefix)+len(localKey))
	result = append(result, s.keyPrefix...)
	result = append(result, localKey...)
	return result
}

func (s *flowStoreImpl) getID(props *storage.NetworkFlowProperties) []byte {
	return s.getFullKey(common.GetID(props))
}

func (s *flowStoreImpl) readFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, lastUpdateTS *types.Timestamp, err error) {
	// The entry for this should be present, but make sure we have a sane default if it is not
	lastUpdateTS = types.TimestampNow()

	err = generic.DefaultForEachItemWithPrefix(s.db, s.keyPrefix, true, func(k, v []byte) error {
		if bytes.Equal(k, updatedTSKey) {
			return proto.Unmarshal(v, lastUpdateTS)
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
