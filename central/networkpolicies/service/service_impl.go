package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	networkBaselineDataStore "github.com/stackrox/rox/central/networkbaseline/datastore"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/networkpolicies/generator"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	networkPolicyConversion "github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkPolicy)): {
			v1.NetworkPolicyService_GetNetworkPolicy_FullMethodName,
			v1.NetworkPolicyService_GetNetworkPolicies_FullMethodName,
			v1.NetworkPolicyService_SimulateNetworkGraph_FullMethodName,
			v1.NetworkPolicyService_GetNetworkGraph_FullMethodName,
			v1.NetworkPolicyService_GetNetworkGraphEpoch_FullMethodName,
			v1.NetworkPolicyService_GetUndoModification_FullMethodName,
			v1.NetworkPolicyService_GetAllowedPeersFromCurrentPolicyForDeployment_FullMethodName,
			v1.NetworkPolicyService_GetDiffFlowsBetweenPolicyAndBaselineForDeployment_FullMethodName,
			v1.NetworkPolicyService_GetUndoModificationForDeployment_FullMethodName,
			v1.NetworkPolicyService_GetDiffFlowsFromUndoModificationForDeployment_FullMethodName,
		},
		user.With(permissions.Modify(resources.NetworkPolicy)): {
			v1.NetworkPolicyService_ApplyNetworkPolicy_FullMethodName,
			v1.NetworkPolicyService_ApplyNetworkPolicyYamlForDeployment_FullMethodName,
		},
		user.With(permissions.Modify(resources.Integration)): {
			v1.NetworkPolicyService_SendNetworkPolicyYAML_FullMethodName,
		},
		user.With(permissions.View(resources.NetworkPolicy), permissions.View(resources.NetworkGraph)): {
			v1.NetworkPolicyService_GenerateNetworkPolicies_FullMethodName,
		},
		user.With(permissions.View(resources.NetworkPolicy), permissions.View(resources.DeploymentExtension)): {
			v1.NetworkPolicyService_GetBaselineGeneratedNetworkPolicyForDeployment_FullMethodName,
		},
	})

	deploymentPredicateFactory = predicate.NewFactory("deployment", &storage.Deployment{})

	networkPolicySAC = sac.ForResource(resources.NetworkPolicy)
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v1.UnimplementedNetworkPolicyServiceServer

	sensorConnMgr    connection.Manager
	clusterStore     clusterDataStore.DataStore
	deployments      deploymentDataStore.DataStore
	externalSrcs     networkEntityDS.EntityDataStore
	graphConfig      graphConfigDS.DataStore
	networkBaselines networkBaselineDataStore.ReadOnlyDataStore
	networkTreeMgr   networktree.Manager
	networkPolicies  npDS.DataStore
	notifierStore    notifierDataStore.DataStore
	graphEvaluator   graph.Evaluator

	clusterSACHelper sachelper.ClusterSacHelper

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
	yaml, err := networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: np}.ToYaml()
	if err != nil {
		np.SetYaml(fmt.Sprintf("Could not render Network Policy YAML: %s", err))
		return
	}
	np.SetYaml(yaml)
}

func (s *serviceImpl) GetNetworkPolicy(ctx context.Context, request *v1.ResourceByID) (*storage.NetworkPolicy, error) {
	networkPolicy, exists, err := s.networkPolicies.GetNetworkPolicy(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "network policy with id '%s' does not exist", request.GetId())
	}
	populateYAML(networkPolicy)
	return networkPolicy, nil
}

func (s *serviceImpl) GetNetworkPolicies(ctx context.Context, request *v1.GetNetworkPoliciesRequest) (*v1.NetworkPoliciesResponse, error) {
	// Check the cluster information.
	if err := s.clusterExists(ctx, request.GetClusterId()); err != nil {
		return nil, err
	}

	// Get the policies in the cluster
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, request.GetClusterId(), request.GetNamespace())
	if err != nil {
		return nil, err
	}

	// If there is a deployment query, filter the policies that apply to the deployments that match the query.
	if request.GetDeploymentQuery() != "" {
		// Get the deployments we want to check connectivity between.
		queryDeployments, err := s.getQueryDeployments(ctx, request.GetClusterId(), request.GetDeploymentQuery())
		if err != nil {
			return nil, err
		}

		networkTree, err := s.getNetworkTree(request.GetClusterId())
		if err != nil {
			return nil, errors.Errorf("unable to get network tree for cluster %s: %v", request.GetClusterId(), err)
		}
		networkPolicies = s.graphEvaluator.GetAppliedPolicies(queryDeployments, networkTree, networkPolicies)
	}

	// Fill in YAML fields where they are not set.
	for _, np := range networkPolicies {
		np.Yaml, err = networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: np}.ToYaml()
		if err != nil {
			return nil, err
		}
	}

	// Get the policies that apply to the fetched deployments.
	npr := &v1.NetworkPoliciesResponse{}
	npr.SetNetworkPolicies(networkPolicies)
	return npr, nil
}

