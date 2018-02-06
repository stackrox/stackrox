package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
)

type alertStore struct {
	db.AlertStorage
}

func newAlertStore(persistent db.AlertStorage) *alertStore {
	return &alertStore{
		AlertStorage: persistent,
	}
}

// GetAlerts retrieves all alerts
func (s *alertStore) GetAlerts(request *v1.GetAlertsRequest) (filtered []*v1.Alert, err error) {
	alerts, err := s.AlertStorage.GetAlerts(request)
	if err != nil {
		return nil, err
	}
	sinceTime, sinceTimeErr := ptypes.Timestamp(request.GetSince())
	untilTime, untilTimeErr := ptypes.Timestamp(request.GetUntil())
	sinceStaleTime, sinceStaleTimeErr := ptypes.Timestamp(request.GetSinceStale())
	untilStaleTime, untilStaleTimeErr := ptypes.Timestamp(request.GetUntilStale())

	severitySet := severitiesWrap(request.GetSeverity()).asSet()
	categoriesSet := categoriesWrap(request.GetCategory()).asSet()
	policyIDsSet := stringWrap(request.GetPolicyId()).asSet()
	policyNamesSet := stringWrap(request.GetPolicyName()).asSet()
	clusterSet := stringWrap(request.GetCluster()).asSet()
	deploymentIDsSet := stringWrap(request.GetDeploymentId()).asSet()
	deploymentNamesSet := stringWrap(request.GetDeploymentName()).asSet()

	for _, alert := range alerts {
		if len(request.GetStale()) == 1 && alert.GetStale() != request.GetStale()[0] {
			continue
		}

		if _, ok := severitySet[alert.GetPolicy().GetSeverity()]; len(severitySet) > 0 && !ok {
			continue
		}

		if len(categoriesSet) > 0 && !s.matchCategories(alert.GetPolicy().GetCategories(), categoriesSet) {
			continue
		}

		if _, ok := policyNamesSet[alert.GetPolicy().GetName()]; len(policyNamesSet) > 0 && !ok {
			continue
		}

		if _, ok := policyIDsSet[alert.GetPolicy().GetId()]; len(policyIDsSet) > 0 && !ok {
			continue
		}

		if _, ok := clusterSet[alert.GetDeployment().GetClusterId()]; len(clusterSet) > 0 && !ok {
			continue
		}

		if _, ok := deploymentIDsSet[alert.GetDeployment().GetId()]; len(deploymentIDsSet) > 0 && !ok {
			continue
		}

		if _, ok := deploymentNamesSet[alert.GetDeployment().GetName()]; len(deploymentNamesSet) > 0 && !ok {
			continue
		}

		if v, ok := alert.GetDeployment().GetLabels()[request.GetLabelKey()]; len(request.GetLabelKey()) > 0 && (!ok || v != request.GetLabelValue()) {
			continue
		}

		if sinceTimeErr == nil && !sinceTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !sinceTime.Before(alertTime) {
				continue
			}
		}

		if untilTimeErr == nil && !untilTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !untilTime.After(alertTime) {
				continue
			}
		}

		if sinceStaleTimeErr == nil && !sinceStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !sinceStaleTime.Before(alertTime) {
				continue
			}
		}

		if untilStaleTimeErr == nil && !untilStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !untilStaleTime.After(alertTime) {
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

func (s *alertStore) matchCategories(alertCategories []v1.Policy_Category, categorySet map[v1.Policy_Category]struct{}) bool {
	for _, c := range alertCategories {
		if _, ok := categorySet[c]; ok {
			return true
		}
	}

	return false
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

type stringWrap []string

func (wrap stringWrap) asSet() map[string]struct{} {
	output := make(map[string]struct{})

	for _, p := range wrap {
		output[p] = struct{}{}
	}

	return output
}
