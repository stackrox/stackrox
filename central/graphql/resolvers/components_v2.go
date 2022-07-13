package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
)

// Top Level Resolvers.
///////////////////////

func (resolver *Resolver) componentV2(ctx context.Context, args IDQuery) (ComponentResolver, error) {
	compRes, err := resolver.imageComponentDataStoreQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	compRes.ctx = ctx
	return compRes, nil
}

func (resolver *Resolver) componentsV2(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return resolver.componentsV2Query(ctx, query)
}

func (resolver *Resolver) imageComponentV2(ctx context.Context, args IDQuery) (ImageComponentResolver, error) {
	res, err := resolver.imageComponentDataStoreQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	res.ctx = ctx
	return res, nil
}

func (resolver *Resolver) imageComponentsV2(ctx context.Context, args PaginatedQuery) ([]ImageComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	resolvers, err := resolver.imageComponentsLoaderQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	ret := make([]ImageComponentResolver, 0, len(resolvers))
	for _, res := range resolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, err
}

func (resolver *Resolver) nodeComponentV2(ctx context.Context, args IDQuery) (NodeComponentResolver, error) {
	nodeCompRes, err := resolver.imageComponentDataStoreQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	nodeCompRes.ctx = ctx
	return nodeCompRes, nil
}

func (resolver *Resolver) nodeComponentsV2(ctx context.Context, args PaginatedQuery) ([]NodeComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	nodeCompResolvers, err := resolver.imageComponentsLoaderQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	ret := make([]NodeComponentResolver, 0, len(nodeCompResolvers))
	for _, res := range nodeCompResolvers {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, err
}

func (resolver *Resolver) componentsV2Query(ctx context.Context, query *v1.Query) ([]ComponentResolver, error) {
	compRes, err := resolver.imageComponentsLoaderQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	ret := make([]ComponentResolver, 0, len(compRes))
	for _, res := range compRes {
		res.ctx = ctx
		ret = append(ret, res)
	}
	return ret, err
}

func (resolver *Resolver) imageComponentDataStoreQuery(ctx context.Context, args IDQuery) (*imageComponentResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("attempted to invoke legacy datastores with postgres enabled")
	}
	component, exists, err := resolver.ImageComponentDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("component not found: %s", string(*args.ID))
	}
	return resolver.wrapImageComponent(component, true, nil)
}

func (resolver *Resolver) imageComponentsLoaderQuery(ctx context.Context, query *v1.Query) ([]*imageComponentResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("attempted to invoke legacy datastores with postgres enabled")
	}
	componentLoader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return nil, err
	}

	return resolver.wrapImageComponents(componentLoader.FromQuery(ctx, query))
}

func (resolver *Resolver) componentCountV2(ctx context.Context, args RawQuery) (int32, error) {
	if features.PostgresDatastore.Enabled() {
		return 0, errors.New("attempted to invoke legacy datastores with postgres enabled")
	}
	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	return resolver.componentCountV2Query(ctx, q)
}

func (resolver *Resolver) componentCountV2Query(ctx context.Context, query *v1.Query) (int32, error) {
	componentLoader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return 0, err
	}

	return componentLoader.CountFromQuery(ctx, query)
}

// Resolvers on Component Object.
/////////////////////////////////

// NodeComponentLastScanned is the last time the node component was scanned in a node.
func (eicr *imageComponentResolver) NodeComponentLastScanned(ctx context.Context) (*graphql.Time, error) {
	if err := readNodes(ctx); err != nil {
		return nil, nil
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}

	componentQuery := eicr.componentQuery()
	componentQuery.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.NodeScanTime.String(),
				Reversed: true,
			},
		},
	}

	nodes, err := nodeLoader.FromQuery(ctx, componentQuery)
	if err != nil || len(nodes) == 0 {
		return nil, err
	} else if len(nodes) > 1 {
		return nil, errors.New("multiple nodes matched for last scanned component query")
	}

	return timestamp(nodes[0].GetScan().GetScanTime())
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *imageComponentResolver) TopVuln(ctx context.Context) (VulnerabilityResolver, error) {
	vulnResolver, err := eicr.unwrappedTopVulnQuery(ctx)
	if err != nil || vulnResolver == nil {
		return nil, err
	}
	return vulnResolver, nil
}

// TopNodeVulnerability returns the first node component vulnerability with the top CVSS score.
func (eicr *imageComponentResolver) TopNodeVulnerability(ctx context.Context) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "TopNodeVulnerability")
	if !features.PostgresDatastore.Enabled() {
		vulnResolver, err := eicr.unwrappedTopVulnQuery(ctx)
		if err != nil || vulnResolver == nil {
			return nil, err
		}
		return vulnResolver, nil
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver TopNodeVulnerability in NodeComponent does not support postgres yet")
}

func (eicr *imageComponentResolver) unwrappedTopVulnQuery(ctx context.Context) (*cVEResolver, error) {
	if eicr.data.GetSetTopCvss() == nil {
		return nil, nil
	}

	query := eicr.componentQuery()
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
			{
				Field:    search.CVE.String(),
				Reversed: true,
			},
		},
		Limit:  1,
		Offset: 0,
	}

	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	vulns, err := vulnLoader.FromQuery(ctx, query)
	if err != nil || len(vulns) == 0 {
		return nil, err
	} else if len(vulns) > 1 {
		return nil, errors.New("multiple vulnerabilities matched for top component vulnerability")
	}

	return &cVEResolver{
		ctx:  eicr.ctx,
		root: eicr.root,
		data: vulns[0],
	}, nil
}

