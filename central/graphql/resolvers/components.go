package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
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
			"vulns: [EmbeddedVulnerability]!",
			"vulnCount: Int!",
			"vulnCounter: VulnerabilityCounter!",
			"lastScanned: Time",
			"images: [Image!]!",
			"imageCount: Int!",
			"imageCount: Int!",
			"deployments: [Deployment!]!",
			"deploymentCount: Int!",
			"priority: Int!",
		}),
		schema.AddExtraResolver("ImageScan", `components: [EmbeddedImageScanComponent!]!`),
		schema.AddQuery("imageComponent(id: ID): EmbeddedImageScanComponent"),
		schema.AddQuery("imageComponents(query: String): [EmbeddedImageScanComponent!]!"),
	)
}

// ImageComponent returns a component based on an input id (name:version)
func (resolver *Resolver) ImageComponent(ctx context.Context, args struct{ *graphql.ID }) (*EmbeddedImageScanComponentResolver, error) {
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

// ImageComponents returns the image scan components that match the input query.
func (resolver *Resolver) ImageComponents(ctx context.Context, q rawQuery) ([]*EmbeddedImageScanComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageComponents")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// Convert to query, but link the fields for the search.
	query, err := search.ParseQuery(q.String(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	// Check that all search inputs apply to vulnerabilities or images.
	if err := search.ValidateQuery(query, mappings.ComponentOptionsMap.Merge(mappings.OptionsMap)); err != nil {
		return nil, err
	}

	return components(ctx, resolver, query)
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

	// Filter the query to just the component portion.
	query = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	if query == nil {
		query = search.EmptyQuery()
	}

	return mapImagesToComponentResolvers(root, images, query)
}

func (resolver *imageScanResolver) Components(ctx context.Context) ([]*EmbeddedImageScanComponentResolver, error) {
	value := resolver.data.GetComponents()
	return resolver.root.wrapEmbeddedImageScanComponents(value, nil)
}

func (resolver *Resolver) wrapEmbeddedImageScanComponents(values []*storage.EmbeddedImageScanComponent, err error) ([]*EmbeddedImageScanComponentResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*EmbeddedImageScanComponentResolver, len(values))
	for i, v := range values {
		output[i] = &EmbeddedImageScanComponentResolver{root: resolver, data: v}
	}
	return output, nil
}

// EmbeddedImageScanComponentResolver resolves data about an image scan component.
type EmbeddedImageScanComponentResolver struct {
	root *Resolver
	data *storage.EmbeddedImageScanComponent

	images      []*imageResolver
	deployments []*deploymentResolver
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
	var latestTime *protoTypes.Timestamp
	for _, image := range eicr.images {
		if latestTime == nil || image.data.GetScan().GetScanTime().Compare(latestTime) > 0 {
			latestTime = image.data.GetScan().GetScanTime()
		}
	}
	return timestamp(latestTime)
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (eicr *EmbeddedImageScanComponentResolver) TopVuln(ctx context.Context) (*EmbeddedVulnerabilityResolver, error) {
	var maxCvss *storage.EmbeddedVulnerability
	for _, vuln := range eicr.data.GetVulns() {
		if maxCvss == nil || vuln.GetCvss() > maxCvss.GetCvss() {
			maxCvss = vuln
		}
	}
	return eicr.root.wrapEmbeddedVulnerability(maxCvss, nil)
}

// Vulns resolves the vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) Vulns(ctx context.Context) ([]*EmbeddedVulnerabilityResolver, error) {
	value := eicr.data.GetVulns()
	return eicr.root.wrapEmbeddedVulnerabilities(value, nil)
}

// VulnCount resolves the number of vulnerabilities contained in the image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCount(ctx context.Context) (int32, error) {
	return int32(len(eicr.data.GetVulns())), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an image component.
func (eicr *EmbeddedImageScanComponentResolver) VulnCounter(ctx context.Context) (*VulnerabilityCounterResolver, error) {
	return mapVulnsToVulnerabilityCounter(eicr.data.GetVulns()), nil
}

// Images are the images that contain the CVE/Vulnerability.
func (eicr *EmbeddedImageScanComponentResolver) Images(ctx context.Context) ([]*imageResolver, error) {
	if eicr.images == nil {
		return nil, errors.New("images not available from vulnerabilites resolved as children of an image")
	}
	return eicr.images, nil
}

// ImageCount is the number of images that contain the CVE/Vulnerability.
func (eicr *EmbeddedImageScanComponentResolver) ImageCount(ctx context.Context) (int32, error) {
	if eicr.images == nil {
		return 0, errors.New("images count not available from vulnerabilites resolved as children of an image")
	}
	return int32(len(eicr.images)), nil
}

// Deployments are the deployments that contain the CVE/Vulnerability.
func (eicr *EmbeddedImageScanComponentResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := eicr.loadDeployments(ctx); err != nil {
		return nil, err
	}
	return eicr.deployments, nil
}

// DeploymentCount is the number deployments that contain the CVE/Vulnerability.
func (eicr *EmbeddedImageScanComponentResolver) DeploymentCount(ctx context.Context) (int32, error) {
	if err := eicr.loadDeployments(ctx); err != nil {
		return 0, err
	}
	return int32(len(eicr.deployments)), nil
}

func (eicr *EmbeddedImageScanComponentResolver) loadDeployments(ctx context.Context) error {
	if eicr.images == nil {
		return errors.New("deployment info not available from vulnerabilites resolved as children of an image")
	} else if eicr.deployments != nil {
		return nil
	}

	// Create a query that finds all of the deployments that contain at least one of the infected images.
	qb := search.NewQueryBuilder()
	for _, image := range eicr.images {
		qb.AddExactMatches(search.ImageSHA, image.data.GetId())
	}
	q := qb.ProtoQuery()

	// Search the deployments.
	listDeps, err := eicr.root.DeploymentDataStore.SearchListDeployments(ctx, q)
	if err != nil {
		return err
	}

	// create resolvers.
	eicr.deployments = make([]*deploymentResolver, 0, len(listDeps))
	for _, listDep := range listDeps {
		eicr.deployments = append(eicr.deployments, &deploymentResolver{
			root: eicr.root,
			list: listDep,
		})
	}

	// Return resolvers.
	return nil
}

// Static helpers.
//////////////////

// Synthetic ID for component objects composed of the name and version of the component.
type componentID struct {
	Name    string
	Version string
}

func componentIDFromString(str string) (*componentID, error) {
	nameAndVersion := strings.Split(str, ":")
	if len(nameAndVersion) != 2 {
		return nil, fmt.Errorf("invalid id: %s", str)
	}
	return &componentID{Name: nameAndVersion[0], Version: nameAndVersion[1]}, nil
}

func (cID *componentID) toString() string {
	return fmt.Sprintf("%s:%s", cID.Name, cID.Version)
}

// Map the images that matched a query to the image components it contains.
func mapImagesToComponentResolvers(root *Resolver, images []*storage.Image, imageQuery *v1.Query) ([]*EmbeddedImageScanComponentResolver, error) {
	pred, err := componentPredicateFactory.GeneratePredicate(imageQuery)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	componentToImages := make(map[componentID][]*imageResolver)
	idToComponent := make(map[componentID]*storage.EmbeddedImageScanComponent)
	for _, image := range images {
		componentsWithImage := make(map[componentID]struct{})
		for _, component := range image.GetScan().GetComponents() {
			if !pred(component) {
				continue
			}
			thisComponentID := componentID{Name: component.GetName(), Version: component.GetVersion()}

			idToComponent[thisComponentID] = component
			if _, hasImageAlready := componentsWithImage[thisComponentID]; !hasImageAlready {
				componentsWithImage[thisComponentID] = struct{}{}
				componentToImages[thisComponentID] = append(componentToImages[thisComponentID], &imageResolver{
					root: root,
					data: image,
				})
			}
		}
	}

	// Create the resolvers.
	var resolvers []*EmbeddedImageScanComponentResolver
	for id, component := range idToComponent {
		resolvers = append(resolvers, &EmbeddedImageScanComponentResolver{
			root:   root,
			data:   component,
			images: componentToImages[id],
		})
	}
	return resolvers, nil
}
