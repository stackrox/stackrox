package env

import "time"

// BackgroundMigrationOverrideSeqNum forces background migrations to start at the specified sequence number.
// A value of -1 (default) means the setting is not active. Requires BackgroundMigrationOverrideTag to be set.
var BackgroundMigrationOverrideSeqNum = RegisterIntegerSetting("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", -1)

// BackgroundMigrationOverrideTag is a free-form string that identifies an override run.
// The tag is persisted to the DB after applying an override. Subsequent replicas that see the
// same tag in the DB will skip the override. Change the tag to trigger a new override run.
var BackgroundMigrationOverrideTag = RegisterSetting("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG")

// SkipBackgroundMigrations is a comma-separated list of background migration sequence numbers to skip.
var SkipBackgroundMigrations = RegisterSetting("ROX_SKIP_BACKGROUND_MIGRATIONS")

// BackgroundIndexTimeout is the per-statement timeout for CREATE INDEX CONCURRENTLY and
// DROP INDEX CONCURRENTLY operations during background index reconciliation.
var BackgroundIndexTimeout = registerDurationSetting("ROX_BACKGROUND_INDEX_TIMEOUT", 2*time.Hour)
