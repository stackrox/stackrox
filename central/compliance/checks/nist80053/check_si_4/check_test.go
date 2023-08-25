package checksi4

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"go.uber.org/mock/gomock"
)

func TestCheckClusterCheckedInInThePastHour(t *testing.T) {
	for _, testCase := range []struct {
		desc               string
		clusterContactTime *types.Timestamp
		shouldPass         bool
	}{
		{
			desc:               "never checked in",
			clusterContactTime: nil,
			shouldPass:         false,
		},
		{
			desc:               "checked in recently",
			clusterContactTime: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-30 * time.Minute)),
			shouldPass:         true,
		},
		{
			desc:               "checked in a long time ago",
			clusterContactTime: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
			shouldPass:         false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			mockData.EXPECT().Cluster().Return(&storage.Cluster{HealthStatus: &storage.ClusterHealthStatus{LastContact: c.clusterContactTime}})
			checkClusterCheckedInInThePastHour(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
