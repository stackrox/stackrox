package resolvers

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/converter/utils"
	distroctx "github.com/stackrox/rox/central/graphql/resolvers/distroctx"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/vulnerabilityrequest/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/stringutils"
)

// V2 Connections to root.
//////////////////////////

var (
	cvePredicateFactory        = predicate.NewFactory("cve", &storage.CVE{})
	cvePostFilteringOptionsMap = func() search.OptionsMap {
		opts := search.Walk(v1.SearchCategory_VULNERABILITIES, "cve", (*storage.CVE)(nil))

		cvss := opts.MustGet(search.CVSS.String())
		severity := opts.MustGet(search.Severity.String())

		return search.NewOptionsMap(v1.SearchCategory_VULNERABILITIES).
			Add(search.CVSS, cvss).
			Add(search.Severity, severity)
	}()
)

func getImageIDFromQuery(q *v1.Query) string {
	if q == nil {
		return ""
	}
	var imageID string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if strings.EqualFold(matchFieldQuery.MatchFieldQuery.GetField(), search.ImageSHA.String()) {
			imageID = matchFieldQuery.MatchFieldQuery.Value
			imageID = strings.TrimRight(imageID, `"`)
			imageID = strings.TrimLeft(imageID, `"`)
		}
	})
	return imageID
}

// AddDistroContext adds the image distribution from the query or scope query if necessary
func (resolver *Resolver) AddDistroContext(ctx context.Context, query, scopeQuery *v1.Query) (context.Context, error) {
	if distro := distroctx.FromContext(ctx); distro != "" {
		return ctx, nil
	}

	scope, hasScope := scoped.GetScopeAtLevel(ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		if image := resolver.getImage(ctx, scope.ID); image != nil {
			return distroctx.Context(ctx, image.GetScan().GetOperatingSystem()), nil
		}
	}

	imageIDFromQuery := getImageIDFromQuery(query)
	imageIDFromScope := getImageIDFromQuery(scopeQuery)

	if imageID := stringutils.FirstNonEmpty(imageIDFromQuery, imageIDFromScope); imageID != "" {
		image, exists, err := resolver.ImageDataStore.GetImageMetadata(ctx, imageID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return ctx, nil
		}
		return distroctx.Context(ctx, image.GetScan().GetOperatingSystem()), nil
	}
	return ctx, nil
}

func filterNamespacedFields(query *v1.Query, cves []*storage.CVE) ([]*storage.CVE, error) {
	vulnQuery, _ := search.FilterQueryWithMap(query, cvePostFilteringOptionsMap)
	vulnPred, err := cvePredicateFactory.GeneratePredicate(vulnQuery)
	if err != nil {
		return nil, err
	}
	filtered := cves[:0]
	for _, cve := range cves {
		if vulnPred.Matches(cve) {
			filtered = append(filtered, cve)
		}
	}
	return filtered, nil
}

func needsPostSorting(query *v1.Query) bool {
	for _, so := range query.GetPagination().GetSortOptions() {
		switch so.GetField() {
		case search.Severity.String(), search.CVSS.String(), search.CVE.String(), search.ImpactScore.String():
			return true
		default:
			return false
		}
	}
	return false
}

func sortNamespacedFields(query *v1.Query, cves []*storage.CVE) ([]*storage.CVE, error) {
	// Currently, only one sort option is supported on this endpoint
	sortOption := query.GetPagination().SortOptions[0]
	switch sortOption.Field {
	case search.Severity.String():
		sort.Slice(cves, func(i, j int) bool {
			var result bool
			if cves[i].GetSeverity() != cves[j].GetSeverity() {
				result = cves[i].GetSeverity() < cves[j].GetSeverity()
			} else {
				result = cves[i].GetCvss() < cves[j].GetCvss()
			}
			if sortOption.GetReversed() {
				return !result
			}
			return result
		})
	case search.CVSS.String():
		sort.Slice(cves, func(i, j int) bool {
			var result bool
			if cves[i].GetCvss() != cves[j].GetCvss() {
				result = cves[i].GetCvss() < cves[j].GetCvss()
			} else {
				result = cves[i].GetSeverity() < cves[j].GetSeverity()
			}
			if sortOption.Reversed {
				return !result
			}
			return result
		})
	case search.CVE.String():
		sort.Slice(cves, func(i, j int) bool {
			var result bool
			if cves[i].GetId() != cves[j].GetId() {
				result = cves[i].GetId() < cves[j].GetId()
			} else {
				result = cves[i].GetSeverity() < cves[j].GetSeverity()
			}
			if sortOption.Reversed {
				return !result
			}
			return result
		})
	case search.ImpactScore.String():
		sort.Slice(cves, func(i, j int) bool {
			var result bool
			if cves[i].GetImpactScore() != cves[j].GetImpactScore() {
				result = cves[i].GetImpactScore() < cves[j].GetImpactScore()
			} else {
				result = cves[i].GetSeverity() < cves[j].GetSeverity()
			}
			if sortOption.Reversed {
				return !result
			}
			return result
		})
	}
	return cves, nil
}

