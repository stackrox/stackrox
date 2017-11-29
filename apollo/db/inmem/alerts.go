package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type alertStore struct {
	alerts     map[string]*v1.Alert
	alertMutex sync.Mutex

	persistent db.Storage
}

func newAlertStore(persistent db.Storage) *alertStore {
	return &alertStore{
		alerts:     make(map[string]*v1.Alert),
		persistent: persistent,
	}
}

func (s *alertStore) loadFromPersistent() error {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	alerts, err := s.persistent.GetAlerts(&v1.GetAlertsRequest{})
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		s.alerts[alert.Id] = alert
	}
	return nil
}

// GetAlerts retrieves all alerts
func (s *alertStore) GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error) {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	alerts := make([]*v1.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alerts = append(alerts, alert)
	}
	if request.Id != "" {
		alert, ok := s.alerts[request.Id]
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

func (s *alertStore) upsertAlert(alert *v1.Alert) {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	s.alerts[alert.Id] = alert
}

// AddAlert adds a new alert
func (s *alertStore) AddAlert(alert *v1.Alert) error {
	if err := s.persistent.AddAlert(alert); err != nil {
		return err
	}
	s.upsertAlert(alert)
	return nil
}

// UpdateAlert updates an alert
func (s *alertStore) UpdateAlert(alert *v1.Alert) error {
	if err := s.persistent.UpdateAlert(alert); err != nil {
		return err
	}
	s.upsertAlert(alert)
	return nil
}

// RemoveAlert removes an alert
func (s *alertStore) RemoveAlert(id string) error {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	if err := s.persistent.RemoveAlert(id); err != nil {
		return err
	}
	delete(s.alerts, id)
	return nil
}
