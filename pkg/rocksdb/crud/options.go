// +build rocksdb

package generic

import "github.com/tecbot/gorocksdb"

// DefaultReadOptions define the default read options to be used for RocksDB
func DefaultReadOptions() *gorocksdb.ReadOptions {
	return gorocksdb.NewDefaultReadOptions()
}

// DefaultWriteOptions defines the default write options to be used for RocksDB
func DefaultWriteOptions() *gorocksdb.WriteOptions {
	return gorocksdb.NewDefaultWriteOptions()
}

// DefaultIteratorOptions defines the default iterator options to be used for RocksDB
func DefaultIteratorOptions() *gorocksdb.ReadOptions {
	readOptions := gorocksdb.NewDefaultReadOptions()
	readOptions.SetFillCache(false) // Avoid filling the cache as we are iterating over the DB
	readOptions.SetPrefixSameAsStart(true)
	return readOptions
}
