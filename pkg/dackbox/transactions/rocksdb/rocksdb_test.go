package rocksdb

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/dackbox/transactions"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRocksDBDackBox(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

type TestSuite struct {
	suite.Suite

	dir     string
	db      *rocksdb.RocksDB
	factory transactions.DBTransactionFactory
}

func (s *TestSuite) SetupTest() {
	db, dir, err := rocksdb.NewTemp("")
	require.NoError(s.T(), err)
	s.dir = dir
	s.db = db
	s.factory = NewRocksDBWrapper(db)
}

func (s *TestSuite) TearDownTest() {
	s.db.Close()
	_ = os.RemoveAll(s.dir)
}

func (s *TestSuite) TestTransactions() {
	key := []byte("key")
	value := []byte("value")
	value2 := []byte("value2")

	txn, err := s.factory.NewTransaction(false)
	s.NoError(err)

	retrievedValue, exists, err := txn.Get(key)
	s.NoError(err)
	s.False(exists)
	s.Nil(retrievedValue)
	txn.Discard()

	txn, err = s.factory.NewTransaction(true)
	s.NoError(err)
	s.NoError(txn.Set(key, value))
	s.NoError(txn.Commit())
	txn.Discard()

	txn, err = s.factory.NewTransaction(false)
	s.NoError(err)
	retrievedValue, exists, err = txn.Get(key)
	s.NoError(err)
	s.True(exists)
	s.Equal(value, retrievedValue)
	txn.Discard()

	txn, err = s.factory.NewTransaction(true)
	s.NoError(err)
	s.NoError(txn.Set(key, value2))
	s.NoError(txn.Commit())
	txn.Discard()

	txn, err = s.factory.NewTransaction(false)
	s.NoError(err)
	retrievedValue, exists, err = txn.Get(key)
	s.NoError(err)
	s.True(exists)
	s.Equal(value2, retrievedValue)
	txn.Discard()
}

func (s *TestSuite) TestTransactionPanicOnUpdate() {
	txn1, err := s.factory.NewTransaction(false)
	s.NoError(err)
	defer txn1.Discard()
	s.Panics(func() {
		_ = txn1.Set([]byte("1"), []byte("2"))
	})

	s.Panics(func() {
		_ = txn1.Delete([]byte("1"), []byte("2"))
	})

}

func (s *TestSuite) TestConcurrentTransactions() {
	key := []byte("key")
	value := []byte("value")
	key2 := []byte("key2")
	value2 := []byte("value2")

	// Create two txns, neither should be able to see the values that the other writes
	txn1, err := s.factory.NewTransaction(true)
	s.NoError(err)
	defer txn1.Discard()
	txn2, err := s.factory.NewTransaction(true)
	s.NoError(err)
	defer txn2.Discard()

	s.NoError(txn1.Set(key, value))
	s.NoError(txn2.Set(key2, value2))

	_, exists, err := txn1.Get(key2)
	s.NoError(err)
	s.False(exists)

	_, exists, err = txn2.Get(key)
	s.NoError(err)
	s.False(exists)

	s.NoError(txn1.Commit())
	// txn2 _still_ shouldn't be able to see the values
	_, exists, err = txn2.Get(key)
	s.NoError(err)
	s.False(exists)
	s.NoError(txn2.Commit())

	// New transactions should see both key and key2

	txn3, err := s.factory.NewTransaction(false)
	s.NoError(err)
	defer txn3.Discard()
	val, exists, err := txn3.Get(key)
	s.NoError(err)
	s.True(exists)
	s.Equal(value, val)

	val2, exists, err := txn3.Get(key2)
	s.NoError(err)
	s.True(exists)
	s.Equal(value2, val2)

	// Delete the values
	txn4, err := s.factory.NewTransaction(true)
	s.NoError(err)
	defer txn4.Discard()
	s.NoError(txn4.Delete(key))
	s.NoError(txn4.Delete(key2))
	s.NoError(txn4.Commit())

	// txn3 should still show them
	val, exists, err = txn3.Get(key)
	s.NoError(err)
	s.True(exists)
	s.Equal(value, val)

	val2, exists, err = txn3.Get(key2)
	s.NoError(err)
	s.True(exists)
	s.Equal(value2, val2)

	// txn5 should have them removed
	txn5, err := s.factory.NewTransaction(false)
	s.NoError(err)
	defer txn5.Discard()

	_, exists, err = txn5.Get(key)
	s.NoError(err)
	s.False(exists)

	_, exists, err = txn5.Get(key2)
	s.NoError(err)
	s.False(exists)
}