func (resolver *cVEResolver) Cvss(ctx context.Context) float64 {
	value := resolver.data.GetCvss()
	return float64(value)
}

func (resolver *cVEResolver) CvssV2(ctx context.Context) (*cVSSV2Resolver, error) {
	value := resolver.data.GetCvssV2()
	return resolver.root.wrapCVSSV2(value, true, nil)
}

func (resolver *cVEResolver) CvssV3(ctx context.Context) (*cVSSV3Resolver, error) {
	value := resolver.data.GetCvssV3()
	return resolver.root.wrapCVSSV3(value, true, nil)
}

func (resolver *Resolver) vulnerabilityV2(ctx context.Context, args IDQuery) (VulnerabilityResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("vulnerabilityV2 not supported on postgres.")
	}
	vulnResolver, err := resolver.unwrappedVulnerabilityV2(ctx, args)
	if err != nil {
		return nil, err
	}
	return vulnResolver, nil
}

func (resolver *Resolver) vulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("vulnerabilitiesV2 not supported on postgres.")
	}
	vulnResolvers, err := resolver.unwrappedVulnerabilitiesV2(ctx, args)
	if err != nil {
		return nil, err
	}
	ret := make([]VulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

func (resolver *Resolver) imageVulnerabilityV2(ctx context.Context, args IDQuery) (ImageVulnerabilityResolver, error) {
	vulnResolver, err := resolver.unwrappedVulnerabilityV2(ctx, args)
	if err != nil {
		return nil, err
	}
	return vulnResolver, nil
}

// imageVulnerabilitiesV2 wraps the resolvers as ImageVulnerabilityResolver objects to restrict the supported API
func (resolver *Resolver) imageVulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	vulnResolvers, err := resolver.unwrappedVulnerabilitiesV2(ctx, args)
	if err != nil {
		return nil, err
	}
	ret := make([]ImageVulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, err
}

func (resolver *Resolver) nodeVulnerabilityV2(ctx context.Context, args IDQuery) (NodeVulnerabilityResolver, error) {
	vulnResolver, err := resolver.unwrappedVulnerabilityV2(ctx, args)
	if err != nil {
		return nil, err
	}
	return vulnResolver, nil
}

func (resolver *Resolver) nodeVulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	vulnResolvers, err := resolver.unwrappedVulnerabilitiesV2(ctx, args)
	if err != nil {
		return nil, err
	}
	ret := make([]NodeVulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

func (resolver *Resolver) clusterVulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]ClusterVulnerabilityResolver, error) {
	vulnResolvers, err := resolver.unwrappedVulnerabilitiesV2(ctx, args)
	if err != nil {
		return nil, err
	}
	ret := make([]ClusterVulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

func (resolver *Resolver) unwrappedVulnerabilityV2(ctx context.Context, args IDQuery) (*cVEResolver, error) {
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	vuln, exists, err := resolver.CVEDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("cve not found: %s", string(*args.ID))
	}
	vulnResolver, err := resolver.wrapCVE(vuln, true, nil)
	if err != nil {
		return nil, err
	}
	vulnResolver.ctx = ctx
	return vulnResolver, nil
}

func (resolver *Resolver) unwrappedVulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]*cVEResolver, error) {
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	scopeQuery, err := args.AsV1ScopeQueryOrEmpty()
	if err != nil {
		return nil, err
	}

	ctx, err = resolver.AddDistroContext(ctx, query, scopeQuery)
	if err != nil {
		return nil, err
	}
	return resolver.unwrappedVulnerabilitiesV2Query(ctx, query)
}

