package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stackrox/stackrox/pkg/bolthelper"
)

func die(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func main() {
	dbPath := flag.String("db-path", "", "the path to the BoltDB dump")
	flag.Parse()
	if *dbPath == "" {
		die("DB Path must be specified.")
	}
	_, err := os.Stat(*dbPath)
	if err != nil {
		die("Invalid DB path: %s", err)
	}
	db, err := bolthelper.New(*dbPath)
	if err != nil {
		die("Failed to open DB: %s", err)
	}
	fmt.Println("Successfully loaded DB!")
	_ = db
	// You can now use the DB to load things by doing stuff like
	//
	// import "github.com/stackrox/stackrox/deployment/store"
	// deploymentStore := store.New(db)
}
