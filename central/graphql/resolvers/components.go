package resolvers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/image/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	componentPredicateFactory = predicate.NewFactory(&storage.EmbeddedImageScanComponent{})
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedImageScanComponent", []string{
			"license: License",
			"id: ID!",
			"name: String!",
			"version: String!",
			"topVuln: EmbeddedVulnerability",
			"vulns(query: String): [EmbeddedVulnerability]!",
			"vulnCount(query: String): Int!",
			"vulnCounter: VulnerabilityCounter!",
			"lastScanned: Time",
			"images(query: String): [Image!]!",
			"imageCount(query: String): Int!",
			"deployments(query: String): [Deployment!]!",
			"deploymentCount(query: String): Int!",
			"priority: Int!",
		}),
		schema.AddExtraResolver("ImageScan", `components(query: String): [EmbeddedImageScanComponent!]!`),
		schema.AddExtraResolver("ImageScan", `componentCount(query: String): Int!`),
		schema.AddQuery("component(id: ID): EmbeddedImageScanComponent"),
		schema.AddQuery("components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!"),
	)
}

// Component returns an image scan component based on an input id (name:version)
func (resolver *Resolver) Component(ctx context.Context, args struct{ *graphql.ID }) (*EmbeddedImageScanComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponent")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	cID, err := componentIDFromString(string(*args.ID))
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
func (resolver *Resolver) Components(ctx context.Context, q paginatedQuery) ([]*EmbeddedImageScanComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// Convert to query, but link the fields for the search.
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	resolvers, err := paginationWrapper{
		pv: query.Pagination,
	}.paginate(components(ctx, resolver, query))
	return resolvers.([]*EmbeddedImageScanComponentResolver), err
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

func (resolver *imageScanResolver) Components(ctx context.Context, args rawQuery) ([]*EmbeddedImageScanComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return mapImagesToComponentResolvers(resolver.root, []*storage.Image{
		{
			Scan: resolver.data,
		},
	}, query)
}

func (resolver *imageScanResolver) ComponentCount(ctx context.Context, args rawQuery) (int32, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	resolvers, err := mapImagesToComponentResolvers(resolver.root, []*storage.Image{
		{
			Scan: resolver.data,
		},
	}, query)
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

// License return the license for the image component.
func (eicr *EmbeddedImageScanComponentResolver) License(ctx context.Context) (*licenseResolver, error) {
	value := eicr.data.GetLicense()
	return eicr.root.wrapLicense(value, true, nil)
}

// ID returns a unique identifier for the component.
func (eicr *EmbeddedImageScanComponentResolver) ID(ctx context.Context) graphql.ID {
	cID := &componentID{
		Name:    eicr.data.GetName(),
		Version: eicr.data.GetVersion(),
	}
	return graphql.ID(cID.toString())
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

// LayerIndex is the index in the parent image.
// TODO: make this only accessable when coming from an image resolver.
func (eicr *EmbeddedImageScanComponentResolver) LayerIndex() *int32 {
	w, ok := eicr.data.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
	if !ok {
		return nil
	}
	v := w.LayerIndex
	return &v
}

// LastScanned is the last time the vulnerability was scanned in an image.
func (eicr *EmbeddedImageScanComponentResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	return timestamp(eicr.lastScanned)
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *EmbeddedImageScanComponentResolver) TopVuln(ctx context.Context) (*EmbeddedVulnerabilityResolver, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) Vulns(ctx context.Context, args rawQuery) ([]*EmbeddedVulnerabilityResolver, error) {
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
		if !vulnPred(vuln) {
			continue
		}
		vulns = append(vulns, &EmbeddedVulnerabilityResolver{
			data:        vuln,
			root:        eicr.root,
			lastScanned: eicr.lastScanned,
		})
	}
	return vulns, nil
}

// VulnCount resolves the number of vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCount(ctx context.Context, args rawQuery) (int32, error) {
	vulns, err := eicr.Vulns(ctx, args)
	if err != nil {
		return 0, err
	}
	return int32(len(vulns)), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return mapVulnsToVulnerabilityCounter(eicr.data.GetVulns()), nil
}

// Images are the images that contain the Component.
func (eicr *EmbeddedImageScanComponentResolver) Images(ctx context.Context, args rawQuery) ([]*imageResolver, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) ImageCount(ctx context.Context, args rawQuery) (int32, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) Deployments(ctx context.Context, args rawQuery) ([]*deploymentResolver, error) {
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
func (eicr *EmbeddedImageScanComponentResolver) DeploymentCount(ctx context.Context, args rawQuery) (int32, error) {
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

	query, err = search.AddAsConjunction(eicr.componentQuery(), query)
	if err != nil {
		return nil, err
	}
	return eicr.root.wrapImages(imageLoader.FromQuery(ctx, query))
}

func (eicr *EmbeddedImageScanComponentResolver) loadDeployments(ctx context.Context, query *v1.Query) ([]*deploymentResolver, error) {
	q, err := eicr.getDeploymentBaseQuery(ctx)
	if err != nil || q == nil {
		return nil, err
	}

	ListDeploymentLoader, err := loaders.GetListDeploymentLoader(ctx)
	if err != nil {
		return nil, err
	}
	return eicr.root.wrapListDeployments(ListDeploymentLoader.FromQuery(ctx, search.ConjunctionQuery(q, query)))
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

// Synthetic ID for component objects composed of the name and version of the component.
type componentID struct {
	Name    string
	Version string
}

func componentIDFromString(str string) (*componentID, error) {
	nameAndVersionEncoded := strings.Split(str, ":")
	if len(nameAndVersionEncoded) != 2 {
		return nil, fmt.Errorf("invalid id: %s", str)
	}
	name, err := base64.URLEncoding.DecodeString(nameAndVersionEncoded[0])
	if err != nil {
		return nil, err
	}
	version, err := base64.URLEncoding.DecodeString(nameAndVersionEncoded[1])
	if err != nil {
		return nil, err
	}
	return &componentID{Name: string(name), Version: string(version)}, nil
}

func (cID *componentID) toString() string {
	nameEncoded := base64.URLEncoding.EncodeToString([]byte(cID.Name))
	versionEncoded := base64.URLEncoding.EncodeToString([]byte(cID.Version))
	return fmt.Sprintf("%s:%s", nameEncoded, versionEncoded)
}

// Map the images that matched a query to the image components it contains.
func mapImagesToComponentResolvers(root *Resolver, images []*storage.Image, query *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	componentPred, err := componentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	idToComponent := make(map[componentID]*EmbeddedImageScanComponentResolver)
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			if !componentPred(component) {
				continue
			}
			thisComponentID := componentID{Name: component.GetName(), Version: component.GetVersion()}
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
