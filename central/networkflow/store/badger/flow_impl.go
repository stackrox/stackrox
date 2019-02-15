package badger

import (
	"bytes"
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	batchSize = 500
)

type flowStoreImpl struct {
	db        *badger.DB
	keyPrefix []byte
}

var updatedTSKey = []byte("\x00")

// GetAllFlows returns all the flows in the store.
func (s *flowStoreImpl) GetAllFlows(since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")
	flows, ts, err = s.readAllFlows(since)
	return flows, ts, err
}

// GetFlow returns the flow for the source and destination, or nil if none exists.
func (s *flowStoreImpl) GetFlow(props *storage.NetworkFlowProperties) (flow *storage.NetworkFlow, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	id := s.getID(flow.GetProps())

	err = s.db.View(func(tx *badger.Txn) error {
		flow, err = readFlow(tx, id)
		if err != nil {
			return err
		}
		return nil
	})
	return
}

// UpsertFlow updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(flows []*storage.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
	tsData, err := protoconv.ConvertTimeToTimestamp(lastUpdatedTS.GoTime()).Marshal()
	if err != nil {
		return err
	}

	kvs := make([]bolthelper.KV, 0, len(flows)+1)
	kvs = append(kvs, bolthelper.KV{Key: updatedTSKey, Value: tsData})

	for _, flow := range flows {
		k := s.getID(flow.GetProps())
		v, err := proto.Marshal(flow)
		if err != nil {
			return err
		}
		kvs = append(kvs, bolthelper.KV{Key: k, Value: v})
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.UpsertAll, "NetworkFlow")

	_, err = badgerhelper.PutAllBatched(s.db, kvs, batchSize)
	return err
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(props *storage.NetworkFlowProperties) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	id := s.getID(props)

	return s.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(id)
	})
}

func (s *flowStoreImpl) getID(props *storage.NetworkFlowProperties) []byte {
	return append(s.keyPrefix, []byte(fmt.Sprintf("%x:%s:%x:%s:%x:%x", props.GetSrcEntity().GetType(), props.GetSrcEntity().GetId(), props.GetDstEntity().GetType(), props.GetDstEntity().GetId(), props.GetDstPort(), props.GetL4Protocol()))...)
}

func (s *flowStoreImpl) readAllFlows(since *types.Timestamp) (flows []*storage.NetworkFlow, lastUpdateTS types.Timestamp, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		return badgerhelper.ForEachWithPrefix(txn, s.keyPrefix, badgerhelper.ForEachOptions{StripKeyPrefix: true},
			func(k, v []byte) error {
				if bytes.Equal(k, updatedTSKey) {
					return proto.Unmarshal(v, &lastUpdateTS)
				}

				flow := new(storage.NetworkFlow)
				if err := proto.Unmarshal(v, flow); err != nil {
					return err
				}
				if since != nil && protoconv.CompareProtoTimestamps(flow.LastSeenTimestamp, since) < 0 {
					return nil
				}
				flows = append(flows, flow)
				return nil
			})
	})
	return
}

// Static helper functions.
/////////////////////////

func readFlow(txn *badger.Txn, id []byte) (flow *storage.NetworkFlow, err error) {
	item, err := txn.Get(id)
	if err != nil {
		return nil, err
	}
	flow = new(storage.NetworkFlow)
	if err := badgerhelper.UnmarshalProtoValue(item, flow); err != nil {
		return nil, err
	}
	return flow, nil
}
