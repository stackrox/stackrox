package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
)

// Write/Read up to 100 items per transaction.
var batchSize = 100

type legacyCrudImpl struct {
	counter *TxnCounter

	duckBox *dackbox.DackBox

	reader     Reader
	listReader Reader
	upserter   Upserter
	deleter    Deleter

	prefix     []byte
	listPrefix []byte
}

// CountImages returns all images regardless of request
func (b *legacyCrudImpl) Count() (int, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	count, err := b.reader.CountIn(b.prefix, branch)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Exists returns if the if image exists in the store
func (b *legacyCrudImpl) Exists(id string) (bool, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	exists, err := b.reader.ExistsIn(badgerhelper.GetBucketKey(b.prefix, []byte(id)), branch)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetImage returns image with given id.
func (b *legacyCrudImpl) Read(id string) (proto.Message, bool, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(b.prefix, []byte(id)), branch)
	if err != nil {
		return nil, false, err
	}

	return msg, msg != nil, nil
}

// GetImage returns image with given id.
func (b *legacyCrudImpl) ReadPartial(id string) (proto.Message, bool, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	msg, err := b.listReader.ReadIn(badgerhelper.GetBucketKey(b.listPrefix, []byte(id)), branch)
	if err != nil {
		return nil, false, err
	}

	return msg, msg != nil, err
}

func (b *legacyCrudImpl) ReadBatch(ids []string) ([]proto.Message, []int, error) {
	var msgs []proto.Message
	var missing []int
	for batch := 0; batch < len(ids); batch += batchSize {
		branch := b.duckBox.NewReadOnlyTransaction()
		defer branch.Discard()

		for idx := batch; idx < len(ids) && idx-batch < batchSize; idx++ {
			msg, err := b.reader.ReadIn(badgerhelper.GetBucketKey(b.prefix, []byte(ids[idx])), branch)
			if err != nil {
				return nil, nil, err
			}
			if msg != nil {
				msgs = append(msgs, msg)
			} else {
				missing = append(missing, idx)
			}
		}

		if err := branch.Commit(); err != nil {
			return nil, nil, err
		}
	}

	return msgs, missing, nil
}

func (b *legacyCrudImpl) ReadBatchPartial(ids []string) ([]proto.Message, []int, error) {
	var msgs []proto.Message
	var missing []int
	for batch := 0; batch < len(ids); batch += batchSize {
		branch := b.duckBox.NewReadOnlyTransaction()
		defer branch.Discard()

		for idx := batch; idx < len(ids) && idx-batch < batchSize; idx++ {
			msg, err := b.listReader.ReadIn(badgerhelper.GetBucketKey(b.listPrefix, []byte(ids[idx])), branch)
			if err != nil {
				return nil, nil, err
			}
			if msg != nil {
				msgs = append(msgs, msg)
			} else {
				missing = append(missing, idx)
			}
		}

		if err := branch.Commit(); err != nil {
			return nil, nil, err
		}
	}

	return msgs, missing, nil
}

func (b *legacyCrudImpl) ReadAll() ([]proto.Message, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	return b.reader.ReadAllIn(b.prefix, branch)
}

func (b *legacyCrudImpl) ReadAllPartial() ([]proto.Message, error) {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	return b.listReader.ReadAllIn(b.listPrefix, branch)
}

func (b *legacyCrudImpl) Create(msg proto.Message) error {
	return b.Upsert(msg)
}

func (b *legacyCrudImpl) CreateBatch(msgs []proto.Message) error {
	return b.UpsertBatch(msgs)
}

func (b *legacyCrudImpl) Update(msg proto.Message) error {
	return b.Upsert(msg)
}

func (b *legacyCrudImpl) UpdateBatch(msgs []proto.Message) error {
	return b.UpsertBatch(msgs)
}

func (b *legacyCrudImpl) Upsert(msg proto.Message) error {
	branch := b.duckBox.NewTransaction()
	defer branch.Discard()

	err := b.upserter.UpsertIn(nil, msg, branch)
	if err != nil {
		return err
	}

	if err := branch.Commit(); err != nil {
		return err
	}
	return b.IncTxnCount()
}

func (b *legacyCrudImpl) UpsertBatch(msgs []proto.Message) error {
	for batch := 0; batch < len(msgs); batch += batchSize {
		branch := b.duckBox.NewTransaction()
		defer branch.Discard()

		for idx := batch; idx < len(msgs) && idx-batch < batchSize; idx++ {
			err := b.upserter.UpsertIn(nil, msgs[idx], branch)
			if err != nil {
				return err
			}
		}

		if err := branch.Commit(); err != nil {
			return err
		}
	}

	return b.IncTxnCount()
}

func (b *legacyCrudImpl) Delete(id string) error {
	branch := b.duckBox.NewTransaction()
	defer branch.Discard()

	err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(b.prefix, []byte(id)), branch)
	if err != nil {
		return err
	}

	if err := branch.Commit(); err != nil {
		return err
	}
	return b.IncTxnCount()
}

func (b *legacyCrudImpl) DeleteBatch(ids []string) error {
	for batch := 0; batch < len(ids); batch += batchSize {
		branch := b.duckBox.NewTransaction()
		defer branch.Discard()

		for idx := batch; idx < len(ids) && idx-batch < batchSize; idx++ {
			err := b.deleter.DeleteIn(badgerhelper.GetBucketKey(b.prefix, []byte(ids[idx])), branch)
			if err != nil {
				return err
			}
		}

		if err := branch.Commit(); err != nil {
			return err
		}
	}

	return b.IncTxnCount()
}

func (b *legacyCrudImpl) GetTxnCount() uint64 {
	return b.counter.GetTxnCount()
}

func (b *legacyCrudImpl) IncTxnCount() error {
	return b.counter.IncTxnCount()
}

func (b *legacyCrudImpl) GetKeys() ([]string, error) {
	branch := b.duckBox.NewTransaction()
	defer branch.Discard()

	var keys []string
	err := badgerhelper.BucketKeyForEach(branch.BadgerTxn(), b.prefix, badgerhelper.ForEachOptions{StripKeyPrefix: true}, func(k []byte) error {
		keys = append(keys, string(k))
		return nil
	})
	return keys, err
}
