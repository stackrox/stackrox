package resolvers

import (
	"context"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/search/scoped"
)

var (
	vulnPredicateFactory = predicate.NewFactory("vulnerability", &storage.EmbeddedVulnerability{})
)

func (resolver *Resolver) wrapEmbeddedVulnerability(value *storage.EmbeddedVulnerability, err error) (*EmbeddedVulnerabilityResolver, error) {
	if err != nil {
		return nil, err
	}
	return &EmbeddedVulnerabilityResolver{root: resolver, data: value}, nil
}

// EmbeddedVulnerabilityResolver resolves data about a CVE/Vulnerability.
// If using the top level vulnerability resolver (as opposed to the resolver under the top level image resolver) you get
// a couple of extensions that allow you to resolve some relationships.
type EmbeddedVulnerabilityResolver struct {
	ctx         context.Context
	root        *Resolver
	lastScanned *protoTypes.Timestamp
	data        *storage.EmbeddedVulnerability
}

// Suppressed returns whether CVE is suppressed (UI term: Snooze) or not
func (evr *EmbeddedVulnerabilityResolver) Suppressed(_ context.Context) bool {
	return evr.data.GetSuppressed()
}

// SuppressActivation returns the time when the CVE was suppressed
func (evr *EmbeddedVulnerabilityResolver) SuppressActivation(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.data.GetSuppressActivation())
}

// SuppressExpiry returns the time when the CVE suppression expires
func (evr *EmbeddedVulnerabilityResolver) SuppressExpiry(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.data.GetSuppressExpiry())
}

// Vectors returns either the CVSSV2 or CVSSV3 data.
func (evr *EmbeddedVulnerabilityResolver) Vectors() *EmbeddedVulnerabilityVectorsResolver {
	if val := evr.data.GetCvssV3(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV3Resolver{evr.ctx, evr.root, val},
		}
	}
	if val := evr.data.GetCvssV2(); val != nil {
		return &EmbeddedVulnerabilityVectorsResolver{
			resolver: &cVSSV2Resolver{evr.ctx, evr.root, val},
		}
	}
	return nil
}

// ID returns the CVE string (which is effectively an id)
func (evr *EmbeddedVulnerabilityResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(evr.data.GetCve())
}

// CVE returns the CVE string (which is effectively an id)
func (evr *EmbeddedVulnerabilityResolver) CVE(_ context.Context) string {
	return evr.data.GetCve()
}

// Cvss returns the CVSS score.
func (evr *EmbeddedVulnerabilityResolver) Cvss(_ context.Context) float64 {
	return float64(evr.data.GetCvss())
}

// Link returns a link to the vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Link(_ context.Context) string {
	return evr.data.GetLink()
}

// Summary returns the summary of the vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Summary(_ context.Context) string {
	return evr.data.GetSummary()
}

// ScoreVersion returns the version of the CVSS score returned.
func (evr *EmbeddedVulnerabilityResolver) ScoreVersion(_ context.Context) string {
	value := evr.data.GetScoreVersion()
	return value.String()
}

// FixedByVersion returns the version of the parent component that removes this CVE.
func (evr *EmbeddedVulnerabilityResolver) FixedByVersion(_ context.Context) (string, error) {
	return evr.data.GetFixedBy(), nil
}

// IsFixable returns whether or not a component with a fix exists.
func (evr *EmbeddedVulnerabilityResolver) IsFixable(_ context.Context, _ RawQuery) (bool, error) {
	return evr.data.GetFixedBy() != "", nil
}

// LastScanned is the last time the vulnerability was scanned in an image.
func (evr *EmbeddedVulnerabilityResolver) LastScanned(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.lastScanned)
}

// CreatedAt is the firsts time the vulnerability was scanned in an image. Unavailable in an image context.
func (evr *EmbeddedVulnerabilityResolver) CreatedAt(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.lastScanned)
}

