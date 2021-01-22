package dackbox

import (
	"time"

	"github.com/stackrox/rox/central/metrics"
	edgeDackBox "github.com/stackrox/rox/central/nodecveedge/dackbox"
	"github.com/stackrox/rox/central/nodecveedge/store"
	"github.com/stackrox/rox/generated/storage"
	pkgBatcher "github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const (
	batchSize = 100

	typ = "NodeCVEEdge"
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
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
	}
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer dackTxn.Discard()

	exists, err := b.reader.ExistsIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count() (int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer dackTxn.Discard()

	count, err := b.reader.CountIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *storeImpl) GetAll() ([]*storage.NodeCVEEdge, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.NodeCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.NodeCVEEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (*storage.NodeCVEEdge, bool, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.NodeCVEEdge), true, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.NodeCVEEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	ret := make([]*storage.NodeCVEEdge, 0, len(ids))
	var missing []int
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			ret = append(ret, msg.(*storage.NodeCVEEdge))
		} else {
			missing = append(missing, idx)
		}
	}

	return ret, missing, nil
}

func (b *storeImpl) Upsert(objs ...*storage.NodeCVEEdge) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Upsert, typ)

	batcher := pkgBatcher.New(len(objs), batchSize)
	for {
		start, end, valid := batcher.Next()
		if !valid {
			break
		}
		if err := b.upsertBatch(objs[start:end]...); err != nil {
			return err
		}
	}
	return nil
}

func (b *storeImpl) Delete(ids ...string) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.RemoveMany, typ)

	batcher := pkgBatcher.New(len(ids), batchSize)
	for {
		start, end, valid := batcher.Next()
		if !valid {
			break
		}
		if err := b.deleteBatch(ids[start:end]...); err != nil {
			return err
		}
	}
	return nil
}

func (b *storeImpl) upsertBatch(objs ...*storage.NodeCVEEdge) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, obj := range objs {
		if err := b.upserter.UpsertIn(nil, obj, dackTxn); err != nil {
			return err
		}
	}

	return dackTxn.Commit()
}

func (b *storeImpl) deleteBatch(ids ...string) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, id := range ids {
		if err := b.deleter.DeleteIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn); err != nil {
			return err
		}
	}

	return dackTxn.Commit()
}
