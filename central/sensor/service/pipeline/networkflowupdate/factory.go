package networkflowupdate

import (
	"fmt"

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
		return nil, fmt.Errorf("creating flow store: %v", err)
	}
	return NewPipeline(clusterID, newFlowStoreUpdater(flowStore)), nil
}
