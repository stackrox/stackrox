package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
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
	bolthelper.RegisterBucket(db, sensorEventBucket)
	return &storeImpl{
		DB: db,
	}
}
