package store

import (
	"bitbucket.org/stack-rox/apollo/central/ranking"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

const deploymentBucket = "deployments"
const deploymentListBucket = "deployments_list"
const deploymentGraveyard = "deployments_graveyard"

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
	GetTombstonedDeployments() ([]*v1.Deployment, error)
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB, ranker *ranking.Ranker) Store {
	bolthelper.RegisterBucketOrPanic(db, deploymentBucket)
	bolthelper.RegisterBucketOrPanic(db, deploymentGraveyard)
	bolthelper.RegisterBucketOrPanic(db, deploymentListBucket)
	return &storeImpl{
		DB:     db,
		ranker: ranker,
	}
}