func (s *serviceImpl) GetNetworkGraph(ctx context.Context, request *v1.GetNetworkGraphRequest) (*v1.NetworkGraph, error) {
	// Check that the cluster exists. If not there is nothing to we can process.
	if err := s.clusterExists(ctx, request.GetClusterId()); err != nil {
		return nil, err
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, request.GetClusterId(), "")
	if err != nil {
		return nil, err
	}

	// Get the deployments we want to check connectivity between.
	queryDeploymentIDs, clusterDeployments, err := s.getDeployments(ctx, request.GetClusterId(), request.GetQuery(), request.GetScope())
	if err != nil {
		return nil, err
	}

	networkTree, err := s.getNetworkTree(request.GetClusterId())
	if err != nil {
		return nil, errors.Errorf("unable to get network tree for cluster %s: %v", request.GetClusterId(), err)
	}
	// Generate the graph.
	return s.graphEvaluator.GetGraph(request.GetClusterId(), queryDeploymentIDs, clusterDeployments, networkTree, networkPolicies, request.GetIncludePorts()), nil
}

func (s *serviceImpl) GetNetworkGraphEpoch(_ context.Context, req *v1.GetNetworkGraphEpochRequest) (*v1.NetworkGraphEpoch, error) {
	nge := &v1.NetworkGraphEpoch{}
	nge.SetEpoch(s.graphEvaluator.Epoch(req.GetClusterId()))
	return nge, nil
}

func (s *serviceImpl) ApplyNetworkPolicy(ctx context.Context, request *v1.ApplyNetworkPolicyYamlRequest) (*v1.Empty, error) {
	undoRecord, err := s.applyModificationAndGetUndoRecord(ctx, request.GetClusterId(), request.GetModification())
	if err != nil {
		return nil, err
	}
	undoRecord.SetClusterId(request.GetClusterId())

	err = s.networkPolicies.UpsertUndoRecord(ctx, undoRecord)
	if err != nil {
		return nil, errors.Errorf("network policy was applied, but undo record could not be stored: %v", err)
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) SimulateNetworkGraph(ctx context.Context, request *v1.SimulateNetworkGraphRequest) (*v1.SimulateNetworkGraphResponse, error) {
	// Check that the cluster exists. If not there is nothing to we can process.
	if err := s.clusterExists(ctx, request.GetClusterId()); err != nil {
		return nil, err
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	networkPoliciesInSimulation, err := s.getNetworkPoliciesInSimulation(ctx, request.GetClusterId(), request.GetModification())
	if err != nil {
		return nil, err
	}

	// Confirm that network policies in restricted namespaces are not changed
	err = validateNoForbiddenModification(networkPoliciesInSimulation)
	if err != nil {
		return nil, err
	}

	// Get the deployments we want to check connectivity between.
	queryDeploymentIDs, clusterDeployments, err := s.getDeployments(ctx, request.GetClusterId(), request.GetQuery(), request.GetScope())
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
			return nil, errors.Errorf("unhandled policy status %v", policyInSim.GetStatus())
		}
	}

	networkTree, err := s.getNetworkTree(request.GetClusterId())
	if err != nil {
		return nil, errors.Errorf("unable to get network tree for cluster %s: %v", request.GetClusterId(), err)
	}

	newGraph := s.graphEvaluator.GetGraph(request.GetClusterId(), queryDeploymentIDs, clusterDeployments, networkTree, newPolicies, request.GetIncludePorts())
	result := &v1.SimulateNetworkGraphResponse{}
	result.SetSimulatedGraph(newGraph)
	result.SetPolicies(networkPoliciesInSimulation)
	if !hasChanges {
		// no need to compute diff - no new policies
		return result, nil
	}

	if !request.GetIncludeNodeDiff() {
		return result, nil
	}

	oldGraph := s.graphEvaluator.GetGraph(request.GetClusterId(), queryDeploymentIDs, clusterDeployments, networkTree, oldPolicies, request.GetIncludePorts())
	removedEdges, addedEdges, err := graph.ComputeDiff(oldGraph, newGraph)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute a network graph diff")
	}

	result.SetRemoved(removedEdges)
	result.SetAdded(addedEdges)
	return result, nil
}