func (resolver *Resolver) vulnerabilitiesV2Query(ctx context.Context, query *v1.Query) ([]VulnerabilityResolver, error) {
	vulnResolvers, err := resolver.unwrappedVulnerabilitiesV2Query(ctx, query)
	if err != nil {
		return nil, err
	}
	ret := make([]VulnerabilityResolver, 0, len(vulnResolvers))
	for _, res := range vulnResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, nil
}

func (resolver *Resolver) unwrappedVulnerabilitiesV2Query(ctx context.Context, query *v1.Query) ([]*cVEResolver, error) {
	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	query = tryUnsuppressedQuery(query)

	originalQuery := query.Clone()
	var queryModified, postSortingNeeded bool

	if distroctx.IsImageScoped(ctx) {
		query, queryModified = search.InverseFilterQueryWithMap(query, cvePostFilteringOptionsMap) // CVE queryModified
		postSortingNeeded = needsPostSorting(originalQuery)
		// We remove pagination since we want to ensure that result is correct by pushing the pagination to happen after the post sorting.
		if postSortingNeeded {
			query.Pagination = nil
		}
	}

	vulns, err := vulnLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	if queryModified {
		vulns, err = filterNamespacedFields(originalQuery, vulns)
		if err != nil {
			return nil, err
		}
	}
	if postSortingNeeded {
		vulns, err = sortNamespacedFields(originalQuery, vulns)
		if err != nil {
			return nil, err
		}
	}

	// If query was modified, it means the result was not paginated since the filtering removes pagination.
	// If post sorting was needed, which means pagination was not performed because it was removed above.
	if queryModified || postSortingNeeded {
		paginatedVulns, err := paginationWrapper{
			pv: originalQuery.GetPagination(),
		}.paginate(vulns, nil)
		if err != nil {
			return nil, err
		}
		vulns = paginatedVulns.([]*storage.CVE)
	}

	vulnResolvers, err := resolver.wrapCVEs(vulns, err)
	return vulnResolvers, err
}

func (resolver *Resolver) vulnerabilityCountV2(ctx context.Context, args RawQuery) (int32, error) {
	if features.PostgresDatastore.Enabled() {
		return 0, errors.New("vulnerabilityCountV2 not supported on postgres.")
	}
	if err := readCVEs(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	return resolver.vulnerabilityCountV2Query(ctx, query)
}

func (resolver *Resolver) vulnerabilityCountV2Query(ctx context.Context, query *v1.Query) (int32, error) {
	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return 0, err
	}

	if distroctx.IsImageScoped(ctx) {
		_, queryModified := search.InverseFilterQueryWithMap(query, cvePostFilteringOptionsMap)
		if queryModified {
			vulns, err := resolver.vulnerabilitiesV2Query(ctx, query)
			if err != nil {
				return 0, err
			}
			return int32(len(vulns)), nil
		}
	}

	query = tryUnsuppressedQuery(query)
	return vulnLoader.CountFromQuery(ctx, query)
}

func (resolver *Resolver) vulnCounterV2(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("vulnerabilityCountV2 not supported on postgres.")
	}
	if err := readCVEs(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.vulnCounterV2Query(ctx, query)
}

func (resolver *Resolver) vulnCounterV2Query(ctx context.Context, query *v1.Query) (*VulnerabilityCounterResolver, error) {
	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	fixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := vulnLoader.FromQuery(ctx, fixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	unFixableVulnsQuery := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableCVEs, err := vulnLoader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	return mapCVEsToVulnerabilityCounter(fixableVulns, unFixableCVEs), nil
}

func (resolver *Resolver) k8sVulnerabilityV2(ctx context.Context, args IDQuery) (VulnerabilityResolver, error) {
	return resolver.vulnerabilityV2(ctx, args)
}

func (resolver *Resolver) k8sVulnerabilitiesV2(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).Query())
	return resolver.vulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

