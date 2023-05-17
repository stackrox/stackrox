package internal

var (
	// CurrentDBVersionSeqNum is the current DB version number.
	// This must be incremented every time we write a migration.
	// It is a shared constant between central and the migrator binary.
	CurrentDBVersionSeqNum = 180

	// MinimumSupportedDBVersionSeqNum is the minimum DB version number
	// that is supported by this database.  This is used in case of rollbacks in
	// the event that a major change introduced an incompatible schema update we
	// can inform that a rollback below this is not supported by the database
	MinimumSupportedDBVersionSeqNum = 180

	// LastRocksDBVersionSeqNum is the sequence number for the last RocksDB version.
	LastRocksDBVersionSeqNum = 112
)
