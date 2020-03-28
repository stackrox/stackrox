package m2to3

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	uuid "github.com/satori/go.uuid"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	boltDB, err := bolthelpers.NewTemp(t.Name())
	require.NoError(t, err)

	cluster1ID := uuid.NewV4().String()
	cluster2ID := uuid.NewV4().String()

	err = boltDB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(boltBucket)
		if err != nil {
			return err
		}
		cluster1B, err := b.CreateBucket([]byte(cluster1ID))
		if err != nil {
			return err
		}

		if err := cluster1B.Put([]byte("key1"), []byte("value1")); err != nil {
			return err
		}
		if err := cluster1B.Put([]byte("key2"), []byte("value2")); err != nil {
			return err
		}

		cluster2B, err := b.CreateBucket([]byte(cluster2ID))
		if err != nil {
			return err
		}

		return cluster2B.Put([]byte("key3"), []byte("value3"))
	})
	require.NoError(t, err)

	type entry struct {
		key, value string
	}

	badgerDB, err := badgerhelpers.NewTemp(t.Name())
	require.NoError(t, err)

	err = migrate(boltDB, badgerDB)
	require.NoError(t, err)

	var entries []entry
	err = badgerDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				entries = append(entries, entry{key: string(key), value: string(val)})
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	expected := []entry{
		{
			key:   fmt.Sprintf("%s\x00%s\x00key1", badgerKeyPrefix, cluster1ID),
			value: "value1",
		},
		{
			key:   fmt.Sprintf("%s\x00%s\x00key2", badgerKeyPrefix, cluster1ID),
			value: "value2",
		},
		{
			key:   fmt.Sprintf("%s\x00%s\x00key3", badgerKeyPrefix, cluster2ID),
			value: "value3",
		},
	}

	assert.ElementsMatch(t, expected, entries)

	err = boltDB.View(func(tx *bolt.Tx) error {
		flowsBucket := tx.Bucket(boltBucket)
		if flowsBucket != nil {
			return errors.New("flows bucket should no longer exist")
		}
		return nil
	})
	assert.NoError(t, err)
}
