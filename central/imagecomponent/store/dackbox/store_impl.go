package dackbox

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	componentDackBox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	"github.com/stackrox/stackrox/central/imagecomponent/store"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	ops "github.com/stackrox/stackrox/pkg/metrics"
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

func (b *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer dackTxn.Discard()

	exists, err := componentDackBox.Reader.ExistsIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, "Image Component")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer dackTxn.Discard()

	count, err := componentDackBox.Reader.CountIn(componentDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetImage returns image with given id.
func (b *storeImpl) Get(ctx context.Context, id string) (image *storage.ImageComponent, exists bool, err error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "Image Component")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := componentDackBox.Reader.ReadIn(componentDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ImageComponent), msg != nil, err
}

// GetImagesBatch returns image with given sha.
func (b *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ImageComponent, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "Image Component")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
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
