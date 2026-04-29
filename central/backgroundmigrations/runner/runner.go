package runner

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/backgroundmigrations"
	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *Runner
)

var log = logging.LoggerForModule()

// Singleton returns the singleton Runner instance.
func Singleton() *Runner {
	once.Do(func() {
		instance = NewRunner(globaldb.GetPostgres(), NewCentralRolloutChecker())
	})
	return instance
}

const (
	// bgMigrationAdvisoryLockID is a unique identifier for the background migration advisory lock.
	// This value is arbitrary but must be consistent across all Central instances and must
	// differ from the migrator advisory lock ID.
	bgMigrationAdvisoryLockID int64 = 2_846_193_750_482_637_519

	retryInterval = 60 * time.Second
)

// Runner executes background migrations after Central is ready.
type Runner struct {
	db             postgres.DB
	rolloutChecker RolloutChecker
	stopper        concurrency.Stopper
	started        atomic.Bool
	targetSeqNum   int
	retryInterval  time.Duration
}

// NewRunner creates a new Runner.
func NewRunner(db postgres.DB, rolloutChecker RolloutChecker) *Runner {
	return &Runner{
		db:             db,
		rolloutChecker: rolloutChecker,
		stopper:        concurrency.NewStopper(),
		targetSeqNum:   backgroundmigrations.CurrentBgMigrationSeqNum,
		retryInterval:  retryInterval,
	}
}

// Start launches the background migration goroutine.
func (r *Runner) Start() {
	if !r.started.CompareAndSwap(false, true) {
		return
	}

	go r.run()
}

// Stop requests graceful shutdown and waits for the runner to finish.
func (r *Runner) Stop() {
	r.stopper.Client().Stop()
	if r.started.Load() {
		_ = r.stopper.Client().Stopped().Wait()
	}
}

func (r *Runner) run() {
	defer r.stopper.Flow().ReportStopped()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-r.stopper.Flow().StopRequested():
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		err := r.runOnce(ctx)
		if err == nil {
			return
		}

		log.Errorf("background migrations failed, retrying in %v: %v", r.retryInterval, err)
		select {
		case <-ctx.Done():
			log.Infof("background migrations stopped")
			return
		case <-time.After(r.retryInterval):
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) error {
	done, err := r.rolloutChecker.IsRolloutDone(ctx)
	if err != nil {
		return errors.Wrap(err, "rollout check")
	}
	if !done {
		return errors.New("rollout not yet complete")
	}

	release, err := r.acquireLock(ctx)
	if err != nil {
		return errors.Wrap(err, "acquiring lock")
	}
	defer release()

	if err := r.runMigrations(ctx); err != nil {
		return errors.Wrap(err, "running background migrations")
	}

	return nil
}

// acquireLock attempts to acquire the advisory lock once.
// Returns (release, nil) on success, or an error if the lock could not be acquired.
func (r *Runner) acquireLock(ctx context.Context) (func(), error) {
	acquired, release, err := dblock.TryAcquireAdvisoryLock(ctx, r.db, bgMigrationAdvisoryLockID)
	if err != nil {
		return nil, errors.Wrap(err, "acquiring advisory lock")
	}
	if !acquired {
		return nil, errors.New("advisory lock held by another instance")
	}
	return release, nil
}

func (r *Runner) runMigrations(ctx context.Context) error {
	dbSeqNum, dbOverrideTag, err := r.readState(ctx)
	if err != nil {
		return errors.Wrap(err, "reading current state")
	}

	overrideSeqNum, overrideTag, shouldOverride := r.checkSeqNumOverrideConfig(r.targetSeqNum, dbOverrideTag)
	if shouldOverride {
		log.Infof("applying override tag %q, resetting seq num from %d to %d", overrideTag, dbSeqNum, overrideSeqNum)
		if err := r.writeState(ctx, overrideSeqNum, overrideTag); err != nil {
			return errors.Wrap(err, "writing override state")
		}
		dbSeqNum = overrideSeqNum
	}

	if !shouldOverride && dbOverrideTag != "" && overrideTag == "" {
		// reset old override tags if it exists
		log.Infof("override env var removed, clearing stale override tag %q from DB", dbOverrideTag)
		if err := r.writeOverrideTag(ctx, ""); err != nil {
			return errors.Wrap(err, "clearing override tag")
		}
	}

	if dbSeqNum > r.targetSeqNum {
		log.Warnf("rollback detected (db=%d, current=%d). Resetting to current seq num.", dbSeqNum, r.targetSeqNum)
		if err := r.writeSeqNum(ctx, r.targetSeqNum); err != nil {
			return errors.Wrap(err, "resetting seq num after rollback")
		}
		dbSeqNum = r.targetSeqNum
	}

	if dbSeqNum == r.targetSeqNum {
		log.Infof("up to date at seq num %d", dbSeqNum)
		return nil
	}

	log.Infof("running migrations from %d to %d", dbSeqNum, r.targetSeqNum)

	for seqNum := dbSeqNum; seqNum < r.targetSeqNum; seqNum++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		migration, ok := migrations.Get(seqNum)
		if !ok {
			return errors.Errorf("no migration found starting at %d", seqNum)
		}

		log.Infof("running migration %d: %s", seqNum, migration.Description)

		if err := migration.Run(ctx, r.db); err != nil {
			return errors.Wrapf(err, "migration %d failed", seqNum)
		}

		if err := r.writeSeqNum(ctx, migration.VersionAfterSeqNum); err != nil {
			return errors.Wrapf(err, "updating seq num to %d", migration.VersionAfterSeqNum)
		}

		log.Infof("completed migration %d, now at seq num %d", seqNum, migration.VersionAfterSeqNum)
	}

	log.Infof("all migrations complete, at seq num %d", r.targetSeqNum)
	return nil
}

