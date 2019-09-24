package main

import (
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/compact"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/pkg/grpc/routes"
)

func main() {
	startProfilingServer()
	if err := run(); err != nil {
		log.WriteToStderrf("Migrator failed: %s", err)
		os.Exit(1)
	}
}

func startProfilingServer() {
	handler := http.NewServeMux()
	for path, debugHandler := range routes.DebugRoutes {
		handler.Handle(path, debugHandler)
	}
	srv := &http.Server{Addr: ":6060", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.WriteToStderrf("Closing profiling server: %v", err)
		}
	}()
}

func run() error {
	if err := compact.Compact(); err != nil {
		log.WriteToStderrf("error compacting DB: %v", err)
	}

	boltDB, err := bolthelpers.Load()
	if err != nil {
		return errors.Wrap(err, "failed to open bolt DB")
	}
	if boltDB == nil {
		log.WriteToStderr("No DB found. Nothing to migrate...")
		return nil
	}

	badgerDB, err := badgerhelpers.NewWithDefaults()
	if err != nil {
		return errors.Wrap(err, "failed to open badger DB")
	}

	defer func() {
		if err := boltDB.Close(); err != nil {
			log.WriteToStderrf("Error closing DB: %v", err)
		}
		if err := badgerDB.Close(); err != nil {
			log.WriteToStderrf("Error closing badger DB: %v", err)
		}
	}()
	err = runner.Run(boltDB, badgerDB)
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}
	return nil
}
