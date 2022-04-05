package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/nodecveedge/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	dackyMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/dackbox/keys"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestGetAllAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	edgeID := string(keys.CreatePairKey([]byte("cve1"), []byte("node1")))
	store := mocks.NewMockStore(ctrl)
	store.EXPECT().Get(edgeID).Return(&storage.NodeCVEEdge{Id: edgeID}, true, nil)

	dataStore := New(dackyMocks.NewMockProvider(ctrl), store)

	node, found, err := dataStore.Get(sac.WithAllAccess(context.Background()), edgeID)
	assert.Equal(t, &storage.NodeCVEEdge{Id: edgeID}, node)
	assert.True(t, found)
	assert.NoError(t, err)
}

func TestGetScopedAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	edgeID := string(keys.CreatePairKey([]byte("cve1"), []byte("node1")))
	store := mocks.NewMockStore(ctrl)
	store.EXPECT().Get(edgeID).Return(&storage.NodeCVEEdge{Id: edgeID}, true, nil)

	mockProvider := dackyMocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().NewGraphView().DoAndReturn(func() graph.DiscardableRGraph {
		g := dackyMocks.NewMockDiscardableRGraph(ctrl)
		g.EXPECT().Discard()
		g.EXPECT().GetRefsToPrefix(gomock.Any(), gomock.Any()).Return([][]byte(nil))
		return g
	})

	dataStore := New(mockProvider, store)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Node)))

	node, found, err := dataStore.Get(ctx, edgeID)
	assert.Equal(t, &storage.NodeCVEEdge{Id: edgeID}, node)
	assert.True(t, found)
	assert.NoError(t, err)

	ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Image)))

	node, found, err = dataStore.Get(ctx, edgeID)
	assert.Nil(t, node)
	assert.False(t, found)
	assert.NoError(t, err)
}

func TestGetNoAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockProvider := dackyMocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().NewGraphView().DoAndReturn(func() graph.DiscardableRGraph {
		g := dackyMocks.NewMockDiscardableRGraph(ctrl)
		g.EXPECT().Discard()
		g.EXPECT().GetRefsToPrefix(gomock.Any(), gomock.Any()).Return([][]byte(nil))
		return g
	})

	dataStore := New(mockProvider, mocks.NewMockStore(ctrl))

	edgeID := string(keys.CreatePairKey([]byte("cve1"), []byte("node1")))
	node, found, err := dataStore.Get(sac.WithNoAccess(context.Background()), edgeID)
	assert.Nil(t, node)
	assert.False(t, found)
	assert.NoError(t, err)
}
