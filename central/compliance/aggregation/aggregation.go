package aggregation

import (
	"strconv"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	minScope  = v1.ComplianceAggregation_STANDARD
	maxScope  = v1.ComplianceAggregation_DEPLOYMENT
	numScopes = maxScope - minScope + 1
)

type passFailCounts struct {
	pass int
	fail int
}

func (c passFailCounts) Add(other passFailCounts) passFailCounts {
	return passFailCounts{
		pass: c.pass + other.pass,
		fail: c.fail + other.fail,
	}
}

func (c passFailCounts) Reduce() passFailCounts {
	if c.fail > 0 {
		return passFailCounts{fail: 1}
	}
	if c.pass > 0 {
		return passFailCounts{pass: 1}
	}
	return passFailCounts{}
}

var (
	log = logging.LoggerForModule()

	passFailCountsByState = map[storage.ComplianceState]passFailCounts{
		storage.ComplianceState_COMPLIANCE_STATE_SUCCESS: {pass: 1},
		storage.ComplianceState_COMPLIANCE_STATE_FAILURE: {fail: 1},
		storage.ComplianceState_COMPLIANCE_STATE_ERROR:   {fail: 1},
	}
)

type groupByKey [numScopes]string

func (k groupByKey) Get(scope v1.ComplianceAggregation_Scope) string {
	if scope < minScope || scope > maxScope {
		log.Errorf("Unknown scope: %v", scope)
		return ""
	}
	return k[int(scope-minScope)]
}

func (k *groupByKey) Set(scope v1.ComplianceAggregation_Scope, value string) {
	if scope < minScope || scope > maxScope {
		log.Errorf("Unknown scope: %v", scope)
		return
	}
	(*k)[int(scope-minScope)] = value
}

// flatCheck is basically all of the check result data flattened into a single object
type flatCheck struct {
	values map[v1.ComplianceAggregation_Scope]string
	state  storage.ComplianceState
}

func (c flatCheck) passFailCounts() passFailCounts {
	return passFailCountsByState[c.state]
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
			Scope: v1.ComplianceAggregation_Scope(i + int(minScope)),
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

// DomainFunc will return a valid storage domain for a given key, if it exists. If multiple domains match, only one will be returned.
type DomainFunc func(i int) *storage.ComplianceDomain

type domainOffsetPair struct {
	offset int
	domain *storage.ComplianceDomain
}

// GetAggregatedResults aggregates the passed results by groupBy and unit
func GetAggregatedResults(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope, runResults []*storage.ComplianceRunResults) ([]*v1.ComplianceAggregation_Result, DomainFunc) {
	var flatChecks []flatCheck
	var domainIndices []domainOffsetPair
	for _, r := range runResults {
		flatChecks = append(flatChecks, getFlatChecksFromRunResult(r)...)
		domainIndices = append(domainIndices, domainOffsetPair{offset: len(flatChecks), domain: r.GetDomain()})
	}

	// Iterate over all of the checks and create a map[groupBy][]flatCheck. Ignore keys where one of the groupBy values
	// would be empty.
	groups := make(map[groupByKey][]flatCheck)
	// Store a valid domain for every groupByKey
	domains := make(map[groupByKey]*storage.ComplianceDomain)
	for i, fc := range flatChecks {
		if i >= domainIndices[0].offset {
			domainIndices = domainIndices[1:]
		}
		var key groupByKey
		valid := true
		for _, s := range groupBy {
			val, ok := fc.values[s]
			if !ok || val == "" {
				valid = false
				break
			}
			key.Set(s, val)
		}
		if valid {
			groups[key] = append(groups[key], fc)
			domains[key] = domainIndices[0].domain
		}
	}

	results := make([]*v1.ComplianceAggregation_Result, 0, len(groups))
	domainMap := make(map[int]*storage.ComplianceDomain)
	for key, checks := range groups {
		unitMap := make(map[string]passFailCounts)
		for i, c := range checks {
			unitKey := c.values[unit]
			// If there is no unit key, then the check doesn't apply in this unit scope
			if unit == v1.ComplianceAggregation_CHECK {
				unitKey = strconv.Itoa(i)
			} else if unitKey == "" {
				continue
			}
			unitMap[unitKey] = unitMap[unitKey].Add(c.passFailCounts())
		}

		// Aggregate over all units for this key
		var counts passFailCounts
		for _, u := range unitMap {
			counts = counts.Add(u.Reduce())
		}

		domainMap[len(results)] = domains[key]
		results = append(results, &v1.ComplianceAggregation_Result{
			AggregationKeys: getAggregationKeys(key),
			Unit:            unit,
			NumPassing:      int32(counts.pass),
			NumFailing:      int32(counts.fail),
		})
	}
	return results, func(i int) *storage.ComplianceDomain { return domainMap[i] }
}
