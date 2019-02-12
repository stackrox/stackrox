package aggregation

import (
	"fmt"
	"strconv"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/compliance/standards"
	standardsIndex "github.com/stackrox/rox/central/compliance/standards/index"
	complianceStore "github.com/stackrox/rox/central/compliance/store"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	namespaceMappings "github.com/stackrox/rox/central/namespace/index/mappings"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	nodeMappings "github.com/stackrox/rox/central/node/index/mappings"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

const (
	minScope  = v1.ComplianceAggregation_STANDARD
	maxScope  = v1.ComplianceAggregation_DEPLOYMENT
	numScopes = maxScope - minScope + 1
)

// Aggregator does compliance aggregation
type Aggregator interface {
	Aggregate(query string, groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) ([]*v1.ComplianceAggregation_Result, []*v1.ComplianceAggregation_Source, DomainFunc, error)
}

// New returns a new aggregator
func New(compliance complianceStore.Store,
	standards standards.Repository,
	clusters clusterDatastore.DataStore,
	namespaces namespaceStore.DataStore,
	nodes nodeStore.GlobalStore) Aggregator {
	return &aggregatorImpl{
		compliance: compliance,
		standards:  standards,
		clusters:   clusters,
		namespaces: namespaces,
		nodes:      nodes,
	}
}

type aggregatorImpl struct {
	compliance complianceStore.Store
	standards  standards.Repository
	clusters   clusterDatastore.DataStore
	namespaces namespaceStore.DataStore
	nodes      nodeStore.GlobalStore
}

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

// Aggregate takes in a search query, groupby scopes and unit scope and returns the results of the aggregation
func (a *aggregatorImpl) Aggregate(queryString string, groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) ([]*v1.ComplianceAggregation_Result, []*v1.ComplianceAggregation_Source, DomainFunc, error) {
	queryMap := search.ParseRawQueryIntoMap(queryString)
	query, err := search.ParseRawQueryOrEmpty(queryString)
	if err != nil {
		return nil, nil, nil, err
	}
	standardIDs, clusterIDs, err := a.getRunParameters(query, queryMap)
	if err != nil {
		return nil, nil, nil, err
	}

	runResults, err := a.compliance.GetLatestRunResultsBatch(clusterIDs, standardIDs, 0)
	if err != nil {
		return nil, nil, nil, err
	}

	validResults, sources := complianceStore.ValidResultsAndSources(runResults)

	mask, err := a.getCheckMask(query, queryMap, standardIDs, clusterIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	results, domainFunc := a.getAggregatedResults(groupBy, unit, validResults, mask)

	return results, sources, domainFunc, nil
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

func getMaskIndex(s v1.ComplianceAggregation_Scope) int {
	return int(s - minScope)
}

func isValidCheck(mask [numScopes]set.StringSet, fc flatCheck) bool {
	for i := range mask {
		scope := v1.ComplianceAggregation_Scope(i) + minScope
		if fc.values[scope] != "" && mask[i].IsInitialized() && !mask[i].Contains(fc.values[scope]) {
			return false
		}
	}
	return true
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

func (a *aggregatorImpl) getCategoryID(controlID string) string {
	category := a.standards.GetCategoryByControl(controlID)
	if category == nil {
		errorhelpers.PanicOnDevelopment(fmt.Errorf("no category found for control %q", controlID))
		return ""
	}
	return category.QualifiedID()
}

func (a *aggregatorImpl) getFlatChecksFromRunResult(runResults *storage.ComplianceRunResults, mask [numScopes]set.StringSet) []flatCheck {
	domain := runResults.GetDomain()
	clusterID := runResults.GetDomain().GetCluster().GetId()
	standardID := runResults.GetRunMetadata().GetStandardId()

	var flatChecks []flatCheck
	for control, r := range runResults.GetClusterResults().GetControlResults() {
		fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
		if isValidCheck(mask, fc) {
			flatChecks = append(flatChecks, fc)
		}
	}
	for n, controlResults := range runResults.GetNodeResults() {
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, n, "", r.GetOverallState())
			if isValidCheck(mask, fc) {
				flatChecks = append(flatChecks, fc)
			}
		}
	}
	for d, controlResults := range runResults.GetDeploymentResults() {
		deployment, ok := domain.Deployments[d]
		if !ok {
			log.Errorf("Okay that's not good, we have a result for a deployment that isn't even in the domain?")
			continue
		}
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, deployment.GetNamespaceId(), standardID, a.getCategoryID(control), control, "", deployment.GetId(), r.GetOverallState())
			if isValidCheck(mask, fc) {
				flatChecks = append(flatChecks, fc)
			}
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

// getAggregatedResults aggregates the passed results by groupBy and unit
func (a *aggregatorImpl) getAggregatedResults(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope, runResults []*storage.ComplianceRunResults, mask [numScopes]set.StringSet) ([]*v1.ComplianceAggregation_Result, DomainFunc) {
	var flatChecks []flatCheck
	var domainIndices []domainOffsetPair
	for _, r := range runResults {
		flatChecks = append(flatChecks, a.getFlatChecksFromRunResult(r, mask)...)
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
	sortAggregations(results)
	return results, func(i int) *storage.ComplianceDomain { return domainMap[i] }
}

// getCheckMask returns an array of ComplianceAggregation scopes that contains a set of IDs that are allowed
// if the set is nil, then it means all are allowed
func (a *aggregatorImpl) getCheckMask(query *v1.Query, queryMap map[string][]string, standardIDs, clusterIDs []string) ([numScopes]set.StringSet, error) {
	var mask [numScopes]set.StringSet
	if search.HasApplicableOptions(queryMap, nodeMappings.OptionsMap) {
		results, err := a.nodes.Search(query)
		if err != nil {
			return mask, err
		}
		mask[getMaskIndex(v1.ComplianceAggregation_NODE)] = set.NewStringSet(search.ResultsToIDs(results)...)
	}
	if search.HasApplicableOptions(queryMap, namespaceMappings.OptionsMap) {
		results, err := a.namespaces.Search(query)
		if err != nil {
			return mask, err
		}
		mask[getMaskIndex(v1.ComplianceAggregation_NAMESPACE)] = set.NewStringSet(search.ResultsToIDs(results)...)
	}
	if search.HasApplicableOptions(queryMap, standardsIndex.ControlOptions) {
		results, err := a.standards.SearchControls(query)
		if err != nil {
			return mask, err
		}
		mask[getMaskIndex(v1.ComplianceAggregation_CONTROL)] = set.NewStringSet(search.ResultsToIDs(results)...)
	}
	return mask, nil
}

// getRunParameters returns the standard IDs and the cluster IDs for the query
func (a *aggregatorImpl) getRunParameters(query *v1.Query, queryMap map[string][]string) (standardIDs, clusterIDs []string, err error) {
	if search.HasApplicableOptions(queryMap, standardsIndex.StandardOptions) {
		results, err := a.standards.SearchStandards(query)
		if err != nil {
			return nil, nil, err
		}
		standardIDs = search.ResultsToIDs(results)
	} else {
		standards, err := a.standards.Standards()
		if err != nil {
			return nil, nil, err
		}
		for _, s := range standards {
			standardIDs = append(standardIDs, s.GetId())
		}
	}

	if search.HasApplicableOptions(queryMap, clusterMappings.OptionsMap) {
		results, err := a.clusters.Search(query)
		if err != nil {
			return nil, nil, err
		}
		clusterIDs = search.ResultsToIDs(results)
	} else {
		clusters, err := a.clusters.GetClusters()
		if err != nil {
			return nil, nil, err
		}
		for _, c := range clusters {
			clusterIDs = append(clusterIDs, c.GetId())
		}
	}

	return
}
