package replica

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/stackrox/pkg/buildinfo"
	migrationtestutils "github.com/stackrox/stackrox/pkg/migrations/testutils"
	"github.com/stackrox/stackrox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	preHistoryVer = versionPair{version: "3.0.56.0", seqNum: 62}
	preVer        = versionPair{version: "3.0.57.0", seqNum: 65}
	currVer       = versionPair{version: "3.0.58.0", seqNum: 65}
	futureVer     = versionPair{version: "10001.0.0.0", seqNum: 6533}

	// Current versions
	rcVer      = versionPair{version: "3.0.58.0-rc.1", seqNum: 65}
	releaseVer = versionPair{version: "3.0.58.0", seqNum: 65}
	devVer     = versionPair{version: "3.0.58.x-19-g6bd31dae22-dirty", seqNum: 65}
	nightlyVer = versionPair{version: "3.0.58.x-nightly-20210407", seqNum: 65}
)

func setVersion(t *testing.T, ver *versionPair) {
	testutils.SetMainVersion(t, ver.version)
	migrationtestutils.SetCurrentDBSequenceNumber(t, ver.seqNum)
}

func TestReplicaMigration(t *testing.T) {
	currVer = releaseVer
	doTestReplicaMigration(t)
	currVer = devVer
	doTestReplicaMigration(t)
	currVer = rcVer
	doTestReplicaMigration(t)
	currVer = nightlyVer
	doTestReplicaMigration(t)
}

func doTestReplicaMigration(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description      string
		fromVersion      *versionPair
		toVersion        *versionPair
		furtherToVersion *versionPair
		enableRollback   bool
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
			description:      "Upgrade from early versions to current with rollback enabled",
			fromVersion:      &preHistoryVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			enableRollback:   true,
		},
		{
			description:      "Upgrade from version 57 to current with rollback enabled",
			fromVersion:      &preVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			enableRollback:   true,
		},
		{
			description:    "Upgrade from current to future with rollback enabled",
			fromVersion:    &currVer,
			toVersion:      &futureVer,
			enableRollback: true,
		},
		{
			description:    "Upgrade from early version to future with rollback enabled",
			fromVersion:    &preHistoryVer,
			toVersion:      &futureVer,
			enableRollback: true,
		},
	}

	// Test normal upgrade
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			mock := createAndRunCentral(t, c.fromVersion)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.enableRollBack(c.enableRollback)
			mock.upgradeCentral(c.toVersion, "")
			if c.furtherToVersion != nil {
				mock.upgradeCentral(c.furtherToVersion, "")
			}
		})
	}
}

func createAndRunCentral(t *testing.T, ver *versionPair) *mockCentral {
	mock := createCentral(t)
	mock.setVersion = setVersion
	mock.setVersion(t, ver)
	mock.runMigrator("", "")
	mock.runCentral()
	return mock
}

func TestReplicaMigrationFailureAndReentry(t *testing.T) {
	currVer = releaseVer
	doTestReplicaMigrationFailureAndReentry(t)
	currVer = devVer
	doTestReplicaMigrationFailureAndReentry(t)
	currVer = rcVer
	doTestReplicaMigrationFailureAndReentry(t)
	currVer = nightlyVer
	doTestReplicaMigrationFailureAndReentry(t)
}

func doTestReplicaMigrationFailureAndReentry(t *testing.T) {
	if buildinfo.ReleaseBuild {
		return
	}
	testCases := []struct {
		description      string
		fromVersion      *versionPair
		toVersion        *versionPair
		furtherToVersion *versionPair
		enableRollback   bool
		breakPoint       string
	}{
		{
			description:      "Upgrade from early versions to current break after scan",
			fromVersion:      &preHistoryVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			breakPoint:       breakAfterScan,
		},
		{
			description:      "Upgrade from version 57 to current break after getting replica",
			fromVersion:      &preVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			breakPoint:       breakAfterGetReplica,
		},
		{
			description: "Upgrade from current to future break after db migration",
			fromVersion: &currVer,
			toVersion:   &futureVer,
			breakPoint:  breakBeforePersist,
		},
		{
			description:      "Upgrade from early versions to current rollback enabled break after db migration",
			fromVersion:      &preHistoryVer,
			toVersion:        &currVer,
			furtherToVersion: &futureVer,
			enableRollback:   true,
			breakPoint:       breakBeforePersist,
		},
		{
			description:    "Upgrade from version 57 to current rollback enabled break after getting replica",
			fromVersion:    &preVer,
			toVersion:      &currVer,
			enableRollback: true,
			breakPoint:     breakAfterGetReplica,
		},
		{
			description:    "Upgrade from current to future enable rollback enabled break after scan",
			fromVersion:    &currVer,
			toVersion:      &futureVer,
			enableRollback: true,
			breakPoint:     breakAfterScan,
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
			mock := createAndRunCentral(t, c.fromVersion)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.enableRollBack(c.enableRollback)
			// Migration aborted
			mock.upgradeCentral(c.toVersion, c.breakPoint)
			if reboot {
				mock.rebootCentral()
			}
			if c.furtherToVersion != nil {
				// Run migrator multiple times
				mock.runMigrator("", "")
				mock.upgradeCentral(c.furtherToVersion, "")
			}
		})
	}
}

