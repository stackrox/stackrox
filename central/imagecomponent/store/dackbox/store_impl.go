package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/central/imagecomponent/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const batchSize = 100

type storeImpl struct {
	keyFence concurrency.KeyFence
	dacky    *dackbox.DackBox
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (store.Store, error) {
	return &storeImpl{
		keyFence: keyFence,
		dacky:    dacky,
	}, nil
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	exists, err := componentDackBox.Reader.ExistsIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) GetAll() ([]*storage.ImageComponent, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs, err := componentDackBox.Reader.ReadAllIn(componentDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ImageComponent, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ImageComponent))
	}

	return ret, nil
}

func (b *storeImpl) Count() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	count, err := componentDackBox.Reader.CountIn(componentDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetImage returns image with given id.
func (b *storeImpl) Get(id string) (image *storage.ImageComponent, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msg, err := componentDackBox.Reader.ReadIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ImageComponent), msg != nil, err
}

// GetImagesBatch returns image with given sha.
func (b *storeImpl) GetBatch(ids []string) ([]*storage.ImageComponent, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids))
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := componentDackBox.Reader.ReadIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		} else {
			missing = append(missing, idx)
		}
	}

	ret := make([]*storage.ImageComponent, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ImageComponent))
	}

	return ret, missing, nil
}

func (b *storeImpl) Upsert(components ...*storage.ImageComponent) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.UpsertAll, "Image Component")

	keysToUpsert := make([][]byte, 0, len(components))
	for _, component := range components {
		keysToUpsert = append(keysToUpsert, componentDackBox.KeyFunc(component))
	}
	lockedKeySet := concurrency.DiscreteKeySet(keysToUpsert...)

	return b.keyFence.DoStatusWithLock(lockedKeySet, func() error {
		batch := batcher.New(len(components), batchSize)
		for {
			start, end, ok := batch.Next()
			if !ok {
				break
			}

			if err := b.upsertNoBatch(components[start:end]...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) upsertNoBatch(components ...*storage.ImageComponent) error {
	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	for _, component := range components {
		err := componentDackBox.Upserter.UpsertIn(nil, component, dackTxn)
		if err != nil {
			return err
		}
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return nil
}

func (b *storeImpl) Delete(ids ...string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.RemoveMany, "Image Component")

	keysToUpsert := make([][]byte, 0, len(ids))
	for _, id := range ids {
		keysToUpsert = append(keysToUpsert, componentDackBox.BucketHandler.GetKey(id))
	}
	lockedKeySet := concurrency.DiscreteKeySet(keysToUpsert...)

	return b.keyFence.DoStatusWithLock(lockedKeySet, func() error {
		batch := batcher.New(len(ids), batchSize)
		for {
			start, end, ok := batch.Next()
			if !ok {
				break
			}

			if err := b.deleteNoBatch(ids[start:end]...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) deleteNoBatch(ids ...string) error {
	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	for _, id := range ids {
		err := componentDackBox.Deleter.DeleteIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return err
		}
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return nil
}
