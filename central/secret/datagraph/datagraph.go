package datagraph

import (
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// DataGraph provides the interface that updates secrets from deployment events.
type DataGraph interface {
	ProcessDeploymentEvent(v1.ResourceAction, *v1.Deployment) error
}

// New returns a new Service instance using the given DB and index.
func New(storage store.Store, indexer index.Indexer) DataGraph {
	return &datagraphImpl{
		storage: storage,
		indexer: indexer,
	}
}
