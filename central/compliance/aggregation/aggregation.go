package aggregation

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	complianceDSTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	standardsIndex "github.com/stackrox/rox/central/compliance/standards/index"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	namespaceMappings "github.com/stackrox/rox/central/namespace/index/mappings"
	nodeDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	nodeMappings "github.com/stackrox/rox/central/node/index/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	passFailCountsByState = map[storage.ComplianceState]passFailCounts{
		storage.ComplianceState_COMPLIANCE_STATE_SUCCESS: {pass: 1},
		storage.ComplianceState_COMPLIANCE_STATE_FAILURE: {fail: 1},
		storage.ComplianceState_COMPLIANCE_STATE_ERROR:   {fail: 1},
		storage.ComplianceState_COMPLIANCE_STATE_SKIP:    {skipped: 1},
		storage.ComplianceState_COMPLIANCE_STATE_NOTE:    {skipped: 1},
	}
)

const (
	minScope  = v1.ComplianceAggregation_STANDARD
	maxScope  = v1.ComplianceAggregation_CHECK
	numScopes = maxScope - minScope + 1
)

type flatCheckValues [numScopes]string

func (f *flatCheckValues) get(scope v1.ComplianceAggregation_Scope) string {
	return f[scope-minScope]
}

type mask [numScopes]set.StringSet

func (m *mask) set(scope v1.ComplianceAggregation_Scope, s set.StringSet) {
	if m == nil {
		return
	}
	m[scope-minScope] = s
}

func (m *mask) get(scope v1.ComplianceAggregation_Scope) set.StringSet {
	if m == nil {
		return nil
	}
	return m[scope-minScope]
}

func (m *mask) matchesValue(scope v1.ComplianceAggregation_Scope, v string) bool {
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
	Aggregate(ctx context.Context, query string, groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) ([]*v1.ComplianceAggregation_Result, []*v1.ComplianceAggregation_Source, map[*v1.ComplianceAggregation_Result]*storage.ComplianceDomain, error)

	GetResultsWithEvidence(ctx context.Context, queryString string) ([]*storage.ComplianceRunResults, error)

	// Search runs search requests in the context of the aggregator.
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// New returns a new aggregator
func New(compliance complianceDS.DataStore,
	standards standards.Repository,
	clusters clusterDatastore.DataStore,
	namespaces namespaceStore.DataStore,
	nodes nodeDatastore.GlobalDataStore,
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
	nodes       nodeDatastore.GlobalDataStore
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
	values *flatCheckValues
	state  storage.ComplianceState
}

func (c flatCheck) passFailCounts() passFailCounts {
	return passFailCountsByState[c.state]
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

func (a *aggregatorImpl) getResultsAndMask(ctx context.Context, queryString string, flags complianceDSTypes.GetFlags) ([]*storage.ComplianceRunResults, []*v1.ComplianceAggregation_Source, *mask, error) {
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
		mask.set(v1.ComplianceAggregation_CLUSTER, set.NewStringSet(clusterIDs...))
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
func (a *aggregatorImpl) Aggregate(ctx context.Context, queryString string, groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) ([]*v1.ComplianceAggregation_Result, []*v1.ComplianceAggregation_Source, map[*v1.ComplianceAggregation_Result]*storage.ComplianceDomain, error) {
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
			v1.ComplianceAggregation_CLUSTER - minScope:    clusterID,
			v1.ComplianceAggregation_NAMESPACE - minScope:  namespaceID,
			v1.ComplianceAggregation_STANDARD - minScope:   standardID,
			v1.ComplianceAggregation_CATEGORY - minScope:   category,
			v1.ComplianceAggregation_CONTROL - minScope:    controlID,
			v1.ComplianceAggregation_NODE - minScope:       nodeID,
			v1.ComplianceAggregation_DEPLOYMENT - minScope: deploymentID,
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
		utils.Should(errors.Errorf("no category found for control %q", controlID))
		return ""
	}
	return category.QualifiedID()
}

func (a *aggregatorImpl) flatChecksFromCluster(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string) []flatCheck {
	if controlSet := mask.get(v1.ComplianceAggregation_CONTROL); controlSet != nil {
		flatChecks := make([]flatCheck, 0, len(controlSet))
		for control := range controlSet {
			r := runResults.GetClusterResults().GetControlResults()[control]
			if r == nil {
				continue
			}
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
			if isValidCheck(mask, fc) {
				flatChecks = append(flatChecks, fc)
			}
		}
		return flatChecks
	}
	flatChecks := make([]flatCheck, 0, len(runResults.GetClusterResults().GetControlResults()))
	for control, r := range runResults.GetClusterResults().GetControlResults() {
		fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, "", "", r.GetOverallState())
		if isValidCheck(mask, fc) {
			flatChecks = append(flatChecks, fc)
		}
	}
	return flatChecks
}

func (a *aggregatorImpl) flatChecksFromNodes(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string) []flatCheck {
	var flatChecks []flatCheck
	if nodeSet := mask.get(v1.ComplianceAggregation_NODE); nodeSet != nil {
		for node := range nodeSet {
			controlResults := runResults.GetNodeResults()[node]
			for control, r := range controlResults.GetControlResults() {
				fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, node, "", r.GetOverallState())
				if isValidCheck(mask, fc) {
					flatChecks = append(flatChecks, fc)
				}
			}
		}
		return flatChecks
	}
	for n, controlResults := range runResults.GetNodeResults() {
		for control, r := range controlResults.GetControlResults() {
			fc := newFlatCheck(clusterID, "", standardID, a.getCategoryID(control), control, n, "", r.GetOverallState())
			if isValidCheck(mask, fc) {
				flatChecks = append(flatChecks, fc)
			}
		}
	}
	return flatChecks
}

