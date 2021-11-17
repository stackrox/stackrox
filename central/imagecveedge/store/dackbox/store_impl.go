package dackbox

import (
	"time"

	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	imgDackBox "github.com/stackrox/rox/central/image/dackbox"
	edgeDackBox "github.com/stackrox/rox/central/imagecveedge/dackbox"
	"github.com/stackrox/rox/central/imagecveedge/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	ops "github.com/stackrox/rox/pkg/metrics"
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
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
		keyFence: keyFence,
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
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, "ImageCVEEdge")

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

func (b *storeImpl) GetAll() ([]*storage.ImageCVEEdge, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, "ImageCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ImageCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ImageCVEEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (cve *storage.ImageCVEEdge, exists bool, err error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "ImageCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ImageCVEEdge), true, err
}

func (b *storeImpl) GetMany(ids []string) ([]*storage.ImageCVEEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	ret := make([]*storage.ImageCVEEdge, 0, len(ids))
	var missing []int
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			ret = append(ret, msg.(*storage.ImageCVEEdge))
		} else {
			missing = append(missing, idx)
		}
	}

	return ret, missing, nil
}

func (b *storeImpl) UpdateVulnState(cve string, images []string, state storage.VulnerabilityState) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.UpdateMany, "ImageCVEEdge")

	edgeIDs := getEdgeIDs(cve, images...)
	graphKeys := gatherKeysForEdge(cve, images...)
	// Lock nodes in the graph and update the image-cve edge in the db.
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(graphKeys...), func() error {
		dackTxn, err := b.dacky.NewTransaction()
		if err != nil {
			return err
		}
		defer dackTxn.Discard()

		for _, edgeID := range edgeIDs {
			msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(edgeID), dackTxn)
			if err != nil {
				return err
			}
			if msg == nil {
				continue
			}
			edge := msg.(*storage.ImageCVEEdge)
			if edge.GetState() == state {
				continue
			}
			edge.State = state
			if err := b.upserter.UpsertIn(nil, edge, dackTxn); err != nil {
				return err
			}
		}
		return dackTxn.Commit()
	})
}

func getEdgeIDs(cve string, imageIDs ...string) []string {
	ids := make([]string, 0, len(imageIDs))
	for _, imgID := range imageIDs {
		ids = append(ids, edges.EdgeID{ParentID: imgID, ChildID: cve}.ToString())
	}
	return ids
}

func gatherKeysForEdge(cve string, imageIDs ...string) [][]byte {
	allKeys := make([][]byte, 0, len(imageIDs)+1)
	for _, imgID := range imageIDs {
		allKeys = append(allKeys, imgDackBox.BucketHandler.GetKey(imgID))
	}
	allKeys = append(allKeys, cveDackBox.BucketHandler.GetKey(cve))
	return allKeys
}
