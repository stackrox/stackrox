// This file was originally generated with
// //go:generate cp ../../../../central/cve/store/dackbox/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	vulnDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/cve"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
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

func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
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

func (b *storeImpl) Count(_ context.Context) (int, error) {
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

func (b *storeImpl) Get(_ context.Context, id string) (cve *storage.CVE, exists bool, err error) {
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

func (b *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.CVE, []int, error) {
	cves, missing, err := b.getMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return cves, missing, nil
}

func (b *storeImpl) getMany(_ context.Context, ids []string) ([]*storage.CVE, []int, error) {
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

func (b *storeImpl) Upsert(_ context.Context, cves ...*storage.CVE) error {
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

func (b *storeImpl) UpsertMany(ctx context.Context, cve []*storage.CVE) error {
	return b.Upsert(ctx, cve...)
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
