package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	acConverter "github.com/stackrox/rox/central/activecomponent/converter"
	"github.com/stackrox/rox/central/graphql/resolvers/deploymentctx"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/image/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
)

// Resolvers on Embedded Scan Object.
/////////////////////////////////////

func (resolver *imageScanResolver) Components(_ context.Context, args PaginatedQuery) ([]*EmbeddedImageScanComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.ClearPagination()

	// This purely exists to make it easier when we do a second pass to remove the storage.Image proto later on, technically this code would work without this.
	if features.FlattenImageData.Enabled() {
		imageV2 := &storage.ImageV2{}
		imageV2.SetScan(resolver.data)
		vulns, err := mapImageV2sToComponentResolvers(resolver.root, []*storage.ImageV2{
			imageV2,
		}, query)

		return paginate(pagination, vulns, err)
	}
	image := &storage.Image{}
	image.SetScan(resolver.data)
	vulns, err := mapImagesToComponentResolvers(resolver.root, []*storage.Image{
		image,
	}, query)

	return paginate(pagination, vulns, err)
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
	os          string
	root        *Resolver
	lastScanned *time.Time
	data        *storage.EmbeddedImageScanComponent
}

// UnusedVarSink represents a query sink
func (eicr *EmbeddedImageScanComponentResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

// License return the license for the image component.
func (eicr *EmbeddedImageScanComponentResolver) License(_ context.Context) (*licenseResolver, error) {
	value := eicr.data.GetLicense()
	return eicr.root.wrapLicense(value, true, nil)
}

// ID returns a unique identifier for the component.
func (eicr *EmbeddedImageScanComponentResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(scancomponent.ComponentID(eicr.data.GetName(), eicr.data.GetVersion(), eicr.os))
}

// Name returns the name of the component.
func (eicr *EmbeddedImageScanComponentResolver) Name(_ context.Context) string {
	return eicr.data.GetName()
}

// Version gives the version of the image component.
func (eicr *EmbeddedImageScanComponentResolver) Version(_ context.Context) string {
	return eicr.data.GetVersion()
}

// Priority returns the priority of the component.
func (eicr *EmbeddedImageScanComponentResolver) Priority(_ context.Context) int32 {
	return int32(eicr.data.GetPriority())
}

// Source returns the source of the component.
func (eicr *EmbeddedImageScanComponentResolver) Source(_ context.Context) string {
	return eicr.data.GetSource().String()
}

// Location returns the location of the component.
func (eicr *EmbeddedImageScanComponentResolver) Location(_ context.Context, _ RawQuery) (string, error) {
	return eicr.data.GetLocation(), nil
}

// FixedIn returns the highest component version in which all the containing vulnerabilities are fixed.
func (eicr *EmbeddedImageScanComponentResolver) FixedIn(_ context.Context) string {
	return eicr.data.GetFixedBy()
}

// RiskScore returns the risk score of the component.
func (eicr *EmbeddedImageScanComponentResolver) RiskScore(_ context.Context) float64 {
	return float64(eicr.data.GetRiskScore())
}

// LayerIndex is the index in the parent image.
func (eicr *EmbeddedImageScanComponentResolver) LayerIndex() (*int32, error) {
	w, ok := eicr.data.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
	if !ok {
		return nil, nil
	}
	v := w.LayerIndex
	return &v, nil
}

// LastScanned is the last time the component was scanned in an image.
func (eicr *EmbeddedImageScanComponentResolver) LastScanned(_ context.Context) (*graphql.Time, error) {
	if eicr.lastScanned == nil {
		return nil, nil
	}
	return &graphql.Time{Time: *eicr.lastScanned}, nil
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *EmbeddedImageScanComponentResolver) TopVuln(_ context.Context) (*EmbeddedVulnerabilityResolver, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) Vulns(_ context.Context, args PaginatedQuery) ([]*EmbeddedVulnerabilityResolver, error) {
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

	vulns, err = paginate(query.GetPagination(), vulns, nil)
	if err != nil {
		return nil, err
	}
	return vulns, nil
}

// VulnCount resolves the number of vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCount(_ context.Context, _ RawQuery) (int32, error) {
	return int32(len(eicr.data.GetVulns())), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCounter(_ context.Context, _ RawQuery) (*VulnerabilityCounterResolver, error) {
	return mapVulnsToVulnerabilityCounter(eicr.data.GetVulns()), nil
}

// Images are the images that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) Images(ctx context.Context, args PaginatedQuery) ([]ImageResolver, error) {
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
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return 0, err
	}
	if features.FlattenImageData.Enabled() {
		imageLoader, err := loaders.GetImageV2Loader(ctx)
		if err != nil {
			return 0, err
		}
		return imageLoader.CountFromQuery(ctx, query)
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
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

// ActiveState shows the activeness of a component in a deployment context.
func (eicr *EmbeddedImageScanComponentResolver) ActiveState(ctx context.Context, _ PaginatedQuery) (*activeStateResolver, error) {
	if !features.ActiveVulnMgmt.Enabled() {
		return &activeStateResolver{}, nil
	}
	deploymentID := deploymentctx.FromContext(ctx)
	if deploymentID == "" {
		return nil, nil
	}
	if eicr.data.GetSource() != storage.SourceType_OS {
		return &activeStateResolver{root: eicr.root, state: Undetermined}, nil
	}

	acID := acConverter.ComposeID(deploymentID, scancomponent.ComponentID(eicr.data.GetName(), eicr.data.GetVersion(), eicr.os))
	found, err := eicr.root.ActiveComponent.Exists(ctx, acID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &activeStateResolver{root: eicr.root, state: Inactive}, nil
	}

	return &activeStateResolver{root: eicr.root, state: Active, activeComponentIDs: []string{acID}}, nil
}

// Nodes are the nodes that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return []*nodeResolver{}, nil
	}
	// Convert to query, but link the fields for the search.
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.ClearPagination()

	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}

	query.SetPagination(pagination)

	return eicr.root.wrapNodes(nodeLoader.FromQuery(ctx, query))
}

