package clone

// DBCloneManager - scans and manage database clones within central.
type DBCloneManager interface {
	// Scan - Looks for database clones
	Scan() error

	// GetCloneToMigrate -- retrieves the clone that needs moved to the active database.
	GetCloneToMigrate() (string, error)

	// Persist -- moves the clone database to be the active database.
	Persist(pgClone string) error
}
