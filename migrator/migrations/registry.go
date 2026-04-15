package migrations

import (
	"fmt"

	"github.com/stackrox/rox/migrator/types"
)

// Migration Auto-Registration Pattern:
// Individual migration files (migrator/migrations/m_*_to_m_*/) use init() for auto-discovery.
// This is an intentional pattern to automatically register migrations without explicit imports.

var (
	migrationRegistry = make(map[int]types.Migration)
)

// MustRegisterMigration registers a Migration, panic-ing if there's an error.
func MustRegisterMigration(m types.Migration) {
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
