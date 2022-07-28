// This file was originally generated with
// //go:generate cp ../../../../central/cve/store/dackbox/store_impl.go .

package legacy

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	vulnDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/cve"
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
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) Store {
	return &storeImpl{
		keyFence: keyFence,
		dacky:    dacky,
	}
}

func (b *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer dackTxn.Discard()

	exists, err := vulnDackBox.Reader.ExistsIn(vulnDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, "CVE")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer dackTxn.Discard()

	count, err := vulnDackBox.Reader.CountIn(vulnDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *storeImpl) Get(ctx context.Context, id string) (cve *storage.CVE, exists bool, err error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "CVE")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := vulnDackBox.Reader.ReadIn(vulnDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.CVE), true, err
}

func (b *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ClusterCVE, []int, error) {
	cves, missing, err := b.getMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	clusterCVEs := make([]*storage.ClusterCVE, 0, len(cves))
	for _, cve := range cves {
		clusterCVEs = append(clusterCVEs, convertCVEToClusterCVE(cve))
	}
	return clusterCVEs, missing, nil
}

func convert2(cve *storage.ClusterCVE) *storage.CVE {
	return &storage.CVE{}
}

func (b *storeImpl) getMany(ctx context.Context, ids []string) ([]*storage.CVE, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "CVE")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids))
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := vulnDackBox.Reader.ReadIn(vulnDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		} else {
			missing = append(missing, idx)
		}
	}

	ret := make([]*storage.CVE, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.CVE))
	}

	return ret, missing, nil
}

// GetIDs returns the keys of all cves stored in RocksDB.
func (b *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	var ids []string
	err = txn.BucketKeyForEach(vulnDackBox.Bucket, true, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}

func (b *storeImpl) Upsert(ctx context.Context, cves ...*storage.CVE) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Upsert, "CVE")

	keysToUpsert := make([][]byte, 0, len(cves))
	for _, vuln := range cves {
		keysToUpsert = append(keysToUpsert, vulnDackBox.KeyFunc(vuln))
	}
	lockedKeySet := concurrency.DiscreteKeySet(keysToUpsert...)

	return b.keyFence.DoStatusWithLock(lockedKeySet, func() error {
		batch := batcher.New(len(cves), batchSize)
		for {
			start, end, ok := batch.Next()
			if !ok {
				break
			}

			if err := b.upsertNoBatch(cves[start:end]...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) UpsertMany(ctx context.Context, clusterCves []*storage.ClusterCVE) error {
	cves := make([]*storage.CVE, 0, len(clusterCves))
	for _, clusterCve := range clusterCves {
		cves = append(cves, convert2(clusterCve))
	}
	return b.Upsert(ctx, cves...)
}

func (b *storeImpl) upsertNoBatch(cves ...*storage.CVE) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, cve := range cves {
		err := vulnDackBox.Upserter.UpsertIn(nil, cve, dackTxn)
		if err != nil {
			return err
		}
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return nil
}

func (b *storeImpl) Delete(ctx context.Context, ids ...string) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.RemoveMany, "CVE")

	keysToUpsert := make([][]byte, 0, len(ids))
	for _, id := range ids {
		keysToUpsert = append(keysToUpsert, vulnDackBox.BucketHandler.GetKey(id))
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
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	for _, id := range ids {
		if err := vulnDackBox.Deleter.DeleteIn(vulnDackBox.BucketHandler.GetKey(id), dackTxn); err != nil {
			return err
		}
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return nil
}

func convertCVEToClusterCVE(cve *storage.CVE) *storage.ClusterCVE {
	return &storage.ClusterCVE{
		Id: cve.GetId(),
		CveBaseInfo: &storage.CVEInfo{
			Cve:          cve.GetId(),
			Summary:      cve.GetSummary(),
			Link:         cve.GetLink(),
			PublishedOn:  cve.GetPublishedOn(),
			CreatedAt:    cve.GetCreatedAt(),
			LastModified: cve.GetLastModified(),
			// ScoreVersion:         cve.GetScoreVersion(),
			CvssV2: cve.GetCvssV2(),
			CvssV3: cve.GetCvssV3(),
			// References:           cve.GetReferences(),
		},
		Cvss:         cve.GetCvss(),
		Severity:     cve.GetSeverity(),
		ImpactScore:  cve.GetImpactScore(),
		Snoozed:      cve.GetSuppressed(),
		SnoozeStart:  cve.GetSuppressActivation(),
		SnoozeExpiry: cve.GetSuppressExpiry(),
		Type:         cve.GetType(),
	}
}

func convertClusterCVeToCVE(clusterCVE *storage.ClusterCVE) *storage.CVE {
	baseInfo := clusterCVE.GetCveBaseInfo()
	distroSpecific := map[string]*storage.CVE_DistroSpecific{
		"os": {
			Severity:     clusterCVE.GetSeverity(),
			Cvss:         clusterCVE.GetCvss(),
			ScoreVersion: clusterCVE.GetScoreVersion,
			CvssV2:       baseInfo.GetCvssV2(),
			CvssV3:       baseInfo.GetCvssV3(),
		},
	}
	return &storage.CVE{
		Id:                 clusterCVE.GetId(),
		Cvss:               clusterCVE.GetCvss(),
		ImpactScore:        clusterCVE.GetImpactScore(),
		Type:               clusterCVE.GetType(),
		Types:              []storage.CVE_CVEType{clusterCVE.GetType()},
		Summary:            baseInfo.GetSummary(),
		Link:               baseInfo.GetLink(),
		PublishedOn:        baseInfo.GetPublishedOn(),
		CreatedAt:          baseInfo.GetCreatedAt(),
		LastModified:       baseInfo.GetLastModified(),
		References:         baseInfo.GetReferences(),
		ScoreVersion:       baseInfo.GetScoreVersion(),
		CvssV2:             baseInfo.GetCvssV2(),
		CvssV3:             baseInfo.GetCvssV3(),
		Suppressed:         clusterCVE.GetSnoozed(),
		SuppressActivation: clusterCVE.GetSnoozeStart(),
		SuppressExpiry:     clusterCVE.GetSnoozeExpiry(),
		DistroSpecifics:    distroSpecific,
		Severity:           clusterCVE.GetSeverity(),
	}
}
