package m31to32

import (
	"bytes"
	"fmt"
	"sort"
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

	batch := db.NewWriteBatch()
	defer batch.Cancel()

	numIndicators := 50000
	for i := 0; i < numIndicators; i++ {
		pi := &storage.ProcessIndicator{
			Id:    fmt.Sprintf("%d", i),
			PodId: fmt.Sprintf("%d", i),
		}
		data, err := proto.Marshal(pi)
		require.NoError(t, err)

		require.NoError(t, batch.Set(getKey(oldProcessBucket, pi.GetId()), data))

		uniqueData, err := getSecondaryKey(pi)
		require.NoError(t, err)
		require.NoError(t, batch.Set(getKey(uniqueProcessBucket, pi.GetId()), uniqueData))

		// Add to expected indicators
		// Change ID of indicator
		id.SetIndicatorID(pi)
		expectedIndicators = append(expectedIndicators, pi)
	}
	require.NoError(t, err)
	require.NoError(t, batch.Flush())

	// Make sure there are 20 entries before running
	numKeys, err := getKeyCount(db)
	require.NoError(t, err)
	// 1 key for process indicator and 1 key for unique indicator
	assert.Equal(t, numIndicators*2, numKeys)

	require.NoError(t, removeUniqueProcessPrefix(nil, db))

	numKeys, err = getKeyCount(db)
	require.NoError(t, err)
	// Should only be the new keys
	assert.Equal(t, numIndicators, numKeys)

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

	sort.Slice(expectedIndicators, func(i, j int) bool {
		return expectedIndicators[i].GetId() < expectedIndicators[j].GetId()
	})
	sort.Slice(actualIndicators, func(i, j int) bool {
		return actualIndicators[i].GetId() < actualIndicators[j].GetId()
	})
	assert.Equal(t, expectedIndicators, actualIndicators)
}
