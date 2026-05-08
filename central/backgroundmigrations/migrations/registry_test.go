package migrations

import (
	"testing"

	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stretchr/testify/assert"
)

func TestRegistryPanicsOnDuplicate(t *testing.T) {
	ResetRegistryForTesting(t)
	MustRegister(types.BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "first"})
	assert.Panics(t, func() {
		MustRegister(types.BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "duplicate"})
	})
}

func TestRegistryGet(t *testing.T) {
	ResetRegistryForTesting(t)
	MustRegister(types.BackgroundMigration{StartingSeqNum: 0, VersionAfterSeqNum: 1, Description: "m0"})
	MustRegister(types.BackgroundMigration{StartingSeqNum: 1, VersionAfterSeqNum: 2, Description: "m1"})

	m, ok := Get(0)
	assert.True(t, ok)
	assert.Equal(t, "m0", m.Description)

	m, ok = Get(1)
	assert.True(t, ok)
	assert.Equal(t, "m1", m.Description)

	_, ok = Get(99)
	assert.False(t, ok)
}
