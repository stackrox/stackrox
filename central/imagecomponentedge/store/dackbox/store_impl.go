package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	edgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	"github.com/stackrox/rox/central/imagecomponentedge/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const batchSize = 100

type storeImpl struct {
	dacky *dackbox.DackBox

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) (store.Store, error) {
	return &storeImpl{
		dacky:    dacky,
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
	}, nil
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	exists, err := b.reader.ExistsIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "ImageComponentEdge")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	count, err := b.reader.CountIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *storeImpl) GetAll() ([]*storage.ImageComponentEdge, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "ImageComponentEdge")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ImageComponentEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ImageComponentEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (cve *storage.ImageComponentEdge, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ImageComponentEdge")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ImageComponentEdge), msg != nil, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.ImageComponentEdge, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ImageComponentEdge")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids)/2)
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		} else {
			missing = append(missing, idx)
		}
	}

	ret := make([]*storage.ImageComponentEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ImageComponentEdge))
	}

	return ret, missing, nil
}

func (b *storeImpl) Upsert(cves ...*storage.ImageComponentEdge) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "ImageComponentEdge")

	for batch := 0; batch < len(cves); batch += batchSize {
		dackTxn := b.dacky.NewTransaction()
		defer dackTxn.Discard()

		for idx := batch; idx < len(cves) && idx < batch+batchSize; idx++ {
			err := b.upserter.UpsertIn(nil, cves[idx], dackTxn)
			if err != nil {
				return err
			}
		}

		if err := dackTxn.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (b *storeImpl) Delete(ids ...string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.RemoveMany, "ImageComponentEdge")

	for batch := 0; batch < len(ids); batch += batchSize {
		dackTxn := b.dacky.NewTransaction()
		defer dackTxn.Discard()

		for idx := batch; idx < len(ids) && idx < batch+batchSize; idx++ {
			err := b.deleter.DeleteIn(edgeDackBox.BucketHandler.GetKey(ids[idx]), dackTxn)
			if err != nil {
				return err
			}
		}

		if err := dackTxn.Commit(); err != nil {
			return err
		}
	}
	return nil
}
