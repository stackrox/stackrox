package resolvers

import (
	"context"
	"fmt"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/image/mappings"
	"github.com/stackrox/rox/central/imagecomponent"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Component returns an image scan component based on an input id (name:version)
func (resolver *Resolver) componentV1(ctx context.Context, args idQuery) (*EmbeddedImageScanComponentResolver, error) {
	cID, err := imagecomponent.FromString(string(*args.ID))
	if err != nil {
		return nil, err
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.Component, cID.Name).
		AddExactMatches(search.ComponentVersion, cID.Version).
		ProtoQuery()
	comps, err := components(ctx, resolver, query)
	if err != nil {
		return nil, err
	} else if len(comps) == 0 {
		return nil, nil
	} else if len(comps) > 1 {
		return nil, fmt.Errorf("multiple components matched: %s this should not happen", string(*args.ID))
	}
	return comps[0], nil
}

// Components returns the image scan components that match the input query.
func (resolver *Resolver) componentsV1(ctx context.Context, q PaginatedQuery) ([]ComponentResolver, error) {
	// Convert to query, but link the fields for the search.
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	resolvers, err := paginationWrapper{
		pv: query.Pagination,
	}.paginate(components(ctx, resolver, query))
	compRes := resolvers.([]*EmbeddedImageScanComponentResolver)

	ret := make([]ComponentResolver, 0, len(compRes))
	for _, resolver := range compRes {
		ret = append(ret, resolver)
	}
	return ret, err
}

// ComponentCount returns count of all clusters across infrastructure
func (resolver *Resolver) componentCountV1(ctx context.Context, args RawQuery) (int32, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	comps, err := components(ctx, resolver, query)
	if err != nil {
		return 0, err
	}
	return int32(len(comps)), nil
}

// Helper function that actually runs the queries and produces the resolvers from the images.
func components(ctx context.Context, root *Resolver, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	// Run search on images.
	images, err := imageLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return mapImagesToComponentResolvers(root, images, query)
}

// Resolvers on Embedded Scan Object.
/////////////////////////////////////

func (resolver *imageScanResolver) Components(ctx context.Context, args PaginatedQuery) ([]*EmbeddedImageScanComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	vulns, err := mapImagesToComponentResolvers(resolver.root, []*storage.Image{
		{
			Scan: resolver.data,
		},
	}, query)

	resolvers, err := paginationWrapper{
		pv: pagination,
	}.paginate(vulns, err)
	return resolvers.([]*EmbeddedImageScanComponentResolver), err
}

func (resolver *imageScanResolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	resolvers, err := resolver.Components(ctx, PaginatedQuery{Query: args.Query})
	if err != nil {
		return 0, err
	}
	return int32(len(resolvers)), nil
}

// EmbeddedImageScanComponentResolver resolves data about an image scan component.
type EmbeddedImageScanComponentResolver struct {
	root        *Resolver
	lastScanned *protoTypes.Timestamp
	data        *storage.EmbeddedImageScanComponent
}

// UnusedVarSink represents a query sink
func (eicr *EmbeddedImageScanComponentResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

// License return the license for the image component.
func (eicr *EmbeddedImageScanComponentResolver) License(ctx context.Context) (*licenseResolver, error) {
	value := eicr.data.GetLicense()
	return eicr.root.wrapLicense(value, true, nil)
}

// ID returns a unique identifier for the component.
func (eicr *EmbeddedImageScanComponentResolver) ID(ctx context.Context) graphql.ID {
	cID := &imagecomponent.ComponentID{
		Name:    eicr.data.GetName(),
		Version: eicr.data.GetVersion(),
	}
	return graphql.ID(cID.ToString())
}

// Name returns the name of the component.
func (eicr *EmbeddedImageScanComponentResolver) Name(ctx context.Context) string {
	return eicr.data.GetName()
}

// Version gives the version of the image component.
func (eicr *EmbeddedImageScanComponentResolver) Version(ctx context.Context) string {
	return eicr.data.GetVersion()
}

// Priority returns the priority of the component.
func (eicr *EmbeddedImageScanComponentResolver) Priority(ctx context.Context) int32 {
	return int32(eicr.data.GetPriority())
}

// Source returns the source of the component.
func (eicr *EmbeddedImageScanComponentResolver) Source(ctx context.Context) string {
	return eicr.data.GetSource().String()
}

// Location returns the location of the component.
func (eicr *EmbeddedImageScanComponentResolver) Location(ctx context.Context, _ RawQuery) (string, error) {
	return eicr.data.GetLocation(), nil
}

// RiskScore returns the risk score of the component.
func (eicr *EmbeddedImageScanComponentResolver) RiskScore(ctx context.Context) float64 {
	return float64(0.0)
}

// LayerIndex is the index in the parent image.
func (eicr *EmbeddedImageScanComponentResolver) LayerIndex() *int32 {
	w, ok := eicr.data.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
	if !ok {
		return nil
	}
	v := w.LayerIndex
	return &v
}

// LastScanned is the last time the component was scanned in an image.
func (eicr *EmbeddedImageScanComponentResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	return timestamp(eicr.lastScanned)
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *EmbeddedImageScanComponentResolver) TopVuln(ctx context.Context) (VulnerabilityResolver, error) {
	var maxCvss *storage.EmbeddedVulnerability
	for _, vuln := range eicr.data.GetVulns() {
		if maxCvss == nil || vuln.GetCvss() > maxCvss.GetCvss() {
			maxCvss = vuln
		}
	}
	if maxCvss == nil {
		return nil, nil
	}
	return eicr.root.wrapEmbeddedVulnerability(maxCvss, nil)
}

// Vulns resolves the vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	vulnQuery, _ := search.FilterQueryWithMap(query, mappings.VulnerabilityOptionsMap)
	vulnPred, err := vulnPredicateFactory.GeneratePredicate(vulnQuery)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	vulns := make([]*EmbeddedVulnerabilityResolver, 0, len(eicr.data.GetVulns()))
	for _, vuln := range eicr.data.GetVulns() {
		if !vulnPred.Matches(vuln) {
			continue
		}
		vulns = append(vulns, &EmbeddedVulnerabilityResolver{
			data:        vuln,
			root:        eicr.root,
			lastScanned: eicr.lastScanned,
		})
	}

	resolvers, err := paginationWrapper{
		pv: query.GetPagination(),
	}.paginate(vulns, nil)
	if err != nil {
		return nil, err
	}
	paginatedVulns := resolvers.([]*EmbeddedVulnerabilityResolver)

	ret := make([]VulnerabilityResolver, 0, len(paginatedVulns))
	for _, resolver := range paginatedVulns {
		ret = append(ret, resolver)
	}
	return ret, err
}

// VulnCount resolves the number of vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCount(ctx context.Context, args RawQuery) (int32, error) {
	return int32(len(eicr.data.GetVulns())), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	return mapVulnsToVulnerabilityCounter(eicr.data.GetVulns()), nil
}

