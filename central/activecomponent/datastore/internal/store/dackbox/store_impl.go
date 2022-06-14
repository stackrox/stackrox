package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	acConverter "github.com/stackrox/rox/central/activecomponent/converter"
	acDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const (
	batchSize = 5000
)

type storeImpl struct {
	dacky    *dackbox.DackBox
	keyFence concurrency.KeyFence

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) store.Store {
	return &storeImpl{
		dacky:    dacky,
		keyFence: keyFence,
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

func (s *storeImpl) UpsertBatch(updates []*acConverter.CompleteActiveComponent) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.UpsertAll, "ActiveComponent")
	batch := batcher.New(len(updates), batchSize)
	for {
		start, end, ok := batch.Next()
		if !ok {
			break
		}

		if err := s.upsertActiveComponents(updates[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) upsertActiveComponents(acs []*acConverter.CompleteActiveComponent) error {
	keysToUpsert := gatherKeysForUpsert(acs)
	keysToLock := concurrency.DiscreteKeySet(keysToUpsert...)
	return s.keyFence.DoStatusWithLock(keysToLock, func() error {
		txn, err := s.dacky.NewTransaction()
		if err != nil {
			return err
		}
		defer txn.Discard()

		g := txn.Graph()
		for _, ac := range acs {
			err = s.upserter.UpsertIn(nil, ac.ActiveComponent, txn)
			if err != nil {
				return err
			}
			acKey := acDackBox.BucketHandler.GetKey(ac.ActiveComponent.GetId())
			g.AddRefs(deploymentDackBox.BucketHandler.GetKey(ac.DeploymentID), acKey)
			g.AddRefs(acKey, componentDackBox.BucketHandler.GetKey(ac.ComponentID))
		}
		return txn.Commit()
	})
}

func (s *storeImpl) DeleteBatch(ids ...string) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.RemoveMany, "ActiveComponent")

	keysToDelete := acDackBox.BucketHandler.GetKeys(ids...)
	keysToLock := concurrency.DiscreteKeySet(keysToDelete...)
	return s.keyFence.DoStatusWithLock(keysToLock, func() error {
		batch := batcher.New(len(keysToDelete), batchSize)
		for {
			start, end, ok := batch.Next()
			if !ok {
				break
			}

			if err := s.deleteNoBatch(keysToDelete[start:end]...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *storeImpl) deleteNoBatch(keys ...[]byte) error {
	dackTxn, err := s.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, key := range keys {
		err := acDackBox.Deleter.DeleteIn(key, dackTxn)
		if err != nil {
			return err
		}
	}

	return dackTxn.Commit()
}

func gatherKeysForUpsert(acs []*acConverter.CompleteActiveComponent) [][]byte {
	var allKeys [][]byte
	for _, ac := range acs {
		allKeys = append(allKeys,
			componentDackBox.BucketHandler.GetKey(ac.ComponentID),
			deploymentDackBox.BucketHandler.GetKey(ac.DeploymentID),
			acDackBox.BucketHandler.GetKey(ac.ActiveComponent.GetId()),
		)
	}
	return allKeys
}
