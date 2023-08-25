package runner

import (
	"testing"

	"github.com/stackrox/rox/migrator/migrations"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stretchr/testify/assert"
)

const (
	// lowestMigrationNumber is the lowest migration number that currently exists
	// we will periodically bump this number and delete outdated migrations once enough releases
	// have passed and we are sure customers are not on those releases
	lowestMigrationNumber = 55
)

func TestValidityOfRegistry(t *testing.T) {
	a := assert.New(t)

	for i := 0; i < lowestMigrationNumber; i++ {
		_, exists := migrations.Get(i)
		a.False(exists)
	}

	for i := lowestMigrationNumber; i < pkgMigrations.CurrentDBVersionSeqNum(); i++ {
		migration, exists := migrations.Get(i)
		a.Equal(i, migration.StartingSeqNum)
		a.Equal(i+1, int(migration.VersionAfter.GetSeqNum()))
		a.True(exists, "No registered migration found for starting seq num: %d", i)
	}

	_, exists := migrations.Get(pkgMigrations.CurrentDBVersionSeqNum())
	a.False(exists, "There should be no migration for the current sequence number")
}
