package aggregation

import (
	"context"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	complianceDSTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	standardsIndex "github.com/stackrox/rox/central/compliance/standards/index"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

const (
	minScope  = storage.ComplianceAggregation_STANDARD
	maxScope  = storage.ComplianceAggregation_CHECK
	numScopes = maxScope - minScope + 1

	checkName = "check"
)

type flatCheckValues [numScopes]string

func (f *flatCheckValues) get(scope storage.ComplianceAggregation_Scope) string {
	if scope == storage.ComplianceAggregation_CHECK {
		return checkName
	}
	return f[scope-minScope]
}

type mask [numScopes]set.StringSet

func (m *mask) set(scope storage.ComplianceAggregation_Scope, s set.StringSet) {
	if m == nil {
		return
	}
	m[scope-minScope] = s
}

func (m *mask) get(scope storage.ComplianceAggregation_Scope) set.StringSet {
	if m == nil {
		return nil
	}
	return m[scope-minScope]
}

func (m *mask) matchesValue(scope storage.ComplianceAggregation_Scope, v string) bool {
	if m == nil {
		return true
	}
	if valueSet := m.get(scope); valueSet != nil {
		return valueSet.Contains(v)
	}
	return true
}

// Aggregator does compliance aggregation
type Aggregator interface {
	Aggregate(ctx context.Context, query string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error)

	GetResultsWithEvidence(ctx context.Context, queryString string) ([]*storage.ComplianceRunResults, error)

	// Search runs search requests in the context of the aggregator.
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	// Count runs requests in the context of the aggregator and return the count of results.
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new aggregator
func New(compliance complianceDS.DataStore,
	standards standards.Repository,
	clusters clusterDatastore.DataStore,
	namespaces namespaceStore.DataStore,
	nodes nodeDatastore.DataStore,
	deployments deploymentStore.DataStore) Aggregator {
	return &aggregatorImpl{
		compliance:  compliance,
		standards:   standards,
		clusters:    clusters,
		namespaces:  namespaces,
		nodes:       nodes,
		deployments: deployments,
	}
}

type aggregatorImpl struct {
	compliance  complianceDS.DataStore
	standards   standards.Repository
	clusters    clusterDatastore.DataStore
	namespaces  namespaceStore.DataStore
	nodes       nodeDatastore.DataStore
	deployments deploymentStore.DataStore
}

func (a *aggregatorImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	var allResults []search.Result
	specifiedFields := getSpecifiedFieldsFromQuery(q)
	for category, searchFuncAndMap := range a.getSearchFuncs() {
		if !search.HasApplicableOptions(specifiedFields, searchFuncAndMap.optionsMap) {
			continue
		}
		results, err := searchFuncAndMap.searchFunc(ctx, q)
		if err != nil {
			return nil, errors.Wrapf(err, "searching category %s", category)
		}
		allResults = append(allResults, results...)
	}
	return allResults, nil
}

// Count returns the number of search results from the query
func (a *aggregatorImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	totalResults := 0
	specifiedFields := getSpecifiedFieldsFromQuery(q)
	for category, searchFuncAndMap := range a.getSearchFuncs() {
		if !search.HasApplicableOptions(specifiedFields, searchFuncAndMap.optionsMap) {
			continue
		}
		numResults, err := searchFuncAndMap.count(ctx, q)
		if err != nil {
			return 0, errors.Wrapf(err, "searching category %s", category)
		}
		totalResults += numResults
	}
	return totalResults, nil
}

type passFailCounts struct {
	pass    int
	fail    int
	skipped int
}

func (c passFailCounts) Add(other passFailCounts) passFailCounts {
	return passFailCounts{
		pass:    c.pass + other.pass,
		fail:    c.fail + other.fail,
		skipped: c.skipped + other.skipped,
	}
}

func (c passFailCounts) Reduce() passFailCounts {
	if c.fail > 0 {
		return passFailCounts{fail: 1}
	}
	if c.pass > 0 {
		return passFailCounts{pass: 1}
	}
	if c.skipped > 0 {
		return passFailCounts{skipped: 1}
	}
	return passFailCounts{}
}

type groupByKey [numScopes]string

func (k groupByKey) Get(scope storage.ComplianceAggregation_Scope) string {
	if scope < minScope || scope > maxScope {
		log.Errorf("Unknown scope: %v", scope)
		return ""
	}
	return k[scope-minScope]
}

func (k *groupByKey) Set(scope storage.ComplianceAggregation_Scope, value string) {
	if scope < minScope || scope > maxScope {
		log.Errorf("Unknown scope: %v", scope)
		return
	}
	(*k)[scope-minScope] = value
}

type groupByValue struct {
	unitMap map[string]*passFailCounts
	domain  *storage.ComplianceDomain
}

func newGroupByValue(domain *storage.ComplianceDomain) *groupByValue {
	return &groupByValue{
		unitMap: make(map[string]*passFailCounts),
		domain:  domain,
	}
}

// flatCheck is basically all of the check result data flattened into a single object
type flatCheck struct {
	values *flatCheckValues
	state  storage.ComplianceState
}

func getSpecifiedFieldsFromQuery(q *v1.Query) []string {
	var querySpecifiedFields []string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		if bq == nil {
			return
		}
		asMFQ, ok := bq.Query.(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		querySpecifiedFields = append(querySpecifiedFields, asMFQ.MatchFieldQuery.GetField())
	})
	return querySpecifiedFields
}

func (a *aggregatorImpl) filterOnRunResult(runResults *storage.ComplianceRunResults, mask *mask) {
	domain := runResults.GetDomain()
	clusterID := runResults.GetDomain().GetCluster().GetId()
	standardID := runResults.GetRunMetadata().GetStandardId()

	for control, r := range runResults.GetClusterResults().GetControlResults() {
		fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
		if !isValidCheck(mask, fc) {
			delete(runResults.GetClusterResults().GetControlResults(), control)
		}
	}
	for n, controlResults := range runResults.GetNodeResults() {
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, n, "", r.GetOverallState())
			if !isValidCheck(mask, fc) {
				delete(controlResults.GetControlResults(), control)
			}
		}
	}
	for d, controlResults := range runResults.GetDeploymentResults() {
		deployment, ok := domain.Deployments[d]
		if !ok {
			log.Error("Okay that's not good, we have a result for a deployment that isn't even in the domain?")
			continue
		}
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, deployment.GetNamespaceId(), standardID, a.getCategoryID(control), control, "", deployment.GetId(), r.GetOverallState())
			if !isValidCheck(mask, fc) {
				delete(controlResults.GetControlResults(), control)
			}
		}
	}
}

