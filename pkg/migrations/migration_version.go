package migrations

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

const (
	// MigrationVersionFile records the latest central version in databases.
	MigrationVersionFile     = "migration_version.yaml"
	migrationVersionFileMode = 0644
	lastRocksDBVersion       = "3.74.0"
)

var (
	log = logging.LoggerForModule()
)

// MigrationVersion is the last central version and migration sequence number
// that run with a database successfully. If central was up and ready to serve,
// it will update the migration version of current database if needed.
type MigrationVersion struct {
	dbPath        string
	MainVersion   string `yaml:"image"`
	SeqNum        int    `yaml:"database"`
	MinimumSeqNum int    `yaml:"mindatabase"`
	LastPersisted time.Time
}
