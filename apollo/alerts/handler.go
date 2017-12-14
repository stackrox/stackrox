package alerts

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.New("alerts")
)

// StalenessHandler handles updating alert staleness.
// Alerts become stale when the associated deployment is updated or deleted.
type StalenessHandler interface {
	UpdateStaleness(event *v1.DeploymentEvent)
}

// NewStalenessHandler returns a new StalenessHandler.
func NewStalenessHandler(storage db.AlertStorage) StalenessHandler {
	return &stalenessHandler{
		storage: storage,
	}
}

type stalenessHandler struct {
	storage db.AlertStorage
}

func (h *stalenessHandler) UpdateStaleness(event *v1.DeploymentEvent) {
	if event.Action == v1.ResourceAction_CREATE_RESOURCE {
		return
	}

	alerts, err := h.storage.GetAlerts(&v1.GetAlertsRequest{
		DeploymentId: []string{event.GetDeployment().GetId()},
	})
	if err != nil {
		logger.Errorf("unable to get alerts for deployment (%s): %s", event.GetDeployment().GetId(), err)
		return
	}

	for _, a := range alerts {
		if a.GetDeployment().GetVersion() != event.GetDeployment().GetVersion() {
			a.Stale = true
			if err := h.storage.UpdateAlert(a); err != nil {
				logger.Errorf("unable to update alert staleness: %s", err)
			}
		}
	}
}
