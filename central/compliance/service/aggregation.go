package service

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	minScope  = v1.ComplianceAggregation_STANDARD
	maxScope  = v1.ComplianceAggregation_DEPLOYMENT
	numScopes = maxScope - minScope + 1
)

var (
	log = logging.LoggerForModule()
)

type groupByKey [numScopes]string

func (k *groupByKey) Set(scope v1.ComplianceAggregation_Scope, value string) {
	if scope < minScope || scope > maxScope {
		log.Errorf("Unknown scope: %s", scope.String())
		return
	}
	(*k)[int(scope-minScope)] = value
}

// flatCheck is basically all of the check result data flattened into a single object
type flatCheck struct {
	values map[v1.ComplianceAggregation_Scope]string
	state  storage.ComplianceState
}

func newFlatCheck(clusterID, namespaceID, standardID, category, controlID, nodeID, deploymentID string, state storage.ComplianceState) flatCheck {
	return flatCheck{
		values: map[v1.ComplianceAggregation_Scope]string{
			v1.ComplianceAggregation_CLUSTER:    clusterID,
			v1.ComplianceAggregation_NAMESPACE:  namespaceID,
			v1.ComplianceAggregation_STANDARD:   standardID,
			v1.ComplianceAggregation_CATEGORY:   category,
			v1.ComplianceAggregation_CONTROL:    controlID,
			v1.ComplianceAggregation_NODE:       nodeID,
			v1.ComplianceAggregation_DEPLOYMENT: deploymentID,
		},
		state: state,
	}
}

func getAggregationKeys(groupByKey groupByKey) []*v1.ComplianceAggregation_AggregationKey {
	var aggregationKeys []*v1.ComplianceAggregation_AggregationKey
	for i, val := range groupByKey {
		if val == "" {
			continue
		}
		aggregationKeys = append(aggregationKeys, &v1.ComplianceAggregation_AggregationKey{
			Scope: v1.ComplianceAggregation_Scope(i),
			Id:    val,
		})
	}
	return aggregationKeys
}

// TODO(cgorman) Look at how to handle category
func getFlatChecksFromRunResult(runResults *storage.ComplianceRunResults) []flatCheck {
	domain := runResults.GetDomain()
	clusterID := runResults.GetDomain().GetCluster().GetId()
	standardID := runResults.GetRunMetadata().GetStandardId()

	var flatChecks []flatCheck
	for control, r := range runResults.GetClusterResults().GetControlResults() {
		flatChecks = append(flatChecks, newFlatCheck(clusterID, "", standardID, "", control, "", "", r.GetOverallState()))
	}
	for n, controlResults := range runResults.GetNodeResults() {
		for control, r := range controlResults.GetControlResults() {
			flatChecks = append(flatChecks, newFlatCheck(clusterID, "", standardID, "", control, n, "", r.GetOverallState()))
		}
	}
	for d, controlResults := range runResults.GetDeploymentResults() {
		deployment, ok := domain.Deployments[d]
		if !ok {
			log.Errorf("Okay that's not good, we have a result for a deployment that isn't even in the domain?")
			continue
		}
		for control, r := range controlResults.GetControlResults() {
			flatChecks = append(flatChecks, newFlatCheck(clusterID, deployment.GetNamespace(), standardID, "", control, "", deployment.GetId(), r.GetOverallState()))
		}
	}
	return flatChecks
}

func getAggregatedResults(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope, runResults []*storage.ComplianceRunResults) []*v1.ComplianceAggregation_Result {
	var flatChecks []flatCheck
	for _, r := range runResults {
		flatChecks = append(flatChecks, getFlatChecksFromRunResult(r)...)
	}
	groups := make(map[groupByKey][]flatCheck)
loop:
	// Iterate over all of the checks create a map[groupBy][]flatCheck
	// as long as one of the groupBy values is not empty
	for _, fc := range flatChecks {
		groupByKey := &groupByKey{}
		for _, s := range groupBy {
			val, ok := fc.values[s]
			if !ok || val == "" {
				continue loop
			}
			groupByKey.Set(s, val)
		}
		groups[*groupByKey] = append(groups[*groupByKey], fc)
	}

	results := make([]*v1.ComplianceAggregation_Result, 0, len(groups))
	for groupKey, checks := range groups {
		resultMap := make(map[string]storage.ComplianceState)
		var fail int32
		for _, c := range checks {
			unitKey := c.values[unit]
			// If there is no unit key, then the check doesn't apply in this unit scope
			if unitKey == "" {
				continue
			}

			if currState, ok := resultMap[unitKey]; !ok {
				resultMap[unitKey] = c.state
				if c.state == storage.ComplianceState_COMPLIANCE_STATE_FAILURE {
					fail++
				}
			} else if currState == storage.ComplianceState_COMPLIANCE_STATE_SUCCESS && c.state == storage.ComplianceState_COMPLIANCE_STATE_FAILURE {
				resultMap[unitKey] = c.state
				fail++
			}
		}

		// If there are no results, then the Unit does not apply to the GroupBys and therefore
		// we will omit the non relevant results
		if len(resultMap) == 0 {
			continue
		}

		results = append(results, &v1.ComplianceAggregation_Result{
			AggregationKeys: getAggregationKeys(groupKey),
			Unit:            unit,
			NumPassing:      int32(len(resultMap)) - fail,
			NumFailing:      fail,
		})
	}
	return results
}
