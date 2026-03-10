//go:build sql_integration

package main

import (
	"context"
	"testing"
	"time"

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
	release, err := lock.AcquireMigrationLock(s.ctx, s.pool, 5*time.Second)
	s.Require().NoError(err)
	defer release()

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)

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
	s.Require().Zero(verAfter.RollbackSeqNum)
	s.Require().Equal(215, verAfter.SeqNum)
}

func (s *UpgradeSuite) TestLockAcquired_OldPod_DBahead() {
	// DB is ahead of this binary, lock can be acquired.
	// Old version should reset SeqNum to it's current SeqNum and rollback markers.
	curSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	futureSeqNum := curSeqNum + 10
	s.setDBVersion(futureSeqNum, "99.0.0")
	migVer.WriteRollbackSeqNum(s.gormDB, curSeqNum)

	err := upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(curSeqNum, afterUpgrade.SeqNum)
	s.Require().Equal(0, afterUpgrade.RollbackSeqNum)
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

func (s *UpgradeSuite) TestFreshInstall() {
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(0, ver.SeqNum)
	s.Require().Equal("0", ver.MainVersion)
	s.Require().Zero(ver.RollbackSeqNum)

	upgradeAcquireLock(s.pool, s.gormDB, s.source)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.Require().Equal(currSeqNum, afterUpgrade.SeqNum,
		"fresh install should not be treated as rollback")
	s.Require().Equal(0, afterUpgrade.RollbackSeqNum)
}

func (s *UpgradeSuite) TestClearRollbackMarker_Idempotent() {
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "4.10.0")

	// Clear when no marker — should succeed.
	err := migVer.ClearRollbackSeqNum(s.gormDB)
	s.Require().NoError(err)

	// Set and clear.
	err = migVer.WriteRollbackSeqNum(s.gormDB, 215)
	s.Require().NoError(err)

	err = migVer.ClearRollbackSeqNum(s.gormDB)
	s.Require().NoError(err)

	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Zero(ver.RollbackSeqNum)

	// Double clear — should succeed.
	err = migVer.ClearRollbackSeqNum(s.gormDB)
	s.Require().NoError(err)
}

// Test that old pod running, new pod starting successfully upgrades, no rollback set, dbSeq number set properly
// Test that old pod restarting, while new pod running migrations, produced rollbackseqnumber
// test that if the new pod finishes migration succesfully it clears rollbackseqnr
// test that the new pod starting when rollbackseqnr is set picks up migration from rollbackseqnr
// Test that old pod starting, while no new pod holds lock resets rollbackseqnumber and dbSeqNum