func (resolver *Resolver) istioVulnerabilityV2(ctx context.Context, args IDQuery) (VulnerabilityResolver, error) {
	return resolver.vulnerabilityV2(ctx, args)
}

func (resolver *Resolver) istioVulnerabilitiesV2(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_ISTIO_CVE.String()).Query())
	return resolver.vulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

func (resolver *Resolver) openShiftVulnerabilityV2(ctx context.Context, args IDQuery) (VulnerabilityResolver, error) {
	return resolver.vulnerabilityV2(ctx, args)
}

func (resolver *Resolver) openShiftVulnerabilitiesV2(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).Query())
	return resolver.vulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

// Implemented Resolver.
////////////////////////

func (resolver *cVEResolver) ID(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *cVEResolver) CVE(ctx context.Context) string {
	if features.PostgresDatastore.Enabled() {
		return resolver.data.GetCve()
	}
	return resolver.data.GetId()
}

func (resolver *cVEResolver) getCVEQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
}

func (resolver *cVEResolver) getCVERawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).Query()
}

// IsFixable returns whether vulnerability is fixable by any component.
func (resolver *cVEResolver) IsFixable(_ context.Context, args RawQuery) (bool, error) {
	// CVE is used in scoping but it's not relevant to IsFixable because it is already scoped to a CVE
	q, err := args.AsV1QueryOrEmpty(search.ExcludeFieldLabel(search.CVE))
	if err != nil {
		return false, err
	}

	ctx, query := resolver.addScopeContext(q)
	if cve.ContainsComponentBasedCVE(resolver.data.GetTypes()) {
		query := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
		count, err := resolver.root.ComponentCVEEdgeDataStore.Count(ctx, query)
		if err != nil {
			return false, err
		}
		if count != 0 {
			return true, nil
		}
	}
	if cve.ContainsClusterCVE(resolver.data.GetTypes()) {
		query := search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.ClusterCVEFixable, true).ProtoQuery())
		count, err := resolver.root.clusterCVEEdgeDataStore.Count(ctx, query)
		if err != nil {
			return false, err
		}
		if count != 0 {
			return true, nil
		}
	}
	return false, nil
}

func (resolver *cVEResolver) withVulnerabilityScope(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		ID:    resolver.data.GetId(),
		Level: v1.SearchCategory_VULNERABILITIES,
	})
}

func (resolver *cVEResolver) getEnvImpactComponentsForImages(ctx context.Context) (numerator, denominator int, err error) {
	allDepsCount, err := resolver.root.DeploymentDataStore.CountDeployments(ctx)
	if err != nil {
		return 0, 0, err
	}
	if allDepsCount == 0 {
		return 0, 0, nil
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, 0, err
	}
	withThisCVECount, err := deploymentLoader.CountFromQuery(resolver.withVulnerabilityScope(ctx), search.EmptyQuery())
	if err != nil {
		return 0, 0, err
	}
	return int(withThisCVECount), allDepsCount, nil
}

func (resolver *cVEResolver) getEnvImpactComponentsForNodes(ctx context.Context) (numerator, denominator int, err error) {
	allNodesCount, err := resolver.root.NodeGlobalDataStore.CountAllNodes(ctx)
	if err != nil {
		return 0, 0, err
	}
	if allNodesCount == 0 {
		return 0, 0, nil
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, 0, err
	}
	withThisCVECount, err := nodeLoader.CountFromQuery(resolver.withVulnerabilityScope(ctx), search.EmptyQuery())
	if err != nil {
		return 0, 0, err
	}
	return int(withThisCVECount), allNodesCount, nil
}