// NodeCount is the number of nodes that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
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

func (eicr *EmbeddedImageScanComponentResolver) loadImages(ctx context.Context, query *v1.Query) ([]ImageResolver, error) {

	pagination := query.GetPagination()
	query.ClearPagination()

	query, err := search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}

	query.SetPagination(pagination)

	if features.FlattenImageData.Enabled() {
		imageV2Loader, err := loaders.GetImageV2Loader(ctx)
		if err != nil {
			return nil, err
		}
		resolvers, err := eicr.root.wrapImageV2s(imageV2Loader.FromQuery(ctx, query))
		res := make([]ImageResolver, 0, len(resolvers))
		for i, resolver := range resolvers {
			res[i] = resolver
		}
		return res, err
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}
	resolvers, err := eicr.root.wrapImages(imageLoader.FromQuery(ctx, query))
	res := make([]ImageResolver, 0, len(resolvers))
	for i, resolver := range resolvers {
		res[i] = resolver
	}
	return res, err
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
	query.ClearPagination()

	query, err = search.AddAsConjunction(deploymentBaseQuery, query)
	if err != nil {
		return nil, err
	}

	query.SetPagination(pagination)

	return eicr.root.wrapListDeployments(ListDeploymentLoader.FromQuery(ctx, query))
}

func (eicr *EmbeddedImageScanComponentResolver) getDeploymentBaseQuery(ctx context.Context) (*v1.Query, error) {
	imageQuery := eicr.componentQuery()
	var results []search.Result
	var err error
	var searchField search.FieldLabel
	if features.FlattenImageData.Enabled() {
		results, err = eicr.root.ImageV2DataStore.Search(ctx, imageQuery)
		searchField = search.ImageID
	} else {
		results, err = eicr.root.ImageDataStore.Search(ctx, imageQuery)
		searchField = search.ImageSHA
	}
	if err != nil || len(results) == 0 {
		return nil, err
	}

	// Create a query that finds all of the deployments that contain at least one of the infected images.
	return search.NewQueryBuilder().AddExactMatches(searchField, search.ResultsToIDs(results)...).ProtoQuery(), nil
}

func (eicr *EmbeddedImageScanComponentResolver) componentQuery() *v1.Query {
	return search.NewQueryBuilder().
		AddExactMatches(search.Component, eicr.data.GetName()).
		AddExactMatches(search.ComponentVersion, eicr.data.GetVersion()).
		ProtoQuery()
}

// Static helpers.
//////////////////

type canGetScan interface {
	GetScan() *storage.ImageScan
}

// Map the image v1s that matched a query to the image components it contains.
func mapImagesToComponentResolvers(root *Resolver, images []*storage.Image, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	mappedImages := make([]canGetScan, 0, len(images))
	for _, image := range images {
		mappedImages = append(mappedImages, image)
	}
	return mapAnyImageToComponentResolvers(root, mappedImages, query)
}

// Map the image v1s that matched a query to the image components it contains.
func mapImageV2sToComponentResolvers(root *Resolver, images []*storage.ImageV2, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	mappedImages := make([]canGetScan, 0, len(images))
	for _, image := range images {
		mappedImages = append(mappedImages, image)
	}
	return mapAnyImageToComponentResolvers(root, mappedImages, query)
}

// Map the images that matched a query to the image components it contains.
func mapAnyImageToComponentResolvers(root *Resolver, images []canGetScan, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	componentPred, err := componentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	idToComponent := make(map[string]*EmbeddedImageScanComponentResolver)
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			if !componentPred.Matches(component) {
				continue
			}
			thisComponentID := scancomponent.ComponentID(component.GetName(), component.GetVersion(), image.GetScan().GetOperatingSystem())
			if _, exists := idToComponent[thisComponentID]; !exists {
				idToComponent[thisComponentID] = &EmbeddedImageScanComponentResolver{
					os:   image.GetScan().GetOperatingSystem(),
					root: root,
					data: component,
				}
			}
			latestTime := protocompat.ConvertTimeToTimestampOrNil(idToComponent[thisComponentID].lastScanned)
			if latestTime == nil || protocompat.CompareTimestamps(image.GetScan().GetScanTime(), latestTime) > 0 {
				imageScanTime := protocompat.ConvertTimestampToTimeOrNil(image.GetScan().GetScanTime())
				idToComponent[thisComponentID].lastScanned = imageScanTime
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
