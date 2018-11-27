package store

import (
	"bytes"
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowStoreImpl struct {
	flowsBucket bolthelper.BucketRef
}

const updatedTSKey = "\x00"

// GetAllFlows returns all the flows in the store.;
func (s *flowStoreImpl) GetAllFlows() (flows []*v1.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")

	err = s.flowsBucket.View(func(b *bolt.Bucket) error {
		var err error
		flows, ts, err = readAllFlows(b)
		return err
	})

	return flows, ts, err
}

// GetFlow returns the flow for the source and destination, or nil if none exists.
func (s *flowStoreImpl) GetFlow(props *v1.NetworkFlowProperties) (flow *v1.NetworkFlow, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	id := getID(flow.GetProps())

	err = s.flowsBucket.View(func(b *bolt.Bucket) error {
		flow, err = readFlow(b, id)
		if err != nil {
			return err
		}
		return nil
	})
	return
}

// UpsertFlow updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(flows []*v1.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
	tsData, err := protoconv.ConvertTimeToTimestamp(lastUpdatedTS.GoTime()).Marshal()
	if err != nil {
		return err
	}

	kvs := make([]bolthelper.KV, len(flows)+1)
	kvs[0] = bolthelper.KV{Key: []byte(updatedTSKey), Value: tsData}

	for i, flow := range flows {
		k := getID(flow.GetProps())
		v, err := proto.Marshal(flow)
		if err != nil {
			return err
		}
		kvs[i+1] = bolthelper.KV{Key: k, Value: v}
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.UpsertAll, "NetworkFlow")

	return s.flowsBucket.Update(func(b *bolt.Bucket) error {
		return bolthelper.PutAll(b, kvs...)
	})
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(props *v1.NetworkFlowProperties) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	id := getID(props)

	return s.flowsBucket.Update(func(b *bolt.Bucket) error {
		return b.Delete(id)
	})
}

// Static helper functions.
/////////////////////////

func readAllFlows(bucket *bolt.Bucket) (flows []*v1.NetworkFlow, lastUpdateTS types.Timestamp, err error) {
	err = bucket.ForEach(func(k, v []byte) error {
		if bytes.Equal(k, []byte(updatedTSKey)) {
			return proto.Unmarshal(v, &lastUpdateTS)
		}
		flow := new(v1.NetworkFlow)

		err = proto.Unmarshal(v, flow)
		if err != nil {
			return err
		}

		flows = append(flows, flow)
		return nil
	})
	return
}

func readFlow(bucket *bolt.Bucket, id []byte) (flow *v1.NetworkFlow, err error) {
	v := bucket.Get(id)
	if v == nil {
		return
	}

	flow = new(v1.NetworkFlow)
	err = proto.Unmarshal(v, flow)
	if err != nil {
		return nil, err
	}
	return
}

func getID(props *v1.NetworkFlowProperties) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d:%d", props.GetSrcDeploymentId(), props.GetDstDeploymentId(), props.GetDstPort(), props.GetL4Protocol()))
}
