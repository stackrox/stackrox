//go:build sql_integration

package clone

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/migrator/clone/metadata"
	"github.com/stackrox/rox/migrator/clone/postgres"
	"github.com/stackrox/rox/migrator/clone/rocksdb"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	migrationtestutils "github.com/stackrox/rox/pkg/migrations/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	preHistoryVer   = versionPair{version: "3.0.56.0", seqNum: 62}
	preVer          = versionPair{version: "3.0.57.0", seqNum: 64}
	currVer         = versionPair{version: "3.0.58.0", seqNum: 65}
	lastLegacyDBVer = versionPair{version: "3.74.0", seqNum: migrations.LastRocksDBVersionSeqNum()}
	postgresDBVer   = versionPair{version: "4.0.0", seqNum: 175}
	futureVer       = versionPair{version: "10001.0.0.0", seqNum: 6533}
	moreFutureVer   = versionPair{version: "10002.0.0.0", seqNum: 7533}

	// Current versions
	rcVer      = versionPair{version: "3.0.58.0-rc.1", seqNum: 65}
	releaseVer = versionPair{version: "3.0.58.0", seqNum: 65}
	devVer     = versionPair{version: "3.0.58.x-19-g6bd31dae22-dirty", seqNum: 65}
	nightlyVer = versionPair{version: "3.0.58.x-nightly-20210407", seqNum: 65}
)

func setVersion(t *testing.T, ver *versionPair) {
	log.Infof("setVersion => %v", ver)
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
}

func TestCloneMigration(t *testing.T) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		currVer = releaseVer
		doTestCloneMigration(t, false)
		currVer = devVer
		doTestCloneMigration(t, false)
		currVer = rcVer
		doTestCloneMigration(t, false)
		currVer = nightlyVer
		doTestCloneMigration(t, false)
	}
}

func TestCloneMigrationRocksToPostgres(t *testing.T) {
	// Run tests with both Rocks and Postgres to make sure migration clone is correctly determined.
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		currVer = releaseVer
		doTestCloneMigrationToPostgres(t, true)
		currVer = devVer
		doTestCloneMigrationToPostgres(t, true)
		currVer = rcVer
		doTestCloneMigrationToPostgres(t, true)
		currVer = nightlyVer
		doTestCloneMigrationToPostgres(t, true)
	}
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
			mock.upgradeCentral(c.toVersion, "")
			if c.furtherToVersion != nil {
				mock.upgradeCentral(c.furtherToVersion, "")
			}
		})
	}
}

func doTestCloneMigrationToPostgres(t *testing.T, runBoth bool) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description          string
		fromVersion          *versionPair
		toVersion            *versionPair
		furtherToVersion     *versionPair
		moreFurtherToVersion *versionPair
	}{
		{
			description:          "Upgrade from version 57 to current with rollback enabled",
			fromVersion:          &preVer,
			toVersion:            &currVer,
			furtherToVersion:     &futureVer,
			moreFurtherToVersion: &moreFutureVer,
		},
		{
			description: "Upgrade from current to future with rollback enabled",
			fromVersion: &currVer,
			toVersion:   &futureVer,
		},
	}

	// Test normal upgrade
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			mock := createAndRunCentralStartRocks(t, c.fromVersion, runBoth)

			defer mock.destroyCentral()
			mock.setVersion = setVersion

			mock.upgradeCentral(c.toVersion, "")
			if c.furtherToVersion != nil {
				mock.upgradeCentral(c.furtherToVersion, "")
				if c.moreFurtherToVersion != nil {
					mock.upgradeCentral(c.moreFurtherToVersion, "")

					// Now try to go back to Rocks and make sure that fails
					// Turn Postgres back off
					require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(false)))
					mock.runMigrator("", "", true)

					// Turn Postgres back on
					require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(true)))
				}
			}
		})
	}
}

func createAndRunCentral(t *testing.T, ver *versionPair, runBoth bool) *mockCentral {
	mock := createCentral(t, runBoth)
	mock.setVersion = setVersion
	mock.setVersion(t, ver)
	mock.runMigrator("", "", false)
	mock.runCentral()
	return mock
}

