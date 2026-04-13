//go:build sql_integration

package app

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
	pkgSchema.ApplySchemaForTable(s.ctx, s.gormDB, pkgSchema.VersionsSchema().Table)
	migVer.SetVersion(s.ctx, s.gormDB, &storage.Version{
		SeqNum:  int32(seqNum),
		Version: version,
	}, false)
}

func (s *UpgradeSuite) TestLockNotAcquired_OldPodProceeds() {
	// DB is at higher version, and the lock is held by another instance.
	// Old pod should proceed without error and without modifying the DB.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum+5, "4.10.0")

	// Acquire the lock to simulate another instance holding it.
	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired, "Lock should have been acquired.")
	defer release()

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	// DB version should remain unchanged.
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(currSeqNum+5, ver.SeqNum)
}

func (s *UpgradeSuite) TestLockAcquired_OldPodOverwritesSeqNum() {
	// DB is ahead of this binary, lock can be acquired.
	// Old version should overwrite SeqNum to its current SeqNum.
	curSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	futureSeqNum := curSeqNum + 10
	s.setDBVersion(futureSeqNum, "99.0.0")

	err := upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(curSeqNum, afterUpgrade.SeqNum)
}

func (s *UpgradeSuite) TestFreshInstall() {
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(0, ver.SeqNum)
	s.Require().Equal("0", ver.MainVersion)

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	afterUpgrade, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.Require().Equal(currSeqNum, afterUpgrade.SeqNum)
}

func (s *UpgradeSuite) TestNewPodUpgrade() {
	// Old pod is running at current seqnum. New pod starts, acquires lock,
	// upgrades successfully.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum, "4.9.0")

	err := upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().NoError(err)

	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Require().Equal(currSeqNum, ver.SeqNum)
}

func (s *UpgradeSuite) TestLockNotAcquired_NewPodFailsFast() {
	// DB is at lower version, and the lock is held by another instance.
	// New pod should fail fast.
	currSeqNum := pkgMigrations.CurrentDBVersionSeqNum()
	s.setDBVersion(currSeqNum-1, "4.8.0")

	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	defer release()

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().Error(err)
}
