package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

func (i *InMemoryStore) loadAlerts() error {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	alerts, err := i.persistent.GetAlerts(&v1.GetAlertsRequest{})
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		i.alerts[alert.Id] = alert
	}
	return nil
}

// GetAlerts retrieves all alerts
func (i *InMemoryStore) GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error) {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	alerts := make([]*v1.Alert, 0, len(i.alerts))
	for _, alert := range i.alerts {
		alerts = append(alerts, alert)
	}
	if request.Id != "" {
		alert, ok := i.alerts[request.Id]
		if ok {
			return []*v1.Alert{alert}, nil
		}
		return alerts, nil
	}
	if request.Severity != v1.Severity_UNSET_SEVERITY {
		filtered := alerts[:0]
		for _, alert := range alerts {
			if alert.Severity == request.Severity {
				filtered = append(filtered, alert)
			}
		}
		alerts = filtered
	}
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].Id < alerts[j].Id })
	return alerts, nil
}

func (i *InMemoryStore) upsertAlert(alert *v1.Alert) {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	i.alerts[alert.Id] = alert
}

// AddAlert adds a new alert
func (i *InMemoryStore) AddAlert(alert *v1.Alert) error {
	if err := i.persistent.AddAlert(alert); err != nil {
		return err
	}
	i.upsertAlert(alert)
	return nil
}

// UpdateAlert updates an alert
func (i *InMemoryStore) UpdateAlert(alert *v1.Alert) error {
	if err := i.persistent.UpdateAlert(alert); err != nil {
		return err
	}
	i.upsertAlert(alert)
	return nil
}

// RemoveAlert removes an alert
func (i *InMemoryStore) RemoveAlert(id string) error {
	i.alertMutex.Lock()
	defer i.alertMutex.Unlock()
	if err := i.persistent.RemoveAlert(id); err != nil {
		return err
	}
	delete(i.alerts, id)
	return nil
}
