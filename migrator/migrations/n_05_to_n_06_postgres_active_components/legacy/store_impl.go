// This file was originally generated with
// //go:generate  cp ../../../../central/activecomponent/datastore/internal/store/postgres/store.go store_impl.go

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	acDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/activecomponent"
	deploymentDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/deployment"
	componentDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecomponent"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
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
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) Store {
	return &storeImpl{
		dacky:    dacky,
		keyFence: keyFence,
		reader:   acDackBox.Reader,
		upserter: acDackBox.Upserter,
		deleter:  acDackBox.Deleter,
	}
}

func (s *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.ActiveComponent, []int, error) {
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

func (s *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	txn, err := s.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	var ids []string
	err = txn.BucketKeyForEach(acDackBox.Bucket, true, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}

func (s *storeImpl) UpsertMany(_ context.Context, updates []*storage.ActiveComponent) error {
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

func (s *storeImpl) upsertActiveComponents(acs []*storage.ActiveComponent) error {
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
			err = s.upserter.UpsertIn(nil, ac, txn)
			if err != nil {
				return err
			}
			acKey := acDackBox.BucketHandler.GetKey(ac.GetId())
			g.AddRefs(deploymentDackBox.BucketHandler.GetKey(ac.GetDeploymentId()), acKey)
			g.AddRefs(acKey, componentDackBox.BucketHandler.GetKey(ac.GetComponentId()))
		}
		return txn.Commit()
	})
}

func gatherKeysForUpsert(acs []*storage.ActiveComponent) [][]byte {
	var allKeys [][]byte
	for _, ac := range acs {
		allKeys = append(allKeys,
			componentDackBox.BucketHandler.GetKey(ac.GetComponentId()),
			deploymentDackBox.BucketHandler.GetKey(ac.GetDeploymentId()),
			acDackBox.BucketHandler.GetKey(ac.GetId()),
		)
	}
	return allKeys
}
