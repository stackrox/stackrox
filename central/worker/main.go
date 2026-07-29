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

	"github.com/stackrox/rox/central/version"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	dbOpenRetries             = 10
	dbTimeBetweenRetries      = 10 * time.Second
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := initDB(ctx)
	defer db.Close()

	ensureDBCurrent(db)

	startHealthServer()

	go startMetricsServer()

	log.Infof("central-worker is ready")

	waitForTerminationSignal()

	log.Infof("central-worker shutting down")
}

func initDB(ctx context.Context) postgres.DB {
	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Fatalf("Could not parse postgres config: %v", err)
	}

	if !pgconfig.IsExternalDatabase() {
		activeDB := pgconfig.GetActiveDB()
		dbConfig.ConnConfig.Database = activeDB
	}

	poolVal := workerPoolSize.IntegerSetting()
	if poolVal < 1 || poolVal > math.MaxInt32 {
		log.Fatalf("ROX_WORKER_DB_POOL_MAX_CONNS must be between 1 and %d, got %d", math.MaxInt32, poolVal)
	}
	dbConfig.MaxConns = int32(poolVal)

	var db postgres.DB
	if err := retry.WithRetry(func() error {
		db, err = postgres.New(ctx, dbConfig)
		return err
	}, retry.Tries(dbOpenRetries), retry.BetweenAttempts(func(attempt int) {
		time.Sleep(dbTimeBetweenRetries)
	}), retry.OnFailedAttempts(func(err error) {
		log.Errorf("open database: %v", err)
	})); err != nil {
		log.Fatalf("Timed out trying to open database: %v", err)
	}

	return db
}

func ensureDBCurrent(db postgres.DB) {
	versionStore := vStore.NewPostgres(db)
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
