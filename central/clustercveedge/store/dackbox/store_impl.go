package dackbox

import (
	"bytes"
	"time"

	"github.com/gogo/protobuf/proto"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	edgeDackBox "github.com/stackrox/rox/central/clustercveedge/dackbox"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/converter"
	vulnDackBox "github.com/stackrox/rox/central/cve/dackbox"
	"github.com/stackrox/rox/central/cve/utils"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
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
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (store.Store, error) {
	return &storeImpl{
		dacky:    dacky,
		keyFence: keyFence,
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
	}, nil
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
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, "ClusterCVEEdge")

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

func (b *storeImpl) GetAll() ([]*storage.ClusterCVEEdge, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, "ClusterCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ClusterCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ClusterCVEEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (edges *storage.ClusterCVEEdge, exists bool, err error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "ClusterCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ClusterCVEEdge), msg != nil, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.ClusterCVEEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "ClusterCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
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

	ret := make([]*storage.ClusterCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ClusterCVEEdge))
	}

	return ret, missing, nil
}

type clusterCVEEdge struct {
	CVE          string
	Edges        []edges.EdgeID
	ClusterIDSet set.StringSet
}

func (b *storeImpl) Upsert(parts ...converter.ClusterCVEParts) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Upsert, "CVE")

	keysToUpdate := gatherKeysForCVEParts(parts...)
	lockedKeySet := concurrency.DiscreteKeySet(keysToUpdate...)

	return b.keyFence.DoStatusWithLock(lockedKeySet, func() error {
		batch := batcher.New(len(parts), batchSize)
		for {
			start, end, ok := batch.Next()
			if !ok {
				break
			}

			if err := b.upsertNoBatch(parts[start:end]...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) upsertNoBatch(parts ...converter.ClusterCVEParts) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, clusterCVE := range parts {
		for _, child := range clusterCVE.Children {
			if err := edgeDackBox.Upserter.UpsertIn(nil, child.Edge, dackTxn); err != nil {
				return err
			}

			dackTxn.Graph().AddRefs(clusterDackBox.BucketHandler.GetKey(child.ClusterID), vulnDackBox.KeyFunc(clusterCVE.CVE))
		}

		currCVEMsg, err := vulnDackBox.Reader.ReadIn(vulnDackBox.BucketHandler.GetKey(clusterCVE.CVE.GetId()), dackTxn)
		if err != nil {
			return err
		}
		if currCVEMsg == nil {
			// Populate the types slice for the new CVE.
			clusterCVE.CVE.Types = []storage.CVE_CVEType{clusterCVE.CVE.GetType()}
		} else {
			currCVE := currCVEMsg.(*storage.CVE)
			clusterCVE.CVE.Suppressed = currCVE.GetSuppressed()
			clusterCVE.CVE.CreatedAt = currCVE.GetCreatedAt()
			clusterCVE.CVE.SuppressActivation = currCVE.GetSuppressActivation()
			clusterCVE.CVE.SuppressExpiry = currCVE.GetSuppressExpiry()

			clusterCVE.CVE.Types = utils.AddCVETypeIfAbsent(currCVE.GetTypes(), clusterCVE.CVE.GetType())
		}

		clusterCVE.CVE.Type = storage.CVE_UNKNOWN_CVE

		if err := vulnDackBox.Upserter.UpsertIn(nil, clusterCVE.CVE, dackTxn); err != nil {
			return err
		}
	}

	return dackTxn.Commit()
}

func (b *storeImpl) Delete(ids ...string) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.RemoveMany, "ClusterCVEEdge")

	parts, err := createClusterCVEEdges(ids)
	if err != nil {
		return err
	}
	batch := batcher.New(len(parts), batchSize)
	for {
		start, end, ok := batch.Next()
		if !ok {
			break
		}

		if err := b.deleteNoBatch(parts[start:end]...); err != nil {
			return err
		}
	}
	return nil
}

func (b *storeImpl) deleteNoBatch(parts ...clusterCVEEdge) error {
	var allKeys [][]byte
	for _, cve := range parts {
		for _, edge := range cve.Edges {
			allKeys = append(allKeys, vulnDackBox.BucketHandler.GetKey(cve.CVE), clusterDackBox.BucketHandler.GetKey(edge.ParentID), edgeDackBox.BucketHandler.GetKey(edge.ToString()))
		}
	}
	lockedKeySet := concurrency.DiscreteKeySet(allKeys...)
	return b.keyFence.DoStatusWithLock(lockedKeySet, func() error {
		dackTxn, err := b.dacky.NewTransaction()
		if err != nil {
			return err
		}
		defer dackTxn.Discard()

		graph := dackTxn.Graph()
		for _, part := range parts {
			cveKey := vulnDackBox.BucketHandler.GetKey(part.CVE)
			for _, edge := range part.Edges {
				if err := edgeDackBox.Deleter.DeleteIn(edgeDackBox.BucketHandler.GetKey(edge.ToString()), dackTxn); err != nil {
					return err
				}
			}

			refs := graph.GetRefsTo(cveKey)
			graph.DeleteRefsTo(cveKey)
			for _, ref := range refs {
				if bytes.HasPrefix(ref, clusterDackBox.BucketHandler.BucketPrefix) && part.ClusterIDSet.Contains(clusterDackBox.BucketHandler.GetID(ref)) {
					continue
				}
				graph.AddRefs(ref, cveKey)
			}
		}
		return dackTxn.Commit()
	})
}

func createClusterCVEEdges(ids []string) ([]clusterCVEEdge, error) {
	var parts []clusterCVEEdge
	cveIndexMap := make(map[string]int)
	for _, id := range ids {
		edge, err := edges.FromString(id)
		if err != nil {
			return nil, err
		}
		if _, ok := cveIndexMap[edge.ChildID]; !ok {
			cveIndexMap[edge.ChildID] = len(parts)
			parts = append(parts, clusterCVEEdge{CVE: edge.ChildID})
		}
		idx := cveIndexMap[edge.ChildID]

		parts[idx].Edges = append(parts[idx].Edges, edge)
		parts[idx].ClusterIDSet.Add(edge.ParentID)
	}
	return parts, nil
}

func gatherKeysForCVEParts(parts ...converter.ClusterCVEParts) [][]byte {
	var allKeys [][]byte
	for _, part := range parts {
		allKeys = append(allKeys, vulnDackBox.KeyFunc(part.CVE))
		for _, child := range part.Children {
			allKeys = append(allKeys, clusterDackBox.BucketHandler.GetKey(child.ClusterID))
		}
	}
	return sortedkeys.Sort(allKeys)
}
