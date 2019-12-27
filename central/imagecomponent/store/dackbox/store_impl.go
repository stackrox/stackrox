package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/central/imagecomponent/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const batchSize = 100

type storeImpl struct {
	counter *crud.TxnCounter
	dacky   *dackbox.DackBox

	reader     crud.Reader
	listReader crud.Reader
	upserter   crud.Upserter
	deleter    crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) (store.Store, error) {
	counter, err := crud.NewTxnCounter(dacky, componentDackBox.Bucket)
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		counter:  counter,
		dacky:    dacky,
		reader:   componentDackBox.Reader,
		upserter: componentDackBox.Upserter,
		deleter:  componentDackBox.Deleter,
	}, nil
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	exists, err := b.listReader.ExistsIn(badgerhelper.GetBucketKey(componentDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) GetAll() ([]*storage.ImageComponent, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(componentDackBox.Bucket, dackTxn)
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

	count, err := b.reader.CountIn(componentDackBox.Bucket, dackTxn)
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

	msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(componentDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return nil, false, err
	}

	return msg.(*storage.ImageComponent), msg != nil, err
}

// GetImagesBatch returns image with given sha.
func (b *storeImpl) GetBatch(ids []string) ([]*storage.ImageComponent, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Image Component")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids)/2)
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(componentDackBox.Bucket, []byte(id)), dackTxn)
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

func (b *storeImpl) Upsert(image *storage.ImageComponent) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "Image Component")

	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	err := b.upserter.UpsertIn(nil, image, dackTxn)
	if err != nil {
		return err
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) UpsertBatch(components []*storage.ImageComponent) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.UpsertAll, "Image Component")

	for batch := 0; batch < len(components); batch += batchSize {
		dackTxn := b.dacky.NewTransaction()
		defer dackTxn.Discard()

		for idx := batch; idx < len(components) && idx < batch+batchSize; idx++ {
			err := b.upserter.UpsertIn(nil, components[idx], dackTxn)
			if err != nil {
				return err
			}
		}

		if err := dackTxn.Commit(); err != nil {
			return err
		}
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Image Component")

	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(componentDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return err
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) DeleteBatch(ids []string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.RemoveMany, "Image Component")

	for batch := 0; batch < len(ids); batch += batchSize {
		dackTxn := b.dacky.NewTransaction()
		defer dackTxn.Discard()

		for idx := batch; idx < len(ids) && idx < batch+batchSize; idx++ {
			err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(componentDackBox.Bucket, []byte(ids[idx])), dackTxn)
			if err != nil {
				return err
			}
		}

		if err := dackTxn.Commit(); err != nil {
			return err
		}
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return b.counter.GetTxnCount(), nil
}

func (b *storeImpl) IncTxnCount() error {
	return b.counter.IncTxnCount()
}