func TestReplicaRestore(t *testing.T) {
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
			description: "Restore to earlier version break after get replica",
			toVersion:   &preHistoryVer,
			breakPoint:  breakAfterGetReplica,
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
			description: "Restore to current versions break after get replica",
			toVersion:   &currVer,
			breakPoint:  breakAfterGetReplica,
		},
		{
			description: "Restore to earlier versions break after scan",
			toVersion:   &currVer,
			breakPoint:  breakBeforePersist,
		},
	}

	// For the parameters that should not matter, run pseudo random to get coverage on different cases
	rand.Seed(888)
	for _, c := range testCases {
		enableRollback := rand.Intn(2) == 1
		if enableRollback {
			c.description = c.description + " rollback enabled"
		}
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}

		t.Run(c.description, func(t *testing.T) {
			mock := createAndRunCentral(t, &preHistoryVer)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.enableRollBack(enableRollback)
			mock.upgradeCentral(&currVer, "")
			mock.restoreCentral(c.toVersion, c.breakPoint)
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
	testCases := []struct {
		description          string
		rollbackEnabled      bool
		forceRollback        string
		withPrevious         bool
		expectedErrorMessage string
		wrongVersion         bool
	}{
		{
			description:          "Rollback disabled without force rollback without previous",
			rollbackEnabled:      false,
			withPrevious:         false,
			forceRollback:        "",
			expectedErrorMessage: errNoPrevious,
		},
		{
			description:          "Rollback disabled with force rollback without previous",
			rollbackEnabled:      false,
			withPrevious:         false,
			forceRollback:        currVer.version,
			expectedErrorMessage: errNoPrevious,
		},
		{
			description:          "Rollback disabled without force rollback with previous",
			rollbackEnabled:      false,
			withPrevious:         true,
			forceRollback:        "",
			expectedErrorMessage: errForceUpgradeDisabled,
		},
		{
			description:          "Rollback disabled with force rollback with wrong previous replica",
			rollbackEnabled:      false,
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: fmt.Sprintf(errPreviousMismatchWithVersions, preVer.version, currVer.version),
			wrongVersion:         true,
		},
		{
			description:          "Rollback enabled without force rollback without previous",
			rollbackEnabled:      true,
			withPrevious:         false,
			forceRollback:        "",
			expectedErrorMessage: errNoPrevious,
		},
		{
			description:          "Rollback enabled with force rollback without previous",
			rollbackEnabled:      true,
			withPrevious:         false,
			forceRollback:        currentReplica,
			expectedErrorMessage: errNoPrevious,
		},
		{
			description:          "Rollback enabled with force rollback with previous",
			rollbackEnabled:      true,
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: "",
		},
		{
			description:          "Rollback enabled without force rollback with previous",
			rollbackEnabled:      true,
			withPrevious:         true,
			forceRollback:        "",
			expectedErrorMessage: errForceUpgradeDisabled,
		},
		{
			description:          "Rollback enabled with force rollback with wrong previous replica",
			rollbackEnabled:      true,
			withPrevious:         true,
			forceRollback:        currVer.version,
			expectedErrorMessage: fmt.Sprintf(errPreviousMismatchWithVersions, preVer.version, currVer.version),
			wrongVersion:         true,
		},
	}
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			ver := &currVer
			if c.wrongVersion {
				ver = &preVer
			}
			mock := createAndRunCentral(t, ver)
			defer mock.destroyCentral()
			mock.enableRollBack(c.withPrevious)
			mock.upgradeCentral(&futureVer, "")
			// Force rollback
			mock.enableRollBack(c.rollbackEnabled)
			setVersion(t, &currVer)
			_, err := Scan(mock.mountPath, c.forceRollback)
			if c.expectedErrorMessage != "" {
				assert.EqualError(t, err, c.expectedErrorMessage)
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
			description: "Rollback to current break after get replica",
			fromVersion: &futureVer,
			toVersion:   &currVer,
			breakPoint:  breakAfterGetReplica,
		},
		{
			description: "Rollback to version 57 break after get replica",
			fromVersion: &currVer,
			toVersion:   &preVer,
			breakPoint:  breakAfterGetReplica,
		},
	}
	rand.Seed(8056)
	for _, c := range testCases {
		reboot := rand.Intn(2) == 1
		if reboot {
			c.description = c.description + " with reboot"
		}

		t.Run(c.description, func(t *testing.T) {
			mock := createAndRunCentral(t, c.toVersion)
			defer mock.destroyCentral()
			mock.setVersion = setVersion
			mock.enableRollBack(true)
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")
			mock.migrateWithVersion(c.fromVersion, c.breakPoint, "")
			mock.rollbackCentral(c.toVersion, "", "")
			mock.upgradeCentral(c.fromVersion, "")
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
				m.restore(&preVer)
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
				mock := createAndRunCentral(t, &preVer)
				defer mock.destroyCentral()
				mock.enableRollBack(true)
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
