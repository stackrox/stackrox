package backgroundmigrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetRegistry() {
	migrations = make(map[int]BackgroundMigration)
}

func TestRegistryPanicsOnDuplicate(t *testing.T) {
	resetRegistry()
	MustRegister(BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "first"})
	assert.Panics(t, func() {
		MustRegister(BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "duplicate"})
	})
}

func TestRegistryGet(t *testing.T) {
	resetRegistry()
	MustRegister(BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0"})
	MustRegister(BackgroundMigration{StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1"})

	m, ok := Get(0)
	assert.True(t, ok)
	assert.Equal(t, "m0", m.Description)

	m, ok = Get(1)
	assert.True(t, ok)
	assert.Equal(t, "m1", m.Description)

	_, ok = Get(99)
	assert.False(t, ok)
}
