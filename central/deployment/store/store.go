package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality.
//go:generate mockgen-wrapper
type Store interface {
	ListDeployment(id string) (*storage.ListDeployment, bool, error)
	ListDeployments() ([]*storage.ListDeployment, error)
	ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error)

	GetDeployment(id string) (*storage.Deployment, bool, error)
	GetDeployments() ([]*storage.Deployment, error)
	GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error)

	CountDeployments() (int, error)
	UpsertDeployment(deployment *storage.Deployment) error
	RemoveDeployment(id string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)

	GetDeploymentIDs() ([]string, error)
}
