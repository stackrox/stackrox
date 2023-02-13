package internal

var (
	// CurrentDBVersionSeqNum is the current DB version number.
	// This must be incremented every time we write a migration.
	// It is a shared constant between central and the migrator binary.
	CurrentDBVersionSeqNum = 172

	// LastRocksDBVersionSeqNum is the sequence number for the last RocksDB version.
	LastRocksDBVersionSeqNum = 112

	// LastRocksToPostgresDBVersionSeqNum is the sequence number for the last legacy to Postgres migration
	LastRocksToPostgresDBVersionSeqNum = 168
)
