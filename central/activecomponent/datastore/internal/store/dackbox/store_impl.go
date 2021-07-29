package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	acDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	dacky *dackbox.DackBox

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) store.Store {
	return &storeImpl{
		dacky:    dacky,
		reader:   acDackBox.Reader,
		upserter: acDackBox.Upserter,
		deleter:  acDackBox.Deleter,
	}
}

func (s *storeImpl) Exists(id string) (bool, error) {
	dackTxn, err := s.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer dackTxn.Discard()

	exists, err := s.reader.ExistsIn(acDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *storeImpl) Get(id string) (*storage.ActiveComponent, bool, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "ActiveComponent")

	dackTxn, err := s.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := s.reader.ReadIn(acDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ActiveComponent), msg != nil, err
}

func (s *storeImpl) GetBatch(ids []string) ([]*storage.ActiveComponent, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "ActiveComponent")

	dackTxn, err := s.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids))
	var missing []int
	for idx, id := range ids {
		msg, err := s.reader.ReadIn(acDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		} else {
			missing = append(missing, idx)
		}
	}

	ret := make([]*storage.ActiveComponent, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ActiveComponent))
	}

	return ret, missing, nil
}
