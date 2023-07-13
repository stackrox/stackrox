package generator

import (
	"context"
	"errors"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/api/core/v1"
)

var (
	// l4ProtoMap translates between `storage.L4Protocol` and `storage.Protocol`, which are
	// separate types for historical reasons.
	l4ProtoMap = map[storage.L4Protocol]storage.Protocol{
		storage.L4Protocol_L4_PROTOCOL_TCP: storage.Protocol_TCP_PROTOCOL,
		storage.L4Protocol_L4_PROTOCOL_UDP: storage.Protocol_UDP_PROTOCOL,
	}

	// l4ProtoByName maps lowercase protocol names, as used by Kubernetes, to `storage.L4Protocol`
	// values.
	l4ProtoByName = map[string]storage.L4Protocol{
		"tcp": storage.L4Protocol_L4_PROTOCOL_TCP,
		"udp": storage.L4Protocol_L4_PROTOCOL_UDP,
	}
)

type portDesc struct {
	l4proto storage.L4Protocol
	port    uint32
}

func (s *portDesc) isZero() bool {
	return s.l4proto == 0 && s.port == 0
}

func (s *portDesc) toNetPolPorts() []*storage.NetworkPolicyPort {
	if s.isZero() {
		return nil
	}

	netPolL4Proto, ok := l4ProtoMap[s.l4proto]
	if !ok {
		// since we can't determine the protocol, conservatively return an "all ports"
		// selector.
		return nil
	}

	return []*storage.NetworkPolicyPort{
		{
			PortRef: &storage.NetworkPolicyPort_Port{
				Port: int32(s.port),
			},
			Protocol: netPolL4Proto,
		},
	}
}

type peers map[*node]struct{}

func (p peers) hasInternetPeer() bool {
	for node := range p {
		if node.entity.Type == storage.NetworkEntityInfo_INTERNET || node.entity.Type == storage.NetworkEntityInfo_EXTERNAL_SOURCE {
			return true
		}
	}
	return false
}

func (p peers) hasMaskedPeer() bool {
	for node := range p {
		if node.masked {
			return true
		}
	}
	return false
}

type ingressInfo struct {
	peers
	exposed bool
}

type node struct {
	entity     networkgraph.Entity
	deployment *storage.Deployment
	selected   bool
	masked     bool
	incoming   map[portDesc]*ingressInfo
	outgoing   map[portDesc]peers
}

func createNode(entity networkgraph.Entity) *node {
	return &node{
		entity:   entity,
		incoming: make(map[portDesc]*ingressInfo),
		outgoing: make(map[portDesc]peers),
	}
}

var (
	externallyExposedExposureLevels = map[storage.PortConfig_ExposureLevel]struct{}{
		storage.PortConfig_NODE:     {},
		storage.PortConfig_ROUTE:    {},
		storage.PortConfig_EXTERNAL: {},
	}
)

func (n *node) populateExposureInfo(perPort bool) {
	for _, deploymentPort := range n.deployment.GetPorts() {
		if _, isExternal := externallyExposedExposureLevels[deploymentPort.GetExposure()]; !isExternal {
			continue
		}

		// Fill in port/protocol information only if we are computing per-port exposure info; otherwise,
		// we leave it as the zero value, which means "all ports/protocols" in port-insensitive mode.
		var port portDesc
		if perPort {
			port.port = uint32(deploymentPort.GetContainerPort())
			l4ProtoName := deploymentPort.GetProtocol()
			if l4ProtoName == "" {
				l4ProtoName = string(v1.ProtocolTCP)
			}
			port.l4proto = l4ProtoByName[strings.ToLower(l4ProtoName)]
		}

		ingInfo := n.incoming[port]
		if ingInfo == nil {
			ingInfo = &ingressInfo{}
			n.incoming[port] = ingInfo
		}
		ingInfo.exposed = true

		if !perPort {
			// early exit on first match in port-insensitive mode
			return
		}
	}
}

