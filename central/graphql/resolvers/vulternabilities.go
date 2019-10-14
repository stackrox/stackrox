package resolvers

import (
	"context"
	"fmt"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/mappings"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	vulnPredicateFactory = predicate.NewFactory(&storage.EmbeddedVulnerability{})
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedVulnerability", []string{
			"cve: String!",
			"cvss: Float!",
			"scoreVersion: String!",
			"vectors: EmbeddedVulnerabilityVectors",
			"link: String!",
			"summary: String!",
			"fixedByVersion: String!",
			"isFixable: Boolean!",
			"lastScanned: Time",
			"components: [EmbeddedImageScanComponent!]!",
			"componentCount: Int!",
			"images: [Image!]!",
			"imageCount: Int!",
			"deployments: [Deployment!]!",
			"deploymentCount: Int!",
		}),
		schema.AddQuery("vulnerability(id: ID): EmbeddedVulnerability"),
		schema.AddQuery("vulnerabilities(query: String): [EmbeddedVulnerability!]!"),
	)
}

// Vulnerability resolves a single vulnerability based on an id (the CVE value).
func (resolver *Resolver) Vulnerability(ctx context.Context, args struct{ *graphql.ID }) (*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Vulnerability")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	query := search.NewQueryBuilder().AddExactMatches(search.CVE, string(*args.ID)).ProtoQuery()
	vulns, err := vulnerabilities(ctx, resolver, query)
	if err != nil {
		return nil, err
	} else if len(vulns) == 0 {
		return nil, nil
	} else if len(vulns) > 1 {
		return nil, fmt.Errorf("multiple vulns matched: %s this should not happen", string(*args.ID))
	}
	return vulns[0], nil
}

// Vulnerabilities resolves a set of vulnerabilities based on a query.
func (resolver *Resolver) Vulnerabilities(ctx context.Context, q rawQuery) ([]*EmbeddedVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Vulnerabilities")
	if err := readImages(ctx); err != nil {
		return nil, err
	}

	// Convert to query, but link the fields for the search.
	query, err := search.ParseQuery(q.String(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	// Check that all search inputs apply to vulnerabilities or images.
	if err := search.ValidateQuery(query, mappings.VulnerabilityOptionsMap.Merge(mappings.OptionsMap)); err != nil {
		return nil, err
	}

	return vulnerabilities(ctx, resolver, query)
}

// Helper function that actually runs the queries and produces the resolvers from the images.
func vulnerabilities(ctx context.Context, root *Resolver, query *v1.Query) ([]*EmbeddedVulnerabilityResolver, error) {
	// Run search on images.
	images, err := root.ImageDataStore.SearchRawImages(ctx, query)
	if err != nil {
		return nil, err
	}

	// Filter the query to just the vulnerability portion.
	query = search.FilterQueryWithMap(query, mappings.VulnerabilityOptionsMap)
	if query == nil {
		query = search.EmptyQuery()
	}

	return mapImagesToVulnerabilityResolvers(root, images, query)
}

func (resolver *Resolver) wrapEmbeddedVulnerabilities(values []*storage.EmbeddedVulnerability, err error) ([]*EmbeddedVulnerabilityResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*EmbeddedVulnerabilityResolver, len(values))
	for i, v := range values {
		output[i] = &EmbeddedVulnerabilityResolver{root: resolver, data: v}
	}
	return output, nil
}

// EmbeddedVulnerabilityResolver resolves data about a CVE/Vulnerability.
// If using the top level vulnerability resolver (as opposed to the resolver under the top level image resolver) you get
// a couple of extensions that allow you to resolve some relationships.
type EmbeddedVulnerabilityResolver struct {
	root *Resolver
	data *storage.EmbeddedVulnerability

	images      []*imageResolver
	components  []*EmbeddedImageScanComponentResolver
	deployments []*deploymentResolver
}

// Vectors returns either the CVSSV2 or CVSSV3 data.
func (evr *EmbeddedVulnerabilityResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	if val := evr.data.GetCvssV2(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV2Resolver{evr.root, val},
		}
	}
	if val := evr.data.GetCvssV3(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV3Resolver{evr.root, val},
		}
	}
	return nil
}

// Cve returns the CVE string (which is effectively an id)
func (evr *EmbeddedVulnerabilityResolver) Cve(ctx context.Context) string {
	return evr.data.GetCve()
}

// Cvss returns the CVSS score.
func (evr *EmbeddedVulnerabilityResolver) Cvss(ctx context.Context) float64 {
	return float64(evr.data.GetCvss())
}

// Link returns a link to the vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Link(ctx context.Context) string {
	return evr.data.GetLink()
}

// Summary returns the summary of the vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Summary(ctx context.Context) string {
	return evr.data.GetSummary()
}

// ScoreVersion returns the version of the CVSS score returned.
func (evr *EmbeddedVulnerabilityResolver) ScoreVersion(ctx context.Context) string {
	value := evr.data.GetScoreVersion()
	return value.String()
}

