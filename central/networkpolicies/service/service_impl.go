package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/notifiers"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkPolicy)): {
			"/v1.NetworkPolicyService/GetNetworkPolicy",
			"/v1.NetworkPolicyService/GetNetworkPolicies",
			"/v1.NetworkPolicyService/GetNetworkGraph",
			"/v1.NetworkPolicyService/GetNetworkGraphEpoch",
		},
		user.With(permissions.Modify(resources.Notifier)): {
			"/v1.NetworkPolicyService/SendNetworkPolicyYaml",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterStore    clusterDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
	notifierStore   notifierStore.Store
	graphEvaluator  networkgraph.Evaluator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNetworkPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNetworkPolicyServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func populateYAML(np *v1.NetworkPolicy) {
	k8sNetworkPolicy := networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: np}.ToKubernetesNetworkPolicy()
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	stringBuilder := &strings.Builder{}
	err := encoder.Encode(k8sNetworkPolicy, stringBuilder)
	if err != nil {
		np.Yaml = fmt.Sprintf("Could not render Network Policy YAML: %s", err)
		return
	}
	np.Yaml = stringBuilder.String()
}

func (s *serviceImpl) GetNetworkPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.NetworkPolicy, error) {
	networkPolicy, exists, err := s.networkPolicies.GetNetworkPolicy(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "network policy with id '%s' does not exist", request.GetId())
	}
	populateYAML(networkPolicy)
	return networkPolicy, nil
}

func (s *serviceImpl) GetNetworkPolicies(ctx context.Context, request *v1.GetNetworkPoliciesRequest) (*v1.NetworkPoliciesResponse, error) {
	if request.GetClusterId() != "" {
		_, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !exists {
			return nil, status.Errorf(codes.InvalidArgument, "cluster with id '%s' doesn't exist", request.GetClusterId())
		}
	}
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(request)
	if err != nil {
		return nil, err
	}
	return &v1.NetworkPoliciesResponse{
		NetworkPolicies: networkPolicies,
	}, nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.GetNetworkGraphRequest) (*v1.GetNetworkGraphResponse, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster ID must be specified")
	}

	// Check that the cluster exists. If not there is nothing to we can process.
	_, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster with ID '%s' does not exist", request.GetClusterId())
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	networkPolicies, err := s.getNetworkPolicies(request.GetClusterId(), request.GetSimulationYaml())
	if err != nil {
		return nil, err
	}

	// Get the deployments we want to check connectivity between.
	deployments, err := s.getDeployments(request.GetClusterId(), request.GetQuery())
	if err != nil {
		return nil, err
	}

	// Generate the graph.
	return s.graphEvaluator.GetGraph(deployments, networkPolicies), nil
}

func (s *serviceImpl) GetNetworkGraphEpoch(context.Context, *v1.Empty) (*v1.GetNetworkGraphEpochResponse, error) {
	return &v1.GetNetworkGraphEpochResponse{
		Epoch: s.graphEvaluator.Epoch(),
	}, nil
}

func (s *serviceImpl) SendNetworkPolicyYAML(ctx context.Context, request *v1.SendNetworkPolicyYamlRequest) (*v1.Empty, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster ID must be specified")
	}

	cluster, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
	if !exists {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	notifierProto, exists, err := s.notifierStore.GetNotifier(request.GetNotifierId())

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Notifier %s not found", request.GetNotifierId()))
	}

	notifier, err := notifiers.CreateNotifier(notifierProto)

	if err != nil {
		return &v1.Empty{}, fmt.Errorf("Error creating notifier with %s (%s) and type %s: %v", notifierProto.GetId(), notifierProto.GetName(), notifierProto.GetType(), err)
	}

	err = notifier.NetworkPolicyYAMLNotify(request.GetYaml(), cluster.GetName())
	if err != nil {
		return &v1.Empty{}, status.Errorf(codes.Internal, fmt.Sprintf("Error sending yaml notification to %s: %v", notifierProto.GetName(), err))
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) getDeployments(clusterID, query string) (deployments []*v1.Deployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()

	var q *v1.Query
	if query != "" {
		q, err = search.ParseRawQuery(query)
		if err != nil {
			return
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	} else {
		q = clusterQuery
	}

	deployments, err = s.deployments.SearchRawDeployments(q)
	return
}

func (s *serviceImpl) getNetworkPolicies(clusterID, simulationYaml string) ([]*v1.NetworkPolicy, error) {
	// Confirm that any input yamls are valid. Do this check first since it is the cheapest.
	additionalPolicies, err := compileValidateYaml(simulationYaml)
	if err != nil {
		return nil, err
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	currentPolicies, err := s.networkPolicies.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: clusterID})
	if err != nil {
		return nil, err
	}

	return replaceOrAddPolicies(additionalPolicies, currentPolicies), nil
}

// replaceOrAddPolicies returns the input slice of policies modified to use newPolicies.
// If oldPolicies contains a network policy with the same name and namespace as newPolicy, we consider newPolicy a
// replacement.
// If oldPolicies does not contain a network policy with a matching namespace and name, we consider it a new additional
// policy.
func replaceOrAddPolicies(newPolicies []*v1.NetworkPolicy, oldPolicies []*v1.NetworkPolicy) (outputPolicies []*v1.NetworkPolicy) {
	if len(newPolicies) == 0 {
		outputPolicies = oldPolicies
		return
	}

	// Add old policies without matching new policies, and new policies that override old policies.
	outputPolicies = make([]*v1.NetworkPolicy, 0, len(oldPolicies))
	for _, oldPolicy := range oldPolicies {
		match := getMatch(newPolicies, oldPolicy)
		if match != nil {
			outputPolicies = append(outputPolicies, match)
		} else {
			outputPolicies = append(outputPolicies, oldPolicy)
		}
	}

	// Add new policies that have no matching old policies.
	for _, newPolicy := range newPolicies {
		match := getMatch(oldPolicies, newPolicy)
		if match == nil {
			outputPolicies = append(outputPolicies, newPolicy)
		}
	}
	return
}

// getMatch finds a matching policy in matchFrom that matches matchTo
func getMatch(matchFrom []*v1.NetworkPolicy, matchTo *v1.NetworkPolicy) *v1.NetworkPolicy {
	for _, match := range matchFrom {
		if matchTo.GetName() == match.GetName() && matchTo.GetNamespace() == match.GetNamespace() {
			return match
		}
	}
	return nil
}

// compileValidateYaml compiles the YAML into a v1.NetworkPolicy, and verifies that a valid namespace exists.
func compileValidateYaml(simulationYaml string) ([]*v1.NetworkPolicy, error) {
	if simulationYaml == "" {
		return nil, nil
	}

	simulationYaml = strings.TrimPrefix(simulationYaml, "---\n")

	// Convert the YAMLs into rox network policy objects.
	policies, err := networkPolicyConversion.YamlWrap{Yaml: simulationYaml}.ToRoxNetworkPolicies()
	if err != nil {
		return nil, err
	}

	// Check that all resulting policies have namespaces.
	for _, policy := range policies {
		if policy.GetNamespace() == "" {
			return nil, fmt.Errorf("yamls tested against must apply to a namespace")
		}
	}

	// Ensure that all resulting policies have IDs.
	for _, policy := range policies {
		if policy.GetId() == "" {
			policy.Id = uuid.NewV4().String()
		}
	}

	return policies, nil
}
