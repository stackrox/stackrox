package getters

import (
	"strconv"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// MockAlertsGetter is a mock AlertsGetter.
type MockAlertsGetter struct {
	Alerts []*v1.ListAlert
}

// ListAlerts supports a limited set of request parameters.
// It only needs to be as specific as the production code.
func (m MockAlertsGetter) ListAlerts(req *v1.ListAlertsRequest) (alerts []*v1.ListAlert, err error) {
	q, err := search.ParseRawQuery(req.GetQuery())
	if err != nil {
		return nil, err
	}

	var staleValue bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.Stale.String() {
			staleValue, err = strconv.ParseBool(mfQ.MatchFieldQuery.GetValue())
			if err != nil {
				panic(err)
			}
		}
	})

	for _, a := range m.Alerts {
		if a.GetStale() == staleValue {
			alerts = append(alerts, a)
		}
	}
	return
}
