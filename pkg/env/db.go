package env

var (
	// DBPath is used to provide the main prevent server with the path to look for the DB
	DBPath = NewSetting("ROX_PREVENT_DB_PATH", WithDefault("/var/lib/prevent"))
)
