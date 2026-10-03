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
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
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
	oldVersion := version.GetMainVersion()
	s.T().Cleanup(func() { versiontest.SetMainVersion(s.T(), oldVersion) })
	versiontest.SetMainVersion(s.T(), "5.1.0")
	s.ctx = sac.WithAllAccess(context.Background())

	s.source = pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(s.source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.Require().NoError(err)
	s.pool = pool

	s.gormDB = pgtest.OpenGormDB(s.T(), s.source)
	s.T().Cleanup(func() { pgtest.CloseGormDB(s.T(), s.gormDB) })
}

func (s *UpgradeSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *UpgradeSuite) setDBVersion(seqNum int, version string) {
	pkgSchema.ApplySchemaForTable(s.ctx, s.gormDB, pkgSchema.VersionsSchema.Table)
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
	s.setDBVersion(currSeqNum, "4.10.0")

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
	s.setDBVersion(currSeqNum-1, "4.10.0")

	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	defer release()

	err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "could not acquire migration lock")
}

func (s *UpgradeSuite) TestCompatibilityGate() {
	for name, tc := range map[string]struct {
		source string
		held   bool
	}{
		"old same sequence":  {source: "4.9.0"},
		"old with lock held": {source: "4.9.0", held: true},
		"unknown source":     {source: ""},
	} {
		s.Run(name, func() {
			s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), tc.source)
			before, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
			s.Require().NoError(err)
			if tc.held {
				acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
				s.Require().NoError(err)
				s.Require().True(acquired)
				defer release()
			}
			err = upgradeAcquireLock(s.pool, s.gormDB, s.source)
			var rejected *pkgMigrations.CompatibilityError
			s.Require().ErrorAs(err, &rejected)
			after, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
			s.Require().NoError(err)
			s.Equal(before, after)
			s.False(s.gormDB.Migrator().HasTable("clusters"))
		})
	}
}

func (s *UpgradeSuite) TestCompatibilityRecheckedUnderLock() {
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "4.10.0")
	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	defer release()
	// Simulate a metadata change after the initial read but before the locked read.
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "4.9.0")
	err = upgradeWithLock(s.ctx, s.pool, s.gormDB, s.source)
	s.Require().ErrorContains(err, "4.9")
	s.False(s.gormDB.Migrator().HasTable("clusters"))
}

func (s *UpgradeSuite) TestUnsafeUpgradeOverride() {
	s.T().Setenv("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", "true")
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "")
	s.Require().NoError(upgradeAcquireLock(s.pool, s.gormDB, s.source))
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Equal("5.1.0", ver.MainVersion)
	s.Equal(220, ver.MinimumSeqNum)
}

func (s *UpgradeSuite) TestOverrideCannotBypassMissingMigrations() {
	s.T().Setenv("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", "true")
	s.setDBVersion(199, "4.4.0")
	s.Require().ErrorContains(upgradeAcquireLock(s.pool, s.gormDB, s.source), "no migration found")
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Equal(199, ver.SeqNum)
	s.False(s.gormDB.Migrator().HasTable("clusters"))
}

func (s *UpgradeSuite) TestRejectedUpgradeReleasesLock() {
	s.setDBVersion(199, "4.4.0")
	s.T().Setenv("ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE", "true")
	s.Require().Error(upgradeAcquireLock(s.pool, s.gormDB, s.source))
	acquired, release, err := lock.TryAcquireMigrationLock(s.ctx, s.pool)
	s.Require().NoError(err)
	s.Require().True(acquired)
	release()
}

func (s *UpgradeSuite) TestSchemaFailureDoesNotPublishTargetVersion() {
	s.setDBVersion(pkgMigrations.CurrentDBVersionSeqNum(), "4.10.0")
	s.Require().PanicsWithValue("schema failure", func() {
		_ = upgradeWithLockAndSchema(s.ctx, s.pool, s.gormDB, s.source, func(context.Context, *gorm.DB) {
			panic("schema failure")
		})
	})
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Equal("4.10.0", ver.MainVersion)
}

func (s *UpgradeSuite) TestInterruptedFreshInstallRemainsFresh() {
	s.Require().PanicsWithValue("interrupted schema setup", func() {
		_ = upgradeWithLockAndSchema(s.ctx, s.pool, s.gormDB, s.source, func(ctx context.Context, db *gorm.DB) {
			s.Require().NoError(db.WithContext(ctx).Exec("CREATE TABLE partial_install (id integer)").Error)
			panic("interrupted schema setup")
		})
	})
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Zero(ver.SeqNum)
	s.False(s.gormDB.Migrator().HasTable("partial_install"))
}

func (s *UpgradeSuite) TestRollbackPreservesMinimumSequence() {
	migVer.SetVersion(s.ctx, s.gormDB, &storage.Version{
		Version: "5.2.0", SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNum() + 1), MinSeqNum: 225,
	}, true)
	s.Require().NoError(upgradeAcquireLock(s.pool, s.gormDB, s.source))
	ver, err := migVer.ReadVersionGormDB(s.ctx, s.gormDB)
	s.Require().NoError(err)
	s.Equal(pkgMigrations.CurrentDBVersionSeqNum(), ver.SeqNum)
	s.Equal("5.1.0", ver.MainVersion)
	s.Equal(225, ver.MinimumSeqNum, "rollback must not reopen an unsafe older rollback path")
}
