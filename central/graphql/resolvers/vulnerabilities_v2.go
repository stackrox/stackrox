package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
)

// V2 Connections to root.
//////////////////////////

func (resolver *Resolver) vulnerabilityV2(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	vuln, exists, err := resolver.CVEDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("cve not found: %s", string(*args.ID))
	}
	return resolver.wrapCVE(vuln, true, nil)
}

func (resolver *Resolver) vulnerabilitiesV2(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.vulnerabilitiesV2Query(ctx, query)
}

func (resolver *Resolver) vulnerabilitiesV2Query(ctx context.Context, query *v1.Query) ([]VulnerabilityResolver, error) {
	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	query = tryUnsuppressedQuery(query)
	vulns, err := resolver.wrapCVEs(vulnLoader.FromQuery(ctx, query))

	ret := make([]VulnerabilityResolver, 0, len(vulns))
	for _, resolver := range vulns {
		ret = append(ret, resolver)
	}
	return ret, err
}

func (resolver *Resolver) vulnerabilityCountV2(ctx context.Context, args RawQuery) (int32, error) {
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
	query = tryUnsuppressedQuery(query)
	return vulnLoader.CountFromQuery(ctx, query)
}

func (resolver *Resolver) vulnCounterV2(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
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
	fixableVulnsQuery := search.NewConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := vulnLoader.FromQuery(ctx, fixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	unFixableVulnsQuery := search.NewConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableCVEs, err := vulnLoader.FromQuery(ctx, unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}
	return mapCVEsToVulnerabilityCounter(fixableVulns, unFixableCVEs), nil
}

func (resolver *Resolver) k8sVulnerabilityV2(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	return resolver.vulnerabilityV2(ctx, args)
}

func (resolver *Resolver) k8sVulnerabilitiesV2(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).Query())
	return resolver.vulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

func (resolver *Resolver) istioVulnerabilityV2(ctx context.Context, args idQuery) (VulnerabilityResolver, error) {
	return resolver.vulnerabilityV2(ctx, args)
}

func (resolver *Resolver) istioVulnerabilitiesV2(ctx context.Context, q PaginatedQuery) ([]VulnerabilityResolver, error) {
	query := search.AddRawQueriesAsConjunction(q.String(),
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_ISTIO_CVE.String()).Query())
	return resolver.vulnerabilitiesV2(ctx, PaginatedQuery{Query: &query, Pagination: q.Pagination})
}

// Implemented Resolver.
////////////////////////

func (resolver *cVEResolver) ID(ctx context.Context) graphql.ID {
	value := resolver.data.GetId()
	return graphql.ID(value)
}

func (resolver *cVEResolver) Cve(ctx context.Context) string {
	return resolver.data.GetId()
}

// IsFixable returns whether vulnerability is fixable by any component.
func (resolver *cVEResolver) IsFixable(ctx context.Context, args RawQuery) (bool, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return false, err
	}

	fixableCVEQuery := search.NewConjunctionQuery(q,
		search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery(),
		search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery(),
	)

	results, err := resolver.root.CVEDataStore.Search(ctx, fixableCVEQuery)
	if err != nil {
		return false, err
	}

	return len(results) != 0, nil
}

// EnvImpact is the fraction of deployments that contains the CVE
func (resolver *cVEResolver) EnvImpact(ctx context.Context) (float64, error) {
	if resolver.data.GetType() == storage.CVE_K8S_CVE {
		return resolver.getEnvImpactForK8sIstioVuln(ctx, converter.K8s)
	} else if resolver.data.GetType() == storage.CVE_ISTIO_CVE {
		return resolver.getEnvImpactForK8sIstioVuln(ctx, converter.Istio)
	}

	allDepsCount, err := resolver.root.DeploymentDataStore.CountDeployments(ctx)
	if err != nil {
		return 0, err
	}
	if allDepsCount == 0 {
		return 0, nil
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	withThisCVECount, err := deploymentLoader.CountFromQuery(ctx, resolver.cveQuery())
	if err != nil {
		return 0, err
	}
	if allDepsCount == 0 {
		return float64(0), nil
	}
	return float64(float64(withThisCVECount) / float64(allDepsCount)), nil
}

func (resolver *cVEResolver) getEnvImpactForK8sIstioVuln(ctx context.Context, ct converter.CVEType) (float64, error) {
	cve := resolver.root.k8sIstioCVEManager.GetNVDCVE(resolver.data.GetId())
	if cve == nil {
		return 0.0, fmt.Errorf("cve: %q not found", resolver.data.GetId())
	}
	p, err := resolver.root.getAffectedClusterPercentage(ctx, cve, ct)
	if err != nil {
		return 0.0, err
	}
	return p, nil
}

// LastScanned is the last time the vulnerability was scanned in an image.
func (resolver *cVEResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	cveQuery := resolver.cveQuery()
	cveQuery.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.ImageScanTime.String(),
				Reversed: true,
			},
		},
	}

	images, err := imageLoader.FromQuery(ctx, cveQuery)
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
			resolver: &cVSSV3Resolver{resolver.root, val},
		}
	}
	if val := resolver.data.GetCvssV2(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV2Resolver{resolver.root, val},
		}
	}
	return nil
}

