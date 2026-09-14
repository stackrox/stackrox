package main

import (
	"context"
	"errors"
	"math"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/stackrox/rox/central/globaldb"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	_ "github.com/stackrox/rox/central/notifiers/all"
	"github.com/stackrox/rox/central/pruning"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	vulnReportV2Scheduler "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/central/version"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	healthAddr                = ":8082"
	defaultWorkerPoolMaxConns = 20
)

var (
	log            = logging.LoggerForModule()
	workerPoolSize = env.RegisterIntegerSetting("ROX_WORKER_DB_POOL_MAX_CONNS", defaultWorkerPoolMaxConns)
)

func main() {
	premain.StartMain()

	log.Infof("Starting central-worker")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	poolVal := workerPoolSize.IntegerSetting()
	if poolVal < 1 || poolVal > math.MaxInt32 {
		log.Fatalf("ROX_WORKER_DB_POOL_MAX_CONNS must be between 1 and %d, got %d", math.MaxInt32, poolVal)
	}
	globaldb.InitializePostgresWithPoolSize(ctx, int32(poolVal))
	defer globaldb.Close()
	log.Infof("DB pool initialized with max_conns=%d", poolVal)

	waitForMigrations(ctx)
	ensureDBCurrent()

	scheduler := vulnReportV2Scheduler.Singleton()
	var listenerStarted atomic.Bool
	startHealthServer(func() bool {
		return ctx.Err() == nil && listenerStarted.Load() && scheduler.Ready()
	})

	go startMetricsServer()

	pruning.Singleton().Start()
	log.Infof("Pruning GC started")

	scheduler.Start(globaldb.GetPostgres())
	var stopListener func()
	defer func() {
		stopBackgroundTasks(scheduler.Stop, pruning.Singleton().Stop, func() {
			if stopListener != nil {
				stopListener()
			}
		})
	}()

	// Do not register cron jobs from notifications before this process owns reporting.
	for !scheduler.Ready() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}

	collectionDatastore, _ := collectionDS.Singleton()
	rl := newReportListener(
		globaldb.GetPostgres(),
		scheduler,
		reportConfigDS.Singleton(),
		reportSnapshotDS.Singleton(),
		collectionDatastore,
		notifierDS.Singleton(),
		notifierProcessor.Singleton(),
	)
	stopListener = rl.start(ctx)
	listenerStarted.Store(true)
	log.Infof("Report LISTEN/NOTIFY listener started")

	log.Infof("central-worker is ready")

	<-ctx.Done()

	log.Infof("central-worker shutting down")
}

func waitForMigrations(ctx context.Context) {
	err := retry.WithRetry(func() error {
		acquired, release, err := dblock.TryAcquireAdvisoryLock(ctx, globaldb.GetPostgres(), dblock.MigrationLockID)
		if err != nil {
			return retry.MakeRetryable(err)
		}
		if !acquired {
			return retry.MakeRetryable(errMigratorRunning)
		}
		release()
		return nil
	}, retry.Tries(30), retry.BetweenAttempts(func(attempt int) {
		log.Infof("Migrator lock held, waiting for migrations to complete (attempt %d)...", attempt+1)
		time.Sleep(10 * time.Second)
	}))
	if err != nil {
		log.Fatalf("Timed out waiting for migrations to complete: %v", err)
	}
	log.Infof("Migrations complete, proceeding with startup")
}

var errMigratorRunning = retryableError("migrator is still running")

type retryableError string

func (e retryableError) Error() string { return string(e) }

func ensureDBCurrent() {
	versionStore := vStore.NewPostgres(globaldb.GetPostgres())
	if err := version.Ensure(versionStore); err != nil {
		log.Fatalf("DB version check failed. Migrations may not be complete: %v", err)
	}
	log.Infof("DB version verified")
}

func startHealthServer(ready func() bool) {
	mux := healthHandler(ready)

	srv := &http.Server{
		Addr:    healthAddr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Fatalf("Health server failed to start: %v", err)
	case <-time.After(1 * time.Second):
	}

	go func() {
		if err := <-errCh; err != nil {
			log.Fatalf("Health server failed: %v", err)
		}
	}()
}

func startMetricsServer() {
	pkgMetrics.NewServer(pkgMetrics.CentralWorkerSubsystem, pkgMetrics.NewTLSConfigurerFromEnv()).RunForever()
	pkgMetrics.GatherThrottleMetricsForever(pkgMetrics.CentralWorkerSubsystem.String())
}

// Stop report scheduling independently of pruning so a long GC cycle does not
// consume the scheduler's time to cancel reports and release its advisory lock.
func stopBackgroundTasks(tasks ...func()) {
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Go(task)
	}
	wg.Wait()
}

func healthHandler(ready func() bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready() {
			http.Error(w, "report scheduler is not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
