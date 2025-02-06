//go:build sql_integration

package clone

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/migrator/clone/metadata"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	preHistoryVer         = versionPair{version: "3.0.56.0", seqNum: 62, minSeqNum: 0}
	preVer                = versionPair{version: "3.0.57.0", seqNum: 64, minSeqNum: 62}
	currVer               = versionPair{version: "3.0.58.0", seqNum: 65, minSeqNum: 62}
	lastLegacyDBVer       = versionPair{version: "3.74.0", seqNum: migrations.LastRocksDBVersionSeqNum()}
	postgresDBVer         = versionPair{version: "4.0.0", seqNum: 175, minSeqNum: 62}
	futureVerDifferentMin = versionPair{version: "10001.0.0.0", seqNum: 6533, minSeqNum: 185}
	futureVer             = versionPair{version: "10001.0.0.0", seqNum: 6533, minSeqNum: 62}

	// Current versions
	rcVer      = versionPair{version: "3.0.58.0-rc.1", seqNum: 65, minSeqNum: 62}
	releaseVer = versionPair{version: "3.0.58.0", seqNum: 65, minSeqNum: 62}
	devVer     = versionPair{version: "3.0.58.x-19-g6bd31dae22-dirty", seqNum: 65, minSeqNum: 62}
	nightlyVer = versionPair{version: "3.0.58.x-nightly-20210407", seqNum: 65, minSeqNum: 62}
)

func setVersion(t *testing.T, ver *versionPair) {
	log.Infof("setVersion => %v", ver)
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
	migrationtestutils.SetCurrentMinDBSequenceNumber(t, ver.minSeqNum)
}

func TestCloneMigration(t *testing.T) {
	currVer = releaseVer
	doTestCloneMigration(t, false)
	currVer = devVer
	doTestCloneMigration(t, false)
	currVer = rcVer
	doTestCloneMigration(t, false)
	currVer = nightlyVer
	doTestCloneMigration(t, false)
}

func doTestCloneMigration(t *testing.T, runBoth bool) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description      string
		fromVersion      *versionPair
		toVersion        *versionPair
		furtherToVersion *versionPair
	}{
		{
			description:      "Upgrade from early versions to current",
			fromVersion:      &preHistoryVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
		},
		{
			description:      "Upgrade from version 57 to current",
			fromVersion:      &preVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
		},
		{
			description: "Upgrade from current to future",
			fromVersion: &currVer,
			toVersion:   &futureVer,
		},
		{
			description: "Upgrade from early version to future",
			fromVersion: &preHistoryVer,
			toVersion:   &futureVer,
		},
	}

	// Test normal upgrade
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			mock := createAndRunCentral(t, c.fromVersion, runBoth)

			defer mock.destroyCentral()
			mock.setVersion = setVersion
			require.NoError(t, mock.upgradeCentral(c.toVersion, ""))
			if c.furtherToVersion != nil {
				require.NoError(t, mock.upgradeCentral(c.furtherToVersion, ""))
			}
		})
	}
}

func createAndRunCentral(t *testing.T, ver *versionPair, runBoth bool) *mockCentral {
	mock := createCentral(t, runBoth)
	mock.setVersion = setVersion
	mock.setVersion(t, ver)
	require.NoError(t, mock.runMigrator("", ""))
	mock.runCentral()
	return mock
}

func TestCloneMigrationFailureAndReentry(t *testing.T) {
	currVer = releaseVer
	doTestCloneMigrationFailureAndReentry(t)
	currVer = devVer
	doTestCloneMigrationFailureAndReentry(t)
	currVer = rcVer
	doTestCloneMigrationFailureAndReentry(t)
	currVer = nightlyVer
	doTestCloneMigrationFailureAndReentry(t)
}

