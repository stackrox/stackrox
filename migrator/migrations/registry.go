package migrations

import (
	"fmt"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/migrations"
)

var (
	migrationRegistry = make(map[int]types.Migration)
)

// MustRegisterMigration registers a Migration, panic-ing if there's an error.
func MustRegisterMigration(m types.Migration) {
	if !features.PostgresDatastore.Enabled() && m.StartingSeqNum > migrations.CurrentDBVersionSeqNumWithoutPostgres() {
		return
	}
	if _, ok := migrationRegistry[m.StartingSeqNum]; ok {
		panic(fmt.Sprintf("Found multiple migrations starting at seq num %d", m.StartingSeqNum))
	}
	migrationRegistry[m.StartingSeqNum] = m
}

// Get the migration starting at the given sequence number.
func Get(startingSeqNum int) (types.Migration, bool) {
	m, ok := migrationRegistry[startingSeqNum]
	return m, ok
}
