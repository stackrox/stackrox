package badgerhelpers

import (
	"errors"

	"github.com/dgraph-io/badger"
)

var (
	// ErrAgain indicates that another transaction needs to take place. All previous changes have been committed.
	ErrAgain = errors.New("another transaction is required")
)

// SplitUpdate calls Update on the given DB. If the error returned by applying the function is ErrTxnTooBig, the
// transaction is committed and ErrAgain is returned.
func SplitUpdate(db *badger.DB, fn func(tx *badger.Txn) error) error {
	return db.Update(func(tx *badger.Txn) error {
		err := fn(tx)
		if err == nil || err != badger.ErrTxnTooBig {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrAgain
	})
}