func doTestCloneMigrationFailureAndReentry(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description      string
		fromVersion      *versionPair
		toVersion        *versionPair
		furtherToVersion *versionPair
		breakPoint       string
	}{
		{
			description:      "Upgrade from early versions to current break after db migration",
			fromVersion:      &preHistoryVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			breakPoint:       breakBeforePersist,
		},
		{
			description: "Upgrade from version 57 to current break after getting clone",
			fromVersion: &preVer,
			toVersion:   &currVer,
			breakPoint:  breakAfterGetClone,
		},
		{
			description: "Upgrade from current to future break after scan",
			fromVersion: &currVer,
			toVersion:   &futureVer,
			breakPoint:  breakAfterScan,
		},
	}
	// For the parameters that should not matter, run pseudo random to get coverage on different cases
	rand.Seed(8181818)
	for _, c := range testCases {
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}
		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			mock := createAndRunCentral(t, c.fromVersion, false)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			// Migration aborted
			require.NoError(t, mock.upgradeCentral(c.toVersion, c.breakPoint))
			if reboot {
				require.NoError(t, mock.rebootCentral())
			}
			if c.furtherToVersion != nil {
				// Run migrator multiple times
				require.NoError(t, mock.runMigrator("", ""))
				require.NoError(t, mock.upgradeCentral(c.furtherToVersion, ""))
			}
		})
	}
}

func TestCloneRestore(t *testing.T) {
	// This will test restore for Rocks -> Rocks or Postgres -> Postgres depending on
	// the test is executed with the Postgres env variable set or not.
	testCloneRestore(t)
}

func testCloneRestore(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description     string
		toVersion       *versionPair
		breakPoint      string
		rocksToPostgres bool
	}{
		{
			description: "Restore to earlier version",
			toVersion:   &preHistoryVer,
		},
		{
			description: "Restore to earlier version break after scan",
			toVersion:   &preHistoryVer,
			breakPoint:  breakAfterScan,
		},
		{
			description: "Restore to earlier version break after get clone",
			toVersion:   &preHistoryVer,
			breakPoint:  breakAfterGetClone,
		},
		{
			description: "Restore to earlier version break before persist",
			toVersion:   &preHistoryVer,
			breakPoint:  breakBeforePersist,
		},
		{
			description: "Restore to previous version",
			toVersion:   &preVer,
		},
		{
			description: "Restore to previous version break after scan",
			toVersion:   &preVer,
			breakPoint:  breakAfterScan,
		},
		{
			description: "Restore to current versions break after get clone",
			toVersion:   &currVer,
			breakPoint:  breakAfterGetClone,
		},
		{
			description: "Restore to earlier versions break after scan",
			toVersion:   &currVer,
			breakPoint:  breakBeforePersist,
		},
		{
			description:     "Restore to earlier RocksDB version",
			toVersion:       &preHistoryVer,
			rocksToPostgres: true,
		},
	}

	for _, c := range testCases {
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}

		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			mock := createAndRunCentral(t, &preHistoryVer, c.rocksToPostgres)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			require.NoError(t, mock.upgradeCentral(&currVer, ""))
			err := mock.restoreCentral(c.toVersion, c.breakPoint, c.rocksToPostgres)
			if c.rocksToPostgres {
				require.EqualError(t, err, "Effective release 4.5, restores from pre-4.0 releases are no longer supported.")
				return
			}

			if reboot {
				require.NoError(t, mock.rebootCentral())
			}
			require.NoError(t, mock.upgradeCentral(&futureVer, ""))
		})
	}
}

func TestForceRollbackRocksToPostgresFailure(t *testing.T) {
	currVer = releaseVer
	doTestForceRollbackRocksToPostgresFailure(t)
	currVer = devVer
	doTestForceRollbackRocksToPostgresFailure(t)
	currVer = rcVer
	doTestForceRollbackRocksToPostgresFailure(t)
	currVer = nightlyVer
	doTestForceRollbackRocksToPostgresFailure(t)
}

