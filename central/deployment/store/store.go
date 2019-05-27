package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	deploymentBucket     = []byte("deployments")
	deploymentListBucket = []byte("deployments_list")
)

var (
	log = logging.LoggerForModule()
)

// Store provides storage functionality for alerts.
type Store interface {
	ListDeployment(id string) (*storage.ListDeployment, bool, error)
	ListDeployments() ([]*storage.ListDeployment, error)
	ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error)

	GetDeployment(id string) (*storage.Deployment, bool, error)
	GetDeployments() ([]*storage.Deployment, error)
	GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error)

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
		return nil, errors.Wrap(err, "failed to initialize ranker")
	}
	return s, nil
}
