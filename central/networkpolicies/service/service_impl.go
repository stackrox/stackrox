package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkpolicies/generator"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/k8sutil"
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
			"/v1.NetworkPolicyService/SimulateNetworkGraph",
			"/v1.NetworkPolicyService/GetNetworkGraph",
			"/v1.NetworkPolicyService/GetNetworkGraphEpoch",
		},
		user.With(permissions.Modify(resources.NetworkPolicy)): {
			"/v1.NetworkPolicyService/ApplyNetworkPolicy",
		},
		user.With(permissions.Modify(resources.Notifier)): {
			"/v1.NetworkPolicyService/SendNetworkPolicyYAML",
		},
		user.With(permissions.View(resources.NetworkPolicy), permissions.View(resources.NetworkGraph)): {
			"/v1.NetworkPolicyService/GenerateNetworkPolicies",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	clusterStore    clusterDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	networkPolicies networkPoliciesStore.Store
	notifierStore   notifierStore.Store
	graphEvaluator  graph.Evaluator

	policyGenerator generator.Generator
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

func populateYAML(np *storage.NetworkPolicy) {
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

func (s *serviceImpl) GetNetworkPolicy(ctx context.Context, request *v1.ResourceByID) (*storage.NetworkPolicy, error) {
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

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.GetNetworkGraphRequest) (*v1.NetworkGraph, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cluster ID must be specified")
	}

	// Check that the cluster exists. If not there is nothing to we can process.
	_, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "cluster with ID '%s' does not exist", request.GetClusterId())
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: request.GetClusterId()})
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

func (s *serviceImpl) GetNetworkGraphEpoch(context.Context, *v1.Empty) (*v1.NetworkGraphEpoch, error) {
	return &v1.NetworkGraphEpoch{
		Epoch: s.graphEvaluator.Epoch(),
	}, nil
}

func (s *serviceImpl) ApplyNetworkPolicy(ctx context.Context, request *v1.ApplyNetworkPolicyYamlRequest) (*v1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Cannot yet apply network policies")
}

func (s *serviceImpl) SimulateNetworkGraph(ctx context.Context, request *v1.SimulateNetworkGraphRequest) (*v1.SimulateNetworkGraphResponse, error) {
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
	networkPoliciesInSimulation, err := s.getNetworkPoliciesInSimulation(request.GetClusterId(), request.GetModification())
	if err != nil {
		return nil, err
	}

	// Get the deployments we want to check connectivity between.
	deployments, err := s.getDeployments(request.GetClusterId(), request.GetQuery())
	if err != nil {
		return nil, err
	}

	// Generate the base graph.
	newPolicies := make([]*storage.NetworkPolicy, 0, len(networkPoliciesInSimulation))
	oldPolicies := make([]*storage.NetworkPolicy, 0, len(networkPoliciesInSimulation))
	var hasChanges bool
	for _, policyInSim := range networkPoliciesInSimulation {
		switch policyInSim.GetStatus() {
		case v1.NetworkPolicyInSimulation_UNCHANGED:
			oldPolicies = append(oldPolicies, policyInSim.GetPolicy())
			newPolicies = append(newPolicies, policyInSim.GetPolicy())
		case v1.NetworkPolicyInSimulation_ADDED:
			newPolicies = append(newPolicies, policyInSim.GetPolicy())
			hasChanges = true
		case v1.NetworkPolicyInSimulation_MODIFIED:
			oldPolicies = append(oldPolicies, policyInSim.GetOldPolicy())
			newPolicies = append(newPolicies, policyInSim.GetPolicy())
			hasChanges = true
		case v1.NetworkPolicyInSimulation_DELETED:
			oldPolicies = append(oldPolicies, policyInSim.GetOldPolicy())
			hasChanges = true
		default:
			return nil, status.Errorf(codes.Internal, "unhandled policy status %v", policyInSim.GetStatus())
		}
	}
	newGraph := s.graphEvaluator.GetGraph(deployments, newPolicies)
	result := &v1.SimulateNetworkGraphResponse{
		SimulatedGraph: newGraph,
		Policies:       networkPoliciesInSimulation,
	}
	if !hasChanges {
		// no need to compute diff - no new policies
		return result, nil
	}

	oldGraph := s.graphEvaluator.GetGraph(deployments, oldPolicies)
	removedEdges, addedEdges, err := graph.ComputeDiff(oldGraph, newGraph)
	if err != nil {
		return nil, fmt.Errorf("could not compute a network graph diff: %v", err)
	}

	result.Removed, result.Added = removedEdges, addedEdges
	return result, nil
}

