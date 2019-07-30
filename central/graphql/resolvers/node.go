package resolvers

import (
	"context"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("node(id:ID!): Node"),
		schema.AddQuery("nodes(query: String): [Node!]!"),
		schema.AddExtraResolver("Node", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddType("NumNodeComplianceControls", []string{"numFailing: Int!", "numPassing: Int!"}),
		schema.AddExtraResolver("Node", "numNodeComplianceControls: NumNodeComplianceControls!"),
		schema.AddExtraResolver("Node", "controlStatus: Boolean!"),
		schema.AddExtraResolver("Node", "failingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Node", "passingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Node", "controls(query: String): [ComplianceControl!]!"),
	)
}

// Node returns a resolver for a matching node, or nil if no node is found in any cluster
func (resolver *Resolver) Node(ctx context.Context, args struct{ graphql.ID }) (*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	clusters, err := resolver.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	var output *nodeResolver
	for _, cluster := range clusters {
		store, err := resolver.NodeGlobalDataStore.GetClusterNodeStore(ctx, cluster.GetId(), false)
		if err != nil {
			return nil, err
		}
		node, err := store.GetNode(string(args.ID))
		if err != nil {
			return nil, err
		}
		if node != nil {
			if output == nil {
				output = &nodeResolver{root: resolver, data: node}
			} else {
				return nil, status.Error(codes.Internal, "multiple matching node ids found")
			}
		}
	}
	return output, nil
}

// Nodes returns resolvers for a matching nodes, or nil if no node is found in any cluster
func (resolver *Resolver) Nodes(ctx context.Context, args rawQuery) ([]*nodeResolver, error) {
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}

	var nodeResolvers []*nodeResolver
	nodes, err := resolver.NodeGlobalDataStore.SearchRawNodes(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		nodeResolvers = append(nodeResolvers, &nodeResolver{root: resolver, data: node})
	}

	return nodeResolvers, nil
}

func (resolver *nodeResolver) ComplianceResults(ctx context.Context, args rawQuery) ([]*controlResultResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nodeID := resolver.data.GetId()
	output.addNodeData(resolver.root, runResults, func(node *storage.Node, _ *v1.ComplianceControl) bool {
		return node.GetId() == nodeID
	})
	return *output, nil
}

func (resolver *nodeResolver) NumNodeComplianceControls(ctx context.Context) (*numNodeComplianceControlsResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	r, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_NODE}, rawQuery{})
	if err != nil {
		return nil, err
	}
	if r == nil {
		return &numNodeComplianceControlsResolver{}, nil
	}
	if len(r) != 1 {
		return &numNodeComplianceControlsResolver{}, errors.Errorf("unexpected node aggregation results length: expected: 1, actual: %d", len(r))
	}
	return &numNodeComplianceControlsResolver{numFailing: r[0].GetNumFailing(), numPassing: r[0].GetNumPassing()}, nil
}

func (resolver *nodeResolver) ControlStatus(ctx context.Context) (bool, error) {
	if err := readCompliance(ctx); err != nil {
		return false, err
	}
	r, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_NODE}, rawQuery{})
	if err != nil || r == nil {
		return false, err
	}
	if len(r) != 1 {
		return false, errors.Errorf("unexpected node aggregation results length: expected: 1, actual: %d", len(r))
	}
	return r[0].GetNumFailing() == 0, nil
}

func (resolver *nodeResolver) getNodeLastSuccessfulComplianceRunAggregatedResult(ctx context.Context, scope []v1.ComplianceAggregation_Scope, args rawQuery) ([]*v1.ComplianceAggregation_Result, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	standardIDs, err := getStandardIDs(ctx, resolver.root.ComplianceStandardStore)
	if err != nil {
		return nil, err
	}
	hasComplianceSuccessfullyRun, err := resolver.root.ComplianceDataStore.IsComplianceRunSuccessfulOnCluster(ctx, resolver.data.GetClusterId(), standardIDs)
	if err != nil || !hasComplianceSuccessfullyRun {
		return nil, err
	}
	query, err := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.NodeID, resolver.data.GetId()).RawQuery()
	if err != nil {
		return nil, err
	}
	if args.Query != nil {
		query = strings.Join([]string{query, args.String()}, "+")
	}
	r, _, _, err := resolver.root.ComplianceAggregator.Aggregate(ctx, query, scope, v1.ComplianceAggregation_CONTROL)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (resolver *nodeResolver) FailingControls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_NODE, v1.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, failing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) PassingControls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_NODE, v1.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, passing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) Controls(ctx context.Context, args rawQuery) ([]*complianceControlResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_NODE, v1.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, any, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func getComplianceControlsFromAggregationResults(results []*v1.ComplianceAggregation_Result, controlType resultType, cs complianceStandards.Repository) ([]*v1.ComplianceControl, error) {
	if cs == nil {
		return nil, errors.Errorf("empty compliance standards store encountered: argument cs is nil")
	}
	var controls []*v1.ComplianceControl
	for _, r := range results {
		if (controlType == passing && r.GetNumPassing() == 0) || (controlType == failing && r.GetNumFailing() == 0) {
			continue
		}
		controlID, err := getScopeIDFromAggregationResult(r, v1.ComplianceAggregation_CONTROL)
		if err != nil {
			continue
		}
		control := cs.Control(controlID)
		if control == nil {
			continue
		}
		controls = append(controls, control)
	}
	return controls, nil
}

type numNodeComplianceControlsResolver struct {
	numFailing int32
	numPassing int32
}

func (resolver *numNodeComplianceControlsResolver) NumFailing() int32 {
	return resolver.numFailing
}

func (resolver *numNodeComplianceControlsResolver) NumPassing() int32 {
	return resolver.numPassing
}