func (s *serviceImpl) SendNetworkPolicyYAML(ctx context.Context, request *v1.SendNetworkPolicyYamlRequest) (*v1.Empty, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID must be specified")
	}
	if len(request.GetNotifierIds()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "Notifier IDs must be specified")
	}
	if request.GetModification().GetApplyYaml() == "" && len(request.GetModification().GetToDelete()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "Modification must have contents")
	}

	clusterName, exists, err := s.clusterStore.GetClusterName(ctx, request.GetClusterId())
	if err != nil {
		return nil, errors.Errorf("failed to retrieve cluster: %s", err.Error())
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "Cluster '%s' not found", request.GetClusterId())
	}

	errorList := errorhelpers.NewErrorList("unable to use all requested notifiers")
	for _, notifierID := range request.GetNotifierIds() {
		notifierProto, exists, err := s.notifierStore.GetNotifier(ctx, notifierID)
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
		netpolNotifier, ok := notifier.(notifiers.NetworkPolicyNotifier)
		if !ok {
			errorList.AddStringf("notifier %s cannot notify on network policies", notifierProto.GetName())
			continue
		}

		err = netpolNotifier.NetworkPolicyYAMLNotify(ctx, request.GetModification().GetApplyYaml(), clusterName)
		if err != nil {
			errorList.AddStringf("error sending yaml notification to %s: %v", notifierProto.GetName(), err)
		}
	}

	err = errorList.ToError()
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GenerateNetworkPolicies(ctx context.Context, req *v1.GenerateNetworkPoliciesRequest) (*v1.GenerateNetworkPoliciesResponse, error) {
	// Default to `none` delete existing mode.
	if req.GetDeleteExisting() == v1.GenerateNetworkPoliciesRequest_UNKNOWN {
		req.SetDeleteExisting(v1.GenerateNetworkPoliciesRequest_NONE)
	}

	if req.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID must be specified")
	}

	generated, toDelete, err := s.policyGenerator.Generate(ctx, req)
	if err != nil {
		return nil, errors.Errorf("error generating network policies: %v", err)
	}

	applyYAML, err := s.generateApplyYamlFromGeneratedPolicies(generated)
	if err != nil {
		return nil, err
	}

	mod := &storage.NetworkPolicyModification{}
	mod.SetApplyYaml(applyYAML)
	mod.SetToDelete(toDelete)

	gnpr := &v1.GenerateNetworkPoliciesResponse{}
	gnpr.SetModification(mod)
	return gnpr, nil
}

func (s *serviceImpl) GetUndoModification(ctx context.Context, req *v1.GetUndoModificationRequest) (*v1.GetUndoModificationResponse, error) {
	undoRecord, exists, err := s.networkPolicies.GetUndoRecord(ctx, req.GetClusterId())
	if err != nil {
		return nil, errors.Errorf("could not query undo store: %v", err)
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "no undo record stored for cluster %q", req.GetClusterId())
	}
	gumr := &v1.GetUndoModificationResponse{}
	gumr.SetUndoRecord(undoRecord)
	return gumr, nil
}

func (s *serviceImpl) generateApplyYamlFromGeneratedPolicies(generatedPolicies []*storage.NetworkPolicy) (string, error) {
	var applyYAML string
	for _, policy := range generatedPolicies {
		yaml, err := networkPolicyConversion.RoxNetworkPolicyWrap{NetworkPolicy: policy}.ToYaml()
		if err != nil {
			return "", errors.Errorf("error converting generated network policy to YAML: %v", err)
		}
		if applyYAML != "" {
			applyYAML += "\n---\n"
		}
		applyYAML += yaml
	}
	return applyYAML, nil
}

func (s *serviceImpl) GetBaselineGeneratedNetworkPolicyForDeployment(ctx context.Context, request *v1.GetBaselineGeneratedPolicyForDeploymentRequest) (*v1.GetBaselineGeneratedPolicyForDeploymentResponse, error) {
	// Currently we don't look at request.GetDeleteExisting. We try to delete the existing baseline generated
	// policy no matter what
	if request.GetDeploymentId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Cluster ID must be specified")
	}

	generated, toDelete, err := s.policyGenerator.GenerateFromBaselineForDeployment(ctx, request)
	if err != nil {
		return nil, errors.Errorf("error generating network policies: %v", err)
	}

	applyYAML, err := s.generateApplyYamlFromGeneratedPolicies(generated)
	if err != nil {
		return nil, err
	}

	npm := &storage.NetworkPolicyModification{}
	npm.SetApplyYaml(applyYAML)
	npm.SetToDelete(toDelete)
	gbgpfdr := &v1.GetBaselineGeneratedPolicyForDeploymentResponse{}
	gbgpfdr.SetModification(npm)
	return gbgpfdr, nil
}

func (s *serviceImpl) GetAllowedPeersFromCurrentPolicyForDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.GetAllowedPeersFromCurrentPolicyForDeploymentResponse, error) {
	dep, networkTree, deploymentsInCluster, err := s.getRelevantClusterObjectsForDeployment(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	allowedPeers, err := s.getAllowedPeersForDeployment(ctx, dep, networkTree, deploymentsInCluster)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetAllowedPeersFromCurrentPolicyForDeploymentResponse{}
	for _, p := range allowedPeers {
		entity := p.entity
		for _, prop := range p.properties {
			nbpe := &v1.NetworkBaselinePeerEntity{}
			nbpe.SetId(entity.GetId())
			nbpe.SetType(entity.GetType())
			nbsp := &v1.NetworkBaselineStatusPeer{}
			nbsp.SetEntity(nbpe)
			nbsp.SetPort(prop.GetPort())
			nbsp.SetProtocol(prop.GetProtocol())
			nbsp.SetIngress(prop.GetIngress())
			resp.SetAllowedPeers(append(resp.GetAllowedPeers(), nbsp))
		}
	}
	return resp, nil
}

func (s *serviceImpl) getRelevantClusterObjectsForDeployment(ctx context.Context, deploymentID string) (*storage.Deployment,
	tree.ReadOnlyNetworkTree, []*storage.Deployment, error) {
	// Get the deployment
	deployment, found, err := s.deployments.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, nil, nil, err
	} else if !found {
		return nil, nil, nil, errors.Wrap(errox.InvalidArgs, "specified deployment not found")
	}

	networkTree, err := s.getNetworkTree(deployment.GetClusterId())
	if err != nil {
		return nil, nil, nil, err
	}
	_, deploymentsInCluster, err := s.getDeployments(ctx, deployment.GetClusterId(), "", nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return deployment, networkTree, deploymentsInCluster, nil
}

func (s *serviceImpl) getAllowedPeersForDeployment(ctx context.Context, deployment *storage.Deployment,
	networkTree tree.ReadOnlyNetworkTree, deploymentsInCluster []*storage.Deployment) (
	groupedEntitiesWithProperties, error) {
	// Get the policies in the cluster
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, deployment.GetClusterId(), "")
	if err != nil {
		return nil, err
	}
	return s.getAllowedPeersForDeploymentWithNetPols(deployment, networkTree, deploymentsInCluster, networkPolicies)
}

