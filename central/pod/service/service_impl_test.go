package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/pod/datastore"
	processIndicatorMocks "github.com/stackrox/stackrox/central/processindicator/datastore/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/grpc/testutils"
	filterMocks "github.com/stackrox/stackrox/pkg/process/filter/mocks"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestGetPods(t *testing.T) {
	cases := []struct {
		name string
		pods []*storage.Pod
	}{
		{
			name: "one pod",
			pods: []*storage.Pod{
				{
					Id: uuid.NewV4().String(),
				},
			},
		},
		{
			name: "multiple pods",
			pods: []*storage.Pod{
				{
					Id: uuid.NewV4().String(),
				},
				{
					Id: uuid.NewV4().String(),
				},
				{
					Id: uuid.NewV4().String(),
				},
				{
					Id: uuid.NewV4().String(),
				},
				{
					Id: uuid.NewV4().String(),
				},
			},
		},
		{
			name: "no pods",
			pods: []*storage.Pod{},
		},
	}

	ctx := sac.WithAllAccess(context.Background())
	mockCtrl := gomock.NewController(t)

	mockFilter := filterMocks.NewMockFilter(mockCtrl)
	mockFilter.EXPECT().UpdateByPod(gomock.Any()).AnyTimes()

	mockIndicators := processIndicatorMocks.NewMockDataStore(mockCtrl)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rocksDB := rocksdbtest.RocksDBForT(t)
			defer rocksDB.Close()

			bleveIndex, err := globalindex.MemOnlyIndex()
			require.NoError(t, err)

			podsDS, err := datastore.NewRocksDB(rocksDB, bleveIndex, mockIndicators, mockFilter)
			require.NoError(t, err)

			for _, pod := range c.pods {
				assert.NoError(t, podsDS.UpsertPod(ctx, pod))
			}

			service := &serviceImpl{
				datastore: podsDS,
			}

			results, err := service.GetPods(ctx, &v1.RawQuery{})
			assert.NoError(t, err)
			assert.ElementsMatch(t, results.Pods, c.pods)
		})
	}
}
