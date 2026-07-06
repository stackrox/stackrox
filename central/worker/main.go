package main

import (
	"context"
	"errors"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/pruning"
	"github.com/stackrox/rox/central/version"
	vStore "github.com/stackrox/rox/central/version/store"
	migratorLock "github.com/stackrox/rox/migrator/lock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/retry"
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

	ctx := context.Background()

	poolVal := workerPoolSize.IntegerSetting()
	if poolVal < 1 || poolVal > math.MaxInt32 {
		log.Fatalf("ROX_WORKER_DB_POOL_MAX_CONNS must be between 1 and %d, got %d", math.MaxInt32, poolVal)
	}
	globaldb.InitializePostgresWithPoolSize(ctx, int32(poolVal))
	log.Infof("DB pool initialized with max_conns=%d", poolVal)

	waitForMigrations(ctx)
	ensureDBCurrent()

	startHealthServer()

	go startMetricsServer()

	pruning.Singleton().Start()
	log.Infof("Pruning GC started")

	log.Infof("central-worker is ready")

	waitForTerminationSignal()

	log.Infof("central-worker shutting down")

	pruning.Singleton().Stop()

	globaldb.Close()
}

func waitForMigrations(ctx context.Context) {
	err := retry.WithRetry(func() error {
		acquired, release, err := migratorLock.TryAcquireMigrationLock(ctx, globaldb.GetPostgres())
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

func startHealthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

func waitForTerminationSignal() {
	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