func (s *serviceImpl) getAllowedPeersForDeploymentWithNetPols(deployment *storage.Deployment,
	networkTree tree.ReadOnlyNetworkTree, deploymentsInCluster []*storage.Deployment, networkPolicies []*storage.NetworkPolicy) (
	groupedEntitiesWithProperties, error) {

	// Only get the network policies that are applied to the deployment
	networkPolicies = s.graphEvaluator.GetAppliedPolicies([]*storage.Deployment{deployment}, networkTree, networkPolicies)
	// Build a graph out of the network policies. We can later remove all the deployments/nodes that do not have any out
	// edge to the deployment we want
	graphWithNetPols :=
		s.graphEvaluator.GetGraph(
			deployment.GetClusterId(),
			set.NewStringSet(deployment.GetId()),
			deploymentsInCluster,
			networkTree,
			networkPolicies,
			true)
	allowedPeers, err := s.getPeersOfDeploymentFromGraph(deployment, graphWithNetPols)
	if err != nil {
		return nil, err
	}
	return allowedPeers, nil
}

type groupedEntitiesWithProperties map[string]*entityWithProperties

func (g groupedEntitiesWithProperties) addProperty(entity *storage.NetworkEntityInfo, property *storage.NetworkBaselineConnectionProperties) {
	entry := g[entity.GetId()]
	if entry == nil {
		entry = &entityWithProperties{entity: entity}
		g[entity.GetId()] = entry
	}
	entry.properties = append(entry.properties, property)
}

type entityWithProperties struct {
	entity     *storage.NetworkEntityInfo
	properties []*storage.NetworkBaselineConnectionProperties
}

func (s *serviceImpl) getPeersOfDeploymentFromGraph(deployment *storage.Deployment, graph *v1.NetworkGraph) (groupedEntitiesWithProperties, error) {
	allowedPeers := make(groupedEntitiesWithProperties)
	// Try to search for the deployment in question
	deploymentIdx := -1
	var deploymentIngressNonIsolated, deploymentEgressNonIsolated bool
	for idx, node := range graph.GetNodes() {
		// The deployment we want is passed in as part of queryDeploymentIDs during getGraph for graph generation
		if !node.GetQueryMatch() {
			continue
		}
		// we are looking at the node which is our deployment. Gather all the egress edges here
		for egressPeerIdx := range node.GetOutEdges() {
			egressPeer := graph.GetNodes()[egressPeerIdx]
			for _, prop := range node.GetOutEdges()[egressPeerIdx].GetProperties() {
				nbcp := &storage.NetworkBaselineConnectionProperties{}
				nbcp.SetIngress(false)
				nbcp.SetPort(prop.GetPort())
				nbcp.SetProtocol(prop.GetProtocol())
				allowedPeers.addProperty(egressPeer.GetEntity(), nbcp)
			}
		}
		// Record the idx
		deploymentIdx = idx
		// Check if the deployment is isolated or not
		deploymentIngressNonIsolated = node.GetNonIsolatedIngress()
		deploymentEgressNonIsolated = node.GetNonIsolatedEgress()
		break
	}
	if deploymentIdx == -1 {
		return nil, errors.Errorf("deployment %q not found in the generated graph", deployment.GetName())
	}
	for _, node := range graph.GetNodes() {
		if node.GetQueryMatch() {
			continue
		}
		// If the peer node is non-isolated, we should add a wildcard flow to result
		if deploymentIngressNonIsolated && node.GetNonIsolatedEgress() {
			entry := allowedPeers[node.GetEntity().GetId()]
			if entry == nil {
				entry = &entityWithProperties{entity: node.GetEntity()}
				allowedPeers[node.GetEntity().GetId()] = entry
			}
			nbcp := &storage.NetworkBaselineConnectionProperties{}
			nbcp.SetIngress(true)
			nbcp.SetPort(0)
			nbcp.SetProtocol(storage.L4Protocol_L4_PROTOCOL_ANY)
			allowedPeers.addProperty(node.GetEntity(), nbcp)
		}
		if deploymentEgressNonIsolated && node.GetNonIsolatedIngress() {
			nbcp := &storage.NetworkBaselineConnectionProperties{}
			nbcp.SetIngress(false)
			nbcp.SetPort(0)
			nbcp.SetProtocol(storage.L4Protocol_L4_PROTOCOL_ANY)
			allowedPeers.addProperty(node.GetEntity(), nbcp)
		}

		// We should try to fill in ingress info for the deployment from this node.
		props, ok := node.GetOutEdges()[int32(deploymentIdx)]
		if !ok {
			continue
		}
		for _, prop := range props.GetProperties() {
			nbcp := &storage.NetworkBaselineConnectionProperties{}
			nbcp.SetIngress(true)
			nbcp.SetPort(prop.GetPort())
			nbcp.SetProtocol(prop.GetProtocol())
			allowedPeers.addProperty(node.GetEntity(), nbcp)
		}
	}
	return allowedPeers, nil
}

