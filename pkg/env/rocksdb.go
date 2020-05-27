package env

var (
	// RocksDB is the variable is the variable used to opt into using RocksDB
	RocksDB = registerBooleanSetting("ROX_ROCKSDB", false)
)