// FixedByVersion returns the version of the parent component that removes this CVE.
// TODO: this and below should only be accessible from a single component context.
func (evr *EmbeddedVulnerabilityResolver) FixedByVersion(ctx context.Context) string {
	return evr.data.GetFixedBy()
}

// IsFixable returns whether or not a component with a fix exists.
func (evr *EmbeddedVulnerabilityResolver) IsFixable(ctx context.Context) bool {
	return evr.data.GetFixedBy() != ""
}

// LastScanned is the last time the vulnerability was scanned in an image.
func (evr *EmbeddedVulnerabilityResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	var latestTime *protoTypes.Timestamp
	for _, image := range evr.images {
		if latestTime == nil || image.data.GetScan().GetScanTime().Compare(latestTime) > 0 {
			latestTime = image.data.GetScan().GetScanTime()
		}
	}
	return timestamp(latestTime)
}

// Components are the components that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Components(ctx context.Context) ([]*EmbeddedImageScanComponentResolver, error) {
	if evr.components == nil {
		return nil, errors.New("components not available from vulnerabilites resolved as children of an image")
	}
	return evr.components, nil
}

// ComponentCount is the number of components that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) ComponentCount(ctx context.Context) (int32, error) {
	if evr.components == nil {
		return 0, errors.New("component count not available from vulnerabilites resolved as children of an image")
	}
	return int32(len(evr.components)), nil
}

// Images are the images that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Images(ctx context.Context) ([]*imageResolver, error) {
	if evr.images == nil {
		return nil, errors.New("images not available from vulnerabilites resolved as children of an image")
	}
	return evr.images, nil
}

// ImageCount is the number of images that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) ImageCount(ctx context.Context) (int32, error) {
	if evr.images == nil {
		return 0, errors.New("images count not available from vulnerabilites resolved as children of an image")
	}
	return int32(len(evr.images)), nil
}

// Deployments are the deployments that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := evr.loadDeployments(ctx); err != nil {
		return nil, err
	}
	return evr.deployments, nil
}

// DeploymentCount is the number of deployments that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) DeploymentCount(ctx context.Context) (int32, error) {
	if err := evr.loadDeployments(ctx); err != nil {
		return 0, err
	}
	return int32(len(evr.deployments)), nil
}

func (evr *EmbeddedVulnerabilityResolver) loadDeployments(ctx context.Context) error {
	if evr.images == nil {
		return errors.New("deployment info not available from vulnerabilites resolved as children of an image")
	} else if evr.deployments != nil {
		return nil
	}

	// Create a query that finds all of the deployments that contain at least one of the infected images.
	qb := search.NewQueryBuilder()
	for _, image := range evr.images {
		qb.AddExactMatches(search.ImageSHA, image.data.GetId())
	}
	q := qb.ProtoQuery()

	// Search the deployments.
	listDeps, err := evr.root.DeploymentDataStore.SearchListDeployments(ctx, q)
	if err != nil {
		return err
	}

	// create resolvers.
	evr.deployments = make([]*deploymentResolver, 0, len(listDeps))
	for _, listDep := range listDeps {
		evr.deployments = append(evr.deployments, &deploymentResolver{
			root: evr.root,
			list: listDep,
		})
	}

	// Return resolvers.
	return nil
}

// Static helpers.
//////////////////

// Map the images that matched a query to the vulnerabilities it contains.
func mapImagesToVulnerabilityResolvers(root *Resolver, images []*storage.Image, query *v1.Query) ([]*EmbeddedVulnerabilityResolver, error) {
	pred, err := vulnPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	// Use the images to map CVEs to the images and components.
	cveToVuln := make(map[string]*storage.EmbeddedVulnerability)
	cveToImages := make(map[string][]*imageResolver)
	cveToComponents := make(map[string][]*EmbeddedImageScanComponentResolver)
	for _, image := range images {
		cvesWithImage := set.NewStringSet()
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !pred(vuln) {
					continue
				}
				if _, exists := cveToVuln[vuln.GetCve()]; !exists {
					cveToVuln[vuln.GetCve()] = vuln
				}
				if !cvesWithImage.Contains(vuln.GetCve()) {
					cvesWithImage.Add(vuln.GetCve())
					cveToImages[vuln.GetCve()] = append(cveToImages[vuln.GetCve()], &imageResolver{
						root: root,
						data: image,
					})
				}
				cveToComponents[vuln.GetCve()] = append(cveToComponents[vuln.GetCve()], &EmbeddedImageScanComponentResolver{
					root: root,
					data: component,
				})
			}
		}
	}

	// Create the resolvers.
	var resolvers []*EmbeddedVulnerabilityResolver
	for cve, vuln := range cveToVuln {
		resolvers = append(resolvers, &EmbeddedVulnerabilityResolver{
			root:       root,
			data:       vuln,
			images:     cveToImages[cve],
			components: cveToComponents[cve],
		})
	}
	return resolvers, nil
}
