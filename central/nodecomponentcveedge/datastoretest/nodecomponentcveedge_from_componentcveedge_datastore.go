package datastoretest

import (
	"context"

	componentCVEEdgeDataStore "github.com/stackrox/rox/central/componentcveedge/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type nodeComponentCVEEdgeFromGenericStore struct {
	genericStore componentCVEEdgeDataStore.DataStore
}

func imageComponentCVEEdgeToNodeComponentCVEEdge(edge *storage.ComponentCVEEdge) *storage.NodeComponentCVEEdge {
	return &storage.NodeComponentCVEEdge{
		Id:              edge.GetId(),
		IsFixable:       edge.GetIsFixable(),
		HasFixedBy:      &storage.NodeComponentCVEEdge_FixedBy{FixedBy: edge.GetFixedBy()},
		NodeComponentId: edge.GetImageComponentId(),
		NodeCveId:       edge.GetImageCveId(),
	}
}

func (s nodeComponentCVEEdgeFromGenericStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.genericStore.Count(ctx, q)
}

func (s nodeComponentCVEEdgeFromGenericStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.genericStore.Search(ctx, q)
}

func (s nodeComponentCVEEdgeFromGenericStore) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.genericStore.SearchEdges(ctx, q)
}

func (s nodeComponentCVEEdgeFromGenericStore) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentCVEEdge, error) {
	genericEdges, err := s.genericStore.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	edges := make([]*storage.NodeComponentCVEEdge, 0, len(genericEdges))
	for _, e := range genericEdges {
		edges = append(edges, imageComponentCVEEdgeToNodeComponentCVEEdge(e))
	}
	return edges, nil
}

func (s nodeComponentCVEEdgeFromGenericStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.genericStore.Exists(ctx, id)
}

func (s nodeComponentCVEEdgeFromGenericStore) Get(ctx context.Context, id string) (*storage.NodeComponentCVEEdge, bool, error) {
	genericEdge, found, err := s.genericStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	return imageComponentCVEEdgeToNodeComponentCVEEdge(genericEdge), true, nil
}
