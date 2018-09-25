package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pborman/uuid"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/data"
)

type flowStoreImpl struct {
	db *bolt.DB

	bucketName string
	bucketUUID uuid.UUID
}

// GetAllFlows returns all the flows in the store.;
func (s *flowStoreImpl) GetAllFlows() (flows []*data.NetworkFlow, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "NetworkFlowFlow")

	err = s.db.View(func(tx *bolt.Tx) error {
		flows, err = s.readAllFlows(tx)
		if err != nil {
			return err
		}
		return nil
	})
	return flows, err
}

// GetFlow returns the flow for the source and destination, or nil if none exists.
func (s *flowStoreImpl) GetFlow(props *data.NetworkFlowProperties) (flow *data.NetworkFlow, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkFlowFlow")

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

// AddFlow adds an flow to the store, returning an error if it is already present.
func (s *flowStoreImpl) AddFlow(flow *data.NetworkFlow) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "NetworkFlowFlow")

	id, err := s.getID(flow.GetProps())
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		if s.hasFlow(tx, id) {
			return fmt.Errorf("flow %s already exists", proto.MarshalTextString(flow.GetProps()))
		}
		return s.writeFlow(tx, id, flow)
	})
}

func (s *flowStoreImpl) UpdateFlow(flow *data.NetworkFlow) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "NetworkFlowFlow")

	id, err := s.getID(flow.GetProps())
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		if !s.hasFlow(tx, id) {
			return fmt.Errorf("flow %s does not exists", proto.MarshalTextString(flow.GetProps()))
		}
		return s.writeFlow(tx, id, flow)
	})
}

// UpsertFlow updates an flow to the store, adding it if not already present.
func (s *flowStoreImpl) UpsertFlow(flow *data.NetworkFlow) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "NetworkFlowFlow")

	id, err := s.getID(flow.GetProps())
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return s.writeFlow(tx, id, flow)
	})
}

// RemoveFlow removes an flow from the store if it is present.
func (s *flowStoreImpl) RemoveFlow(props *data.NetworkFlowProperties) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkFlowFlow")

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

func (s *flowStoreImpl) readAllFlows(tx *bolt.Tx) (flows []*data.NetworkFlow, err error) {
	bucket := tx.Bucket([]byte(s.bucketName))
	err = bucket.ForEach(func(k, v []byte) error {
		flow := new(data.NetworkFlow)

		err = proto.Unmarshal(v, flow)
		if err != nil {
			return err
		}

		flows = append(flows, flow)
		return nil
	})
	return
}

func (s *flowStoreImpl) readFlow(tx *bolt.Tx, id []byte) (flow *data.NetworkFlow, err error) {
	bucket := tx.Bucket([]byte(s.bucketName))

	v := bucket.Get(id)
	if v == nil {
		return
	}

	flow = new(data.NetworkFlow)
	err = proto.Unmarshal(v, flow)
	if err != nil {
		return nil, err
	}
	return
}

func (s *flowStoreImpl) hasFlow(tx *bolt.Tx, id []byte) bool {
	bucket := tx.Bucket([]byte(s.bucketName))

	bytes := bucket.Get(id)
	if bytes == nil {
		return false
	}
	return true
}

func (s *flowStoreImpl) writeFlow(tx *bolt.Tx, id []byte, flow *data.NetworkFlow) error {
	v, err := proto.Marshal(flow)
	if err != nil {
		return err
	}

	bucket := tx.Bucket([]byte(s.bucketName))
	bucket.Put(id, v)
	return nil
}

func (s *flowStoreImpl) removeFlow(tx *bolt.Tx, id []byte) error {
	bucket := tx.Bucket([]byte(s.bucketName))
	bucket.Delete(id)
	return nil
}

func (s *flowStoreImpl) getID(props *data.NetworkFlowProperties) (serialized []byte, err error) {
	var marshaledProps []byte
	marshaledProps, err = proto.Marshal(props)
	if err != nil {
		return
	}
	serialized, err = uuid.NewSHA1(s.bucketUUID, marshaledProps).MarshalText()
	return
}
