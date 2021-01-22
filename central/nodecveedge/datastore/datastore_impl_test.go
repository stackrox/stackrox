package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/nodecveedge/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	dackyMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestGetAllAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)
	store.EXPECT().Get("id").Return(&storage.NodeCVEEdge{Id: "id"}, true, nil)

	dataStore := New(dackyMocks.NewMockProvider(ctrl), store)

	node, found, err := dataStore.Get(sac.WithAllAccess(context.Background()), "id")
	assert.Equal(t, &storage.NodeCVEEdge{Id: "id"}, node)
	assert.True(t, found)
	assert.NoError(t, err)
}

func TestGetScopedAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)
	store.EXPECT().Get("id").Return(&storage.NodeCVEEdge{Id: "id"}, true, nil)

	dataStore := New(dackyMocks.NewMockProvider(ctrl), store)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Node)))

	node, found, err := dataStore.Get(ctx, "id")
	assert.Equal(t, &storage.NodeCVEEdge{Id: "id"}, node)
	assert.True(t, found)
	assert.NoError(t, err)

	ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Image)))

	node, found, err = dataStore.Get(ctx, "id")
	assert.Nil(t, node)
	assert.False(t, found)
	assert.NoError(t, err)
}

func TestGetNoAccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	dataStore := New(dackyMocks.NewMockProvider(ctrl), mocks.NewMockStore(ctrl))

	node, found, err := dataStore.Get(sac.WithNoAccess(context.Background()), "id")
	assert.Nil(t, node)
	assert.False(t, found)
	assert.NoError(t, err)
}
