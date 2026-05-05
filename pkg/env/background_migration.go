package env

// BackgroundMigrationOverrideSeqNum forces background migrations to start at the specified sequence number.
// A value of -1 (default) means the setting is not active. Requires BackgroundMigrationOverrideTag to be set.
var BackgroundMigrationOverrideSeqNum = RegisterIntegerSetting("ROX_BACKGROUND_MIGRATION_OVERRIDE_SEQ_NUM", -1)

// BackgroundMigrationOverrideTag is a free-form string that identifies an override run.
// The tag is persisted to the DB after applying an override. Subsequent replicas that see the
// same tag in the DB will skip the override. Change the tag to trigger a new override run.
var BackgroundMigrationOverrideTag = RegisterSetting("ROX_BACKGROUND_MIGRATION_OVERRIDE_TAG")
