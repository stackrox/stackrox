package util

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

var networkPolicyFields = map[string]struct{}{
	augmentedobjs.MissingIngressPolicyCustomTag: {},
	augmentedobjs.MissingEgressPolicyCustomTag:  {},
}

func checkNetworkPolicyField(p *storage.Policy) bool {
	for _, section := range p.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if _, ok := networkPolicyFields[group.GetFieldName()]; ok {
				return true
			}
		}
	}
	return false
}

func RemoveAlertsWithNetworkPolicyFields(alerts []*storage.Alert) ([]*storage.Alert, bool) {
	newAlerts := []*storage.Alert{}
	foundNetworkPolicy := false
	for _, a := range alerts {
		if checkNetworkPolicyField(a.GetPolicy()) {
			foundNetworkPolicy = true
			continue
		}
		newAlerts = append(newAlerts, a)
	}
	return newAlerts, foundNetworkPolicy
}