func (s *serviceImpl) applyModificationAndGetUndoRecord(
	ctx context.Context,
	clusterID string,
	modification *storage.NetworkPolicyModification,
) (*storage.NetworkPolicyApplicationUndoRecord, error) {
	if strings.TrimSpace(modification.GetApplyYaml()) == "" && len(modification.GetToDelete()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "Modification must have contents")
	}

	// Check that:
	// - all network policies can be parsed
	// - all network policies have a non-empty namespace field
	// - the user has write access to all namespaces where the application takes place
	if nsSet, err := getNamespacesFromModification(modification); err != nil {
		return nil, errors.Wrap(err, "failed to determine network policy namespaces")
	} else if nsSet.Contains("") {
		return nil, status.Error(codes.InvalidArgument, "network policy has empty namespace")
	} else if err := checkAllNamespacesWriteAllowed(ctx, clusterID, nsSet.AsSlice()...); err != nil {
		return nil, err
	}

	conn := s.sensorConnMgr.GetConnection(clusterID)
	if conn == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "no active connection to cluster %q", clusterID)
	}

	undoMod, err := conn.NetworkPolicies().ApplyNetworkPolicies(ctx, modification)
	if err != nil {
		return nil, errors.Errorf("could not apply network policy modification: %v", err)
	}

	var user string
	identity := authn.IdentityFromContextOrNil(ctx)
	if identity != nil {
		user = identity.FriendlyName()
		if ap := identity.ExternalAuthProvider(); ap != nil {
			user += fmt.Sprintf(" [%s]", ap.Name())
		}
	}
	undoRecord := &storage.NetworkPolicyApplicationUndoRecord{}
	undoRecord.SetUser(user)
	undoRecord.SetApplyTimestamp(protocompat.TimestampNow())
	undoRecord.SetOriginalModification(modification)
	undoRecord.SetUndoModification(undoMod)
	return undoRecord, nil
}

func (s *serviceImpl) ApplyNetworkPolicyYamlForDeployment(ctx context.Context, request *v1.ApplyNetworkPolicyYamlForDeploymentRequest) (*v1.Empty, error) {
	// Get the deployment
	deployment, found, err := s.deployments.GetDeployment(ctx, request.GetDeploymentId())
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errors.Wrapf(errox.NotFound, "requested deployment %q not found", request.GetDeploymentId())
	}

	undoRecord, err := s.applyModificationAndGetUndoRecord(ctx, deployment.GetClusterId(), request.GetModification())
	if err != nil {
		return nil, err
	}

	npaudr := &storage.NetworkPolicyApplicationUndoDeploymentRecord{}
	npaudr.SetDeploymentId(request.GetDeploymentId())
	npaudr.SetUndoRecord(undoRecord)
	err = s.networkPolicies.UpsertUndoDeploymentRecord(
		ctx, npaudr)
	if err != nil {
		return nil, errors.Errorf("network policy was applied, but undo deployment record could not be stored: %v", err)
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GetUndoModificationForDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.GetUndoModificationForDeploymentResponse, error) {
	// Try getting the deployment first
	_, found, err := s.deployments.GetDeployment(ctx, request.GetId())
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errors.Wrapf(errox.NotFound, "deployment with ID %q not found", request.GetId())
	}

	undoRecord, found, err := s.networkPolicies.GetUndoDeploymentRecord(ctx, request.GetId())
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errors.Wrapf(errox.NotFound, "no undo record stored for deployment %q", request.GetId())
	}
	gumfdr := &v1.GetUndoModificationForDeploymentResponse{}
	gumfdr.SetUndoRecord(undoRecord.GetUndoRecord())
	return gumfdr, nil
}

type nameNSPair struct {
	name      string
	namespace string
}

