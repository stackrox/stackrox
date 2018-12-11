package store

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

const deploymentBucket = "deployments"
const deploymentListBucket = "deployments_list"

var (
	log = logging.LoggerForModule()
)

// Store provides storage functionality for alerts.
type Store interface {
	ListDeployment(id string) (*storage.ListDeployment, bool, error)
	ListDeployments() ([]*storage.ListDeployment, error)

	GetDeployment(id string) (*storage.Deployment, bool, error)
	GetDeployments() ([]*storage.Deployment, error)
	CountDeployments() (int, error)
	UpsertDeployment(deployment *storage.Deployment) error
	UpdateDeployment(deployment *storage.Deployment) error
	RemoveDeployment(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) (Store, error) {
	bolthelper.RegisterBucketOrPanic(db, deploymentBucket)
	bolthelper.RegisterBucketOrPanic(db, deploymentListBucket)
	s := &storeImpl{
		DB: db,
	}
	if err := s.initializeRanker(); err != nil {
		return nil, fmt.Errorf("failed to initialize ranker: %s", err)
	}
	return s, nil
}
