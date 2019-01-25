package registry

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

var (
	migrationRegistry = make(map[int]Migration)
)

// A Migration represents a migration.
type Migration struct {
	StartingSeqNum int
	Run            func() error
	VersionAfter   storage.Version
}

// MustRegisterMigration registers a Migration, panic-ing if there's an error.
func MustRegisterMigration(m Migration) {
	if _, ok := migrationRegistry[m.StartingSeqNum]; ok {
		panic(fmt.Sprintf("Found multiple migrations starting at seq num %d", m.StartingSeqNum))
	}
	migrationRegistry[m.StartingSeqNum] = m
}

// CurrentSeqNum returns the current seq num.
func CurrentSeqNum() int32 {
	max := int32(1)
	for _, m := range migrationRegistry {
		seqNum := m.VersionAfter.GetSeqNum()
		if seqNum > max {
			max = seqNum
		}
	}
	return max
}