func (a *aggregatorImpl) getResultsAndMask(ctx context.Context, queryString string, flags complianceDSTypes.GetFlags) ([]*storage.ComplianceRunResults, []*storage.ComplianceAggregation_Source, *mask, error) {
	query, err := search.ParseQuery(queryString, search.MatchAllIfEmpty())
	if err != nil {
		return nil, nil, nil, err
	}
	querySpecifiedFields := getSpecifiedFieldsFromQuery(query)

	standardIDs, err := a.getStandardsToRun(ctx, query, querySpecifiedFields)
	if err != nil {
		return nil, nil, nil, err
	}

	clusterIDs, clusterQueryWasApplicable, err := a.getClustersToRun(ctx, query, querySpecifiedFields)
	if err != nil {
		return nil, nil, nil, err
	}

	runResults, err := a.compliance.GetLatestRunResultsBatch(ctx, clusterIDs, standardIDs, flags)
	if err != nil {
		return nil, nil, nil, err
	}

	validResults, sources := complianceDS.ValidResultsAndSources(runResults)

	mask, err := a.getCheckMask(ctx, query, querySpecifiedFields)
	if err != nil {
		return nil, nil, nil, err
	}

	if clusterQueryWasApplicable {
		mask.set(storage.ComplianceAggregation_CLUSTER, set.NewStringSet(clusterIDs...))
	}
	return validResults, sources, mask, err
}

