package main

import (
	"net/http"
	"os"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/compact"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/runner"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/tecbot/gorocksdb"
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
	conf, err := config.ReadConfig()
	if err != nil {
		log.WriteToStderrf("error reading configuration: %v. Skipping migrator", err)
		return nil
	}

	if conf.Maintenance.SafeMode {
		log.WriteToStderr("configuration has safe mode set. Skipping migrator")
		return nil
	}

	if err := compact.Compact(conf); err != nil {
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

	var rocksdb *gorocksdb.DB
	var rocksDBSeqNum int
	if env.RocksDB.BooleanSetting() {
		rocksdb, err = rockshelper.New()
		if err != nil {
			return errors.Wrap(err, "failed to open rocksdb")
		}

		rocksDBSeqNum, err = runner.GetCurrentSeqNumRocksDB(rocksdb)
		if err != nil {
			return errors.Wrap(err, "failed to fetch sequence number from rocksdb")
		}
	}

	var badgerDB *badger.DB
	if rocksdb == nil || rocksDBSeqNum == 0 {
		badgerDB, err = badgerhelpers.NewWithDefaults()
		if err != nil {
			return errors.Wrap(err, "failed to open badger DB")
		}
	}

	defer func() {
		if err := boltDB.Close(); err != nil {
			log.WriteToStderrf("Error closing DB: %v", err)
		}
		if badgerDB != nil {
			if err := badgerDB.Close(); err != nil {
				log.WriteToStderrf("Error closing badger DB: %v", err)
			}
		}

		if rocksdb != nil {
			rocksdb.Close()
		}
	}()
	err = runner.Run(&types.Databases{
		BoltDB:   boltDB,
		BadgerDB: badgerDB,
		RocksDB:  rocksdb,
	})
	if err != nil {
		return errors.Wrap(err, "migrations failed")
	}
	return nil
}
