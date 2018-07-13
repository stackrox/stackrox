package datagraph

import (
	"bitbucket.org/stack-rox/apollo/central/secret/index"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataGraph provides the interface that updates secrets from deployment events.
type DataGraph interface {
	ProcessDeploymentEvent(event *v1.DeploymentEvent) error
}

// New returns a new Service instance using the given DB and index.
func New(storage store.Store, indexer index.Indexer) DataGraph {
	return &datagraphImpl{
		storage: storage,
		indexer: indexer,
	}
}