func (r *Runner) readState(ctx context.Context) (int, string, error) {
	row := r.db.QueryRow(ctx, "SELECT seqnum, override_tag FROM "+schema.BackgroundMigrationVersionsTableName+" LIMIT 1")
	var seqNum int32
	var overrideTag string
	if err := row.Scan(&seqNum, &overrideTag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := r.seedInitialRow(ctx); err != nil {
				return 0, "", errors.Wrap(err, "seeding initial row")
			}
			return 0, "", nil
		}
		return 0, "", err
	}
	return int(seqNum), overrideTag, nil
}

func (r *Runner) seedInitialRow(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "INSERT INTO "+schema.BackgroundMigrationVersionsTableName+" (seqnum, override_tag) VALUES (0, '')")
	return err
}

func (r *Runner) writeSeqNum(ctx context.Context, seqNum int) error {
	_, err := r.db.Exec(ctx, "UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = $1", int32(seqNum))
	return err
}

func (r *Runner) writeOverrideTag(ctx context.Context, overrideTag string) error {
	_, err := r.db.Exec(ctx, "UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET override_tag = $1", overrideTag)
	return err
}

func (r *Runner) writeState(ctx context.Context, seqNum int, overrideTag string) error {
	_, err := r.db.Exec(ctx, "UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = $1, override_tag = $2", int32(seqNum), overrideTag)
	return err
}

// checkSeqNumOverrideConfig checks env configuration for sequence number overrides and applies them.
// returns the configuration and whether it needs to be applied
func (r *Runner) checkSeqNumOverrideConfig(currSeqNum int, dbOverrideTag string) (seqNum int, tag string, shouldOverride bool) {
	seqNum = env.BackgroundMigrationOverrideSeqNum.IntegerSetting()
	tag = env.BackgroundMigrationOverrideTag.Setting()

	if seqNum < 0 || tag == "" || tag == dbOverrideTag {
		return seqNum, tag, shouldOverride
	}

	if seqNum >= currSeqNum {
		log.Infof("override background seq num %d is greater or equal current seq num %d, ignoring override", seqNum, currSeqNum)
		return seqNum, tag, shouldOverride
	}

	return seqNum, tag, true
}