// Vulns resolves the vulnerabilities contained in the image component.
func (eicr *imageComponentResolver) Vulns(_ context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	scopeQuery, err := args.AsV1ScopeQueryOrEmpty()
	if err != nil {
		return nil, err
	}

	ctx, err := eicr.root.AddDistroContext(eicr.ctx, query, scopeQuery)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}
	query.Pagination = pagination
	return eicr.root.vulnerabilitiesV2Query(scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    eicr.data.GetId(),
	}), query)
}

// VulnCount resolves the number of vulnerabilities contained in the image component.
func (eicr *imageComponentResolver) VulnCount(_ context.Context, args RawQuery) (int32, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	scopeQuery, err := args.AsV1ScopeQueryOrEmpty()
	if err != nil {
		return 0, err
	}
	ctx, err := eicr.root.AddDistroContext(eicr.ctx, query, scopeQuery)
	if err != nil {
		return 0, err
	}

	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return 0, err
	}

	return eicr.root.vulnerabilityCountV2Query(scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    eicr.data.GetId(),
	}), query)
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (eicr *imageComponentResolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}

	fixableVulnsQuery := search.ConjunctionQuery(eicr.componentQuery(), search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())
	fixableVulns, err := vulnLoader.FromQuery(scoped.Context(eicr.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    eicr.data.GetId(),
	}), fixableVulnsQuery)
	if err != nil {
		return nil, err
	}

	unFixableVulnsQuery := search.ConjunctionQuery(eicr.componentQuery(), search.NewQueryBuilder().AddBools(search.Fixable, false).ProtoQuery())
	unFixableCVEs, err := vulnLoader.FromQuery(scoped.Context(eicr.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    eicr.data.GetId(),
	}), unFixableVulnsQuery)
	if err != nil {
		return nil, err
	}
	return mapCVEsToVulnerabilityCounter(cveToVulnerabilityWithSeverity(fixableVulns), cveToVulnerabilityWithSeverity(unFixableCVEs)), nil
}

// NodeVulnerabilities resolves the node vulnerabilities contained in the node component.
func (eicr *imageComponentResolver) NodeVulnerabilities(_ context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "NodeVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		return eicr.root.NodeVulnerabilities(eicr.imageComponentScopeContext(), args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver NodeVulnerabilities in NodeComponent does not support postgres yet")
}

// NodeVulnerabilityCount resolves the number of node vulnerabilities contained in the node component.
func (eicr *imageComponentResolver) NodeVulnerabilityCount(_ context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "NodeVulnerabilityCount")
	if !features.PostgresDatastore.Enabled() {
		return eicr.root.NodeVulnerabilityCount(eicr.imageComponentScopeContext(), args)
	}
	// TODO : Add postgres support
	return 0, errors.New("Sub-resolver NodeVulnerabilityCount in NodeComponent does not support postgres yet")
}

// NodeVulnerabilityCounter resolves the number of different types of node vulnerabilities contained in a node component.
func (eicr *imageComponentResolver) NodeVulnerabilityCounter(_ context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "NodeVulnerabilityCounter")
	if !features.PostgresDatastore.Enabled() {
		return eicr.root.NodeVulnerabilityCounter(eicr.imageComponentScopeContext(), args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver NodeVulnerabilityCounter in NodeComponent does not support postgres yet")
}

func (eicr *imageComponentResolver) imageComponentScopeContext() context.Context {
	return scoped.Context(eicr.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    eicr.data.GetId(),
	})
}

// Nodes are the nodes that contain the Component.
func (eicr *imageComponentResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return []*nodeResolver{}, nil
	}

	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	pagination := query.GetPagination()
	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}
	query.Pagination = pagination
	return eicr.root.wrapNodes(nodeLoader.FromQuery(ctx, query))
}

// NodeCount is the number of nodes that contain the Component.
func (eicr *imageComponentResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readNodes(ctx); err != nil {
		return 0, nil
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return 0, err
	}
	return nodeLoader.CountFromQuery(ctx, query)
}

// PlottedVulns returns the data required by top risky component scatter-plot on vuln mgmt dashboard
func (eicr *imageComponentResolver) PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("PlottedVulns resolver is not support on postgres. Use PlottedImageVulnerabilities.")
	}
	query := search.AddRawQueriesAsConjunction(args.String(), eicr.componentRawQuery())
	return newPlottedVulnerabilitiesResolver(ctx, eicr.root, RawQuery{Query: &query})
}

// PlottedNodeVulnerabilities returns the data required by top risky component scatter-plot on vuln mgmt dashboard
func (eicr *imageComponentResolver) PlottedNodeVulnerabilities(ctx context.Context, args RawQuery) (*PlottedNodeVulnerabilitiesResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "PlottedNodeVulnerabilities")
	if !features.PostgresDatastore.Enabled() {
		return newPlottedNodeVulnerabilitiesResolver(eicr.imageComponentScopeContext(), eicr.root, args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver PlottedNodeVulnerabilities in NodeComponent does not support postgres yet")
}
