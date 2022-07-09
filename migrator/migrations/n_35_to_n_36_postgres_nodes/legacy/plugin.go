package legacy

import (
	"context"

	nodeDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/node"
)

// GetIDs returns the keys of all images stored in RocksDB.
func (b *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	var ids []string
	err = txn.BucketKeyForEach(nodeDackBox.Bucket, true, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}