func (a *aggregatorImpl) GetResultsWithEvidence(ctx context.Context, queryString string) ([]*storage.ComplianceRunResults, error) {
	validResults, _, mask, err := a.getResultsAndMask(ctx, queryString, complianceDSTypes.RequireMessageStrings)
	if err != nil {
		return nil, err
	}
	for _, r := range validResults {
		a.filterOnRunResult(r, mask)
	}
	return validResults, nil
}

// Aggregate takes in a search query, groupby scopes and unit scope and returns the results of the aggregation
func (a *aggregatorImpl) Aggregate(ctx context.Context, queryString string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error) {
	for _, group := range groupBy {
		if group < minScope {
			return nil, nil, nil, errors.Errorf("group %s is not a valid scope to run aggregation on", group)
		}
	}
	if unit < minScope {
		return nil, nil, nil, errors.Errorf("unit %s is not a valid scope to run aggregation on", unit)
	}

	validResults, sources, mask, err := a.getResultsAndMask(ctx, queryString, 0)
	if err != nil {
		return nil, nil, nil, err
	}

	results, domainMap := a.getAggregatedResults(groupBy, unit, validResults, mask)

	return results, sources, domainMap, nil

}

func newFlatCheck(clusterID, namespaceID, standardID, category, controlID, nodeID, deploymentID string, state storage.ComplianceState) flatCheck {
	return flatCheck{
		values: &flatCheckValues{
			storage.ComplianceAggregation_CLUSTER - minScope:    clusterID,
			storage.ComplianceAggregation_NAMESPACE - minScope:  namespaceID,
			storage.ComplianceAggregation_STANDARD - minScope:   standardID,
			storage.ComplianceAggregation_CATEGORY - minScope:   category,
			storage.ComplianceAggregation_CONTROL - minScope:    controlID,
			storage.ComplianceAggregation_NODE - minScope:       nodeID,
			storage.ComplianceAggregation_DEPLOYMENT - minScope: deploymentID,
		},
		state: state,
	}
}

func isValidCheck(mask *mask, fc flatCheck) bool {
	if mask == nil {
		return true
	}
	var failedOnEmpty, hadMatch bool
	for i := range mask {
		if valueSet := mask[i]; valueSet != nil {
			if !valueSet.Contains(fc.values[i]) {
				if fc.values[i] == "" {
					failedOnEmpty = true
					continue
				}
				return false
			}
			hadMatch = true
		}
	}
	if failedOnEmpty && !hadMatch {
		return false
	}
	return true
}

func getAggregationKeys(groupByKey groupByKey) []*storage.ComplianceAggregation_AggregationKey {
	var aggregationKeys []*storage.ComplianceAggregation_AggregationKey
	for i, val := range groupByKey {
		if val == "" {
			continue
		}
		aggregationKeys = append(aggregationKeys, &storage.ComplianceAggregation_AggregationKey{
			Scope: storage.ComplianceAggregation_Scope(i + int(minScope)),
			Id:    val,
		})
	}
	return aggregationKeys
}

func (a *aggregatorImpl) getCategoryID(controlID string) string {
	// Controls can now be removed with the addition of the compliance operator
	// All controls should have categories if they exist
	category := a.standards.GetCategoryByControl(controlID)
	if category == nil {
		return ""
	}
	return category.QualifiedID()
}

// DomainFunc will return a valid storage domain for a given key, if it exists. If multiple domains match, only one will be returned.
type DomainFunc func(i int) *storage.ComplianceDomain

func isValidUnit(unit storage.ComplianceAggregation_Scope, fc flatCheck) bool {
	return unit == storage.ComplianceAggregation_CHECK || fc.values.get(unit) != ""
}

