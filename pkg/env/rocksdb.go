package env

var (
	// RocksDB is the variable is the variable used to opt into using RocksDB
	RocksDB = RegisterBooleanSetting("ROX_ROCKSDB", true)
)