func (s *serviceImpl) SendNetworkPolicyYAML(ctx context.Context, request *v1.SendNetworkPolicyYamlRequest) (*v1.Empty, error) {
	if request.GetClusterId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Cluster ID must be specified")
	}
	if len(request.GetNotifierIds()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Notifier IDs must be specified")
	}

	cluster, exists, err := s.clusterStore.GetCluster(request.GetClusterId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve cluster: %s", err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Cluster '%s' not found", request.GetClusterId())
	}

	errorList := errorhelpers.NewErrorList("unable to use all requested notifiers")
	for _, notifierID := range request.GetNotifierIds() {
		notifierProto, exists, err := s.notifierStore.GetNotifier(notifierID)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if !exists {
			errorList.AddStringf("notifier with id:%s not found", notifierID)
			continue
		}

		notifier, err := notifiers.CreateNotifier(notifierProto)
		if err != nil {
			errorList.AddStringf("error creating notifier with id:%s (%s) and type %s: %v", notifierProto.GetId(), notifierProto.GetName(), notifierProto.GetType(), err)
			continue
		}

		err = notifier.NetworkPolicyYAMLNotify(request.GetModification().GetApplyYaml(), cluster.GetName())
		if err != nil {
			errorList.AddStringf("error sending yaml notification to %s: %v", notifierProto.GetName(), err)
		}
	}

	err = errorList.ToError()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GenerateNetworkPolicies(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (*v1.GenerateNetworkPoliciesResponse, error) {
	if s.policyGenerator == nil || !features.NetworkPolicyGenerator.Enabled() {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}

	// Default to `none` delete existing mode.
	if req.DeleteExisting == v1.GenerateNetworkPoliciesRequest_UNKNOWN {
		req.DeleteExisting = v1.GenerateNetworkPoliciesRequest_NONE
	}

	generated, toDelete, err := s.policyGenerator.Generate(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error generating network policies: %v", err)
	}

	var applyYAML string
	for _, generatedPolicy := range generated {
		yaml, err := networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: generatedPolicy}.ToYaml()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error converting generated network policy to YAML: %v", err)
		}
		if applyYAML != "" {
			applyYAML += "\n---\n"
		}
		applyYAML += yaml
	}

	mod := &v1.NetworkPolicyModification{
		ApplyYaml: applyYAML,
		ToDelete:  toDelete,
	}

	return &v1.GenerateNetworkPoliciesResponse{
		Modification: mod,
	}, nil
}

