package internal

var (
	// MinimumSupportedDBVersionSeqNum is the minimum DB version number
	// that is supported by this database.  This is used in case of rollbacks in
	// the event that a major change introduced an incompatible schema update we
	// can inform that a rollback below this is not supported by the database
	MinimumSupportedDBVersionSeqNum = 186
)
