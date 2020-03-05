package m31to32

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getKey(prefix []byte, key string) []byte {
	result := append([]byte{}, prefix...)
	result = append(result, []byte(key)...)
	return result
}

func getSecondaryKey(indicator *storage.ProcessIndicator) ([]byte, error) {
	uniqueKey := &storage.ProcessIndicatorUniqueKey{
		PodId:               indicator.GetPodId(),
		ContainerName:       indicator.GetContainerName(),
		ProcessExecFilePath: indicator.GetSignal().GetExecFilePath(),
		ProcessName:         indicator.GetSignal().GetName(),
		ProcessArgs:         indicator.GetSignal().GetArgs(),
	}
	return proto.Marshal(uniqueKey)
}

func getKeyCount(db *badger.DB) (int, error) {
	var count int
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	err := db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

func TestMigration(t *testing.T) {
	db, err := badgerhelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	var expectedIndicators []*storage.ProcessIndicator
	err = db.Update(func(tx *badger.Txn) error {
		for i := 0; i < 10; i++ {
			pi := &storage.ProcessIndicator{
				Id:    fmt.Sprintf("%d", i),
				PodId: fmt.Sprintf("%d", i),
			}
			data, err := proto.Marshal(pi)
			if err != nil {
				return err
			}

			if err := tx.Set(getKey(oldProcessBucket, pi.GetId()), data); err != nil {
				return err
			}

			uniqueData, err := getSecondaryKey(pi)
			if err != nil {
				return err
			}
			if err := tx.Set(getKey(uniqueProcessBucket, pi.GetId()), uniqueData); err != nil {
				return err
			}

			// Add to expected indicators
			// Change ID of indicator
			id.SetIndicatorID(pi)
			expectedIndicators = append(expectedIndicators, pi)
		}
		return nil
	})
	require.NoError(t, err)

	// Make sure there are 20 entries before running
	numKeys, err := getKeyCount(db)
	require.NoError(t, err)
	assert.Equal(t, 20, numKeys)

	require.NoError(t, removeUniqueProcessPrefix(nil, db))

	numKeys, err = getKeyCount(db)
	require.NoError(t, err)
	assert.Equal(t, 10, numKeys)

	// Validate the results
	var actualIndicators []*storage.ProcessIndicator
	err = db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			assert.True(t, bytes.HasPrefix(item.Key(), newProcessBucket))

			var pi storage.ProcessIndicator
			err := item.Value(func(val []byte) error {
				return proto.Unmarshal(val, &pi)
			})
			require.NoError(t, err)

			actualIndicators = append(actualIndicators, &pi)
		}
		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedIndicators, actualIndicators)
}
