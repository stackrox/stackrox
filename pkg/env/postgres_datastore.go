package env

var (
	// PostgresDatastoreEnabled toggles whether to use Postgres for the datastore or not.
	PostgresDatastoreEnabled = RegisterBooleanSetting("ROX_POSTGRES_DATASTORE", false)
)
