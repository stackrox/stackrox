package migrations

import (
	"testing"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stretchr/testify/assert"
)

func TestValidityOfRegistry(t *testing.T) {
	a := assert.New(t)

	for startingSeqNum, m := range migrationRegistry {
		a.Equal(startingSeqNum, m.StartingSeqNum)
		a.Equal(startingSeqNum+1, int(m.VersionAfter.GetSeqNum()))
	}

	for i := 1; i < migrations.CurrentDBVersionSeqNum; i++ {
		_, exists := migrationRegistry[i]
		a.True(exists, "No registered migration found for starting seq num: %d", i)
	}

	a.Equal(migrations.CurrentDBVersionSeqNum, len(migrationRegistry)+1,
		"The DB version number and the migration counts aren't in sync!")
}
