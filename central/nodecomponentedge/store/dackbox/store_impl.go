package dackbox

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/metrics"
	edgeDackBox "github.com/stackrox/stackrox/central/nodecomponentedge/dackbox"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	ops "github.com/stackrox/stackrox/pkg/metrics"
)

const (
	typ = "NodeComponentEdge"
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

func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
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

func (b *storeImpl) Count(_ context.Context) (int, error) {
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

func (b *storeImpl) GetAll(_ context.Context) ([]*storage.NodeComponentEdge, error) {
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
	ret := make([]*storage.NodeComponentEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.NodeComponentEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(_ context.Context, id string) (*storage.NodeComponentEdge, bool, error) {
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

	return msg.(*storage.NodeComponentEdge), true, err
}

func (b *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.NodeComponentEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids))
	var missing []int
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

	ret := make([]*storage.NodeComponentEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.NodeComponentEdge))
	}

	return ret, missing, nil
}
