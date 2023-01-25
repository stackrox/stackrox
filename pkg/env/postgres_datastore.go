package env

var (
	// PostgresDatastoreEnabled toggles whether to use Postgres for the datastore or not.
	PostgresDatastoreEnabled = RegisterUnchangeableBooleanSetting("ROX_POSTGRES_DATASTORE", true)
)