func doTestForceRollbackRocksToPostgresFailure(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}

	testCases := []struct {
		description          string
		forceRollback        string
		withPrevious         bool
		expectedErrorMessage string
		wrongVersion         bool
		toVersion            versionPair
	}{
		{
			description:          "without force rollback without previous",
			withPrevious:         false,
			forceRollback:        "",
			toVersion:            futureVerDifferentMin,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, currVer.seqNum, futureVerDifferentMin.minSeqNum),
		},
		{
			// Any rollbacks to 4.1 or later will only use central_active
			description:          "with force rollback without previous min sequence not supported",
			withPrevious:         false,
			forceRollback:        currVer.version,
			toVersion:            futureVerDifferentMin,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, currVer.seqNum, futureVerDifferentMin.minSeqNum),
		},
		{
			description:          "force rollback with previous",
			withPrevious:         true,
			forceRollback:        currVer.version,
			toVersion:            futureVer,
			expectedErrorMessage: "",
		},
		{
			description:          "without force rollback with previous",
			withPrevious:         true,
			forceRollback:        "",
			toVersion:            futureVerDifferentMin,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, currVer.seqNum, futureVerDifferentMin.minSeqNum),
		},
		{
			description:          "with force rollback code does not support min sequence in DB",
			withPrevious:         true,
			forceRollback:        currVer.version,
			toVersion:            futureVerDifferentMin,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrSoftwareNotCompatibleWithDatabase, currVer.seqNum, futureVerDifferentMin.minSeqNum),
			wrongVersion:         true,
		},
	}
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			ver := &currVer
			if c.wrongVersion {
				ver = &preVer
			}
			mock := createAndRunCentral(t, ver, false)
			defer mock.destroyCentral()
			require.NoError(t, mock.upgradeCentral(&c.toVersion, ""))
			if !c.withPrevious {
				mock.removePreviousClone()
			}
			// Force rollback
			setVersion(t, &currVer)

			var dbm DBCloneManager

			expectedError := c.expectedErrorMessage

			source := pgtest.GetConnectionString(t)
			sourceMap, _ := pgconfig.ParseSource(source)
			config, err := postgres.ParseConfig(source)
			require.NoError(t, err)

			dbm = NewPostgres(mock.mountPath, c.forceRollback, config, sourceMap)

			err = dbm.Scan()
			if expectedError != "" {
				assert.EqualError(t, err, expectedError)
			} else {
				assert.NoError(t, err)
				require.NoError(t, mock.rollbackCentral(&currVer, "", c.forceRollback))
			}

		})
	}
}

func TestRollback(t *testing.T) {
	currVer = releaseVer
	doTestRollback(t)
	currVer = devVer
	doTestRollback(t)
	currVer = rcVer
	doTestRollback(t)
	currVer = nightlyVer
	doTestRollback(t)
}

func doTestRollback(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description string
		fromVersion *versionPair
		toVersion   *versionPair // version to failback to
		breakPoint  string
	}{
		{
			description: "Rollback to current",
			fromVersion: &futureVer,
			toVersion:   &currVer,
		},
		{
			description: "Rollback to version 57",
			fromVersion: &currVer,
			toVersion:   &preVer,
		},
		{
			description: "Rollback to current break before persist",
			fromVersion: &futureVer,
			toVersion:   &currVer,
			breakPoint:  breakBeforePersist,
		},
		{
			description: "Rollback to version 57 break before persist",
			fromVersion: &currVer,
			toVersion:   &preVer,
			breakPoint:  breakBeforePersist,
		},
		{
			description: "Rollback to current break after scan",
			fromVersion: &futureVer,
			toVersion:   &currVer,
			breakPoint:  breakAfterScan,
		},
		{
			description: "Rollback to version 57 break after scan",
			fromVersion: &currVer,
			toVersion:   &preVer,
			breakPoint:  breakAfterScan,
		},
		{
			description: "Rollback to current break after get clone",
			fromVersion: &futureVer,
			toVersion:   &currVer,
			breakPoint:  breakAfterGetClone,
		},
		{
			description: "Rollback to version 57 break after get clone",
			fromVersion: &currVer,
			toVersion:   &preVer,
			breakPoint:  breakAfterGetClone,
		},
	}
	rand.Seed(8056)
	for _, c := range testCases {
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}
		log.Infof("Test = %q", c.description)

		t.Run(c.description, func(t *testing.T) {
			mock := createAndRunCentral(t, c.toVersion, false)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			require.NoError(t, mock.upgradeCentral(c.fromVersion, c.breakPoint))
			require.NoError(t, mock.rollbackCentral(c.toVersion, "", c.toVersion.version))
			require.NoError(t, mock.upgradeCentral(c.fromVersion, ""))
		})
	}
}

