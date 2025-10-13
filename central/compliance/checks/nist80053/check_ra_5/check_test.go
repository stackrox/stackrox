package checkra5

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"go.uber.org/mock/gomock"
)

type alert struct {
	policyID string
}

func TestCheckNoUnresolvedAlertsForPolicies(t *testing.T) {
	for _, testCase := range []struct {
		desc       string
		policyIDs  set.StringSet
		alerts     []alert
		shouldPass bool
	}{
		{
			desc:       "some alerts, but no policy IDs",
			policyIDs:  nil,
			alerts:     []alert{{"1"}},
			shouldPass: true,
		},
		{
			desc:       "some alerts, but no relevant policy IDs",
			policyIDs:  set.NewStringSet("2"),
			alerts:     []alert{{"1"}},
			shouldPass: true,
		},
		{
			desc:       "no unresolved alerts",
			policyIDs:  set.NewStringSet("1", "2"),
			alerts:     nil,
			shouldPass: true,
		},
		{
			desc:       "yes, an unresolved alerts",
			policyIDs:  set.NewStringSet("1", "2"),
			alerts:     []alert{{"1"}},
			shouldPass: false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)

			convertedAlerts := make([]*storage.ListAlert, 0, len(c.alerts))
			for _, alert := range c.alerts {
				convertedAlerts = append(convertedAlerts, &storage.ListAlert{
					Policy: &storage.ListAlertPolicy{Id: alert.policyID},
				})
			}

			mockData.EXPECT().UnresolvedAlerts().Return(convertedAlerts)
			checkNoUnresolvedAlertsForPolicies(mockCtx, c.policyIDs)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}

}
