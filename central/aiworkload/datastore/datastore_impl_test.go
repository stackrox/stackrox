package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/aiworkload/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpsertAIWorkload_RejectsEmptyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := storeMocks.NewMockAIWorkloadStore(ctrl)
	ds := newDatastoreImpl(store)

	err := ds.UpsertAIWorkload(context.Background(), &storage.AIWorkload{Name: "no-id"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "without an id")
}

func TestUpsertAIWorkload_SetsLastUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := storeMocks.NewMockAIWorkloadStore(ctrl)
	ds := newDatastoreImpl(store)

	store.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, workloads []*storage.AIWorkload) error {
			require.Len(t, workloads, 1)
			assert.NotNil(t, workloads[0].GetLastUpdated())
			return nil
		},
	)

	err := ds.UpsertAIWorkload(context.Background(), &storage.AIWorkload{Id: "test-id", Name: "test"})
	require.NoError(t, err)
}