// createAndRunCentralStartRocks - creates a central that has both Rocks and Postgres but it only
// starts Rocks to help simulate the condition of having a Rocks and then upgrading to Postgres.
func createAndRunCentralStartRocks(t *testing.T, ver *versionPair, runBoth bool) *mockCentral {
	mock := createCentral(t, runBoth)
	mock.setVersion = setVersion
	mock.setVersion(t, ver)
	// First get a Rocks up and current.  This way when we do the next upgrade we should get a previous rocks.
	require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(false)))

	mock.runMigrator("", "", false)
	mock.runCentral()

	// Turn Postgres back on
	require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(true)))
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
			mock.upgradeCentral(c.toVersion, c.breakPoint)
			if reboot {
				mock.rebootCentral()
			}
			if c.furtherToVersion != nil {
				// Run migrator multiple times
				mock.runMigrator("", "", false)
				mock.upgradeCentral(c.furtherToVersion, "")
			}
		})
	}
}

func TestCloneRestore(t *testing.T) {
	// This will test restore for Rocks -> Rocks or Postgres -> Postgres depending on
	// the test is executed with the Postgres env variable set or not.
	testCloneRestore(t, false)

	// Test restore again for the case of restoring Rocks -> Postgres
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testCloneRestore(t, true)
	}
}

func testCloneRestore(t *testing.T, rocksToPostgres bool) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description string
		toVersion   *versionPair
		breakPoint  string
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
	}

	for _, c := range testCases {
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}

		if rocksToPostgres {
			c.description = c.description + " rocksDB to Postgres"
		}

		t.Run(c.description, func(t *testing.T) {
			log.Infof("Test = %q", c.description)
			mock := createAndRunCentral(t, &preHistoryVer, rocksToPostgres)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.upgradeCentral(&currVer, "")
			mock.restoreCentral(c.toVersion, c.breakPoint, rocksToPostgres)
			if reboot {
				mock.rebootCentral()
			}
			mock.upgradeCentral(&futureVer, "")
		})
	}
}

func TestForceRollbackFailure(t *testing.T) {
	currVer = releaseVer
	doTestForceRollbackFailure(t)
	currVer = devVer
	doTestForceRollbackFailure(t)
	currVer = rcVer
	doTestForceRollbackFailure(t)
	currVer = nightlyVer
	doTestForceRollbackFailure(t)
}

func doTestForceRollbackFailure(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	var forceRollbackClone string
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		forceRollbackClone = postgres.CurrentClone
	} else {
		forceRollbackClone = rocksdb.CurrentClone
	}
	testCases := []struct {
		description             string
		forceRollback           string
		withPrevious            bool
		expectedErrorMessage    string
		postgresDevErrorMessage string
		wrongVersion            bool
	}{
		{
			description:             "without force rollback without previous",
			withPrevious:            false,
			forceRollback:           "",
			expectedErrorMessage:    metadata.ErrNoPrevious,
			postgresDevErrorMessage: metadata.ErrNoPreviousInDevEnv,
		},
		{
			description:             "with force rollback without previous",
			withPrevious:            false,
			forceRollback:           forceRollbackClone,
			expectedErrorMessage:    metadata.ErrNoPrevious,
			postgresDevErrorMessage: metadata.ErrNoPreviousInDevEnv,
		},
		{
			description:          "with force rollback with previous",
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: "",
		},
		{
			description:          "without force rollback with previous",
			withPrevious:         true,
			forceRollback:        "",
			expectedErrorMessage: metadata.ErrForceUpgradeDisabled,
		},
		{
			description:          "with force rollback with wrong previous clone",
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrPreviousMismatchWithVersions, preVer.version, currVer.version),
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
			mock.upgradeCentral(&futureVer, "")
			if !c.withPrevious {
				mock.removePreviousClone()
			}
			// Force rollback
			setVersion(t, &currVer)

			var dbm DBCloneManager

			expectedError := c.expectedErrorMessage
			if env.PostgresDatastoreEnabled.BooleanSetting() {
				source := pgtest.GetConnectionString(t)
				sourceMap, _ := pgconfig.ParseSource(source)
				config, err := pgxpool.ParseConfig(source)
				require.NoError(t, err)

				dbm = NewPostgres(mock.mountPath, c.forceRollback, config, sourceMap)

				// Since postgres version no longer makes a previous if the sequence number doesn't change
				// the error message for a dev build may differ
				if c.postgresDevErrorMessage != "" && currVer != releaseVer {
					expectedError = c.postgresDevErrorMessage
				}
			} else {
				dbm = New(mock.mountPath, c.forceRollback)
			}

			err := dbm.Scan()
			if expectedError != "" {
				assert.EqualError(t, err, expectedError)
			} else {
				assert.NoError(t, err)
				mock.rollbackCentral(&currVer, "", c.forceRollback)
			}
		})
	}
}

