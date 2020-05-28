package transactions

// DBTransactionFactory describes a creator for a transaction
type DBTransactionFactory interface {
	NewTransaction(update bool) (DBTransaction, error)
}

// DBTransaction is an abstraction of a database transaction
type DBTransaction interface {
	Delete(key ...[]byte) error
	Get(key []byte) ([]byte, bool, error)
	Set(key, value []byte) error
	BucketForEach(graphPrefix []byte, stripPrefix bool, fn func(k, v []byte) error) error
	BucketKeyForEach(graphPrefix []byte, stripPrefix bool, fn func(k []byte) error) error
	BucketKeyCount(prefix []byte) (int, error)

	Commit() error
	Discard()
}