func (s *serviceImpl) GetDiffFlowsFromUndoModificationForDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.GetDiffFlowsResponse, error) {
	// First get allowed peers from current network policies
	dep, networkTree, deploymentsInCluster, err := s.getRelevantClusterObjectsForDeployment(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	currentAllowedPeers, err := s.getAllowedPeersForDeployment(ctx, dep, networkTree, deploymentsInCluster)
	if err != nil {
		return nil, err
	}

	undoRecord, found, err := s.networkPolicies.GetUndoDeploymentRecord(ctx, request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if !found {
		return nil, status.Errorf(codes.NotFound, "no undo record stored for deployment %q", request.GetId())
	}

	undoModification := undoRecord.GetUndoRecord().GetUndoModification()
	// Get the policies in the cluster
	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, dep.GetClusterId(), "")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	conflictingNetPols := make(map[nameNSPair]struct{})
	for _, toDelete := range undoModification.GetToDelete() {
		conflictingNetPols[nameNSPair{name: toDelete.GetName(), namespace: toDelete.GetNamespace()}] = struct{}{}
	}

	policiesViaUndo, err := compileValidateYaml(undoModification.GetApplyYaml())
	if err != nil {
		return nil, err
	}
	for _, p := range policiesViaUndo {
		conflictingNetPols[nameNSPair{name: p.GetName(), namespace: p.GetName()}] = struct{}{}
	}

	networkPoliciesPostUndo := policiesViaUndo
	for _, netPol := range networkPolicies {
		if _, isConflicting := conflictingNetPols[nameNSPair{name: netPol.GetName(), namespace: netPol.GetNamespace()}]; !isConflicting {
			networkPoliciesPostUndo = append(networkPoliciesPostUndo, netPol)
		}
	}

	allowedPeersPostUndo, err := s.getAllowedPeersForDeploymentWithNetPols(dep, networkTree, deploymentsInCluster, networkPoliciesPostUndo)
	if err != nil {
		return nil, err
	}
	return s.computeDiffBetweenPeerGroups(currentAllowedPeers, allowedPeersPostUndo), nil
}

func (s *serviceImpl) GetDiffFlowsBetweenPolicyAndBaselineForDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.GetDiffFlowsResponse, error) {
	// First get allowed peers from current network policies
	dep, networkTree, deploymentsInCluster, err := s.getRelevantClusterObjectsForDeployment(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	currentAllowedPeers, err := s.getAllowedPeersForDeployment(ctx, dep, networkTree, deploymentsInCluster)
	if err != nil {
		return nil, err
	}

	gbgpfdr := &v1.GetBaselineGeneratedPolicyForDeploymentRequest{}
	gbgpfdr.SetDeploymentId(request.GetId())
	gbgpfdr.SetIncludePorts(true)
	generated, toDelete, err := s.policyGenerator.GenerateFromBaselineForDeployment(ctx, gbgpfdr)
	if err != nil {
		return nil, errors.Errorf("error generating network policies: %v", err)
	}

	networkPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, dep.GetClusterId(), "")
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	conflictingNetPols := make(map[nameNSPair]struct{})
	for _, toDel := range toDelete {
		conflictingNetPols[nameNSPair{name: toDel.GetName(), namespace: toDel.GetNamespace()}] = struct{}{}
	}

	for _, p := range generated {
		conflictingNetPols[nameNSPair{name: p.GetName(), namespace: p.GetName()}] = struct{}{}
	}

	networkPoliciesPostGeneration := generated
	for _, netPol := range networkPolicies {
		if _, isConflicting := conflictingNetPols[nameNSPair{name: netPol.GetName(), namespace: netPol.GetNamespace()}]; !isConflicting {
			networkPoliciesPostGeneration = append(networkPoliciesPostGeneration, netPol)
		}
	}

	allowedPeersPostGeneration, err := s.getAllowedPeersForDeploymentWithNetPols(dep, networkTree, deploymentsInCluster, networkPoliciesPostGeneration)
	if err != nil {
		return nil, err
	}
	return s.computeDiffBetweenPeerGroups(currentAllowedPeers, allowedPeersPostGeneration), nil
}

func (s *serviceImpl) computeDiffBetweenPeerGroups(
	previousPeers, currentPeers groupedEntitiesWithProperties,
) *v1.GetDiffFlowsResponse {
	rsp := &v1.GetDiffFlowsResponse{}
	for _, currentPeer := range currentPeers {
		entity := currentPeer.entity
		if previousPeer, ok := previousPeers[entity.GetId()]; !ok {
			// Previous peer not found in the list of current peers. This is a newly added peer
			gdfgf := &v1.GetDiffFlowsGroupedFlow{}
			gdfgf.SetEntity(currentPeer.entity)
			gdfgf.SetProperties(currentPeer.properties)
			rsp.SetAdded(append(rsp.GetAdded(), gdfgf))
		} else {
			// A new set of flows might be configured for this entity. Reconcile the difference if there is any
			rsp.SetReconciled(append(rsp.GetReconciled(), s.reconcileFlowDifferences(entity, previousPeer.properties, currentPeer.properties)))
			delete(previousPeers, entity.GetId())
		}
	}

	// Since we have deleted matched peers from the previous peers map, the peers left
	// are removed in the diff.
	for _, previousPeer := range previousPeers {
		gdfgf := &v1.GetDiffFlowsGroupedFlow{}
		gdfgf.SetEntity(previousPeer.entity)
		gdfgf.SetProperties(previousPeer.properties)
		rsp.SetRemoved(append(rsp.GetRemoved(), gdfgf))
	}

	return rsp
}

type connectionProperties struct {
	ingress  bool
	port     uint32
	protocol storage.L4Protocol
}

