package backgroundmigrations

// CurrentBgMigrationSeqNum is the current background migration sequence number.
// This must be incremented every time a new background migration is added.
// It is independent from the schema migration sequence number in pkg/migrations.
const CurrentBgMigrationSeqNum = 0
