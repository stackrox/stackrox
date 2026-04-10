package backgroundmigrations

import (
	"fmt"
)

var migrations map[int]BackgroundMigration = make(map[int]BackgroundMigration)

// MustRegister adds a background migration to the registry. It panics on error.
func MustRegister(m BackgroundMigration) {
	if _, ok := migrations[m.StartingSeqNum]; ok {
		panic(fmt.Sprintf("Found multiple migrations starting at seq num %d", m.StartingSeqNum))
	}
	migrations[m.StartingSeqNum] = m
}

// Get the BackgroundMigration at the given sequence number.
func Get(startingSeqNum int) (BackgroundMigration, bool) {
	m, ok := migrations[startingSeqNum]
	return m, ok
}