func (s *serviceImpl) toConnectionPropertiesStruct(properties *storage.NetworkBaselineConnectionProperties) connectionProperties {
	return connectionProperties{
		ingress:  properties.GetIngress(),
		port:     properties.GetPort(),
		protocol: properties.GetProtocol(),
	}
}

func (s *serviceImpl) reconcileFlowDifferences(entity *storage.NetworkEntityInfo, allowedProperties,
	baselineProperties []*storage.NetworkBaselineConnectionProperties) *v1.GetDiffFlowsReconciledFlow {
	result := &v1.GetDiffFlowsReconciledFlow{}
	result.SetEntity(entity)
	// Convert allowedProperties to set for easy lookup
	allowedPropertiesSet := make(map[connectionProperties]struct{})
	for _, properties := range allowedProperties {
		allowedPropertiesSet[s.toConnectionPropertiesStruct(properties)] = struct{}{}
	}

	// Loop through baseline properties and fill the flow info
	for _, properties := range baselineProperties {
		converted := s.toConnectionPropertiesStruct(properties)
		if _, ok := allowedPropertiesSet[converted]; !ok {
			// This set of baseline connection properties if not currently allowed
			result.SetAdded(append(result.GetAdded(), properties))
		} else {
			// This set of properties currently exists.
			result.SetUnchanged(append(result.GetUnchanged(), properties))
			delete(allowedPropertiesSet, converted)
		}
	}
	// Since we have deleted matched properties from the currently allowed properties set. The properties left
	// are the ones that will be removed.
	for properties := range allowedPropertiesSet {
		nbcp := &storage.NetworkBaselineConnectionProperties{}
		nbcp.SetIngress(properties.ingress)
		nbcp.SetPort(properties.port)
		nbcp.SetProtocol(properties.protocol)
		result.SetRemoved(append(result.GetRemoved(), nbcp))
	}

	return result
}

func (s *serviceImpl) getQueryDeployments(ctx context.Context, clusterID, query string) ([]*storage.Deployment, error) {
	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	q := clusterQuery
	if query != "" {
		var err error
		q, err = search.ParseQuery(query)
		if err != nil {
			return nil, err
		}
		q = search.ConjunctionQuery(q, clusterQuery)
	}

	deps, err := s.deployments.SearchRawDeployments(ctx, q)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func (s *serviceImpl) getDeployments(ctx context.Context, clusterID, rawQ string, scope *v1.NetworkGraphScope) (set.StringSet, []*storage.Deployment, error) {
	depQ, scopeQ, err := networkgraph.GetFilterAndScopeQueries(clusterID, rawQ, scope)
	if err != nil {
		return nil, nil, err
	}

	clusterDeployments, err := s.deployments.SearchRawDeployments(ctx, scopeQ)
	if err != nil {
		return nil, nil, err
	}

	depQuery, _ := search.FilterQueryWithMap(depQ, deployments.OptionsMap)
	pred, err := deploymentPredicateFactory.GeneratePredicate(depQuery)
	if err != nil {
		return nil, nil, err
	}
	queryDeploymentIDs := set.NewStringSet()
	for _, dep := range clusterDeployments {
		if pred.Matches(dep) {
			queryDeploymentIDs.Add(dep.GetId())
		}
	}
	return queryDeploymentIDs, clusterDeployments, nil
}

func (s *serviceImpl) getNetworkTree(clusterID string) (tree.ReadOnlyNetworkTree, error) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	cfg, err := s.graphConfig.GetNetworkGraphConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	if cfg.GetHideDefaultExternalSrcs() {
		return s.networkTreeMgr.GetReadOnlyNetworkTree(ctx, clusterID), nil
	}

	return tree.NewMultiNetworkTree(
		s.networkTreeMgr.GetReadOnlyNetworkTree(ctx, clusterID),
		s.networkTreeMgr.GetDefaultNetworkTree(ctx),
	), nil
}

