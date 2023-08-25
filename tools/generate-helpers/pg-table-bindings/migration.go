package main

import "regexp"

var (
	migrateFromRegex = regexp.MustCompile(`^(rocksdb|boltdb|dackbox)$`)
)

// MigrationOptions hold options to generate migrations
type MigrationOptions struct {
	MigrateFromDB   string
	MigrateSequence int
	Dir             string
	SingletonStore  bool
	BatchSize       int
}
