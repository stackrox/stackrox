package getters

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AlertGetter provides the required access to alerts for risk scoring.
type AlertGetter interface {
	ListAlerts(request *v1.ListAlertsRequest) ([]*storage.ListAlert, error)
}
