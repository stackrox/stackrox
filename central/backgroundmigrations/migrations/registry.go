package migrations

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/backgroundmigrations/types"
)

var migrations map[int]types.BackgroundMigration = make(map[int]types.BackgroundMigration)

// MustRegister adds a background migration to the registry. It panics on error.
func MustRegister(m types.BackgroundMigration) {
	if m.VersionAfterSeqNum != m.StartingSeqNum+1 {
		panic(fmt.Sprintf("Background Migration at seq num %d has VersionAfterSeqNum %d, expected %d", m.StartingSeqNum, m.VersionAfterSeqNum, m.StartingSeqNum+1))
	}
	if _, ok := migrations[m.StartingSeqNum]; ok {
		panic(fmt.Sprintf("Found multiple background migrations starting at seq num %d", m.StartingSeqNum))
	}
	migrations[m.StartingSeqNum] = m
}

// Get the BackgroundMigration at the given sequence number.
func Get(startingSeqNum int) (types.BackgroundMigration, bool) {
	m, ok := migrations[startingSeqNum]
	return m, ok
}

// ResetRegistryForTesting clears all registered migrations. Test use only.
func ResetRegistryForTesting(t *testing.T) {
	migrations = make(map[int]types.BackgroundMigration)
}
