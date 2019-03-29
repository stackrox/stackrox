package networkflowupdate

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

// NewFactory returns a new instance of a FragmentFactory that returns Fragments handling NetworkFlowUpdate messages
// from sensor.
func NewFactory(clusterStore store.ClusterStore) pipeline.FragmentFactory {
	return &factoryImpl{
		clusterStore: clusterStore,
	}
}

type factoryImpl struct {
	clusterStore store.ClusterStore
}

// GetFragment returns a new pipeline fragment for the given cluster.
func (s *factoryImpl) GetFragment(clusterID string) (pipeline.Fragment, error) {
	flowStore, err := s.clusterStore.CreateFlowStore(clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "creating flow store")
	}
	return NewPipeline(clusterID, newFlowStoreUpdater(flowStore)), nil
}