// EnvImpact is the fraction of deployments that contains the CVE
func (resolver *cVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	var numerator, denominator int

	for _, vulnType := range resolver.data.GetTypes() {
		var n, d int
		var err error

		switch vulnType {
		case storage.CVE_K8S_CVE:
			n, d, err = resolver.getEnvImpactComponentsForPerClusterVuln(ctx, utils.K8s)
		case storage.CVE_ISTIO_CVE:
			n, d, err = resolver.getEnvImpactComponentsForPerClusterVuln(ctx, utils.Istio)
		case storage.CVE_OPENSHIFT_CVE:
			n, d, err = resolver.getEnvImpactComponentsForPerClusterVuln(ctx, utils.OpenShift)
		case storage.CVE_IMAGE_CVE:
			n, d, err = resolver.getEnvImpactComponentsForImages(ctx)
		case storage.CVE_NODE_CVE:
			n, d, err = resolver.getEnvImpactComponentsForNodes(ctx)
		default:
			return 0, errors.Errorf("unknown CVE type: %s", vulnType)
		}

		if err != nil {
			return 0, err
		}

		numerator += n
		denominator += d
	}

	if denominator == 0 {
		return 0, nil
	}

	return float64(numerator) / float64(denominator), nil
}

func (resolver *cVEResolver) getEnvImpactComponentsForPerClusterVuln(ctx context.Context, ct utils.CVEType) (int, int, error) {
	clusters, err := resolver.root.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return 0, 0, err
	}
	affectedClusters, err := resolver.root.orchestratorIstioCVEManager.GetAffectedClusters(ctx, resolver.data.GetId(), ct, resolver.root.cveMatcher)
	if err != nil {
		return 0, 0, err
	}
	return len(affectedClusters), len(clusters), nil
}

// LastScanned is the last time the vulnerability was scanned in an image.
func (resolver *cVEResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.ImageScanTime.String(),
				Reversed: true,
			},
		},
	}

	images, err := imageLoader.FromQuery(resolver.withVulnerabilityScope(ctx), q)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned vulnerability query")
	}

	return timestamp(images[0].GetScan().GetScanTime())
}

func (resolver *cVEResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	if val := resolver.data.GetCvssV3(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV3Resolver{resolver.ctx, resolver.root, val},
		}
	}
	if val := resolver.data.GetCvssV2(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV2Resolver{resolver.ctx, resolver.root, val},
		}
	}
	return nil
}

func (resolver *cVEResolver) VulnerabilityType() string {
	return resolver.data.GetType().String()
}

func (resolver *cVEResolver) VulnerabilityTypes() []string {
	vulnTypes := make([]string, 0, len(resolver.data.GetTypes()))
	for _, vulnType := range resolver.data.GetTypes() {
		vulnTypes = append(vulnTypes, vulnType.String())
	}
	return vulnTypes
}

// Components are the components that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "Components")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	cveQuery := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
	query, err = search.AddAsConjunction(cveQuery, query)
	if err != nil {
		return nil, err
	}

	return resolver.root.componentsV2Query(resolver.addScopeContext(query))
}

// ComponentCount is the number of components that contain the CVE/Vulnerability.
func (resolver *cVEResolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	cveQuery := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
	query, err = search.AddAsConjunction(cveQuery, query)
	if err != nil {
		return 0, err
	}

	componentLoader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return componentLoader.CountFromQuery(resolver.addScopeContext(query))
}

func (resolver *cVEResolver) ImageComponents(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "ImageComponents")
	return resolver.root.ImageComponents(resolver.withVulnerabilityScope(ctx), args)
}

func (resolver *cVEResolver) ImageComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "ImageComponentCount")
	return resolver.root.ImageComponentCount(resolver.withVulnerabilityScope(ctx), args)
}

// NodeComponents are the node components that contain the CVE/Vulnerability.
func (resolver *cVEResolver) NodeComponents(_ context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "NodeComponents")
	if !features.PostgresDatastore.Enabled() {
		query := search.AddRawQueriesAsConjunction(args.String(), resolver.getCVERawQuery())
		return resolver.root.NodeComponents(resolver.withVulnerabilityScope(resolver.ctx), PaginatedQuery{Query: &query, Pagination: args.Pagination})
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver NodeComponents in NodeVulnerability does not support postgres yet")
}

// NodeComponentCount is the number of node components that contain the CVE/Vulnerability.
func (resolver *cVEResolver) NodeComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "NodeComponentCount")
	if !features.PostgresDatastore.Enabled() {
		query := search.AddRawQueriesAsConjunction(args.String(), resolver.getCVERawQuery())
		return resolver.root.NodeComponentCount(resolver.withVulnerabilityScope(resolver.ctx), RawQuery{Query: &query})
	}
	// TODO : Add postgres support
	return 0, errors.New("Sub-resolver NodeComponentCount in NodeVulnerability does not support postgres yet")
}

