package main

import (
	"fmt"
	"os"

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
	db, err := bolthelpers.New()
	if err != nil {
		return fmt.Errorf("failed to open DB: %s", err)
	}
	if db == nil {
		log.WriteToStderr("No DB found. Nothing to migrate...")
		return nil
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.WriteToStderr("Error closing DB: %v", err)
		}
	}()
	err = runner.Run(db)
	if err != nil {
		return fmt.Errorf("migrations failed: %s", err)
	}
	return nil
}
