package env

var (
	// There are two ways of applying schema changes supported, a continuous
	// synching via GORM and classical migrations via tern. This configuration
	// is internal only and should not be advertised to users.
	TernMigrations = RegisterBooleanSetting("ROX_TERN_MIGRATIONS", false)

	// Where to look for migration files. If empty, migrator will try to find
	// the directory assuming it's invoked from the repository.
	TernMigrationsDir = RegisterSetting("ROX_TERN_MIGRATIONS_DIR", AllowEmpty())
)
