package runner

import (
	"testing"

	"github.com/stackrox/rox/migrator/migrations"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestValidityOfRegistry(t *testing.T) {
	a := assert.New(t)

	mainVersion := version.GetMainVersion()
	if _, err := pkgMigrations.MinimumSupportedForVersion(mainVersion); err != nil {
		mainVersion = pkgMigrations.LatestReleaseVersion().Version
	}
	setMainVersion(t, mainVersion)

	lowestMigrationNumber, err := pkgMigrations.MinimumSupportedDBVersionSeqNum()
	a.NoError(err)

	for i := range lowestMigrationNumber {
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
