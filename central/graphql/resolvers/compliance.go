package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/namespace"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	complianceOnce sync.Once
)

func init() {
	InitCompliance()
}

// InitCompliance is a function that registers compliance graphql resolvers with the static schema. It's exposed for
// feature flag / unit test reasons. Once the flag is gone, this can be folded into the normal init() method.
func InitCompliance() {
	complianceOnce.Do(func() {
		schema := getBuilder()
		utils.Must(
			schema.AddQuery("complianceStandard(id:ID!): ComplianceStandardMetadata"),
			schema.AddQuery("complianceStandards: [ComplianceStandardMetadata!]!"),
			schema.AddQuery("aggregatedResults(groupBy:[ComplianceAggregation_Scope!],unit:ComplianceAggregation_Scope!,where:String): ComplianceAggregation_Response!"),
			schema.AddQuery("complianceControl(id:ID!): ComplianceControl"),
			schema.AddQuery("complianceControlGroup(id:ID!): ComplianceControlGroup"),
			schema.AddExtraResolver("ComplianceStandardMetadata", "controls: [ComplianceControl!]!"),
			schema.AddExtraResolver("ComplianceStandardMetadata", "groups: [ComplianceControlGroup!]!"),
			schema.AddUnionType("ComplianceDomainKey", []string{"ComplianceStandardMetadata", "ComplianceControlGroup", "ComplianceControl", "Cluster", "Deployment", "Node", "Namespace"}),
			schema.AddExtraResolver("ComplianceAggregation_Result", "keys: [ComplianceDomainKey!]!"),
			schema.AddUnionType("Resource", []string{"Deployment", "Cluster", "Node"}),
			schema.AddType("ControlResult", []string{"resource: Resource", "control: ComplianceControl", "value: ComplianceResultValue"}),
			schema.AddExtraResolver("ComplianceStandardMetadata", "complianceResults: [ControlResult!]!"),
			schema.AddExtraResolver("ComplianceControl", "complianceResults: [ControlResult!]!"),
		)
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

// ComplianceControl retrieves an individual control by ID
func (resolver *Resolver) ComplianceControl(ctx context.Context, args struct{ graphql.ID }) (*complianceControlResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	control := resolver.ComplianceStandardStore.Control(string(args.ID))
	return resolver.wrapComplianceControl(control, control != nil, nil)
}

// ComplianceControlGroup retrieves a control group by ID
func (resolver *Resolver) ComplianceControlGroup(ctx context.Context, args struct{ graphql.ID }) (*complianceControlGroupResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	group := resolver.ComplianceStandardStore.Group(string(args.ID))
	return resolver.wrapComplianceControlGroup(group, group != nil, nil)
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
	var where string
	if args.Where != nil {
		where = *args.Where
	}

	groupBy := toComplianceAggregation_Scopes(args.GroupBy)
	unit := toComplianceAggregation_Scope(&args.Unit)

	validResults, sources, domainMap, err := resolver.ComplianceAggregator.Aggregate(ctx, where, groupBy, unit)
	if err != nil {
		return nil, err
	}

	return &complianceAggregationResponseWithDomainResolver{
		complianceAggregation_ResponseResolver: complianceAggregation_ResponseResolver{
			root: resolver,
			data: &v1.ComplianceAggregation_Response{
				Results: validResults,
				Sources: sources,
			},
		},
		domainMap: domainMap,
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

func (resolver *complianceDomainKeyResolver) ToNamespace() (*namespaceResolver, bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_NAMESPACE {
		receivedNS, found, err := namespace.ResolveByID(context.TODO(), resolver.key.GetId(), resolver.root.NamespaceDataStore,
			resolver.root.DeploymentDataStore, resolver.root.SecretsDataStore, resolver.root.NetworkPoliciesStore)
		if err == nil && found {
			return &namespaceResolver{resolver.root, receivedNS}, true
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

func (resolver *complianceDomainKeyResolver) ToComplianceStandardMetadata() (standard *complianceStandardMetadataResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_STANDARD {
		standard, found, err := resolver.root.ComplianceStandardStore.StandardMetadata(resolver.key.GetId())
		if err == nil && found {
			return &complianceStandardMetadataResolver{resolver.root, standard}, found
		}
	}
	return nil, false
}

// ToComplianceControl returns a resolver for a control if the domain key refers to a control and it exists
func (resolver *complianceDomainKeyResolver) ToComplianceControl() (control *complianceControlResolver, found bool) {
	if resolver.key.GetScope() == v1.ComplianceAggregation_CONTROL {
		controlID := resolver.key.GetId()
		control := resolver.root.ComplianceStandardStore.Control(controlID)
		if control != nil {
			return &complianceControlResolver{
				root: resolver.root,
				data: control,
			}, true
		}
	}
	return nil, false
}

// ToComplianceControlGroup returns a resolver for a group if the domain key refers to a control group and it exists
func (resolver *complianceDomainKeyResolver) ToComplianceControlGroup() (group *complianceControlGroupResolver, found bool) {
	if (resolver.key.GetScope()) == v1.ComplianceAggregation_CATEGORY {
		groupID := resolver.key.GetId()
		control := resolver.root.ComplianceStandardStore.Group(groupID)
		if control != nil {
			return &complianceControlGroupResolver{
				root: resolver.root,
				data: control,
			}, true
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

func allStandards(repository standards.Repository) []string {
	s, err := repository.Standards()
	if err != nil {
		return nil
	}
	ids := make([]string, len(s))
	for i, st := range s {
		ids[i] = st.GetId()
	}
	return ids
}

type controlResultResolver struct {
	root       *Resolver
	controlID  string
	value      *storage.ComplianceResultValue
	deployment *storage.Deployment
	cluster    *storage.Cluster
	node       *storage.Node
}

type bulkControlResults []*controlResultResolver

func newBulkControlResults() *bulkControlResults {
	output := make(bulkControlResults, 0)
	return &output
}

func (container *bulkControlResults) addDeploymentData(root *Resolver, data map[compliance.ClusterStandardPair]store.ResultsWithStatus, filter func(*storage.Deployment, *v1.ComplianceControl) bool) {
	for _, v := range data {
		for did, res := range v.LastSuccessfulResults.GetDeploymentResults() {
			deployment := v.LastSuccessfulResults.GetDomain().GetDeployments()[did]
			results := res.GetControlResults()
			for controlID, result := range results {
				if filter == nil || filter(deployment, root.ComplianceStandardStore.Control(controlID)) {
					*container = append(*container, &controlResultResolver{
						root:       root,
						controlID:  controlID,
						value:      result,
						deployment: deployment,
					})
				}
			}
		}
	}
}

func (container *bulkControlResults) addClusterData(root *Resolver, data map[compliance.ClusterStandardPair]store.ResultsWithStatus, filter func(control *v1.ComplianceControl) bool) {
	for _, v := range data {
		res := v.LastSuccessfulResults.GetClusterResults()
		results := res.GetControlResults()
		for controlID, result := range results {
			if filter == nil || filter(root.ComplianceStandardStore.Control(controlID)) {
				*container = append(*container, &controlResultResolver{
					root:      root,
					controlID: controlID,
					value:     result,
					cluster:   v.LastSuccessfulResults.GetDomain().GetCluster(),
				})
			}
		}
	}
}

func (container *bulkControlResults) addNodeData(root *Resolver, data map[compliance.ClusterStandardPair]store.ResultsWithStatus, filter func(node *storage.Node, control *v1.ComplianceControl) bool) {
	for _, v := range data {
		for nodeID, res := range v.LastSuccessfulResults.GetNodeResults() {
			node := v.LastSuccessfulResults.GetDomain().GetNodes()[nodeID]
			results := res.GetControlResults()
			for controlID, result := range results {
				if filter == nil || filter(node, root.ComplianceStandardStore.Control(controlID)) {
					*container = append(*container, &controlResultResolver{
						root:      root,
						controlID: controlID,
						value:     result,
						node:      node,
					})
				}
			}
		}
	}
}

func (resolver *controlResultResolver) Resource(ctx context.Context) *controlResultResolver {
	return resolver
}

func (resolver *controlResultResolver) Control(ctx context.Context) (*complianceControlResolver, error) {
	return resolver.root.ComplianceControl(ctx, struct{ graphql.ID }{graphql.ID(resolver.controlID)})
}

func (resolver *controlResultResolver) ToDeployment() (*deploymentResolver, bool) {
	if resolver.deployment == nil {
		return nil, false
	}
	return &deploymentResolver{resolver.root, resolver.deployment, nil}, true
}

func (resolver *controlResultResolver) ToCluster() (*clusterResolver, bool) {
	if resolver.cluster == nil {
		return nil, false
	}
	return &clusterResolver{root: resolver.root, data: resolver.cluster}, true
}

func (resolver *controlResultResolver) ToNode() (*nodeResolver, bool) {
	if resolver.node == nil {
		return nil, false
	}
	return &nodeResolver{root: resolver.root, data: resolver.node}, true
}

func (resolver *controlResultResolver) Value(ctx context.Context) *complianceResultValueResolver {
	return &complianceResultValueResolver{
		root: resolver.root,
		data: resolver.value,
	}
}

func allClusters(ctx context.Context, resolver *Resolver) []string {
	clusters, err := resolver.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil
	}
	output := make([]string, 0, len(clusters))
	for _, cl := range clusters {
		output = append(output, cl.GetId())
	}
	return output
}

func (resolver *complianceStandardMetadataResolver) ComplianceResults(ctx context.Context) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	data, err := resolver.root.ComplianceDataStore.GetLatestRunResultsBatch(allClusters(ctx, resolver.root), []string{resolver.data.GetId()}, store.RequireMessageStrings)
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	output.addClusterData(resolver.root, data, nil)
	output.addDeploymentData(resolver.root, data, nil)
	output.addNodeData(resolver.root, data, nil)

	return *output, nil
}

func (resolver *complianceControlResolver) ComplianceResults(ctx context.Context) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	data, err := resolver.root.ComplianceDataStore.GetLatestRunResultsBatch(allClusters(ctx, resolver.root), []string{resolver.data.GetStandardId()}, store.RequireMessageStrings)
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	output.addClusterData(resolver.root, data, func(control *v1.ComplianceControl) bool {
		return control.GetId() == resolver.data.GetId()
	})
	output.addDeploymentData(resolver.root, data, func(deployment *storage.Deployment, control *v1.ComplianceControl) bool {
		return control.GetId() == resolver.data.GetId()
	})
	output.addNodeData(resolver.root, data, func(node *storage.Node, control *v1.ComplianceControl) bool {
		return control.GetId() == resolver.data.GetId()
	})

	return *output, nil
}
