package internal

var (
	// CurrentDBVersionSeqNum is the current DB version number.
	// This must be incremented every time we write a migration.
	// It is a shared constant between central and the migrator binary.
	CurrentDBVersionSeqNum int = 97
)
