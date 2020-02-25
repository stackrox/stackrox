package badgerutils

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLoad(t *testing.T, backupfile, key, value string) {
	db, path, err := badgerhelper.NewTemp(backupfile)
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	defer utils.IgnoreError(db.Close)

	file, err := os.Open(backupfile)
	require.NoError(t, err)

	err = Load(file, db)
	require.NoError(t, err)

	err = db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte(key))
		if err != nil {
			return err
		}
		valueBytes, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		assert.Equal(t, value, string(valueBytes))
		return nil
	})
	assert.NoError(t, err)
}

func TestNewFormat(t *testing.T) {
	testLoad(t, "backupwithmagic", "backup", "old")
}

func TestOldFormat(t *testing.T) {
	testLoad(t, "oldbackup", "backup", "old")
}
