package main

import "regexp"

var (
	migrateFromRegex = regexp.MustCompile(`^(rocksdb|boltdb):\S+$`)
)

type MigrationOptions struct {
	MigrateFromDB     string
	MigrateFromBucket string
	MigrateSequence   int
}
