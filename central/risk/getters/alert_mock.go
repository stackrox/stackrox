package getters

import (
	"context"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// MockAlertsSearcher is a mock AlertsSearcher.
type MockAlertsSearcher struct {
	Alerts []*storage.ListAlert
}

// SearchListAlerts implements the AlertsSearcher interface
func (m MockAlertsSearcher) SearchListAlerts(_ context.Context, q *v1.Query) (alerts []*storage.ListAlert, err error) {
	state := storage.ViolationState_ACTIVE.String()
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.ViolationState.String() {
			state = mfQ.MatchFieldQuery.GetValue()
		}
	})

	for _, a := range m.Alerts {
		if a.GetState().String() == strings.Trim(state, "\"") {
			alerts = append(alerts, a)
		}
	}
	return
}
