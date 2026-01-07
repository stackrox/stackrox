package migrations

import (
	"time"
)

// MigrationVersion is the last central version and migration sequence number
// that run with a database successfully. If central was up and ready to serve,
// it will update the migration version of current database if needed.
type MigrationVersion struct {
	MainVersion   string `yaml:"image"`
	SeqNum        int    `yaml:"database"`
	MinimumSeqNum int    `yaml:"mindatabase"`
	LastPersisted time.Time
}
