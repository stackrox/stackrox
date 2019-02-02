package resolvers

import (
	"context"
	"strings"
	"sync"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/compliance/aggregation"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log            = logging.LoggerForModule()
	complianceOnce sync.Once
)

func init() {
	InitCompliance()
}

// InitCompliance is a function that registers compliance graphql resolvers with the static schema. It's exposed for
// feature flag / unit test reasons. Once the flag is gone, this can be folded into the normal init() method.
func InitCompliance() {
	if !features.Compliance.Enabled() {
		return
	}
	complianceOnce.Do(func() {
		schema := getBuilder()
		schema.AddQuery("complianceStandard(id:ID!): ComplianceStandardMetadata")
		schema.AddQuery("complianceStandards: [ComplianceStandardMetadata!]!")
		schema.AddQuery("aggregatedResults(groupBy:[ComplianceAggregation_Scope!],unit:ComplianceAggregation_Scope!,where:String): ComplianceAggregation_Response!")
		schema.AddExtraResolver("ComplianceStandardMetadata", "controls: [ComplianceControl!]!")
		schema.AddExtraResolver("ComplianceStandardMetadata", "groups: [ComplianceControlGroup!]!")
		schema.AddType("Namespace", []string{"cluster: Cluster", "name: String!"})
		schema.AddUnionType("ComplianceDomainKey", []string{"ComplianceStandardMetadata", "ComplianceControl", "Cluster", "Deployment", "Node", "Namespace"})
		schema.AddExtraResolver("ComplianceAggregation_Result", "keys: [ComplianceDomainKey!]!")
	})
}

// ComplianceStandards returns graphql resolvers for all compliance standards
func (resolver *Resolver) ComplianceStandards(ctx context.Context) ([]*complianceStandardMetadataResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComplianceStandardMetadatas(
		resolver.ComplianceStandardStore.Standards())
}

// ComplianceStandard returns a graphql resolver for a named compliance standard
func (resolver *Resolver) ComplianceStandard(ctx context.Context, args struct{ graphql.ID }) (*complianceStandardMetadataResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComplianceStandardMetadata(
		resolver.ComplianceStandardStore.StandardMetadata(string(args.ID)))
}

type aggregatedResultQuery struct {
	GroupBy *[]string
	Unit    string
	Where   *string
}

// AggregatedResults returns the aggregration of the last runs aggregated by scope, unit and filtered by a query
func (resolver *Resolver) AggregatedResults(ctx context.Context, args aggregatedResultQuery) (*complianceAggregationResponseWithDomainResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	standards, err := resolver.ComplianceStandardStore.Standards()
	if err != nil {
		return nil, err
	}

	clusters, err := resolver.ClusterDataStore.GetClusters()
	if err != nil {
		return nil, err
	}

	var clusterIDs, standardIDs []string
	if args.Where != nil {
		searchMap := search.ParseRawQueryIntoMap(*args.Where)
		standardIDs = aggregation.FilterStandards(standards, searchMap[search.Standard.String()])
		clusterIDs = aggregation.FilterClusters(clusters, searchMap[search.Cluster.String()])
	} else {
		standardIDs = aggregation.FilterStandards(standards, nil)
		clusterIDs = aggregation.FilterClusters(clusters, nil)
	}

	runResults, err := resolver.ComplianceDataStore.GetLatestRunResultsBatch(clusterIDs, standardIDs)
	if err != nil {
		return nil, err
	}

	validResults, sources := store.ValidResultsAndSources(runResults)
	results, domainFunc := aggregation.GetAggregatedResults(toComplianceAggregation_Scopes(args.GroupBy), toComplianceAggregation_Scope(&args.Unit), validResults)

	return &complianceAggregationResponseWithDomainResolver{
		complianceAggregation_ResponseResolver: complianceAggregation_ResponseResolver{
			root: resolver,
			data: &v1.ComplianceAggregation_Response{
				Results: results,
				Sources: sources,
			},
		},
		domainFunc: domainFunc,
	}, nil
}

