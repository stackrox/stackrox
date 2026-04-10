package backgroundmigrations

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *Runner
)

// Singleton returns the singleton Runner instance.
func Singleton() *Runner {
	once.Do(func() {
		instance = NewRunner(globaldb.GetPostgres(), NewCentralRolloutChecker())
	})
	return instance
}

// Runner executes background migrations after Central is ready.
type Runner struct {
	db                  postgres.DB
	rolloutChecker      RolloutChecker
	stopper             concurrency.Stopper
	started             bool
	currentBgSeqNumFunc func() int
}

// NewRunner creates a new Runner.
func NewRunner(db postgres.DB, rolloutChecker RolloutChecker) *Runner {
	return &Runner{
		db:                  db,
		rolloutChecker:      rolloutChecker,
		stopper:             concurrency.NewStopper(),
		currentBgSeqNumFunc: func() int { return CurrentBgMigrationSeqNum },
	}
}

// Start launches the background migration goroutine.
func (r *Runner) Start() {
	r.started = true
	go r.run()
}

// Stop requests graceful shutdown and waits for the runner to finish.
func (r *Runner) Stop() {
	r.stopper.Client().Stop()
	if r.started {
		_ = r.stopper.Client().Stopped().Wait()
	}
}

func (r *Runner) run() {
	defer r.stopper.Flow().ReportStopped()
	log := logging.LoggerForModule()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-r.stopper.Flow().StopRequested()
		cancel()
	}()

	if err := r.rolloutChecker.WaitForRolloutComplete(ctx); err != nil {
		log.Infof("Background migrations: rollout check cancelled: %v", err)
		return
	}

	dbBgSeqNum, err := r.readSeqNum(ctx)
	if err != nil {
		log.Errorf("Background migrations: failed to read current seq num: %v", err)
		return
	}

	currentSeqNum := r.currentBgSeqNumFunc()

	if dbBgSeqNum > currentSeqNum {
		log.Warnf("Background migrations: rollback detected (db=%d, current=%d). Resetting to current seq num.", dbBgSeqNum, currentSeqNum)
		if err := r.writeSeqNum(ctx, currentSeqNum); err != nil {
			log.Errorf("Background migrations: failed to reset seq num: %v", err)
			return
		}
		dbBgSeqNum = currentSeqNum
	}

	if dbBgSeqNum == currentSeqNum {
		log.Infof("Background migrations: up to date at seq num %d", dbBgSeqNum)
		return
	}

	log.Infof("Background migrations: running migrations from %d to %d", dbBgSeqNum, currentSeqNum)

	for seqNum := dbBgSeqNum; seqNum < currentSeqNum; seqNum++ {
		select {
		case <-r.stopper.Flow().StopRequested():
			log.Infof("Background migrations: shutdown requested, stopping at seq num %d", seqNum)
			return
		default:
		}

		migration, ok := Get(seqNum)
		if !ok {
			log.Errorf("Background migrations: no migration found starting at %d", seqNum)
			return
		}

		log.Infof("Background migrations: running migration %d: %s", seqNum, migration.Description)

		if err := migration.Run(ctx, r.db); err != nil {
			if ctx.Err() != nil {
				log.Infof("Background migrations: migration %d cancelled during shutdown", seqNum)
				return
			}
			log.Errorf("Background migrations: migration %d failed: %v. Will retry on next restart.", seqNum, err)
			return
		}

		if err := r.writeSeqNum(ctx, migration.VersionAfterSeqNum); err != nil {
			log.Errorf("Background migrations: failed to update seq num to %d: %v", migration.VersionAfterSeqNum, err)
			return
		}

		log.Infof("Background migrations: completed migration %d, now at seq num %d", seqNum, migration.VersionAfterSeqNum)
	}

	log.Infof("Background migrations: all migrations complete, at seq num %d", currentSeqNum)
}

func (r *Runner) readSeqNum(ctx context.Context) (int, error) {
	row := r.db.QueryRow(ctx, "SELECT seqnum FROM "+schema.BackgroundMigrationVersionsTableName+" LIMIT 1")
	var seqNum int32
	if err := row.Scan(&seqNum); err != nil {
		return 0, err
	}
	return int(seqNum), nil
}

func (r *Runner) writeSeqNum(ctx context.Context, seqNum int) error {
	_, err := r.db.Exec(ctx, "UPDATE "+schema.BackgroundMigrationVersionsTableName+" SET seqnum = $1", int32(seqNum))
	return err
}
