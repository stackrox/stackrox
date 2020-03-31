package badger

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dbhelper"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowStoreImpl struct {
	db        *badger.DB
	keyPrefix []byte
}

var (
	updatedTSKey = []byte("\x00")
	idSeparator  = []byte(":")
)

// GetAllFlows returns all the flows in the store.
func (s *flowStoreImpl) GetAllFlows(since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")
	flows, ts, err = s.readFlows(nil, since)
	return flows, ts, err
}

func (s *flowStoreImpl) GetMatchingFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "NetworkFlow")
	flows, ts, err = s.readFlows(pred, since)
	return flows, ts, err
}

// GetFlow returns the flow for the source and destination, or nil if none exists.
func (s *flowStoreImpl) GetFlow(props *storage.NetworkFlowProperties) (flow *storage.NetworkFlow, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

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
	kvs = append(kvs, bolthelper.KV{Key: s.getFullKey(updatedTSKey), Value: tsData})

	for _, flow := range flows {
		k := s.getID(flow.GetProps())
		v, err := proto.Marshal(flow)
		if err != nil {
			return err
		}
		kvs = append(kvs, bolthelper.KV{Key: k, Value: v})
	}

	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.UpsertAll, "NetworkFlow")
	batch := s.db.NewWriteBatch()
	defer batch.Cancel()

	for _, kv := range kvs {
		if err := batch.Set(kv.Key, kv.Value); err != nil {
			return errors.Wrap(err, "error setting network flows")
		}
	}
	if err := batch.Flush(); err != nil {
		return errors.Wrap(err, "error flushing network flows")
	}
	return nil
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(props *storage.NetworkFlowProperties) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	id := s.getID(props)

	return badgerhelper.RetryableUpdate(s.db, func(tx *badger.Txn) error {
		return tx.Delete(id)
	})
}

func (s *flowStoreImpl) RemoveFlowsForDeployment(id string) error {
	var keysToDelete [][]byte

	idBytes := []byte(id)
	err := s.db.View(func(tx *badger.Txn) error {
		return badgerhelper.ForEachOverKeySet(tx, s.keyPrefix, badgerhelper.ForEachOptions{StripKeyPrefix: true, IteratorOptions: &badger.IteratorOptions{PrefetchValues: false}},
			func(k []byte) error {
				if bytes.Equal(k, updatedTSKey) {
					return nil
				}
				srcID, dstID := s.getDeploymentIDsFromKey(k)
				if bytes.Equal(idBytes, srcID) || bytes.Equal(idBytes, dstID) {
					keysToDelete = append(keysToDelete, s.getFullKey(k))
					return nil
				}
				return nil
			})
	})
	if err != nil {
		return err
	}
	batch := s.db.NewWriteBatch()
	defer batch.Cancel()
	for _, key := range keysToDelete {
		if err := batch.Delete(key); err != nil {
			return err
		}
	}
	return batch.Flush()
}

func (s *flowStoreImpl) RemoveMatchingFlows(keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	var keysToDelete [][]byte
	err := s.db.View(func(tx *badger.Txn) error {
		return badgerhelper.ForEachOverKeySet(tx, s.keyPrefix, badgerhelper.ForEachOptions{IteratorOptions: &badger.IteratorOptions{PrefetchValues: false}},
			func(k []byte) error {
				strippedKey := dbhelper.StripPrefix(s.keyPrefix, k)
				if bytes.Equal(strippedKey, updatedTSKey) {
					return nil
				}
				props, err := parseID(strippedKey)
				if err != nil {
					return err
				}
				if keyMatchFn != nil && !keyMatchFn(props) {
					return nil
				}
				// No need to read the flow if valueMatchFn is nil
				if valueMatchFn != nil {
					flow, err := readFlow(tx, k)
					if err != nil {
						return err
					}
					if !valueMatchFn(flow) {
						return nil
					}
				}

				keysToDelete = append(keysToDelete, sliceutils.ByteClone(k))
				return nil
			})
	})
	if err != nil {
		return err
	}
	batch := s.db.NewWriteBatch()
	defer batch.Cancel()
	for _, key := range keysToDelete {
		if err := batch.Delete(key); err != nil {
			return err
		}
	}
	return batch.Flush()
}

func (s *flowStoreImpl) getFullKey(localKey []byte) []byte {
	result := make([]byte, 0, len(s.keyPrefix)+len(localKey))
	result = append(result, s.keyPrefix...)
	result = append(result, localKey...)
	return result
}

func (s *flowStoreImpl) getID(props *storage.NetworkFlowProperties) []byte {
	return s.getFullKey(getID(props))
}

func (s *flowStoreImpl) getDeploymentIDsFromKey(id []byte) ([]byte, []byte) {
	bytesSlices := bytes.Split(id, idSeparator)
	return bytesSlices[1], bytesSlices[3]
}

func (s *flowStoreImpl) readFlows(pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) (flows []*storage.NetworkFlow, lastUpdateTS types.Timestamp, err error) {
	// The entry for this should be present, but make sure we have a sane default if it is not
	lastUpdateTS = *types.TimestampNow()

	iteratorOpts := badgerhelper.DefaultIteratorOptions()
	err = s.db.View(func(txn *badger.Txn) error {
		return badgerhelper.ForEachItemWithPrefix(txn, s.keyPrefix, badgerhelper.ForEachOptions{StripKeyPrefix: true, IteratorOptions: iteratorOpts},
			func(k []byte, item *badger.Item) error {
				if bytes.Equal(k, updatedTSKey) {
					return badgerhelper.UnmarshalProtoValue(item, &lastUpdateTS)
				}

				if pred != nil {
					props, err := parseID(k)
					if err != nil {
						return err
					}
					if !pred(props) {
						return nil
					}
				}

				flow := new(storage.NetworkFlow)
				if err := badgerhelper.UnmarshalProtoValue(item, flow); err != nil {
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
	})
	return
}

// Static helper functions.
/////////////////////////

func getID(props *storage.NetworkFlowProperties) []byte {
	return []byte(fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(props.GetSrcEntity().GetType()), props.GetSrcEntity().GetId(), int32(props.GetDstEntity().GetType()), props.GetDstEntity().GetId(), props.GetDstPort(), int32(props.GetL4Protocol())))
}

func parseID(id []byte) (*storage.NetworkFlowProperties, error) {
	parts := strings.Split(string(id), ":")
	if len(parts) != 6 {
		return nil, errors.Errorf("expected 6 parts when parsing network flow ID, got %d", len(parts))
	}

	srcType, err := strconv.ParseInt(parts[0], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing source type of network flow ID")
	}
	dstType, err := strconv.ParseInt(parts[2], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dest type of network flow ID")
	}
	dstPort, err := strconv.ParseUint(parts[4], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dest port of network flow ID")
	}
	l4proto, err := strconv.ParseInt(parts[5], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "parsing l4 proto of network flow ID")
	}

	result := &storage.NetworkFlowProperties{
		SrcEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_Type(srcType),
			Id:   parts[1],
		},
		DstEntity: &storage.NetworkEntityInfo{
			Type: storage.NetworkEntityInfo_Type(dstType),
			Id:   parts[3],
		},
		DstPort:    uint32(dstPort),
		L4Protocol: storage.L4Protocol(l4proto),
	}
	return result, nil
}

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
