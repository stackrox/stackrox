package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/pod/datastore"
	processIndicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	filterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"
	"github.com/stackrox/rox/pkg/sac"
	testutils2 "github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
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
	mockIndicators.EXPECT().RemoveProcessIndicatorsOfStaleContainersByPod(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			badgerDB := testutils2.BadgerDBForT(t)
			defer utils.IgnoreError(badgerDB.Close)

			bleveIndex, err := globalindex.MemOnlyIndex()
			require.NoError(t, err)

			podsDS, err := datastore.New(badgerDB, bleveIndex, mockIndicators, mockFilter)
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
