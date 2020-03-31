package badger

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/transactions"
)

type badgerWrapper struct {
	db *badger.DB
}

func (b *badgerWrapper) NewTransaction(update bool) transactions.DBTransaction {
	return &txnWrapper{
		txn: b.db.NewTransaction(update),
	}
}

// NewBadgerWrapper wraps a BadgerDB so it implements the TransactionFactory
func NewBadgerWrapper(db *badger.DB) transactions.DBTransactionFactory {
	return &badgerWrapper{
		db: db,
	}
}

type txnWrapper struct {
	txn *badger.Txn
}

func (t *txnWrapper) Delete(keys ...[]byte) error {
	for _, k := range keys {
		if err := t.txn.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (t *txnWrapper) Get(key []byte) ([]byte, bool, error) {
	item, err := t.txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (t *txnWrapper) Set(key, value []byte) error {
	return t.txn.Set(key, value)
}

func (t *txnWrapper) BucketForEach(graphPrefix []byte, stripPrefix bool, fn func(k, v []byte) error) error {
	return badgerhelper.BucketForEach(t.txn, graphPrefix, badgerhelper.ForEachOptions{
		StripKeyPrefix: stripPrefix,
	}, fn)
}

func (t *txnWrapper) BucketKeyForEach(graphPrefix []byte, stripPrefix bool, fn func(k []byte) error) error {
	return badgerhelper.BucketKeyForEach(t.txn, graphPrefix, badgerhelper.ForEachOptions{
		StripKeyPrefix: stripPrefix,
	}, fn)
}

func (t *txnWrapper) BucketKeyCount(prefix []byte) (int, error) {
	return badgerhelper.BucketKeyCount(t.txn, prefix)
}

func (t *txnWrapper) Commit() error {
	return t.txn.Commit()
}

func (t *txnWrapper) Discard() {
	t.txn.Discard()
}
