package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	acConverter "github.com/stackrox/stackrox/central/activecomponent/converter"
	"github.com/stackrox/stackrox/central/graphql/resolvers/deploymentctx"
	"github.com/stackrox/stackrox/central/graphql/resolvers/loaders"
	"github.com/stackrox/stackrox/central/metrics"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/features"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/scoped"
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
	component, exists, err := resolver.ImageComponentDataStore.Get(ctx, string(*args.ID))
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.Errorf("component not found: %s", string(*args.ID))
	}
	return resolver.wrapImageComponent(component, true, nil)
}

func (resolver *Resolver) imageComponentsLoaderQuery(ctx context.Context, query *v1.Query) ([]*imageComponentResolver, error) {
	componentLoader, err := loaders.GetComponentLoader(ctx)
	if err != nil {
		return nil, err
	}

	return resolver.wrapImageComponents(componentLoader.FromQuery(ctx, query))
}

func (resolver *Resolver) componentCountV2(ctx context.Context, args RawQuery) (int32, error) {
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

// ID returns a unique identifier for the component. Need to implement this on top of 'Id' so that we can implement
// the same interface as the non-generated embedded resolver used in v1.
func (eicr *imageComponentResolver) ID(ctx context.Context) graphql.ID {
	return graphql.ID(eicr.data.GetId())
}

// LastScanned is the last time the component was scanned in an image.
func (eicr *imageComponentResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	componentQuery := eicr.componentQuery()
	componentQuery.Pagination = &v1.QueryPagination{
		Limit:  1,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.ImageScanTime.String(),
				Reversed: true,
			},
		},
	}

	images, err := imageLoader.FromQuery(ctx, componentQuery)
	if err != nil || len(images) == 0 {
		return nil, err
	} else if len(images) > 1 {
		return nil, errors.New("multiple images matched for last scanned component query")
	}

	return timestamp(images[0].GetScan().GetScanTime())
}

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

func (eicr *imageComponentResolver) TopImageVulnerability(ctx context.Context) (ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "TopImageVulnerability")
	if !features.PostgresDatastore.Enabled() {
		vulnResolver, err := eicr.unwrappedTopVulnQuery(ctx)
		if err != nil || vulnResolver == nil {
			return nil, err
		}
		return vulnResolver, nil
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver TopImageVulnerability in ImageComponent does not support postgres yet")
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
	return mapCVEsToVulnerabilityCounter(fixableVulns, unFixableCVEs), nil
}

func (eicr *imageComponentResolver) ImageVulnerabilities(_ context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilities")
	return eicr.root.ImageVulnerabilities(eicr.imageComponentScopeContext(), args)
}

func (eicr *imageComponentResolver) ImageVulnerabilityCount(_ context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCount")
	return eicr.root.ImageVulnerabilityCount(eicr.imageComponentScopeContext(), args)
}

func (eicr *imageComponentResolver) ImageVulnerabilityCounter(_ context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ImageComponents, "ImageVulnerabilityCounter")
	return eicr.root.ImageVulnerabilityCounter(eicr.imageComponentScopeContext(), args)
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
		return eicr.root.NodeVulnCounter(eicr.imageComponentScopeContext(), args)
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

// Images are the images that contain the Component.
func (eicr *imageComponentResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
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
	return eicr.root.wrapImages(imageLoader.FromQuery(ctx, query))
}

// ImageCount is the number of images that contain the Component.
func (eicr *imageComponentResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
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
	return imageLoader.CountFromQuery(ctx, query)
}

// Deployments are the deployments that contain the Component.
func (eicr *imageComponentResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
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

	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	return eicr.root.wrapDeployments(deploymentLoader.FromQuery(ctx, query))
}

