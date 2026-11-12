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
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/premain"
	"k8s.io/apimachinery/pkg/util/wait"
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
	var ready atomic.Bool
	startHealthServer(&ready)

	globaldb.InitializePostgresWithPoolSize(ctx, int32(poolVal))
	defer globaldb.Close()
	log.Infof("DB pool initialized with max_conns=%d", poolVal)

	if err := waitForMigrations(ctx, globaldb.GetPostgres(), 10*time.Second); err != nil {
		log.Infof("Stopping migration wait: %v", err)
		return
	}
	log.Infof("Migrations complete, proceeding with startup")

	go startMetricsServer()

	scheduler := vulnReportV2Scheduler.Singleton()
	scheduler.Start(globaldb.GetPostgres())
	log.Infof("Vulnerability report scheduler started")

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
	rl.start(ctx)
	log.Infof("Report LISTEN/NOTIFY listener started")

	ready.Store(true)
	log.Infof("central-worker is ready")

	<-ctx.Done()
	ready.Store(false)

	log.Infof("central-worker shutting down")

	scheduler.Stop()
}

// waitForMigrations waits for startup migrations, regardless of their duration.
// Background migrations use a separate lock and do not block Worker startup.
func waitForMigrations(ctx context.Context, pool postgres.DB, interval time.Duration) error {
	return wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		if err := checkMigrations(ctx, pool); err != nil {
			log.Infof("Migrations not yet complete, waiting: %v", err)
			return false, nil
		}
		return true, nil
	})
}

func checkMigrations(ctx context.Context, pool postgres.DB) error {
	acquired, release, err := dblock.TryAcquireAdvisoryLock(ctx, pool, dblock.MigrationLockID)
	if err != nil {
		return err
	}
	if !acquired {
		return errors.New("migrator is still running")
	}
	defer release()
	return version.Ensure(vStore.NewPostgres(pool))
}

func newHealthHandler(ready *atomic.Bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func startHealthServer(ready *atomic.Bool) {
	srv := &http.Server{
		Addr:    healthAddr,
		Handler: newHealthHandler(ready),
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
