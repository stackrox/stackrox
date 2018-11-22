package store

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
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
	ListDeployment(id string) (*v1.ListDeployment, bool, error)
	ListDeployments() ([]*v1.ListDeployment, error)

	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	CountDeployments() (int, error)
	UpsertDeployment(deployment *v1.Deployment) error
	UpdateDeployment(deployment *v1.Deployment) error
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