func (resolver *cVEResolver) Severity(ctx context.Context) string {
	if resolver.data.GetScoreVersion() == storage.CVE_V3 {
		return resolver.data.GetCvssV3().GetSeverity().String()
	}
	return resolver.data.GetCvssV2().GetSeverity().String()
}

func (resolver *cVEResolver) VulnerabilityType() string {
	return resolver.data.GetType().String()
}

// Components are the components that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.CVEs, "Components")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	componentQuery := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()

	pagination := query.GetPagination()
	query.Pagination = nil
	query, err = search.AddAsConjunction(componentQuery, query)
	if err != nil {
		return nil, err
	}
	query.Pagination = pagination

	return resolver.root.componentsV2Query(ctx, query)
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
	return componentLoader.CountFromQuery(ctx, query)
}

// Images are the images that contain the CVE/Vulnerability.
func (resolver *cVEResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query, err = search.AddAsConjunction(resolver.cveQuery(), query)
	if err != nil {
		return nil, err
	}
	query.Pagination = pagination

	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapImages(imageLoader.FromQuery(ctx, query))
}

// ImageCount is the number of images that contain the CVE/Vulnerability.
func (resolver *cVEResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	cveQuery := search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
	query, err = search.AddAsConjunction(cveQuery, query)
	if err != nil {
		return 0, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return 0, err
	}
	return imageLoader.CountFromQuery(ctx, query)
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

	pagination := query.GetPagination()
	query, err = search.AddAsConjunction(resolver.cveQuery(), query)
	if err != nil {
		return nil, err
	}
	query.Pagination = pagination

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.root.wrapDeployments(deploymentLoader.FromQuery(ctx, query))
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
	query, err = search.AddAsConjunction(resolver.cveQuery(), query)
	if err != nil {
		return 0, err
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return deploymentLoader.CountFromQuery(ctx, query)
}

func (resolver *cVEResolver) cveQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, resolver.data.GetId()).ProtoQuery()
}

// These return dummy values, as they should not be accessed from the top level vuln resolver, but the embedded
// version instead.

// FixedByVersion returns the version of the parent component that removes this CVE.
func (resolver *cVEResolver) FixedByVersion(ctx context.Context, args RawQuery) (string, error) {
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return "", err
	}

	return resolver.getCVEFixedByVersion(ctx, q)
}

// UnusedVarSink represents a query sink
func (resolver *cVEResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func (resolver *cVEResolver) getCVEFixedByVersion(ctx context.Context, q *v1.Query) (string, error) {
	if resolver.data.GetType() == storage.CVE_IMAGE_CVE {
		return resolver.getComponentFixedByVersion(ctx, q)
	}
	return resolver.getClusterFixedByVersion(ctx, q)
}

func (resolver *cVEResolver) getComponentFixedByVersion(ctx context.Context, q *v1.Query) (string, error) {
	q, containsUnmatchableFields := search.FilterQueryWithMap(q, componentMappings.OptionsMap)
	if q == nil || containsUnmatchableFields {
		return "", nil
	}

	results, err := resolver.root.ImageComponentDataStore.Search(ctx, q)
	if err != nil || len(results) == 0 {
		return "", err
	} else if len(results) > 1 {
		return "", errors.New("multiple components matched for vulnerability fixedByVersion query")
	}

	edgeID := edges.EdgeID{ParentID: results[0].ID, ChildID: resolver.data.GetId()}.ToString()
	edge, found, err := resolver.root.ComponentCVEEdgeDataStore.Get(ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetFixedBy(), nil
}

func (resolver *cVEResolver) getClusterFixedByVersion(ctx context.Context, q *v1.Query) (string, error) {
	q, containsUnmatchableFields := search.FilterQueryWithMap(q, clusterMappings.OptionsMap)
	if q == nil || containsUnmatchableFields {
		return "", nil
	}

	results, err := resolver.root.ClusterDataStore.Search(ctx, q)
	if err != nil || len(results) == 0 {
		return "", err
	} else if len(results) > 1 {
		return "", errors.New("multiple clusters matched for vulnerability fixedByVersion query")
	}

	edgeID := edges.EdgeID{ParentID: results[0].ID, ChildID: resolver.data.GetId()}.ToString()
	edge, found, err := resolver.root.clusterCVEEdgeDataStore.Get(ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetFixedBy(), nil
}
