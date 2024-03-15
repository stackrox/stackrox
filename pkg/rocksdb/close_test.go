package rocksdb

import (
	"testing"
	"time"

	"github.com/DataDog/gorocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSynchronousCloseBaseCase(t *testing.T) {
	db, err := NewTemp(t.Name())
	require.NoError(t, err)
	require.NotNil(t, db)

	assert.NoError(t, db.IncRocksDBInProgressOps())
	db.DecRocksDBInProgressOps()
	db.Close()

	assert.Error(t, db.IncRocksDBInProgressOps())
}

func TestConcurrentWritesAndCloses(t *testing.T) {
	db, err := NewTemp(t.Name())
	require.NoError(t, err)
	require.NotNil(t, db)

	for i := 0; i < 10; i++ {
		go func() {
			if err := db.IncRocksDBInProgressOps(); err != nil {
				return
			}
			defer db.DecRocksDBInProgressOps()

			_, err := db.Get(gorocksdb.NewDefaultReadOptions(), []byte("key"))
			assert.NoError(t, err)
		}()
	}
	time.Sleep(10 * time.Millisecond)
	db.Close()
}