// This is a completely white box test to cover the racing condition while persisting changes.
// These conditions are theoretically possible but chance is very slim but we should handle that.
func TestRacingConditionInPersist(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description string
		preRun      func(m *mockCentral)
	}{
		{
			description: "Restore breaks in persist",
			preRun: func(m *mockCentral) {
				m.restore(&preVer, false)
			},
		},
		{
			description: "Upgrade breaks in persist",
			preRun: func(m *mockCentral) {
				setVersion(t, &futureVer)
			},
		},
		{
			description: "Rollback breaks in persist",
			preRun: func(m *mockCentral) {
				require.NoError(t, m.migrateWithVersion(&futureVer, breakBeforePersist, ""))
				setVersion(t, &currVer)
			},
		},
	}
	for _, c := range testCases {
		run := func(desc string, breakpoint string) {
			t.Run(desc, func(t *testing.T) {
				log.Infof("Test = %q", c.description)
				mock := createAndRunCentral(t, &preVer, false)
				defer mock.destroyCentral()
				require.NoError(t, mock.upgradeCentral(&currVer, ""))
				c.preRun(mock)
				mock.runMigratorWithBreaksInPersist(breakpoint)
				require.NoError(t, mock.rebootCentral())
			})
		}

		for _, breakpoint := range []string{breakBeforeCleanUp, breakBeforeCommitCurrent, breakAfterRemove} {
			desc := c.description + " at " + breakpoint
			run(desc, breakpoint)
		}
	}
}

func TestUpgradeFromLastRocksDB(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description    string
		previousVerion *versionPair
		fromVersion    *versionPair
		toVersion      *versionPair
		fromRocks      bool
	}{
		{
			description:    "Upgrade from fresh install of 3.74",
			previousVerion: nil,
			fromVersion:    &lastLegacyDBVer, // versionPair{version: "3.74.0", seqNum: migrations.LastRocksDBVersionSeqNum()}
			toVersion:      &postgresDBVer,   // versionPair{version: "4.0.0", seqNum: 175}
			fromRocks:      false,
		},
		{
			description:    "Upgrade from 3.74 with previous",
			previousVerion: &preVer,          // versionPair{version: "3.0.57.0", seqNum: 64}
			fromVersion:    &lastLegacyDBVer, // versionPair{version: "3.74.0", seqNum: migrations.LastRocksDBVersionSeqNum()}
			toVersion:      &postgresDBVer,   // versionPair{version: "4.0.0", seqNum: 175}
			fromRocks:      true,
		},
		{
			// We require upgrade from 3.74 to Postgres. This is not a recommended upgrade path, but
			// internally we still need to test the data on stackrox-db PVC is in valid state,
			// so we can recover from it.
			description: "Upgrade directly from earlier versions",
			fromVersion: &preVer,        // versionPair{version: "3.0.57.0", seqNum: 64}
			toVersion:   &postgresDBVer, // versionPair{version: "4.0.0", seqNum: 175}
			fromRocks:   true,
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			startVer := c.fromVersion
			if c.previousVerion != nil {
				startVer = c.previousVerion
			}
			mock := createCentral(t, true)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.setVersion(t, startVer)

			// Doesn't make sense to run this on the fresh install case
			if c.fromRocks {
				// With the flag permanently set now we are no longer able to toggle it.
				// So we need to explicitly update RocksDB
				// if we want tests to work as if RocksDB was the original DB.
				mock.legacyUpgrade(t, c.fromVersion, c.previousVerion)
			}

			mock.setVersion(t, c.toVersion)
			err := mock.runMigrator("", "")
			if c.fromRocks && err != nil {
				require.EqualError(t, err, "Effective release 4.5, upgrades from pre-4.0 releases are no longer supported.")
				return
			}

			require.NoError(t, err)

			mock.runCentral()
			err = mock.upgradeCentral(c.toVersion, "")
			if c.fromRocks {
				require.EqualError(t, err, "Effective release 4.5, upgrades from pre-4.0 releases are no longer supported.")
				return
			}
			require.NoError(t, err)

			mock.verifyCurrent()

			// Again on the fresh install case we don't touch rocks so it won't be updated.
			if c.fromRocks && err == nil {
				require.Fail(t, "This case is not valid")
			}
			// Rocks should have then empty version if we fresh installed Postgres.
			mock.verifyClone(rocksdb.CurrentClone, &versionPair{version: "0", seqNum: 0})
		})
	}
}