func (g *generator) buildGraph(ctx context.Context, clusterID string, selectedDeployments []*storage.Deployment, okFlows, missingInfoFlows []*storage.NetworkFlow, includePorts bool) (map[networkgraph.Entity]*node, error) {
	// Determine the deployments that are still missing.
	missingDeploymentIDs := set.NewStringSet()
	for _, flow := range missingInfoFlows {
		props := flow.GetProps()
		// It is sufficient to check the source entity, because we know that we have information regarding the
		// destination deployment entity -- otherwise, we would not retrieve the flow in the first place.
		if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetSrcEntity().GetDeployment() == nil {
			missingDeploymentIDs.Add(props.GetSrcEntity().GetId())
		}
	}

	// Not all of these deployments may be visible, but that's okay.
	var unselectedButVisibleDeployments []*storage.Deployment
	if missingDeploymentIDs.Cardinality() > 0 {
		var err error
		unselectedButVisibleDeployments, err = g.deploymentStore.GetDeployments(ctx, missingDeploymentIDs.AsSlice())
		if err != nil {
			return nil, err
		}

		// Retain only deployments that are not visible or deleted.
		for _, deployment := range unselectedButVisibleDeployments {
			missingDeploymentIDs.Remove(deployment.GetId())
		}
	}

	nodesByKey := make(map[networkgraph.Entity]*node)

	allFlows := make([]*storage.NetworkFlow, 0, len(okFlows)+len(missingInfoFlows))
	allFlows = append(allFlows, okFlows...)
	allFlows = append(allFlows, missingInfoFlows...)

	// Add nodes and edges for all flows.
	for _, flow := range allFlows {
		srcKey := networkgraph.EntityFromProto(flow.GetProps().GetSrcEntity())
		srcNode := nodesByKey[srcKey]
		if srcNode == nil {
			srcNode = createNode(srcKey)
			nodesByKey[srcKey] = srcNode
		}

		dstKey := networkgraph.EntityFromProto(flow.GetProps().GetDstEntity())
		dstNode := nodesByKey[dstKey]
		if dstNode == nil {
			dstNode = createNode(dstKey)
			nodesByKey[dstKey] = dstNode
		}

		// In port-insensitive mode, portInfo is left as the zero value, which means "all ports".
		var portInfo portDesc
		if includePorts {
			portInfo.port = flow.GetProps().GetDstPort()
			portInfo.l4proto = flow.GetProps().GetL4Protocol()
		}

		dstIncomingPeers := dstNode.incoming[portInfo]
		if dstIncomingPeers == nil {
			dstIncomingPeers = &ingressInfo{
				peers: make(peers),
			}
			dstNode.incoming[portInfo] = dstIncomingPeers
		}
		dstIncomingPeers.peers[srcNode] = struct{}{}

		srcOutgoingPeers := srcNode.outgoing[portInfo]
		if srcOutgoingPeers == nil {
			srcOutgoingPeers = make(peers)
			srcNode.outgoing[portInfo] = srcOutgoingPeers
		}
		srcOutgoingPeers[dstNode] = struct{}{}
	}

	// Populate deployment data for all deployments that we can see (either selected by the query, or relevant to
	// one of selected deployments and visible).
	// Remaining deployments will leave the `deployment` field of the nodes as nil, which will be interpreted as a
	// masked peer.
	for _, deployment := range selectedDeployments {
		key := networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}

		deploymentNode := nodesByKey[key]
		if deploymentNode == nil {
			deploymentNode = createNode(key)
			nodesByKey[key] = deploymentNode
		}
		deploymentNode.deployment = deployment
		deploymentNode.selected = true

		// Exposure information is only relevant for deployments for which a network policy is generated,
		// not for unselected peers.
		deploymentNode.populateExposureInfo(includePorts)
	}

	for _, deployment := range unselectedButVisibleDeployments {
		key := networkgraph.Entity{
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
			ID:   deployment.GetId(),
		}

		deploymentNode := nodesByKey[key]
		if deploymentNode == nil {
			continue
		}
		deploymentNode.deployment = deployment
	}

	if missingDeploymentIDs.Cardinality() > 0 {
		// Finally, do a deployments query with elevated privileges to know which deployments are invisible. These will
		// be then marked as masked.
		// This step exists to ensure that a recently deleted deployment is not interpreted as a masked deployment, which
		// would be extremely bad user experience as the generated policies for its peers would be useless.
		q := search.NewQueryBuilder().AddDocIDSet(missingDeploymentIDs).ProtoQuery()
		viewAllDeploymentsInClusterCtx := sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedClusterLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Deployment),
				sac.ClusterScopeKeys(clusterID)))
		results, err := g.deploymentStore.Search(viewAllDeploymentsInClusterCtx, q)
		if err != nil {
			return nil, err
		}
		for _, maskedDeploymentID := range search.ResultsToIDs(results) {
			key := networkgraph.Entity{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				ID:   maskedDeploymentID,
			}

			deploymentNode := nodesByKey[key]
			if deploymentNode == nil {
				continue // shouldn't happen
			}
			deploymentNode.masked = true
		}
	}

	return nodesByKey, nil
}

func (g *generator) generateNodeFromBaselineForDeployment(
	ctx context.Context,
	deployment *storage.Deployment,
	includePorts bool,
) (*node, error) {
	if isProtectedDeployment(deployment) {
		return nil, errors.New("cannot generate policy for a protected deployment")
	}

	// We get the baseline for deployment, and construct a node object for this deployment with info of its baseline
	baseline, ok, err := g.networkBaselines.GetNetworkBaseline(ctx, deployment.GetId())
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("network baseline not found for deployment")
	}

	deploymentNode := createNode(networkgraph.Entity{Type: storage.NetworkEntityInfo_DEPLOYMENT, ID: deployment.GetId()})
	deploymentNode.deployment = deployment

	// Temporarily elevate permissions to obtain all deployments in cluster.
	elevatedCtx := sac.WithAllAccess(ctx)
	// Since we only generate ingress flows, we only look at ingress peers for policy generation
	for _, peer := range baseline.GetPeers() {
		peerNode := g.populateNode(elevatedCtx, peer.GetEntity().GetInfo().GetId(), peer.GetEntity().GetInfo().GetType())
		if peerNode == nil {
			// Peer deployment probably has been deleted.
			continue
		}
		for _, props := range peer.GetProperties() {
			if !props.GetIngress() {
				continue
			}
			var port portDesc
			if includePorts {
				port.port = props.GetPort()
				port.l4proto = props.GetProtocol()
			}
			currentPeers, ok := deploymentNode.incoming[port]
			if !ok {
				currentPeers = &ingressInfo{peers: make(peers)}
			}
			currentPeers.peers[peerNode] = struct{}{}
			deploymentNode.incoming[port] = currentPeers
		}
	}

	return deploymentNode, nil
}
