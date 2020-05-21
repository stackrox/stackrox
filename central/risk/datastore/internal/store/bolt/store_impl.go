package bolt

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	bucketName = []byte("risk")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.Risk).GetId())
}

func alloc() proto.Message {
	return new(storage.Risk)
}

// New creates a new risk store based on BoltDB
func New(db *bbolt.DB) (store.Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	return &storeImpl{crud: newCrud}, nil
}

func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Risk")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *storeImpl) Get(id string) (*storage.Risk, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Risk")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	risk := msg.(*storage.Risk)
	return risk, true, nil
}

func (s *storeImpl) GetMany(ids []string) ([]*storage.Risk, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Risk")
	msgs, missingIndices, err := s.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	storedKeys := make([]*storage.Risk, 0, len(msgs))
	for _, msg := range msgs {
		storedKeys = append(storedKeys, msg.(*storage.Risk))
	}
	return storedKeys, missingIndices, nil
}

func (s *storeImpl) Walk(fn func(risk *storage.Risk) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Risk")
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if err := fn(m.(*storage.Risk)); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) Upsert(risk *storage.Risk) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Risk")
	_, _, err := s.crud.Upsert(risk)
	return err
}
