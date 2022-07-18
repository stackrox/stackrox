package datastoretest

import (
	"context"

	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type nodeComponentFromImageComponentDataStore struct {
	imageComponentStore imageComponentDataStore.DataStore
}

func convertImageComponentToNodeComponent(imageComponent *storage.ImageComponent) *storage.NodeComponent {
	return &storage.NodeComponent{
		Id:              imageComponent.Id,
		Name:            imageComponent.Name,
		Version:         imageComponent.Version,
		Priority:        imageComponent.Priority,
		RiskScore:       imageComponent.RiskScore,
		SetTopCvss:      &storage.NodeComponent_TopCvss{TopCvss: imageComponent.GetTopCvss()},
		OperatingSystem: imageComponent.OperatingSystem,
	}
}

func (s *nodeComponentFromImageComponentDataStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.imageComponentStore.Search(ctx, q)
}

func (s *nodeComponentFromImageComponentDataStore) SearchNodeComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.imageComponentStore.SearchImageComponents(ctx, q)
}

func (s *nodeComponentFromImageComponentDataStore) SearchRawNodeComponents(ctx context.Context, q *v1.Query) ([]*storage.NodeComponent, error) {
	components, err := s.imageComponentStore.SearchRawImageComponents(ctx, q)
	if err != nil {
		return nil, err
	}
	nodeComponents := make([]*storage.NodeComponent, 0, len(components))
	for _, cmp := range components {
		nodeComponents = append(nodeComponents, convertImageComponentToNodeComponent(cmp))
	}
	return nodeComponents, nil
}

func (s *nodeComponentFromImageComponentDataStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.imageComponentStore.Exists(ctx, id)
}

func (s *nodeComponentFromImageComponentDataStore) Get(ctx context.Context, id string) (*storage.NodeComponent, bool, error) {
	component, found, err := s.imageComponentStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	return convertImageComponentToNodeComponent(component), true, nil
}

func (s *nodeComponentFromImageComponentDataStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.imageComponentStore.Count(ctx, q)
}

func (s *nodeComponentFromImageComponentDataStore) GetBatch(ctx context.Context, id []string) ([]*storage.NodeComponent, error) {
	components, err := s.imageComponentStore.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	nodeComponents := make([]*storage.NodeComponent, 0, len(components))
	for _, cmp := range components {
		nodeComponents = append(nodeComponents, convertImageComponentToNodeComponent(cmp))
	}
	return nodeComponents, nil
}
