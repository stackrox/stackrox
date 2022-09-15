package resolvers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
)

// Top Level Resolvers.
///////////////////////

func (resolver *Resolver) componentV2(ctx context.Context, args IDQuery) (ComponentResolver, error) {
	component, err := resolver.imageComponentDataStoreQuery(ctx, args)
	compRes, err := resolver.wrapImageComponent(component, true, err)
	if err != nil || compRes == nil {
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
	component, err := resolver.imageComponentDataStoreQuery(ctx, args)
	res, err := resolver.wrapImageComponent(component, true, err)
	if err != nil || res == nil {
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

	resolvers, err := resolver.wrapImageComponents(resolver.imageComponentsLoaderQuery(ctx, query))
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
	component, err := resolver.imageComponentDataStoreQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	nodeComponent, err := imageComponentToNodeComponent(component)
	nodeCompRes, err := resolver.wrapNodeComponent(nodeComponent, true, err)
	if err != nil || nodeCompRes == nil {
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

	components, err := resolver.imageComponentsLoaderQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	nodeComponets, err := imageComponentsToNodeComponents(components)
	nodeCompResolvers, err := resolver.wrapNodeComponents(nodeComponets, err)
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
	compRes, err := resolver.wrapImageComponents(resolver.imageComponentsLoaderQuery(ctx, query))
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

func (resolver *Resolver) imageComponentDataStoreQuery(ctx context.Context, args IDQuery) (*storage.ImageComponent, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("attempted to invoke legacy datastores with postgres enabled")
	}
	component, exists, err := resolver.ImageComponentDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("component not found: %s", string(*args.ID))
	}
	return component, err
}

func (resolver *Resolver) imageComponentsLoaderQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error) {
	if features.PostgresDatastore.Enabled() {
		return nil, errors.New("attempted to invoke legacy datastores with postgres enabled")
	}
	componentLoader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return nil, err
	}

	return componentLoader.FromQuery(ctx, query)
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

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *imageComponentResolver) TopVuln(ctx context.Context) (VulnerabilityResolver, error) {
	vulnResolver, err := eicr.unwrappedTopVulnQuery(ctx)
	if err != nil || vulnResolver == nil {
		return nil, err
	}
	return vulnResolver, nil
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