// DiscoveredAtImage is the first time the vulnerability was discovered in the parent image.
func (evr *EmbeddedVulnerabilityResolver) DiscoveredAtImage(_ context.Context, _ RawQuery) (*graphql.Time, error) {
	return timestamp(evr.data.FirstImageOccurrence)
}

// VulnerabilityType returns the type of vulnerability
func (evr *EmbeddedVulnerabilityResolver) VulnerabilityType() string {
	return evr.data.VulnerabilityType.String()
}

// VulnerabilityTypes returns the types of the vulnerability
func (evr *EmbeddedVulnerabilityResolver) VulnerabilityTypes() []string {
	vulnTypes := make([]string, 0, len(evr.data.GetVulnerabilityTypes()))
	for _, vulnType := range evr.data.GetVulnerabilityTypes() {
		vulnTypes = append(vulnTypes, vulnType.String())
	}
	return vulnTypes
}

// Images are the images that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Images(ctx context.Context, args PaginatedQuery) ([]*imageResolver, error) {
	if err := readImages(ctx); err != nil {
		return []*imageResolver{}, nil
	}
	// Convert to query, but link the fields for the search.
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	images, err := evr.loadImages(ctx, query)
	if err != nil {
		return nil, err
	}
	return images, nil
}

// ImageCount is the number of images that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) ImageCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readImages(ctx); err != nil {
		return 0, nil
	}
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	query, err = search.AddAsConjunction(evr.vulnQuery(), query)
	if err != nil {
		return 0, err
	}
	return imageLoader.CountFromQuery(ctx, query)
}

// Deployments are the deployments that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Deployments(ctx context.Context, args PaginatedQuery) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return evr.loadDeployments(ctx, query)
}

// DeploymentCount is the number of deployments that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) DeploymentCount(ctx context.Context, args RawQuery) (int32, error) {
	if err := readDeployments(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	deploymentBaseQuery, err := evr.getDeploymentBaseQuery(ctx)
	if err != nil || deploymentBaseQuery == nil {
		return 0, err
	}
	deploymentLoader, err := loaders.GetDeploymentLoader(ctx)
	if err != nil {
		return 0, err
	}
	return deploymentLoader.CountFromQuery(ctx, search.ConjunctionQuery(deploymentBaseQuery, query))
}

// Nodes are the nodes that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, nil
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	query, err = search.AddAsConjunction(evr.vulnQuery(), query)
	if err != nil {
		return nil, err
	}

	query.Pagination = pagination

	return evr.root.wrapNodes(nodeLoader.FromQuery(ctx, query))
}

// NodeCount is the number of nodes that contain the CVE/Vulnerability.
func (evr *EmbeddedVulnerabilityResolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
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
	query, err = search.AddAsConjunction(evr.vulnQuery(), query)
	if err != nil {
		return 0, err
	}
	return nodeLoader.CountFromQuery(ctx, query)
}

// Clusters returns resolvers for clusters affected by cluster vulnerability.
func (evr *EmbeddedVulnerabilityResolver) Clusters(ctx context.Context, args PaginatedQuery) ([]*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "Clusters")

	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	query := search.AddRawQueriesAsConjunction(args.String(), evr.vulnRawQuery())
	return evr.root.Clusters(ctx, PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// ClusterCount returns a number of clusters affected by cluster vulnerability.
func (evr *EmbeddedVulnerabilityResolver) ClusterCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.ClusterCVEs, "ClusterCount")

	if err := readClusters(ctx); err != nil {
		return 0, err
	}
	query := search.AddRawQueriesAsConjunction(args.String(), evr.vulnRawQuery())
	return evr.root.ClusterCount(ctx, RawQuery{Query: &query})
}

func (resolver *Resolver) getComponentsForAffectedCluster(ctx context.Context, cve *schema.NVDCVEFeedJSON10DefCVEItem, ct utils.CVEType) (int, int, error) {
	clusters, err := resolver.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return 0, 0, err
	}
	if len(clusters) == 0 {
		return 0, 0, nil
	}

	affectedClusters, err := resolver.cveMatcher.GetAffectedClusters(ctx, cve)
	if err != nil {
		return 0, 0, errors.Errorf("unknown CVE type: %s", ct)
	}
	return len(affectedClusters), len(clusters), nil
}