type complianceDomainKeyResolver struct {
	root   *Resolver
	domain *storage.ComplianceDomain
	key    *v1.ComplianceAggregation_AggregationKey
}

func (resolver *complianceDomainKeyResolver) ToCluster() (cluster *clusterResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_CLUSTER {
		if resolver.domain.GetCluster() != nil {
			return &clusterResolver{resolver.root, resolver.domain.GetCluster()}, true
		}
	}
	return nil, false
}

func (resolver *complianceDomainKeyResolver) ToDeployment() (deployment *deploymentResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_DEPLOYMENT {
		deployment, found := resolver.domain.GetDeployments()[resolver.key.GetId()]
		if found {
			return &deploymentResolver{resolver.root, deployment, nil}, found
		}
	}
	return nil, false
}

func (resolver *complianceDomainKeyResolver) ToNode() (node *nodeResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_NODE {
		node, found := resolver.domain.GetNodes()[resolver.key.GetId()]
		if found {
			return &nodeResolver{resolver.root, node}, found
		}
	}
	return nil, false
}

type namespaceResolver struct {
	root    *Resolver
	name    string
	cluster *storage.Cluster
}

func (resolver *namespaceResolver) Name() string {
	return resolver.name
}

func (resolver *namespaceResolver) Cluster() *clusterResolver {
	if resolver.cluster == nil {
		return nil
	}
	return &clusterResolver{
		resolver.root,
		resolver.cluster,
	}
}

func (resolver *complianceDomainKeyResolver) ToNamespace() (namespace *namespaceResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_NAMESPACE {
		return &namespaceResolver{resolver.root, resolver.key.GetId(), resolver.domain.GetCluster()}, true
	}
	return nil, false
}

func (resolver *complianceDomainKeyResolver) ToComplianceStandardMetadata() (standard *complianceStandardMetadataResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_STANDARD {
		standard, found, err := resolver.root.ComplianceStandardStore.StandardMetadata(resolver.key.GetId())
		if err == nil && found {
			return &complianceStandardMetadataResolver{resolver.root, standard}, found
		}
	}
	return nil, false
}

func (resolver *complianceDomainKeyResolver) ToComplianceControl() (control *complianceControlResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_CONTROL {
		controlID := resolver.key.GetId()
		parts := strings.SplitN(controlID, ":", 2)
		if len(parts) == 2 {
			controls, err := resolver.root.ComplianceStandardStore.Controls(parts[0])
			if err == nil {
				for _, c := range controls {
					if c.GetId() == parts[1] {
						return &complianceControlResolver{resolver.root, c}, true
					}
				}
			}
		}
	}
	return nil, false
}

// ComplianceDomain returns a graphql resolver that loads the underlying object for an aggregation key
func (resolver *complianceAggregationResultWithDomainResolver) Keys() ([]*complianceDomainKeyResolver, error) {
	output := make([]*complianceDomainKeyResolver, len(resolver.data.AggregationKeys))
	for i, v := range resolver.data.AggregationKeys {
		output[i] = &complianceDomainKeyResolver{
			root:   resolver.root,
			domain: resolver.domain,
			key:    v,
		}
	}
	return output, nil
}

// ComplianceResults returns graphql resolvers for all matching compliance results
func (resolver *Resolver) ComplianceResults(ctx context.Context, query rawQuery) ([]*complianceControlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	q, err := query.AsV1Query()
	if err != nil {
		return nil, err
	}
	return resolver.wrapComplianceControlResults(
		resolver.ComplianceDataStore.QueryControlResults(q))
}

func (resolver *complianceStandardMetadataResolver) Controls(ctx context.Context) ([]*complianceControlResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapComplianceControls(
		resolver.root.ComplianceStandardStore.Controls(resolver.data.GetId()))
}

func (resolver *complianceStandardMetadataResolver) Groups(ctx context.Context) ([]*complianceControlGroupResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapComplianceControlGroups(
		resolver.root.ComplianceStandardStore.Groups(resolver.data.GetId()))
}
