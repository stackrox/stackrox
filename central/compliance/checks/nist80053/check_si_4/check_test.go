package checksi4

import (
	"testing"
	"time"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"go.uber.org/mock/gomock"
)

func getClusterWithLastContactTime(timestamp *time.Time) *storage.Cluster {
	if timestamp == nil {
		chs := &storage.ClusterHealthStatus{}
		chs.ClearLastContact()
		cluster := &storage.Cluster{}
		cluster.SetHealthStatus(chs)
		return cluster
	}
	chs := &storage.ClusterHealthStatus{}
	chs.SetLastContact(protoconv.ConvertTimeToTimestamp(*timestamp))
	cluster := &storage.Cluster{}
	cluster.SetHealthStatus(chs)
	return cluster
}

func TestCheckClusterCheckedInInThePastHour(t *testing.T) {
	nowMinus30Minutes := time.Now().Add(-30 * time.Minute)
	nowMinus2Hours := time.Now().Add(-2 * time.Hour)
	for _, testCase := range []struct {
		desc               string
		clusterContactTime *time.Time
		shouldPass         bool
	}{
		{
			desc:               "never checked in",
			clusterContactTime: nil,
			shouldPass:         false,
		},
		{
			desc:               "checked in recently",
			clusterContactTime: &nowMinus30Minutes,
			shouldPass:         true,
		},
		{
			desc:               "checked in a long time ago",
			clusterContactTime: &nowMinus2Hours,
			shouldPass:         false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			mockData.EXPECT().Cluster().Return(getClusterWithLastContactTime(c.clusterContactTime))
			checkClusterCheckedInInThePastHour(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