func (evr *EmbeddedVulnerabilityResolver) getEnvImpactComponentsForPerClusterVuln(ctx context.Context, ct utils.CVEType) (int, int, error) {
	clusters, err := evr.root.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return 0, 0, err
	}
	affectedClusters, err := evr.root.orchestratorIstioCVEManager.GetAffectedClusters(ctx, evr.data.Cve, ct, evr.root.cveMatcher)
	if err != nil {
		return 0, 0, err
	}
	return len(affectedClusters), len(clusters), nil
}

func (evr *EmbeddedVulnerabilityResolver) getEnvImpactComponentsForImages(ctx context.Context) (numerator, denominator int, err error) {
	allDepsCount, err := evr.root.DeploymentDataStore.CountDeployments(ctx)
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
	withThisCVECount, err := deploymentLoader.CountFromQuery(evr.scopeContext(ctx), search.EmptyQuery())
	if err != nil {
		return 0, 0, err
	}
	return int(withThisCVECount), allDepsCount, nil
}

func (evr *EmbeddedVulnerabilityResolver) getEnvImpactComponentsForNodes(ctx context.Context) (numerator, denominator int, err error) {
	allNodesCount, err := evr.root.NodeDataStore.CountNodes(ctx)
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
	withThisCVECount, err := nodeLoader.CountFromQuery(evr.scopeContext(ctx), search.EmptyQuery())
	if err != nil {
		return 0, 0, err
	}
	return int(withThisCVECount), allNodesCount, nil
}

func (evr *EmbeddedVulnerabilityResolver) scopeContext(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		ID:    evr.data.GetCve(),
		Level: v1.SearchCategory_VULNERABILITIES,
	})
}

