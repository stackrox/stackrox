package bolt

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	bucketName = []byte("processWhitelistResults")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.ProcessWhitelistResults).GetDeploymentId())
}

func alloc() proto.Message {
	return new(storage.ProcessWhitelistResults)
}

// NewBoltStore returns the bolt store for process whitelist results
func NewBoltStore(db *bbolt.DB) (store.Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	return &storeImpl{crud: newCrud}, nil
}

func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "WhitelistResults")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *storeImpl) Get(id string) (*storage.ProcessWhitelistResults, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "WhitelistResults")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	whitelistresults := msg.(*storage.ProcessWhitelistResults)
	return whitelistresults, true, nil
}

func (s *storeImpl) Upsert(whitelistresults *storage.ProcessWhitelistResults) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "WhitelistResults")
	_, _, err := s.crud.Upsert(whitelistresults)
	return err
}
