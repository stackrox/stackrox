package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackrox/rox/central/version"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	dbOpenRetries         = 10
	dbTimeBetweenRetries  = 10 * time.Second
	healthAddr            = ":8082"
)

var log = logging.LoggerForModule()

func main() {
	premain.StartMain()

	log.Infof("Starting central-worker")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := initDB(ctx)
	defer db.Close()

	ensureDBCurrent(db)

	healthSrv := startHealthServer()

	log.Infof("central-worker is ready")

	waitForTerminationSignal()

	log.Infof("central-worker shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Health server shutdown error: %v", err)
	}
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

	poolMaxConns := env.RegisterIntegerSetting("ROX_WORKER_DB_POOL_MAX_CONNS", 20)
	dbConfig.MaxConns = int32(poolMaxConns.IntegerSetting())

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

func startHealthServer() *http.Server {
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
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Health server error: %v", err)
		}
	}()
	return srv
}

func waitForTerminationSignal() {
	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