// Images are the images that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	if err := readImages(ctx); err != nil {
		return []*imageResolver{}, nil
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapImages(imageLoader.FromQuery(resolver.addScopeContext(query)))
}

// ImageCount is the number of images that contain the CVE/Vulnerability.
func (resolver *cVEResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readImages(ctx); err != nil {
		return 0, nil
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return 0, err
	}
	return imageLoader.CountFromQuery(resolver.addScopeContext(query))
}

// Deployments are the deployments that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(deploymentLoader.FromQuery(resolver.addScopeContext(query)))
}

// DeploymentCount is the number of deployments that contain the CVE/Vulnerability.
func (resolver *cVEResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return deploymentLoader.CountFromQuery(resolver.addScopeContext(query))
}

// Nodes are the nodes that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return []*nodeResolver{}, nil
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapNodes(nodeLoader.FromQuery(resolver.addScopeContext(query)))
}

// NodeCount is the number of nodes that contain the CVE/Vulnerability.
func (resolver *cVEResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readNodes(ctx); err != nil {
		return 0, nil
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, err
	}

	return nodeLoader.CountFromQuery(resolver.addScopeContext(query))
}

// These return dummy values, as they should not be accessed from the top level vuln resolver, but the embedded
// version instead.

// FixedByVersion returns the version of the parent component that removes this CVE.
func (resolver *cVEResolver) FixedByVersion(ctx context.Context) (string, error) {
	return resolver.getCVEFixedByVersion(ctx)
}

// UnusedVarSink represents a query sink
func (resolver *cVEResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func (resolver *cVEResolver) getCVEFixedByVersion(ctx context.Context) (string, error) {
	if cve.ContainsComponentBasedCVE(resolver.data.GetTypes()) {
		return resolver.getComponentFixedByVersion(ctx)
	}
	return resolver.getClusterFixedByVersion(ctx)
}

func (resolver *cVEResolver) getComponentFixedByVersion(_ context.Context) (string, error) {
	scope, hasScope := scoped.GetScope(resolver.ctx)
	if !hasScope {
		return "", nil
	}
	if scope.Level != v1.SearchCategory_IMAGE_COMPONENTS {
		return "", nil
	}

	edgeID := edges.EdgeID{ParentID: scope.ID, ChildID: resolver.data.GetId()}.ToString()
	edge, found, err := resolver.root.ComponentCVEEdgeDataStore.Get(resolver.ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetFixedBy(), nil
}

func (resolver *cVEResolver) getClusterFixedByVersion(_ context.Context) (string, error) {
	scope, hasScope := scoped.GetScope(resolver.ctx)
	if !hasScope {
		return "", nil
	}
	if scope.Level != v1.SearchCategory_CLUSTERS {
		return "", nil
	}

	edgeID := edges.EdgeID{ParentID: scope.ID, ChildID: resolver.data.GetId()}.ToString()
	edge, found, err := resolver.root.clusterCVEEdgeDataStore.Get(resolver.ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetFixedBy(), nil
}

func (resolver *cVEResolver) DiscoveredAtImage(ctx context.Context, args RawQuery) (*graphql.Time, error) {
	if !cve.ContainsCVEType(resolver.data.GetTypes(), storage.CVE_IMAGE_CVE) {
		return nil, nil
	}

	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	} else {
		var err error
		imageID, err = getImageIDFromIfImageShaQuery(ctx, resolver.root, args)
		if err != nil {
			return nil, errors.Wrap(err, "could not determine vulnerability discovered time in image")
		}
	}

	if imageID == "" {
		return nil, nil
	}

	edgeID := edges.EdgeID{
		ParentID: imageID,
		ChildID:  resolver.data.GetId(),
	}.ToString()

	edge, found, err := resolver.root.ImageCVEEdgeDataStore.Get(resolver.ctx, edgeID)
	if err != nil || !found {
		return nil, err
	}
	return timestamp(edge.GetFirstImageOccurrence())
}

// ActiveState shows the activeness of a vulnerability in a deployment context.
func (resolver *cVEResolver) ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error) {
	scopeQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	deploymentID := getDeploymentScope(scopeQuery, ctx, resolver.ctx)
	if deploymentID == "" {
		return nil, nil
	}
	// We only support OS level component. The active state is not determined if there is no OS level component associate with this vuln.
	query := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).AddStrings(search.ComponentSource, storage.SourceType_OS.String()).ProtoQuery()
	osLevelComponents, err := resolver.root.ImageComponentDataStore.Count(ctx, query)
	if err != nil {
		return nil, err
	}
	if osLevelComponents == 0 {
		return &activeStateResolver{root: resolver.root, state: Undetermined}, nil
	}

	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentID)
	imageID := getImageIDFromQuery(scopeQuery)
	if imageID != "" {
		qb.AddExactMatches(search.ImageSHA, imageID)
	}
	query = search.ConjunctionQuery(resolver.getCVEQuery(), qb.ProtoQuery())

	results, err := resolver.root.ActiveComponent.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	state := Inactive
	if len(ids) != 0 {
		state = Active
	}
	return &activeStateResolver{root: resolver.root, state: state, activeComponentIDs: ids, imageScope: imageID}, nil
}