// EnvImpact is the fraction of deployments that contains the CVE
func (evr *EmbeddedVulnerabilityResolver) EnvImpact(ctx context.Context) (float64, error) {
	var numerator, denominator int
	for _, vulnType := range evr.data.GetVulnerabilityTypes() {
		var n, d int
		var err error

		switch vulnType {
		case storage.EmbeddedVulnerability_K8S_VULNERABILITY:
			n, d, err = evr.getEnvImpactComponentsForPerClusterVuln(ctx, utils.K8s)
		case storage.EmbeddedVulnerability_ISTIO_VULNERABILITY:
			n, d, err = evr.getEnvImpactComponentsForPerClusterVuln(ctx, utils.Istio)
		case storage.EmbeddedVulnerability_OPENSHIFT_VULNERABILITY:
			n, d, err = evr.getEnvImpactComponentsForPerClusterVuln(ctx, utils.OpenShift)
		case storage.EmbeddedVulnerability_IMAGE_VULNERABILITY:
			n, d, err = evr.getEnvImpactComponentsForImages(ctx)
		case storage.EmbeddedVulnerability_NODE_VULNERABILITY:
			n, d, err = evr.getEnvImpactComponentsForNodes(ctx)
		default:
			return 0, errors.Errorf("unknown CVE type: %s", evr.data.GetVulnerabilityType())
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

// Severity return the severity of the vulnerability (CVSSv3 or CVSSv2).
func (evr *EmbeddedVulnerabilityResolver) Severity(_ context.Context) string {
	return evr.data.GetSeverity().String()
}

// PublishedOn is the time the vulnerability was published (ref: NVD).
func (evr *EmbeddedVulnerabilityResolver) PublishedOn(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.data.GetPublishedOn())
}

// LastModified is the time the vulnerability was last modified (ref: NVD).
func (evr *EmbeddedVulnerabilityResolver) LastModified(_ context.Context) (*graphql.Time, error) {
	return timestamp(evr.data.GetLastModified())
}

// ImpactScore returns the impact score of the vulnerability.
func (evr *EmbeddedVulnerabilityResolver) ImpactScore(_ context.Context) float64 {
	if val := evr.data.GetCvssV3(); val != nil {
		return float64(evr.data.GetCvssV3().GetImpactScore())
	}
	if val := evr.data.GetCvssV2(); val != nil {
		return float64(evr.data.GetCvssV2().GetImpactScore())
	}
	return float64(0.0)
}

// UnusedVarSink represents a query sink
func (evr *EmbeddedVulnerabilityResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

func (evr *EmbeddedVulnerabilityResolver) loadImages(ctx context.Context, query *v1.Query) ([]*imageResolver, error) {
	imageLoader, err := loaders.GetImageLoader(ctx)
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	query, err = search.AddAsConjunction(evr.vulnQuery(), query)
	if err != nil {
		return nil, err
	}

	query.Pagination = pagination

	return evr.root.wrapImages(imageLoader.FromQuery(ctx, query))
}

func (evr *EmbeddedVulnerabilityResolver) loadDeployments(ctx context.Context, query *v1.Query) ([]*deploymentResolver, error) {
	deploymentBaseQuery, err := evr.getDeploymentBaseQuery(ctx)
	if err != nil || deploymentBaseQuery == nil {
		return nil, err
	}
	// Search the deployments.
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

	return evr.root.wrapListDeployments(ListDeploymentLoader.FromQuery(ctx, query))
}

// ActiveState shows the activeness of a vulnerability in a deployment context.
func (evr *EmbeddedVulnerabilityResolver) ActiveState(ctx context.Context, _ RawQuery) (*activeStateResolver, error) {
	if !features.ActiveVulnMgmt.Enabled() {
		return &activeStateResolver{}, nil
	}
	deploymentID := getDeploymentScope(nil, ctx, evr.ctx)
	if deploymentID == "" {
		return nil, nil
	}

	// We only support OS level component. The active state is not determined if there is no OS level component associate with this vuln.
	query := search.NewQueryBuilder().AddExactMatches(search.CVE, evr.data.GetCve()).AddExactMatches(search.ComponentSource, storage.SourceType_OS.String()).ProtoQuery()
	osLevelComponents, err := evr.root.ImageComponentDataStore.Count(ctx, query)
	if err != nil {
		return nil, err
	}
	if osLevelComponents == 0 {
		return &activeStateResolver{root: evr.root, state: Undetermined}, nil
	}

	query = search.ConjunctionQuery(evr.vulnQuery(), search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentID).ProtoQuery())
	results, err := evr.root.ActiveComponent.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	state := Inactive
	if len(ids) != 0 {
		state = Active
	}
	return &activeStateResolver{root: evr.root, state: state, activeComponentIDs: ids}, nil
}

// VulnerabilityState return the effective state of this vulnerability (observed, deferred or marked as false positive).
func (evr *EmbeddedVulnerabilityResolver) VulnerabilityState(_ context.Context) string {
	return evr.data.GetState().String()
}

func (evr *EmbeddedVulnerabilityResolver) getDeploymentBaseQuery(ctx context.Context) (*v1.Query, error) {
	imageQuery := evr.vulnQuery()
	results, err := evr.root.ImageDataStore.Search(ctx, imageQuery)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	// Create a query that finds all of the deployments that contain at least one of the infected images.
	return search.NewQueryBuilder().AddExactMatches(search.ImageSHA, search.ResultsToIDs(results)...).ProtoQuery(), nil
}

func (evr *EmbeddedVulnerabilityResolver) vulnQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, evr.data.GetCve()).ProtoQuery()
}

func (evr *EmbeddedVulnerabilityResolver) vulnRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.CVE, evr.data.GetCve()).Query()
}

// EffectiveVulnerabilityRequest is not implemented for v1.
func (evr *EmbeddedVulnerabilityResolver) EffectiveVulnerabilityRequest(_ context.Context) (*VulnerabilityRequestResolver, error) {
	return nil, nil
}
