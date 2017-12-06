package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
)

type alertStore struct {
	alerts     map[string]*v1.Alert
	alertMutex sync.Mutex

	persistent db.AlertStorage
}

func newAlertStore(persistent db.AlertStorage) *alertStore {
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

func (s *alertStore) GetAlert(id string) (d *v1.Alert, exist bool, err error) {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	d, exist = s.alerts[id]
	return
}

// GetAlerts retrieves all alerts
func (s *alertStore) GetAlerts(request *v1.GetAlertsRequest) (filtered []*v1.Alert, err error) {
	s.alertMutex.Lock()
	defer s.alertMutex.Unlock()
	alerts := make([]*v1.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alerts = append(alerts, alert)
	}

	requestTime, requestTimeErr := ptypes.Timestamp(request.GetSince())
	severitySet := severitiesWrap(request.GetSeverity()).asSet()
	categoriesSet := categoriesWrap(request.GetCategory()).asSet()
	policiesSet := policiesWrap(request.GetPolicyName()).asSet()

	for _, alert := range alerts {
		if _, ok := severitySet[alert.GetSeverity()]; len(severitySet) > 0 && !ok {
			continue
		}

		if _, ok := categoriesSet[alert.GetPolicy().GetCategory()]; len(categoriesSet) > 0 && !ok {
			continue
		}

		if _, ok := policiesSet[alert.GetPolicy().GetName()]; len(policiesSet) > 0 && !ok {
			continue
		}

		if requestTimeErr == nil && !requestTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !requestTime.Before(alertTime) {
				continue
			}
		}

		filtered = append(filtered, alert)
	}

	// Sort by descending timestamp.
	sort.SliceStable(filtered, func(i, j int) bool {
		if sI, sJ := filtered[i].GetTime().GetSeconds(), filtered[j].GetTime().GetSeconds(); sI != sJ {
			return sI > sJ
		}

		return filtered[i].GetTime().GetNanos() > filtered[j].GetTime().GetNanos()
	})

	return
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

type severitiesWrap []v1.Severity

func (wrap severitiesWrap) asSet() map[v1.Severity]struct{} {
	output := make(map[v1.Severity]struct{})

	for _, s := range wrap {
		output[s] = struct{}{}
	}

	return output
}

type categoriesWrap []v1.Policy_Category

func (wrap categoriesWrap) asSet() map[v1.Policy_Category]struct{} {
	output := make(map[v1.Policy_Category]struct{})

	for _, c := range wrap {
		output[c] = struct{}{}
	}

	return output
}

type policiesWrap []string

func (wrap policiesWrap) asSet() map[string]struct{} {
	output := make(map[string]struct{})

	for _, p := range wrap {
		output[p] = struct{}{}
	}

	return output
}