func TestForceRollbackRocksToPostgresFailure(t *testing.T) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		currVer = releaseVer
		doTestForceRollbackRocksToPostgresFailure(t)
		currVer = devVer
		doTestForceRollbackRocksToPostgresFailure(t)
		currVer = rcVer
		doTestForceRollbackRocksToPostgresFailure(t)
		currVer = nightlyVer
		doTestForceRollbackRocksToPostgresFailure(t)
	}
}

func doTestForceRollbackRocksToPostgresFailure(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	var forceRollbackClone string
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		forceRollbackClone = postgres.CurrentClone
	} else {
		forceRollbackClone = rocksdb.CurrentClone
	}
	testCases := []struct {
		description             string
		forceRollback           string
		withPrevious            bool
		expectedErrorMessage    string
		postgresDevErrorMessage string
		wrongVersion            bool
	}{
		{
			description:             "without force rollback without previous",
			withPrevious:            false,
			forceRollback:           "",
			expectedErrorMessage:    metadata.ErrNoPrevious,
			postgresDevErrorMessage: metadata.ErrNoPreviousInDevEnv,
		},
		{
			description:             "with force rollback without previous",
			withPrevious:            false,
			forceRollback:           forceRollbackClone,
			expectedErrorMessage:    metadata.ErrNoPrevious,
			postgresDevErrorMessage: metadata.ErrNoPreviousInDevEnv,
		},
		{
			description:          "force rollback with previous",
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: "",
		},
		{
			description:          "without force rollback with previous",
			withPrevious:         true,
			forceRollback:        "",
			expectedErrorMessage: metadata.ErrForceUpgradeDisabled,
		},
		{
			description:          "with force rollback with wrong previous clone",
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: fmt.Sprintf(metadata.ErrPreviousMismatchWithVersions, preVer.version, currVer.version),
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
			mock.upgradeCentral(&futureVer, "")
			if !c.withPrevious {
				mock.removePreviousClone()
			}
			// Force rollback
			setVersion(t, &currVer)

			var dbm DBCloneManager

			expectedError := c.expectedErrorMessage

			if env.PostgresDatastoreEnabled.BooleanSetting() {
				source := pgtest.GetConnectionString(t)
				sourceMap, _ := pgconfig.ParseSource(source)
				config, err := pgxpool.ParseConfig(source)
				require.NoError(t, err)

				dbm = NewPostgres(mock.mountPath, c.forceRollback, config, sourceMap)

				if c.postgresDevErrorMessage != "" && currVer != releaseVer {
					expectedError = c.postgresDevErrorMessage
				}
			} else {
				dbm = New(mock.mountPath, c.forceRollback)
			}

			err := dbm.Scan()
			if expectedError != "" {
				assert.EqualError(t, err, expectedError)
			} else {
				assert.NoError(t, err)
				mock.rollbackCentral(&currVer, "", c.forceRollback)
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
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")
			mock.rollbackCentral(c.toVersion, "", "")
			mock.upgradeCentral(c.fromVersion, "")
		})
	}
}

