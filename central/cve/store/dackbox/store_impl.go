package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	vulnDackBox "github.com/stackrox/rox/central/cve/dackbox"
	"github.com/stackrox/rox/central/cve/store"
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

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) (store.Store, error) {
	counter, err := crud.NewTxnCounter(dacky, vulnDackBox.Bucket)
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		counter:  counter,
		dacky:    dacky,
		reader:   vulnDackBox.Reader,
		upserter: vulnDackBox.Upserter,
		deleter:  vulnDackBox.Deleter,
	}, nil
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	exists, err := b.reader.ExistsIn(badgerhelper.GetBucketKey(vulnDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "CVE")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	count, err := b.reader.CountIn(vulnDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *storeImpl) GetAll() ([]*storage.CVE, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "CVE")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(vulnDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.CVE, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.CVE))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (cve *storage.CVE, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "CVE")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(vulnDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return nil, false, err
	}

	return msg.(*storage.CVE), msg != nil, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.CVE, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "CVE")

	dackTxn := b.dacky.NewReadOnlyTransaction()
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids)/2)
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(vulnDackBox.Bucket, []byte(id)), dackTxn)
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

// UpdateImage updates a image to bolt.
func (b *storeImpl) Upsert(cve *storage.CVE) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "CVE")

	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	err := b.upserter.UpsertIn(nil, cve, dackTxn)
	if err != nil {
		return err
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) UpsertBatch(cves []*storage.CVE) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "CVE")

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
	return b.counter.IncTxnCount()
}

func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "CVE")

	dackTxn := b.dacky.NewTransaction()
	defer dackTxn.Discard()

	err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(vulnDackBox.Bucket, []byte(id)), dackTxn)
	if err != nil {
		return err
	}

	if err := dackTxn.Commit(); err != nil {
		return err
	}
	return b.counter.IncTxnCount()
}

func (b *storeImpl) DeleteBatch(ids []string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.RemoveMany, "CVE")

	for batch := 0; batch < len(ids); batch += batchSize {
		dackTxn := b.dacky.NewTransaction()
		defer dackTxn.Discard()

		for idx := batch; idx < len(ids) && idx < batch+batchSize; idx++ {
			err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(vulnDackBox.Bucket, []byte(ids[idx])), dackTxn)
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