func getGroupByKey(groupBy []storage.ComplianceAggregation_Scope, fc flatCheck) (groupByKey, bool) {
	var key groupByKey
	valid := true
	for _, s := range groupBy {
		val := fc.values.get(s)
		if val == "" {
			valid = false
			break
		}
		key.Set(s, val)
	}
	return key, valid
}

func handleResult(groups map[groupByKey]*groupByValue, key groupByKey, unitKey string, domain *storage.ComplianceDomain, r *storage.ComplianceResultValue) {
	groupByValue := groups[key]
	if groupByValue == nil {
		groupByValue = newGroupByValue(domain)
		groups[key] = groupByValue
	}
	pfCounts := groupByValue.unitMap[unitKey]
	if pfCounts == nil {
		pfCounts = new(passFailCounts)
		groupByValue.unitMap[unitKey] = pfCounts
	}
	switch r.OverallState {
	case storage.ComplianceState_COMPLIANCE_STATE_SUCCESS:
		pfCounts.pass++
	case storage.ComplianceState_COMPLIANCE_STATE_FAILURE, storage.ComplianceState_COMPLIANCE_STATE_ERROR, storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN:
		pfCounts.fail++
	case storage.ComplianceState_COMPLIANCE_STATE_SKIP, storage.ComplianceState_COMPLIANCE_STATE_NOTE:
		pfCounts.skipped++
	}
}

func processFlatCheck(groups map[groupByKey]*groupByValue, fc flatCheck, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope,
	mask *mask, runResults *storage.ComplianceRunResults, r *storage.ComplianceResultValue) {
	if !isValidUnit(unit, fc) || !isValidCheck(mask, fc) {
		return
	}
	groupKey, valid := getGroupByKey(groupBy, fc)
	if !valid {
		return
	}
	handleResult(groups, groupKey, fc.values.get(unit), runResults.GetDomain(), r)
}

func (a *aggregatorImpl) aggregateFromCluster(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string,
	groups map[groupByKey]*groupByValue, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) {
	if controlSet := mask.get(storage.ComplianceAggregation_CONTROL); controlSet != nil {
		for control := range controlSet {
			r := runResults.GetClusterResults().GetControlResults()[control]
			if r == nil {
				// Check to see if the control is a machine config control
				for _, controlResults := range runResults.GetMachineConfigResults() {
					if r := controlResults.GetControlResults()[control]; r != nil {
						fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
						processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
					}
				}
				continue
			}
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
			processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
		}
		return
	}
	for control, r := range runResults.GetClusterResults().GetControlResults() {
		fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
		processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
	}

	for _, controlResults := range runResults.GetMachineConfigResults() {
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
			processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
		}
	}
}

func (a *aggregatorImpl) aggregateFromNodes(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string, groups map[groupByKey]*groupByValue,
	groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) {
	if nodeSet := mask.get(storage.ComplianceAggregation_NODE); nodeSet != nil {
		for node := range nodeSet {
			controlResults := runResults.GetNodeResults()[node]
			for control, r := range controlResults.GetControlResults() {
				fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, node, "", r.GetOverallState())
				processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
			}
		}
		return
	}
	for node, controlResults := range runResults.GetNodeResults() {
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, node, "", r.GetOverallState())
			processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
		}
	}
}

func (a *aggregatorImpl) aggregateFromDeployments(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string,
	groups map[groupByKey]*groupByValue, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) {
	domain := runResults.GetDomain()
	if deploymentSet := mask.get(storage.ComplianceAggregation_DEPLOYMENT); deploymentSet != nil {
		for deploymentID := range deploymentSet {
			deployment := domain.Deployments[deploymentID]
			if deployment == nil {
				continue
			}
			if !mask.matchesValue(storage.ComplianceAggregation_NAMESPACE, deployment.GetNamespaceId()) {
				continue
			}
			for control, r := range runResults.GetDeploymentResults()[deploymentID].GetControlResults() {
				fc := newFlatCheck(clusterID, deployment.GetNamespaceId(), standardID, a.getCategoryID(control), control, "", deployment.GetId(), r.GetOverallState())
				processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
			}
		}
		return
	}

	for d, controlResults := range runResults.GetDeploymentResults() {
		deployment, ok := domain.Deployments[d]
		if !ok {
			log.Errorf("result for deployment %s exists, but it is not included in the domain", d)
			continue
		}
		if !mask.matchesValue(storage.ComplianceAggregation_NAMESPACE, deployment.GetNamespaceId()) {
			continue
		}
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, deployment.GetNamespaceId(), standardID, a.getCategoryID(control), control, "", deployment.GetId(), r.GetOverallState())
			processFlatCheck(groups, fc, groupBy, unit, mask, runResults, r)
		}
	}
}

