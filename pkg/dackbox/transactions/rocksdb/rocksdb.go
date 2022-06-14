package rocksdb

import (
	"github.com/stackrox/stackrox/pkg/dackbox/transactions"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	generic "github.com/stackrox/stackrox/pkg/rocksdb/crud"
	"github.com/tecbot/gorocksdb"
)

type rocksDBWrapper struct {
	db *rocksdb.RocksDB
}

func (b *rocksDBWrapper) NewTransaction(update bool) (transactions.DBTransaction, error) {
	if err := b.db.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}

	snapshot := b.db.NewSnapshot()
	readOpts := gorocksdb.NewDefaultReadOptions()
	readOpts.SetSnapshot(snapshot)

	itOpts := gorocksdb.NewDefaultReadOptions()
	itOpts.SetSnapshot(snapshot)
	itOpts.SetPrefixSameAsStart(true)
	itOpts.SetFillCache(false)

	wrapper := &txnWrapper{
		db:       b.db,
		isUpdate: update,

		snapshot: snapshot,
		readOpts: readOpts,
		itOpts:   itOpts,
	}
	if update {
		wrapper.batch = gorocksdb.NewWriteBatch()
	}
	return wrapper, nil
}

// NewRocksDBWrapper is a wrapper around a rocksDB so it implements the DBTransactionFactory interface
func NewRocksDBWrapper(db *rocksdb.RocksDB) transactions.DBTransactionFactory {
	return &rocksDBWrapper{
		db: db,
	}
}

type txnWrapper struct {
	db       *rocksdb.RocksDB
	isUpdate bool

	hasDecrementedInProgressOp bool
	batch                      *gorocksdb.WriteBatch
	readOpts                   *gorocksdb.ReadOptions
	itOpts                     *gorocksdb.ReadOptions
	snapshot                   *gorocksdb.Snapshot
}

func (t *txnWrapper) Delete(keys ...[]byte) {
	if !t.isUpdate {
		panic("trying to delete a key during a read txn")
	}
	for _, k := range keys {
		t.batch.Delete(k)
	}
}

func (t *txnWrapper) Get(key []byte) ([]byte, bool, error) {
	data, err := t.db.GetBytes(t.readOpts, key)
	return data, data != nil, err
}

func (t *txnWrapper) Set(key, value []byte) {
	if !t.isUpdate {
		panic("trying to set during a read txn")
	}

	t.batch.Put(key, value)
}

func (t *txnWrapper) BucketForEach(graphPrefix []byte, stripPrefix bool, fn func(k, v []byte) error) error {
	return generic.BucketForEach(t.db, t.itOpts, graphPrefix, stripPrefix, fn)
}

func (t *txnWrapper) BucketKeyForEach(graphPrefix []byte, stripPrefix bool, fn func(k []byte) error) error {
	return generic.BucketKeyForEach(t.db, t.itOpts, graphPrefix, stripPrefix, fn)
}

func (t *txnWrapper) BucketKeyCount(prefix []byte) (int, error) {
	var count int
	err := generic.BucketKeyForEach(t.db, t.itOpts, prefix, false, func(k []byte) error {
		count++
		return nil
	})
	return count, err
}

func (t *txnWrapper) Commit() error {
	defer t.Discard()

	writeOpts := generic.DefaultWriteOptions()
	defer writeOpts.Destroy()

	return t.db.Write(writeOpts, t.batch)
}

func (t *txnWrapper) Discard() {
	if t.batch != nil {
		t.batch.Destroy()
		t.batch = nil
	}
	if t.readOpts != nil {
		t.readOpts.Destroy()
		t.readOpts = nil
	}
	if t.itOpts != nil {
		t.itOpts.Destroy()
		t.itOpts = nil
	}
	if t.snapshot != nil {
		t.db.ReleaseSnapshot(t.snapshot)
		t.snapshot = nil
	}
	if !t.hasDecrementedInProgressOp {
		t.db.DecRocksDBInProgressOps()
		t.hasDecrementedInProgressOp = true
	}
}