func (s *serviceImpl) getDeployments(clusterID, query string) (deployments []*storage.Deployment, err error) {
	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	q := clusterQuery
	if query != "" {
		q, err = search.ParseRawQuery(query)
		if err != nil {
			return
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	}

	deployments, err = s.deployments.SearchRawDeployments(q)
	return
}

func (s *serviceImpl) getNetworkPoliciesInSimulation(clusterID string, modification *v1.NetworkPolicyModification) ([]*v1.NetworkPolicyInSimulation, error) {
	// Confirm that any input yamls are valid. Do this check first since it is the cheapest.
	additionalPolicies, err := compileValidateYaml(modification.GetApplyYaml())
	if err != nil {
		return nil, err
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	currentPolicies, err := s.networkPolicies.GetNetworkPolicies(&v1.GetNetworkPoliciesRequest{ClusterId: clusterID})
	if err != nil {
		return nil, err
	}

	return applyPolicyModification(policyModification{
		ExistingPolicies: currentPolicies,
		NewPolicies:      additionalPolicies,
		ToDelete:         modification.GetToDelete(),
	})
}

type policyModification struct {
	ExistingPolicies []*storage.NetworkPolicy
	ToDelete         []*v1.NetworkPolicyReference
	NewPolicies      []*storage.NetworkPolicy
}

// applyPolicyModification returns the input slice of policies modified to use newPolicies.
// If oldPolicies contains a network policy with the same name and namespace as newPolicy, we consider newPolicy a
// replacement.
// If oldPolicies does not contain a network policy with a matching namespace and name, we consider it a new additional
// policy.
func applyPolicyModification(policies policyModification) (outputPolicies []*v1.NetworkPolicyInSimulation, err error) {
	outputPolicies = make([]*v1.NetworkPolicyInSimulation, 0, len(policies.NewPolicies)+len(policies.ExistingPolicies))
	policiesByRef := make(map[k8sutil.NSObjRef]*v1.NetworkPolicyInSimulation, len(policies.ExistingPolicies))
	for _, oldPolicy := range policies.ExistingPolicies {
		simPolicy := &v1.NetworkPolicyInSimulation{
			Policy: oldPolicy,
			Status: v1.NetworkPolicyInSimulation_UNCHANGED,
		}
		outputPolicies = append(outputPolicies, simPolicy)
		policiesByRef[k8sutil.RefOf(oldPolicy)] = simPolicy
	}

	// Delete policies that should be deleted
	for _, toDeleteRef := range policies.ToDelete {
		ref := k8sutil.RefOf(toDeleteRef)
		simPolicy := policiesByRef[ref]
		if simPolicy == nil {
			return nil, fmt.Errorf("policy %s in namespace %s marked for deletion does not exist", toDeleteRef.GetName(), toDeleteRef.GetNamespace())
		}

		if simPolicy.OldPolicy == nil {
			simPolicy.OldPolicy = simPolicy.Policy
		}
		simPolicy.Policy = nil
		simPolicy.Status = v1.NetworkPolicyInSimulation_DELETED
	}

	// Add new policies that have no matching old policies.
	for _, newPolicy := range policies.NewPolicies {
		oldPolicySim := policiesByRef[k8sutil.RefOf(newPolicy)]
		if oldPolicySim != nil {
			oldPolicySim.Status = v1.NetworkPolicyInSimulation_MODIFIED
			if oldPolicySim.OldPolicy == nil {
				oldPolicySim.OldPolicy = oldPolicySim.Policy
			}
			oldPolicySim.Policy = newPolicy
			continue
		}
		newPolicySim := &v1.NetworkPolicyInSimulation{
			Status: v1.NetworkPolicyInSimulation_ADDED,
			Policy: newPolicy,
		}
		outputPolicies = append(outputPolicies, newPolicySim)
	}

	// Fix IDs: For all modified policies, the ID of the new and old policies should be the same (that way the
	// diff does not get cluttered with just policy ID changes); for all new policies, we generate new, fresh UUIDs
	// that do not collide with any other IDs.
	// Rationale: IDs are (almost) meaningless - IDs from the simulation YAML will be changed by kubectl create/apply
	// anyway.
	for _, policy := range outputPolicies {
		if policy.GetStatus() == v1.NetworkPolicyInSimulation_MODIFIED {
			policy.Policy.Id = policy.GetOldPolicy().GetId()
		} else if policy.GetStatus() == v1.NetworkPolicyInSimulation_ADDED {
			policy.Policy.Id = uuid.NewV4().String()
		}
	}
	return
}

// compileValidateYaml compiles the YAML into a storage.NetworkPolicy, and verifies that a valid namespace exists.
func compileValidateYaml(simulationYaml string) ([]*storage.NetworkPolicy, error) {
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

	return policies, nil
}