func (s *serviceImpl) getNetworkPoliciesInSimulation(ctx context.Context, clusterID string, modification *storage.NetworkPolicyModification) ([]*v1.NetworkPolicyInSimulation, error) {
	additionalPolicies, err := compileValidateYaml(modification.GetApplyYaml())
	if err != nil {
		return nil, err
	}

	// Gather all of the network policies that apply to the cluster and add the addition we are testing if applicable.
	currentPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, clusterID, "")
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
	ToDelete         []*storage.NetworkPolicyReference
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
		simPolicy := &v1.NetworkPolicyInSimulation{}
		simPolicy.SetPolicy(oldPolicy)
		simPolicy.SetStatus(v1.NetworkPolicyInSimulation_UNCHANGED)
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

		if simPolicy.GetOldPolicy() == nil {
			simPolicy.SetOldPolicy(simPolicy.GetPolicy())
		}
		simPolicy.ClearPolicy()
		simPolicy.SetStatus(v1.NetworkPolicyInSimulation_DELETED)
	}

	// Add new policies that have no matching old policies.
	for _, newPolicy := range policies.NewPolicies {
		oldPolicySim := policiesByRef[k8sutil.RefOf(newPolicy)]
		if oldPolicySim != nil {
			oldPolicySim.SetStatus(v1.NetworkPolicyInSimulation_MODIFIED)
			if oldPolicySim.GetOldPolicy() == nil {
				oldPolicySim.SetOldPolicy(oldPolicySim.GetPolicy())
			}
			oldPolicySim.SetPolicy(newPolicy)
			continue
		}
		newPolicySim := &v1.NetworkPolicyInSimulation{}
		newPolicySim.SetStatus(v1.NetworkPolicyInSimulation_ADDED)
		newPolicySim.SetPolicy(newPolicy)
		outputPolicies = append(outputPolicies, newPolicySim)
	}

	// Fix IDs: For all modified policies, the ID of the new and old policies should be the same (that way the
	// diff does not get cluttered with just policy ID changes); for all new policies, we generate new, fresh UUIDs
	// that do not collide with any other IDs.
	// Rationale: IDs are (almost) meaningless - IDs from the simulation YAML will be changed by kubectl create/apply
	// anyway.
	for _, policy := range outputPolicies {
		if policy.GetStatus() == v1.NetworkPolicyInSimulation_MODIFIED {
			policy.GetPolicy().SetId(policy.GetOldPolicy().GetId())
		} else if policy.GetStatus() == v1.NetworkPolicyInSimulation_ADDED {
			policy.GetPolicy().SetId(uuid.NewV4().String())
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
			return nil, errox.InvalidArgs.CausedBy("yamls tested against must apply to a namespace")
		}
	}

	return policies, nil
}

// validateNoForbiddenModification verifies whether network policy changes are not applied to 'stackrox' and 'kube-system' namespace
func validateNoForbiddenModification(networkPoliciesInSimulation []*v1.NetworkPolicyInSimulation) error {
	for _, policyInSim := range networkPoliciesInSimulation {
		policyNamespace := policyInSim.GetOldPolicy().GetNamespace()
		if policyNamespace == "" {
			policyNamespace = policyInSim.GetPolicy().GetNamespace()
		}

		if policyNamespace != namespaces.StackRox && policyNamespace != namespaces.KubeSystem {
			continue
		}

		if policyInSim.GetStatus() == v1.NetworkPolicyInSimulation_UNCHANGED {
			continue
		}

		policyName := policyInSim.GetPolicy().GetName()
		if policyInSim.GetStatus() != v1.NetworkPolicyInSimulation_MODIFIED {
			if policyInSim.GetStatus() != v1.NetworkPolicyInSimulation_ADDED {
				policyName = policyInSim.GetOldPolicy().GetName()
			}
			return errors.Errorf("%q cannot be applied since network policy change in '%q' namespace is forbidden", policyName, policyNamespace)
		}

		err := validateNoPolicyDiff(policyInSim.GetPolicy(), policyInSim.GetOldPolicy())
		if err != nil {
			return errors.Errorf("%q cannot be applied since network policy change in '%q' namespace is forbidden", policyName, policyNamespace)
		}
	}

	return nil
}

// validateNoPolicyDiff returns an error if the YAML of two network policies is different
func validateNoPolicyDiff(applyPolicy *storage.NetworkPolicy, currPolicy *storage.NetworkPolicy) error {
	if applyPolicy.GetYaml() != currPolicy.GetYaml() {
		return errors.New("network policies do not match")
	}

	return nil
}

func (s *serviceImpl) clusterExists(ctx context.Context, clusterID string) error {
	if clusterID == "" {
		return errors.Wrap(errox.InvalidArgs, "cluster ID must be specified")
	}
	requestedResourcesWithAccess := []permissions.ResourceWithAccess{permissions.View(resources.NetworkPolicy)}
	exists, err := s.clusterSACHelper.IsClusterVisibleForPermissions(ctx, clusterID, requestedResourcesWithAccess)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "cluster with ID %q doesn't exist", clusterID)
	}
	return nil
}

func getNamespacesFromModification(modification *storage.NetworkPolicyModification) (set.StringSet, error) {
	result := set.NewStringSet()
	for _, toDelete := range modification.GetToDelete() {
		result.Add(toDelete.GetNamespace())
	}

	if applyYaml := strings.TrimSpace(modification.GetApplyYaml()); applyYaml != "" {
		netPols, err := networkPolicyConversion.YamlWrap{Yaml: modification.GetApplyYaml()}.ToKubernetesNetworkPolicies()
		if err != nil {
			return nil, errors.Wrap(err, "error parsing network policies")
		}
		for _, np := range netPols {
			result.Add(np.GetNamespace())
		}
	}
	return result, nil
}

func checkAllNamespacesWriteAllowed(ctx context.Context, clusterID string, namespaces ...string) error {
	nsScopeKeys := make([][]sac.ScopeKey, 0, len(namespaces))
	for _, ns := range namespaces {
		nsScopeKeys = append(nsScopeKeys, []sac.ScopeKey{sac.NamespaceScopeKey(ns)})
	}
	return sac.VerifyAuthzOK(
		networkPolicySAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ClusterID(clusterID).AllAllowed(
			nsScopeKeys), nil)
}