// DeploymentCount is the number of deployments that contain the Component.
func (eicr *imageComponentResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readDeployments(ctx); err != nil {
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
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return deploymentLoader.CountFromQuery(ctx, query)
}

// ActiveState shows the activeness of a component in a deployment context.
func (eicr *imageComponentResolver) ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error) {
	if !features.ActiveVulnManagement.Enabled() {
		return nil, nil
	}
	scopeQuery, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	deploymentID := getDeploymentScope(scopeQuery, eicr.ctx)
	if deploymentID == "" {
		return nil, nil
	}

	if eicr.data.GetSource() != storage.SourceType_OS {
		return &activeStateResolver{root: eicr.root, state: Undetermined}, nil
	}
	acID := acConverter.ComposeID(deploymentID, eicr.data.GetId())

	var found bool
	imageID := getImageIDFromQuery(scopeQuery)
	if imageID == "" {
		found, err = eicr.root.ActiveComponent.Exists(ctx, acID)
		if err != nil {
			return nil, err
		}
	} else {
		query := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).ProtoQuery()
		results, err := eicr.root.ActiveComponent.Search(ctx, query)
		if err != nil {
			return nil, err
		}
		found = search.ResultsToIDSet(results).Contains(acID)
	}

	if !found {
		return &activeStateResolver{root: eicr.root, state: Inactive}, nil
	}

	return &activeStateResolver{root: eicr.root, state: Active, activeComponentIDs: []string{acID}, imageScope: imageID}, nil
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

// Helper functions.
////////////////////

func (eicr *imageComponentResolver) componentQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ComponentID, eicr.data.GetId()).ProtoQuery()
}

func (eicr *imageComponentResolver) componentRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.ComponentID, eicr.data.GetId()).Query()
}

// These return dummy values, as they should not be accessed from the top level component resolver, but the embedded
// version instead.

// Location returns the location of the component.
func (eicr *imageComponentResolver) Location(ctx context.Context, args RawQuery) (string, error) {
	var imageID string
	scope, hasScope := scoped.GetScope(eicr.ctx)
	if hasScope && scope.Level == v1.SearchCategory_IMAGES {
		imageID = scope.ID
	} else if !hasScope || scope.Level != v1.SearchCategory_IMAGES {
		var err error
		imageID, err = getImageIDFromIfImageShaQuery(ctx, eicr.root, args)
		if err != nil {
			return "", errors.Wrap(err, "could not determine component location")
		}
	}

	if imageID == "" {
		return "", nil
	}

	edgeID := edges.EdgeID{ParentID: imageID, ChildID: eicr.data.GetId()}.ToString()
	edge, found, err := eicr.root.ImageComponentEdgeDataStore.Get(ctx, edgeID)
	if err != nil || !found {
		return "", err
	}
	return edge.GetLocation(), nil
}

func (eicr *imageComponentResolver) FixedIn(ctx context.Context) string {
	return eicr.data.GetFixedBy()
}

// LayerIndex is the index in the parent image.
func (eicr *imageComponentResolver) LayerIndex() *int32 {
	return nil
}

// PlottedVulns returns the data required by top risky component scatter-plot on vuln mgmt dashboard
func (eicr *imageComponentResolver) PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
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

// UnusedVarSink represents a query sink
func (eicr *imageComponentResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func getDeploymentIDFromQuery(q *v1.Query) string {
	if q == nil {
		return ""
	}
	var deploymentID string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if strings.EqualFold(matchFieldQuery.MatchFieldQuery.GetField(), search.DeploymentID.String()) {
			deploymentID = matchFieldQuery.MatchFieldQuery.Value
			deploymentID = strings.TrimRight(deploymentID, `"`)
			deploymentID = strings.TrimLeft(deploymentID, `"`)
		}
	})
	return deploymentID
}

func getDeploymentScope(scopeQuery *v1.Query, contexts ...context.Context) string {
	var deploymentID string
	for _, ctx := range contexts {
		deploymentID = deploymentctx.FromContext(ctx)
		if deploymentID != "" {
			return deploymentID
		}
	}
	if scopeQuery != nil {
		deploymentID = getDeploymentIDFromQuery(scopeQuery)
	}
	return deploymentID
}
