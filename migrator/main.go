package main

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/runner"
)

func main() {
	if err := run(); err != nil {
		log.WriteToStderr("Migrator failed: %s", err)
		os.Exit(1)
	}
}

func run() error {
	boltDB, err := bolthelpers.Load()
	if err != nil {
		return fmt.Errorf("failed to open bolt DB: %v", err)
	}
	if boltDB == nil {
		log.WriteToStderr("No DB found. Nothing to migrate...")
		return nil
	}

	badgerDB, err := badgerhelpers.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to open badger DB: %v", err)
	}

	defer func() {
		if err := boltDB.Close(); err != nil {
			log.WriteToStderr("Error closing DB: %v", err)
		}
		if err := badgerDB.Close(); err != nil {
			log.WriteToStderr("Error closing badger DB: %v", err)
		}
	}()
	err = runner.Run(boltDB, badgerDB)
	if err != nil {
		return fmt.Errorf("migrations failed: %s", err)
	}
	return nil
}