func (a *aggregatorImpl) getAggregatedResults(groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope, runResults []*storage.ComplianceRunResults, mask *mask) ([]*storage.ComplianceAggregation_Result, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain) {
	groups := make(map[groupByKey]*groupByValue)
	for _, r := range runResults {
		clusterID := r.GetDomain().GetCluster().GetId()
		standardID := r.GetRunMetadata().GetStandardId()

		a.aggregateFromCluster(r, mask, clusterID, standardID, groups, groupBy, unit)
		a.aggregateFromNodes(r, mask, clusterID, standardID, groups, groupBy, unit)
		a.aggregateFromDeployments(r, mask, clusterID, standardID, groups, groupBy, unit)
	}

	results := make([]*storage.ComplianceAggregation_Result, 0, len(groups))
	domainMap := make(map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain)
	for key, groupByValue := range groups {
		var counts passFailCounts
		if unit != storage.ComplianceAggregation_CHECK {
			for _, u := range groupByValue.unitMap {
				counts = counts.Add(u.Reduce())
			}
		} else {
			// If looking at a check, don't reduce and instead just take the value
			// of the unit map
			counts = *groupByValue.unitMap[checkName]
		}

		result := &storage.ComplianceAggregation_Result{
			AggregationKeys: getAggregationKeys(key),
			Unit:            unit,
			NumPassing:      int32(counts.pass),
			NumFailing:      int32(counts.fail),
			NumSkipped:      int32(counts.skipped),
		}
		results = append(results, result)
		domainMap[result] = groupByValue.domain
	}
	sortAggregations(results)
	return results, domainMap
}

type searchFuncAndOptionsMap struct {
	// The function used for searching. Must not be nil.
	searchFunc func(context.Context, *v1.Query) ([]search.Result, error)
	// The function used for counting. In contrast to searchFunc, this may be nil, in
	// which case counting is done via searching.
	countFunc  func(context.Context, *v1.Query) (int, error)
	optionsMap search.OptionsMap
}

