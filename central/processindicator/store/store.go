package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

var (
	processIndicatorBucket = []byte("process_indicators")
	uniqueProcessesBucket  = []byte("process_indicators_unique")
)

// Store provides storage functionality for alerts.
type Store interface {
	GetProcessIndicator(id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators() ([]*storage.ProcessIndicator, error)
	GetProcessInfoToArgs() (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error)
	AddProcessIndicator(*storage.ProcessIndicator) (string, error)
	AddProcessIndicators(...*storage.ProcessIndicator) ([]string, error)
	RemoveProcessIndicator(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processIndicatorBucket)
	bolthelper.RegisterBucketOrPanic(db, uniqueProcessesBucket)
	s := &storeImpl{
		DB: db,
	}
	return s
}