// Images are the images that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	// Convert to query, but link the fields for the search.
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	images, err := eicr.loadImages(ctx, query)
	if err != nil {
		return nil, err
	}
	return images, nil
}

// ImageCount is the number of images that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return eicr.loadDeployments(ctx, query)
}

// DeploymentCount is the number of deployments that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	deploymentBaseQuery, err := eicr.getDeploymentBaseQuery(ctx)
	if err != nil || deploymentBaseQuery == nil {
		return 0, err
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return deploymentLoader.CountFromQuery(ctx, search.ConjunctionQuery(deploymentBaseQuery, query))
}

func (eicr *EmbeddedImageScanComponentResolver) loadImages(ctx context.Context, query *v1.Query) ([]*imageResolver, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}

	query.Pagination = pagination

	return eicr.root.wrapImages(imageLoader.FromQuery(ctx, query))
}

func (eicr *EmbeddedImageScanComponentResolver) loadDeployments(ctx context.Context, query *v1.Query) ([]*deploymentResolver, error) {
	deploymentBaseQuery, err := eicr.getDeploymentBaseQuery(ctx)
	if err != nil || deploymentBaseQuery == nil {
		return nil, err
	}

	ListDeploymentLoader, err := loaders.GetListDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	query, err = search.AddAsConjunction(deploymentBaseQuery, query)
	if err != nil {
		return nil, err
	}

	query.Pagination = pagination

	return eicr.root.wrapListDeployments(ListDeploymentLoader.FromQuery(ctx, query))
}

func (eicr *EmbeddedImageScanComponentResolver) getDeploymentBaseQuery(ctx context.Context) (*v1.Query, error) {
	imageQuery := eicr.componentQuery()
	results, err := eicr.root.ImageDataStore.Search(ctx, imageQuery)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	// Create a query that finds all of the deployments that contain at least one of the infected images.
	var qb []*v1.Query
	for _, id := range search.ResultsToIDs(results) {
		qb = append(qb, search.NewQueryBuilder().AddExactMatches(search.ImageSHA, id).ProtoQuery())
	}
	return search.DisjunctionQuery(qb...), nil
}

func (eicr *EmbeddedImageScanComponentResolver) componentQuery() *v1.Query {
	return search.NewQueryBuilder().
		AddExactMatches(search.Component, eicr.data.GetName()).
		AddExactMatches(search.ComponentVersion, eicr.data.GetVersion()).
		ProtoQuery()
}

// Static helpers.
//////////////////

// Map the images that matched a query to the image components it contains.
func mapImagesToComponentResolvers(root *Resolver, images []*storage.Image, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	componentPred, err := componentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	idToComponent := make(map[imagecomponent.ComponentID]*EmbeddedImageScanComponentResolver)
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			if !componentPred.Matches(component) {
				continue
			}
			thisComponentID := imagecomponent.ComponentID{Name: component.GetName(), Version: component.GetVersion()}
			if _, exists := idToComponent[thisComponentID]; !exists {
				idToComponent[thisComponentID] = &EmbeddedImageScanComponentResolver{
					root: root,
					data: component,
				}
			}
			latestTime := idToComponent[thisComponentID].lastScanned
			if latestTime == nil || image.GetScan().GetScanTime().Compare(latestTime) > 0 {
				idToComponent[thisComponentID].lastScanned = image.GetScan().GetScanTime()
			}
		}
	}

	// Create the resolvers.
	resolvers := make([]*EmbeddedImageScanComponentResolver, 0, len(idToComponent))
	for _, component := range idToComponent {
		resolvers = append(resolvers, component)
	}
	return resolvers, nil
}
