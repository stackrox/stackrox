package main

import "regexp"

var (
	migrateFromRegex = regexp.MustCompile(`^(rocksdb|boltdb|dackbox):\S+$`)
)

// MigrationOptions hold options to generate migrations
type MigrationOptions struct {
	MigrateFromDB     string
	MigrateFromBucket string
	MigrateSequence   int
	Dir               string
	SingletonStore    bool
}
