//go:build sql_integration

package main

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/lock"
	migVer "github.com/stackrox/rox/migrator/version"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type UpgradeSuite struct {
	suite.Suite
	pool   postgres.DB
	source string
	gormDB *gorm.DB
	ctx    context.Context
}

func TestUpgradeSuite(t *testing.T) {
	suite.Run(t, new(UpgradeSuite))
}

func (s *UpgradeSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())

	s.source = pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(s.source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.pool = pool

	s.gormDB = pgtest.OpenGormDB(s.T(), s.source)
}

func (s *UpgradeSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *UpgradeSuite) setDBVersion(seqNum int, version string) {
	pkgSchema.ApplySchemaForTable(s.ctx, s.gormDB, pkgSchema.VersionsSchema.Table)
	migVer.SetVersionGormDB(s.ctx, s.gormDB, &storage.Version{
		SeqNum:  int32(seqNum),
		Version: version,
	}, false)
}

func (s *UpgradeSuite) TestLockNotAcquired_OldPodWritesRollbackMarker() {
	// Simulate: DB is at higher version, and the lock is held by another instance.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum+5, "4.10.0")

	// Acquire the lock to simulate another instance holding it.
	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired, "Lock should have been acquired.")
	defer release()

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	// Verify marker was written by re-reading the version.
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().NotZero(ver.RollbackSeqNum)
	s.Require().Equal(currSeqNum, ver.RollbackSeqNum)
}

func (s *UpgradeSuite) TestLockAcquired_RollbackMarkerHonored() {
	// Simulate: A previous upgrade failed. DB is at seqnum 218, rollback marker is 215.
	s.setDBVersion(218, "4.10.0")
	err := migVer.WriteRollbackSeqNum(s.gormDB, 215)
	s.Require().NoError(err)

	// Read the version — marker should be 215.
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().NotZero(ver.RollbackSeqNum)
	s.Require().Equal(215, ver.RollbackSeqNum)
	s.Require().Equal(218, ver.SeqNum)

	checkAndResetRollbackMarker(s.ctx, s.pool, ver)
	// The check should set SeqNum to the RollbackSeqNum and remove the marker
	verAfter, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Assert().Zero(verAfter.RollbackSeqNum)
	s.Assert().Equal(215, verAfter.SeqNum)
	// Make sure other fields are not wiped by the reset
	s.Assert().Equal(ver.MainVersion, verAfter.MainVersion)
	s.Assert().Equal(ver.MinimumSeqNum, verAfter.MinimumSeqNum)
}

func (s *UpgradeSuite) TestLockAcquired_OldPod_ResetSeqAndRollback() {
	// DB is ahead of this binary, lock can be acquired.
	// Old version should reset SeqNum to it's current SeqNum and remove rollback markers.
	curSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	futureSeqNum := curSeqNum + 10
	s.setDBVersion(futureSeqNum, "99.0.0")
	err := migVer.WriteRollbackSeqNum(s.gormDB, curSeqNum)
	s.Require().NoError(err)

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(curSeqNum, afterUpgrade.SeqNum)
	s.Require().Equal(0, afterUpgrade.RollbackSeqNum)
}

func (s *UpgradeSuite) TestFreshInstall() {
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(0, ver.SeqNum)
	s.Require().Equal("0", ver.MainVersion)
	s.Require().Zero(ver.RollbackSeqNum)

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.Require().Equal(currSeqNum, afterUpgrade.SeqNum,
		"fresh install should not be treated as rollback")
	s.Require().Equal(0, afterUpgrade.RollbackSeqNum)
}

func (s *UpgradeSuite) TestNewPodUpgrade_NoRollbackMarker() {
	// Old pod is running at current seqnum. New pod starts, acquires lock,
	// upgrades successfully. No rollback marker should be set.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum, "4.9.0")

	err := upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(currSeqNum, ver.SeqNum)
	s.Require().Zero(ver.RollbackSeqNum, "no rollback marker should be set after successful upgrade")
}

func (s *UpgradeSuite) TestNewPodUpgrade_ClearsRollbackMarker() {
	// A previous upgrade failed, leaving a rollback marker. New pod starts,
	// acquires lock, resets seqnum to marker, runs migrations, and clears marker.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum+5, "4.10.0")
	err := migVer.WriteRollbackSeqNum(s.gormDB, currSeqNum)
	s.Require().NoError(err)

	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(currSeqNum, ver.RollbackSeqNum)

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	verAfter, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Zero(verAfter.RollbackSeqNum,
		"rollback marker should be cleared after successful upgrade")
	s.Require().Equal(currSeqNum, verAfter.SeqNum,
		"seqnum should match current binary version after upgrade")
}

func (s *UpgradeSuite) TestWriteRollbackMarker_LowestWins() {
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "4.10.0")

	// First write: 215
	err := migVer.WriteRollbackSeqNum(s.gormDB, 215)
	s.Require().NoError(err)

	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().NotZero(ver.RollbackSeqNum)
	s.Require().Equal(215, ver.RollbackSeqNum)

	// Second write with a higher value: should NOT overwrite.
	err = migVer.WriteRollbackSeqNum(s.gormDB, 220)
	s.Require().NoError(err)

	ver, err = migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().NotZero(ver.RollbackSeqNum)
	s.Require().Equal(215, ver.RollbackSeqNum, "lower marker should be preserved")

	// Third write with a lower value: should overwrite.
	err = migVer.WriteRollbackSeqNum(s.gormDB, 210)
	s.Require().NoError(err)

	ver, err = migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().NotZero(ver.RollbackSeqNum)
	s.Require().Equal(210, ver.RollbackSeqNum, "even lower marker should win")
}