// TestRollbackPostgresToRocks - set of tests that will test rolling back to Rocks from Postgres.
func TestRollbackPostgresToRocks(t *testing.T) {
	// Run tests with both Rocks and Postgres to make sure migration clone is correctly determined.
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		currVer = releaseVer
		doTestRollbackPostgresToRocks(t)
		currVer = devVer
		doTestRollbackPostgresToRocks(t)
		currVer = rcVer
		doTestRollbackPostgresToRocks(t)
		currVer = nightlyVer
		doTestRollbackPostgresToRocks(t)
	}
}

func doTestRollbackPostgresToRocks(t *testing.T) {
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
			fromVersion: &postgresDBVer,
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
			fromVersion: &postgresDBVer,
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
			fromVersion: &postgresDBVer,
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
			fromVersion: &postgresDBVer,
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
			mock := createAndRunCentralStartRocks(t, c.toVersion, true)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")

			// Turn Postgres back off so we will rollback to Rocks
			require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(false)))

			mock.rollbackCentral(c.toVersion, "", c.toVersion.version)
			mock.upgradeCentral(c.fromVersion, "")

			// We turned off Postgres.  That means we are testing
			// rollback from Postgres to Rocks.  So we need to turn Postgres back on for the next run.
			require.NoError(t, os.Setenv(env.PostgresDatastoreEnabled.EnvVar(), strconv.FormatBool(true)))
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
				m.migrateWithVersion(&futureVer, breakBeforePersist, "")
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
				mock.upgradeCentral(&currVer, "")
				c.preRun(mock)
				mock.runMigratorWithBreaksInPersist(breakpoint)
				mock.rebootCentral()
			})
		}

		for _, breakpoint := range []string{breakBeforeCleanUp, breakBeforeCommitCurrent, breakAfterRemove} {
			desc := c.description + " at " + breakpoint
			run(desc, breakpoint)
		}
	}
}

func TestUpgradeFromLastRocksDB(t *testing.T) {
	t.Skip("ROX-15123: Skip Rollback to RocksDB test")
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description    string
		previousVerion *versionPair
		fromVersion    *versionPair
		toVersion      *versionPair
	}{
		{
			description:    "Upgrade from fresh install of 3.74",
			previousVerion: nil,
			fromVersion:    &lastLegacyDBVer,
			toVersion:      &postgresDBVer,
		},
		{
			description:    "Upgrade from 3.74 with previous",
			previousVerion: &preVer,
			fromVersion:    &lastLegacyDBVer,
			toVersion:      &postgresDBVer,
		},
		{
			// We require upgrade from 3.74 to Postgres. This is not a recommended upgrade path, but
			// internally we still need to test the data on stackrox-db PVC is in valid state,
			// so we can recover from it.
			description: "Upgrade directly from earlier versions",
			fromVersion: &preVer,
			toVersion:   &postgresDBVer,
		},
	}

	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			startVer := c.fromVersion
			if c.previousVerion != nil {
				startVer = c.previousVerion
			}
			mock := createAndRunCentralStartRocks(t, startVer, true)
			defer mock.destroyCentral()
			mock.setVersion = setVersion

			if c.previousVerion != nil {
				mock.legacyUpgrade(t, c.fromVersion)
			}

			mock.upgradeCentral(c.toVersion, "")
			mock.verifyCurrent()
			if strings.HasPrefix(c.fromVersion.version, "3.74.") {
				mock.verifyClone(rocksdb.CurrentClone, &versionPair{version: c.fromVersion.version, seqNum: migrations.LastRocksDBVersionSeqNum()})
			} else {
				mock.verifyClone(rocksdb.CurrentClone, &versionPair{version: "3.74.0", seqNum: migrations.LastRocksDBVersionSeqNum()})
			}
			if c.previousVerion != nil {
				mock.verifyClone(rocksdb.PreviousClone, &versionPair{version: c.previousVerion.version, seqNum: c.previousVerion.seqNum})
			}
		})
	}
}
