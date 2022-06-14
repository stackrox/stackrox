package store

import (
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var logsBucket = []byte("logs")

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	GetLogs() ([]string, error)
	GetLogsRange() (start int64, end int64, err error)
	AddLog(log string) error
	RemoveLogs(from, to int64) error
}

// newStore returns a new Store instance using the provided bolt DB instance.
func newStore(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, logsBucket)
	return &storeImpl{
		DB: db,
	}
}
