package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const (
	processIndicatorBucket = "process_indicators"
	uniqueProcessesBucket  = "process_indicators_unique"
)

// Store provides storage functionality for alerts.
type Store interface {
	GetProcessIndicator(id string) (*v1.ProcessIndicator, bool, error)
	GetProcessIndicators() ([]*v1.ProcessIndicator, error)
	AddProcessIndicator(*v1.ProcessIndicator) error
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
