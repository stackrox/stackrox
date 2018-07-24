package getters

import "bitbucket.org/stack-rox/apollo/generated/api/v1"

// AlertGetter provides the required access to alerts for risk scoring.
type AlertGetter interface {
	ListAlerts(request *v1.ListAlertsRequest) ([]*v1.ListAlert, error)
}