func (a *aggregatorImpl) getFlatChecksFromDeployments(runResults *storage.ComplianceRunResults, mask *mask, clusterID, standardID string) []flatCheck {
	domain := runResults.GetDomain()
	var flatChecks []flatCheck
	if deploymentSet := mask.get(v1.ComplianceAggregation_DEPLOYMENT); deploymentSet != nil {
		for deploymentID := range deploymentSet {
			deployment := domain.Deployments[deploymentID]
			if deployment == nil {
				continue
			}
			if !mask.matchesValue(v1.ComplianceAggregation_NAMESPACE, deployment.GetNamespaceId()) {
				continue
			}
			for control, r := range runResults.GetDeploymentResults()[deploymentID].GetControlResults() {
				fc := newFlatCheck(clusterID, deployment.GetNamespaceId(), standardID, a.getCategoryID(control), control, "", deployment.GetId(), r.GetOverallState())
				if isValidCheck(mask, fc) {
					flatChecks = append(flatChecks, fc)
				}
			}
		}
		return flatChecks
	}

	for d, controlResults := range runResults.GetDeploymentResults() {
		deployment, ok := domain.Deployments[d]
		if !ok {
			log.Errorf("result for deployment %s exists, but it is not included in the domain", d)
			continue
		}
		if !mask.matchesValue(v1.ComplianceAggregation_NAMESPACE, deployment.GetNamespaceId()) {
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

func (a *aggregatorImpl) getFlatChecksFromRunResult(runResults *storage.ComplianceRunResults, mask *mask) []flatCheck {
	clusterID := runResults.GetDomain().GetCluster().GetId()
	standardID := runResults.GetRunMetadata().GetStandardId()

	flatChecks := a.flatChecksFromCluster(runResults, mask, clusterID, standardID)
	flatChecks = append(flatChecks, a.flatChecksFromNodes(runResults, mask, clusterID, standardID)...)
	flatChecks = append(flatChecks, a.getFlatChecksFromDeployments(runResults, mask, clusterID, standardID)...)
	return flatChecks
}

// DomainFunc will return a valid storage domain for a given key, if it exists. If multiple domains match, only one will be returned.
type DomainFunc func(i int) *storage.ComplianceDomain

type domainOffsetPair struct {
	offset int
	domain *storage.ComplianceDomain
}

// getAggregatedResults aggregates the passed results by groupBy and unit
func (a *aggregatorImpl) getAggregatedResults(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope, runResults []*storage.ComplianceRunResults, mask *mask) ([]*v1.ComplianceAggregation_Result, map[*v1.ComplianceAggregation_Result]*storage.ComplianceDomain) {
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
			val := fc.values.get(s)
			if val == "" {
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
	domainMap := make(map[*v1.ComplianceAggregation_Result]*storage.ComplianceDomain)
	for key, checks := range groups {
		unitMap := make(map[string]passFailCounts)
		for i, c := range checks {
			unitKey := c.values.get(unit)
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

		result := &v1.ComplianceAggregation_Result{
			AggregationKeys: getAggregationKeys(key),
			Unit:            unit,
			NumPassing:      int32(counts.pass),
			NumFailing:      int32(counts.fail),
			NumSkipped:      int32(counts.skipped),
		}
		domainMap[result] = domains[key]
		results = append(results, result)
	}
	sortAggregations(results)
	return results, domainMap
}

type searchFuncAndOptionsMap struct {
	searchFunc func(context.Context, *v1.Query) ([]search.Result, error)
	optionsMap search.OptionsMap
}

func wrapContextLessSearchFunc(f func(*v1.Query) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error) {
	return func(_ context.Context, q *v1.Query) ([]search.Result, error) {
		return f(q)
	}
}

func (a *aggregatorImpl) getSearchFuncs() map[v1.ComplianceAggregation_Scope]searchFuncAndOptionsMap {
	// Careful: If you modify something here, be sure to also modify the options multimap in
	// `compliance/search/options.go`.
	return map[v1.ComplianceAggregation_Scope]searchFuncAndOptionsMap{
		v1.ComplianceAggregation_STANDARD: {
			searchFunc: wrapContextLessSearchFunc(a.standards.SearchStandards),
			optionsMap: standardsIndex.StandardOptions,
		},
		v1.ComplianceAggregation_CLUSTER: {
			searchFunc: a.clusters.Search,
			optionsMap: clusterMappings.OptionsMap,
		},
		v1.ComplianceAggregation_NODE: {
			searchFunc: a.nodes.Search,
			optionsMap: nodeMappings.OptionsMap,
		},
		v1.ComplianceAggregation_NAMESPACE: {
			searchFunc: a.namespaces.Search,
			optionsMap: namespaceMappings.OptionsMap,
		},
		v1.ComplianceAggregation_CONTROL: {
			searchFunc: wrapContextLessSearchFunc(a.standards.SearchControls),
			optionsMap: standardsIndex.ControlOptions,
		},
		v1.ComplianceAggregation_DEPLOYMENT: {
			searchFunc: a.deployments.Search,
			optionsMap: deployments.OptionsMap,
		},
	}
}

func (a *aggregatorImpl) getResultsFromScope(ctx context.Context, scope v1.ComplianceAggregation_Scope, query *v1.Query, querySpecifiedFields []string) (results []search.Result, wasApplicable bool, err error) {
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

func (a *aggregatorImpl) addSetToMaskIfOptionsApplicable(ctx context.Context, scope v1.ComplianceAggregation_Scope, mask *mask,
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

	err := a.addSetToMaskIfOptionsApplicable(ctx, v1.ComplianceAggregation_NODE, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, v1.ComplianceAggregation_NAMESPACE, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, v1.ComplianceAggregation_CONTROL, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	err = a.addSetToMaskIfOptionsApplicable(ctx, v1.ComplianceAggregation_DEPLOYMENT, &mask, query, querySpecifiedFields)
	if err != nil {
		return nil, err
	}

	return &mask, nil
}

func (a *aggregatorImpl) getStandardsToRun(ctx context.Context, query *v1.Query, querySpecifiedFields []string) ([]string, error) {
	results, wasApplicable, err := a.getResultsFromScope(ctx, v1.ComplianceAggregation_STANDARD, query, querySpecifiedFields)
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
	results, wasApplicable, err := a.getResultsFromScope(ctx, v1.ComplianceAggregation_CLUSTER, query, querySpecifiedFields)
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
