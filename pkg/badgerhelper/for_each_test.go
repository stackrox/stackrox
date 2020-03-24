package badgerhelper

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachWithPrefix(t *testing.T) {
	if devbuild.IsEnabled() {
		defer func() {
			obj := recover()
			assert.NotNil(t, obj)
		}()
	}
	db, dir, err := NewTemp("TestForEachWithPrefix")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	defer utils.IgnoreError(db.Close)

	err = db.Update(func(tx *badger.Txn) error {
		if err := tx.Set([]byte("01"), []byte("1")); err != nil {
			return err
		}
		// 2 is set to an empty byte slice
		if err := tx.Set([]byte("02"), []byte{}); err != nil {
			return err
		}
		return tx.Set([]byte("03"), []byte("1"))
	})
	require.NoError(t, err)

	foundIDs := set.NewStringSet()
	err = db.View(func(tx *badger.Txn) error {
		return ForEachWithPrefix(tx, []byte("0"), ForEachOptions{StripKeyPrefix: true}, func(k, v []byte) error {
			foundIDs.Add(string(k))
			assert.NotEmpty(t, v)
			return nil
		})
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "3"}, foundIDs.AsSlice())
}
