package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pborman/uuid"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowStoreImpl struct {
	db *bolt.DB

	bucketName         string
	updateTSBucketName string
	bucketUUID         uuid.UUID
}

// GetAllFlows returns all the flows in the store.;
func (s *flowStoreImpl) GetAllFlows() (flows []*v1.NetworkFlow, ts types.Timestamp, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlow")

	err = s.db.View(func(tx *bolt.Tx) error {
		flows, err = s.readAllFlows(tx)
		if err != nil {
			return err
		}
		bucket := tx.Bucket([]byte(s.updateTSBucketName))

		bytes := bucket.Get([]byte(s.bucketUUID))
		if bytes == nil {
			return fmt.Errorf("unable to get last update timestamp for flows from cluster %s", s.bucketUUID)
		}
		err = ts.Unmarshal(bytes)
		if err != nil {
			return err
		}
		return nil
	})

	return flows, ts, err
}

// GetFlow returns the flow for the source and destination, or nil if none exists.
func (s *flowStoreImpl) GetFlow(props *v1.NetworkFlowProperties) (flow *v1.NetworkFlow, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	id, err := s.getID(flow.GetProps())
	if err != nil {
		return nil, err
	}

	err = s.db.View(func(tx *bolt.Tx) error {
		flow, err = s.readFlow(tx, id)
		if err != nil {
			return err
		}
		return nil
	})
	return
}

// UpsertFlow updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlows(flows []*v1.NetworkFlow, lastUpdatedTS timestamp.MicroTS) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.UpsertAll, "NetworkFlow")

	return s.db.Update(func(tx *bolt.Tx) error {
		t, err := protoconv.ConvertTimeToTimestamp(lastUpdatedTS.GoTime()).Marshal()
		if err != nil {
			return err
		}
		bucket := tx.Bucket([]byte(s.updateTSBucketName))
		bucket.Put([]byte(s.bucketUUID), t)

		return s.writeFlows(tx, flows...)

	})
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(props *v1.NetworkFlowProperties) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	id, err := s.getID(props)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return s.removeFlow(tx, id)
	})
}

// Member helper functions.
/////////////////////////

func (s *flowStoreImpl) readAllFlows(tx *bolt.Tx) (flows []*v1.NetworkFlow, err error) {

	bucket := tx.Bucket([]byte(s.bucketName))
	err = bucket.ForEach(func(k, v []byte) error {
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

func (s *flowStoreImpl) readFlow(tx *bolt.Tx, id []byte) (flow *v1.NetworkFlow, err error) {
	bucket := tx.Bucket([]byte(s.bucketName))

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

func (s *flowStoreImpl) writeFlows(tx *bolt.Tx, flows ...*v1.NetworkFlow) error {
	bucket := tx.Bucket([]byte(s.bucketName))

	for _, flow := range flows {
		id, err := s.getID(flow.GetProps())
		if err != nil {
			return err
		}

		v, err := proto.Marshal(flow)
		if err != nil {
			return err
		}

		bucket.Put(id, v)
	}
	return nil
}

func (s *flowStoreImpl) removeFlow(tx *bolt.Tx, id []byte) error {
	bucket := tx.Bucket([]byte(s.bucketName))
	bucket.Delete(id)
	return nil
}

func (s *flowStoreImpl) getID(props *v1.NetworkFlowProperties) (serialized []byte, err error) {
	var marshaledProps []byte
	marshaledProps, err = proto.Marshal(props)
	if err != nil {
		return
	}
	serialized, err = uuid.NewSHA1(s.bucketUUID, marshaledProps).MarshalText()
	return
}
