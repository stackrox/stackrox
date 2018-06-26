package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const deploymentEventBucket = "deploymentEvents"

// Store provides storage functionality for alerts.
type Store interface {
	GetDeploymentEvent(id uint64) (*v1.DeploymentEvent, bool, error)
	GetDeploymentEventIds(clusterID string) ([]uint64, map[string]uint64, error)
	AddDeploymentEvent(deployment *v1.DeploymentEvent) (uint64, error)
	UpdateDeploymentEvent(id uint64, deployment *v1.DeploymentEvent) error
	RemoveDeploymentEvent(id uint64) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, deploymentEventBucket)
	return &storeImpl{
		DB: db,
	}
}
