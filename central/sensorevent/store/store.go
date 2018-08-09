package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const sensorEventBucket = "sensorEvents"

// Store provides storage functionality for alerts.
//go:generate mockery -name=Store
type Store interface {
	GetSensorEvent(id uint64) (*v1.SensorEvent, bool, error)
	GetSensorEventIds(clusterID string) ([]uint64, map[string]uint64, error)
	AddSensorEvent(deployment *v1.SensorEvent) (uint64, error)
	UpdateSensorEvent(id uint64, event *v1.SensorEvent) error
	RemoveSensorEvent(id uint64) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, sensorEventBucket)
	return &storeImpl{
		DB: db,
	}
}
