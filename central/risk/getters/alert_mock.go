package getters

import (
	"fmt"

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
	parsedRequest, err := search.ParseRawQuery(req.GetQuery())
	if err != nil {
		return nil, err
	}

	for _, a := range m.Alerts {
		match := true
		staleValues := parsedRequest.Fields[search.Stale].GetValues()
		if len(staleValues) != 0 {
			match = false
			for _, v := range staleValues {
				if fmt.Sprintf("%t", a.Stale) == v {
					match = true
				}
			}
		}
		if match {
			alerts = append(alerts, a)
		}
	}
	return
}
