package badgerhelper

import "github.com/dgraph-io/badger"

// RetryableUpdate wraps a badger txn in a form that retries when there are conflicts
func RetryableUpdate(db *badger.DB, fn func(tx *badger.Txn) error) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = db.Update(fn); err != badger.ErrConflict {
			return err
		}
		log.Info("Trying to run update again after conflict")
	}
	return err
}