func (s *searchFuncAndOptionsMap) count(ctx context.Context, q *v1.Query) (int, error) {
	if s.countFunc != nil {
		return s.countFunc(ctx, q)
	}
	results, err := s.searchFunc(ctx, q)
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

func wrapContextLessSearchFunc(f func(*v1.Query) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error) {
	return func(_ context.Context, q *v1.Query) ([]search.Result, error) {
		return f(q)
	}
}

func (a *aggregatorImpl) getSearchFuncs() map[storage.ComplianceAggregation_Scope]searchFuncAndOptionsMap {
	// Careful: If you modify something here, be sure to also modify the options multimap in
	// `compliance/search/options.go`.
	return map[storage.ComplianceAggregation_Scope]searchFuncAndOptionsMap{
		storage.ComplianceAggregation_STANDARD: {
			searchFunc: wrapContextLessSearchFunc(a.standards.SearchStandards),
			optionsMap: standardsIndex.StandardOptions,
		},
		storage.ComplianceAggregation_CLUSTER: {
			searchFunc: a.clusters.Search,
			countFunc:  a.clusters.Count,
			optionsMap: schema.ClustersSchema.OptionsMap,
		},
		storage.ComplianceAggregation_NODE: {
			searchFunc: a.nodes.Search,
			countFunc:  a.nodes.Count,
			optionsMap: schema.NodesSchema.OptionsMap,
		},
		storage.ComplianceAggregation_NAMESPACE: {
			searchFunc: a.namespaces.Search,
			countFunc:  a.namespaces.Count,
			optionsMap: schema.NamespacesSchema.OptionsMap,
		},
		storage.ComplianceAggregation_CONTROL: {
			searchFunc: wrapContextLessSearchFunc(a.standards.SearchControls),
			optionsMap: standardsIndex.ControlOptions,
		},
		storage.ComplianceAggregation_DEPLOYMENT: {
			searchFunc: a.deployments.Search,
			countFunc:  a.deployments.Count,
			optionsMap: deployments.OptionsMap,
		},
	}
}

func (a *aggregatorImpl) getResultsFromScope(ctx context.Context, scope storage.ComplianceAggregation_Scope, query *v1.Query, querySpecifiedFields []string) (results []search.Result, wasApplicable bool, err error) {
	funcAndMap, ok := a.getSearchFuncs()[scope]
	// Programming error.
	if !ok {
		utils.Should(errors.Errorf("No search func registered for scope: %s", scope))
		return
	}
	wasApplicable = search.HasApplicableOptions(querySpecifiedFields, funcAndMap.optionsMap)
	if !wasApplicable {
		return
	}
	results, err = funcAndMap.searchFunc(ctx, query)
	return
}

func (a *aggregatorImpl) addSetToMaskIfOptionsApplicable(ctx context.Context, scope storage.ComplianceAggregation_Scope, mask *mask,
	query *v1.Query, querySpecifiedFields []string) error {

	results, wasApplicable, err := a.getResultsFromScope(ctx, scope, query, querySpecifiedFields)
	if err != nil {
		return err
	}
	if !wasApplicable {
		return nil
	}

	mask.set(scope, search.ResultsToIDSet(results))
	return nil
}

// getCheckMask returns an array of ComplianceAggregation scopes that contains a set of IDs that are allowed
// if the set is nil, then it means all are allowed
func (a *aggregatorImpl) getCheckMask(ctx context.Context, query *v1.Query, querySpecifiedFields []string) (*mask, error) {
	var mask mask

	err := a.addSetToMaskIfOptionsApplicable(ctx, storage.ComplianceAggregation_NODE, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, storage.ComplianceAggregation_NAMESPACE, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, storage.ComplianceAggregation_CONTROL, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, storage.ComplianceAggregation_DEPLOYMENT, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	return &mask, nil
}

func (a *aggregatorImpl) getStandardsToRun(ctx context.Context, query *v1.Query, querySpecifiedFields []string) ([]string, error) {
	results, wasApplicable, err := a.getResultsFromScope(ctx, storage.ComplianceAggregation_STANDARD, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}
	if wasApplicable {
		return search.ResultsToIDs(results), nil
	}
	stds, err := a.standards.Standards()
	if err != nil {
		return nil, err
	}
	standardIDs := make([]string, 0, len(stds))
	for _, s := range stds {
		standardIDs = append(standardIDs, s.GetId())
	}
	return standardIDs, nil
}

func (a *aggregatorImpl) getClustersToRun(ctx context.Context, query *v1.Query, querySpecifiedFields []string) ([]string, bool, error) {
	results, wasApplicable, err := a.getResultsFromScope(ctx, storage.ComplianceAggregation_CLUSTER, query, querySpecifiedFields)
	if err != nil {
		return nil, false, err
	}
	if wasApplicable {
		return search.ResultsToIDs(results), true, nil
	}
	clusters, err := a.clusters.GetClusters(ctx)
	if err != nil {
		return nil, false, err
	}
	clusterIDs := make([]string, 0, len(clusters))
	for _, c := range clusters {
		clusterIDs = append(clusterIDs, c.GetId())
	}
	return clusterIDs, false, nil
}
