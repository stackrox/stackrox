package checksi4

import (
	"testing"
	"time"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"go.uber.org/mock/gomock"
)

func TestCheckClusterCheckedInInThePastHour(t *testing.T) {
	for _, testCase := range []struct {
		desc                  string
		hasClusterContactTime bool
		clusterContactTime    time.Time
		shouldPass            bool
	}{
		{
			desc:                  "never checked in",
			hasClusterContactTime: false,
			shouldPass:            false,
		},
		{
			desc:                  "checked in recently",
			hasClusterContactTime: true,
			clusterContactTime:    time.Now().Add(-30 * time.Minute),
			shouldPass:            true,
		},
		{
			desc:                  "checked in a long time ago",
			hasClusterContactTime: true,
			clusterContactTime:    time.Now().Add(-2 * time.Hour),
			shouldPass:            false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			returnedCluster := &storage.Cluster{HealthStatus: &storage.ClusterHealthStatus{}}
			if c.hasClusterContactTime {
				returnedCluster.HealthStatus.LastContact = protoconv.MustConvertTimeToTimestamp(c.clusterContactTime)
			} else {
				returnedCluster.HealthStatus.LastContact = nil
			}
			mockData.EXPECT().Cluster().Return(returnedCluster)
			checkClusterCheckedInInThePastHour(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