// VulnerabilityState return the effective state of this vulnerability (observed, deferred or marked as false positive).
func (resolver *cVEResolver) VulnerabilityState(ctx context.Context) string {
	if resolver.data.GetSuppressed() {
		return storage.VulnerabilityState_DEFERRED.String()
	}

	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	}

	if imageID == "" {
		return ""
	}

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		log.Error(errors.Wrap(err, "getting image loader"))
		return ""
	}
	img, err := imageLoader.FromID(ctx, imageID)
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching image with id %s", imageID))
		return ""
	}

	states, err := resolver.root.vulnReqQueryMgr.VulnsWithState(resolver.ctx,
		common.VulnReqScope{
			Registry: img.GetName().GetRegistry(),
			Remote:   img.GetName().GetRemote(),
			Tag:      img.GetName().GetTag(),
		})
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching vuln requests for image %s/%s:%s", img.GetName().GetRegistry(), img.GetName().GetRemote(), img.GetName().GetTag()))
		return ""
	}
	if s, ok := states[resolver.data.GetId()]; ok {
		return s.String()
	}

	return storage.VulnerabilityState_OBSERVED.String()
}

func (resolver *cVEResolver) addScopeContext(query *v1.Query) (context.Context, *v1.Query) {
	ctx := resolver.ctx
	scope, ok := scoped.GetScope(ctx)
	if !ok {
		return resolver.withVulnerabilityScope(ctx), query
	}
	// If the scope is not set to vulnerabilities then
	// we need to add a query to scope the search to the current vuln
	if scope.Level != v1.SearchCategory_VULNERABILITIES {
		return ctx, search.ConjunctionQuery(query, resolver.getCVEQuery())
	}
	return ctx, query
}

// EffectiveVulnerabilityRequest returns the effective vulnerability request i.e. the request that directly impacts
// this vulnerability in the given image scope.
func (resolver *cVEResolver) EffectiveVulnerabilityRequest(ctx context.Context) (*VulnerabilityRequestResolver, error) {
	var imageID string
	scope, hasScope := scoped.GetScopeAtLevel(resolver.ctx, v1.SearchCategory_IMAGES)
	if hasScope {
		imageID = scope.ID
	}

	if imageID == "" {
		return nil, errors.Errorf("image scope must be provided for determining effective vulnerability request for cve %s", resolver.data.GetId())
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting image loader")
	}
	img, err := imageLoader.FromID(ctx, imageID)
	if err != nil {
		log.Error(errors.Wrapf(err, "fetching image with id %s", imageID))
		return nil, nil
	}

	req, err := resolver.root.vulnReqQueryMgr.EffectiveVulnReq(ctx, resolver.data.GetId(),
		common.VulnReqScope{
			Registry: img.GetName().GetRegistry(),
			Remote:   img.GetName().GetRemote(),
			Tag:      img.GetName().GetTag(),
		})
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapVulnerabilityRequest(req, nil)
}
