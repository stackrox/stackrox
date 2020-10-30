package graph

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/set"
)

type graphBuilder struct {
	namespacesByName map[string]*storage.NamespaceMetadata
	allDeployments   []*node
	knownExtSrcs     []*node
	knownExtSrcsByID map[string]*node
	internetSrc      *node
	networkTree      *tree.NetworkTreeWrapper
	deploymentsByNS  map[*storage.NamespaceMetadata][]*node
}

func newGraphBuilder(deployments []*storage.Deployment, externalSrcs []*storage.NetworkEntityInfo, namespacesByID map[string]*storage.NamespaceMetadata) *graphBuilder {
	b := &graphBuilder{}
	b.init(deployments, externalSrcs, namespacesByID)
	return b
}

func (b *graphBuilder) init(deployments []*storage.Deployment, externalSrcs []*storage.NetworkEntityInfo, namespacesByID map[string]*storage.NamespaceMetadata) {
	b.allDeployments = make([]*node, 0, len(deployments))
	b.knownExtSrcs = make([]*node, 0, len(externalSrcs))
	b.knownExtSrcsByID = make(map[string]*node)
	b.namespacesByName = make(map[string]*storage.NamespaceMetadata)
	b.deploymentsByNS = make(map[*storage.NamespaceMetadata][]*node)

	for _, ns := range namespacesByID {
		b.namespacesByName[ns.GetName()] = ns
	}

	for _, deployment := range deployments {
		node := newDeploymentNode(deployment)
		b.allDeployments = append(b.allDeployments, node)

		ns := b.namespacesByName[deployment.GetNamespace()]
		if ns == nil {
			ns = namespacesByID[deployment.GetNamespaceId()]
			if ns == nil {
				ns = &storage.NamespaceMetadata{
					Name: deployment.GetNamespace(),
					Id:   deployment.GetNamespaceId(),
				}
			}
			b.namespacesByName[ns.GetName()] = ns
		}
		b.deploymentsByNS[ns] = append(b.deploymentsByNS[ns], node)
	}

	for _, extSrc := range externalSrcs {
		node := newExternalSrcNode(extSrc)
		b.knownExtSrcs = append(b.knownExtSrcs, node)
		b.knownExtSrcsByID[extSrc.GetId()] = node
	}

	// We do not return the error since INTERNET gets added anyway i.e. external connections do not get excluded.
	var err error
	if b.networkTree, err = tree.NewNetworkTreeWrapper(externalSrcs); err != nil {
		log.Errorf("failed to create network tree for network policy builder: %v", err)
	}

	b.internetSrc = newExternalSrcNode(networkgraph.InternetEntity().ToProto())
}

func (b *graphBuilder) evaluatePeers(currentNS *storage.NamespaceMetadata, peers []*storage.NetworkPolicyPeer) ([]*node, bool) {
	if len(peers) == 0 {
		// An empty peers list means all possible peers are allowed. We skip adding any known external sources, since
		// there could be many. Instead only all INTERNET node, which abstracts all the external sources.
		allNodes := make([]*node, 0, len(b.allDeployments)+1)
		allNodes = append(allNodes, b.allDeployments...)

		if !features.NetworkGraphExternalSrcs.Enabled() {
			return allNodes, true
		}

		allNodes = append(allNodes, b.internetSrc)
		return allNodes, true
	}

	allPeers := make(map[*node]struct{})
	internetAccess := false
	for _, peer := range peers {
		if peer.GetIpBlock() != nil {
			internetAccess = true
		}

		// +1 for INTERNET
		if len(allPeers) == len(b.allDeployments)+len(b.knownExtSrcs)+1 {
			break
		}

		matchedPeers := b.evaluatePeer(currentNS, peer)
		for _, ep := range matchedPeers {
			allPeers[ep] = struct{}{}
		}
	}

	allPeerSlice := make([]*node, 0, len(allPeers))
	for pd := range allPeers {
		allPeerSlice = append(allPeerSlice, pd)
	}
	return allPeerSlice, internetAccess
}

func (b *graphBuilder) evaluatePeer(currentNS *storage.NamespaceMetadata, peer *storage.NetworkPolicyPeer) []*node {
	if peer.GetIpBlock() != nil {
		// TODO(ROX-5370): We assume all CIDR blocks always match all deployments and overlapping external CIDRs.
		//  This is probably wrong, but we don't really have a good way of determining otherwise. Except for maybe
		//  look at Pod IPs. Which we actually could.
		allNodes := make([]*node, 0, len(b.allDeployments)+len(b.knownExtSrcs)+1)
		allNodes = append(allNodes, b.allDeployments...)

		if !features.NetworkGraphExternalSrcs.Enabled() {
			return allNodes
		}

		allNodes = append(allNodes, b.evaluateExternalPeer(peer.GetIpBlock())...)
		return allNodes
	}

	var deploymentsInNSs []*node
	if peer.GetNamespaceSelector() == nil {
		if peer.GetPodSelector() == nil {
			// A peer with neither pod nor namespace selector set matches nothing.
			return nil
		}
		// Otherwise, the pod selector is applied to all pods in the policy's namespace.
		deploymentsInNSs = b.deploymentsByNS[currentNS]
	} else {
		nsSel, err := labels.CompileSelector(peer.GetNamespaceSelector())
		if err != nil {
			log.Errorf("Failed to compile namespace selector for network policy in namespace %s: %v", currentNS, err)
			return nil
		}
		if nsSel.MatchesAll() {
			deploymentsInNSs = b.allDeployments
		} else if !nsSel.MatchesNone() {
			for ns, deployments := range b.deploymentsByNS {
				if nsSel.Matches(ns.GetLabels()) {
					deploymentsInNSs = append(deploymentsInNSs, deployments...)
				}
			}
		}
	}

	if len(deploymentsInNSs) == 0 {
		return nil
	}

	if peer.GetPodSelector() == nil {
		// Non-nil namespace selector + nil pod selector => match all pods in all matched namespaces.
		return deploymentsInNSs
	}

	podSel, err := labels.CompileSelector(peer.GetPodSelector())
	if err != nil {
		log.Errorf("Failed to compile pod selector for network policy in namespace %s: %v", currentNS.GetName(), err)
		return nil
	}

	return matchDeployments(deploymentsInNSs, podSel)
}

func (b *graphBuilder) evaluateExternalPeer(ipBlock *storage.IPBlock) []*node {
	if ipBlock.GetCidr() == "" {
		return nil
	}

	// a. Find known external network that fully contains the netpol ipBlock.
	// b. If no such network is found, find all the known external sources that are fully contained by the netpol ipBlock.
	// c. Finally remove any external peer that is fully contained by the netpol "except" networks.

	if extSrc := b.networkTree.GetMatchingSupernetForCIDR(ipBlock.GetCidr(), func(entity *storage.NetworkEntityInfo) bool {
		return entity.GetId() != networkgraph.InternetExternalSourceID
	}); extSrc != nil {
		return []*node{b.knownExtSrcsByID[extSrc.GetId()]}
	}

	allMatchedPeers := b.networkTree.GetSubnetsForCIDR(ipBlock.GetCidr())
	netsToExclude := set.NewStringSet()
	for _, except := range ipBlock.GetExcept() {
		for _, extSrc := range b.networkTree.GetSubnetsForCIDR(except) {
			netsToExclude.Add(extSrc.GetId())
		}
	}

	peers := make([]*node, 0, len(allMatchedPeers))
	// No single known external network fully contains the ipBlock, hence add INTERNET.
	peers = append(peers, b.internetSrc)
	for _, extSrc := range allMatchedPeers {
		if !netsToExclude.Contains(extSrc.GetId()) {
			peers = append(peers, b.knownExtSrcsByID[extSrc.GetId()])
		}
	}
	return peers
}

func (b *graphBuilder) getOrCreateEdge(src, tgt *node, egress bool) *edge {
	if src.adjacentNodes == nil {
		src.adjacentNodes = make(map[*node]struct{})
	}
	src.adjacentNodes[tgt] = struct{}{}

	other := tgt
	edgeMap := &src.egressEdges
	if !egress {
		other = src
		edgeMap = &tgt.ingressEdges
	}

	if e := (*edgeMap)[other]; e != nil {
		return e
	}

	e := &edge{
		src: src,
		tgt: tgt,
	}

	if *edgeMap == nil {
		*edgeMap = make(map[*node]*edge)
	}
	(*edgeMap)[other] = e
	return e
}

func (b *graphBuilder) addEdgesForNetworkPolicy(netPol *storage.NetworkPolicy, currNS *storage.NamespaceMetadata, matchedDeployments []*node) {
	ingressPolicy := hasIngress(netPol.GetSpec().GetPolicyTypes())
	egressPolicy := hasEgress(netPol.GetSpec().GetPolicyTypes())

	for _, matched := range matchedDeployments {
		if ingressPolicy {
			matched.isIngressIsolated = true
		}
		if egressPolicy {
			matched.isEgressIsolated = true
		}
		matched.applyingPoliciesIDs = append(matched.applyingPoliciesIDs, netPol.GetId())
	}

	for _, ingRule := range netPol.GetSpec().GetIngress() {
		peers, _ := b.evaluatePeers(currNS, ingRule.GetFrom())
		for _, matched := range matchedDeployments {
			portDescs := matched.resolvePorts(ingRule.GetPorts())
			if len(portDescs) == 0 {
				continue
			}
			for _, p := range peers {
				if matched == p {
					continue
				}

				e := b.getOrCreateEdge(p, matched, false)
				e.ports = append(e.ports, portDescs...)
			}
		}
	}

	for _, egRule := range netPol.GetSpec().GetEgress() {
		peers, internetAccess := b.evaluatePeers(currNS, egRule.GetTo())
		if internetAccess {
			for _, matched := range matchedDeployments {
				matched.internetAccess = true
			}
		}

		for _, p := range peers {
			portDescs := p.resolvePorts(egRule.GetPorts())
			if len(portDescs) == 0 {
				continue
			}
			for _, matched := range matchedDeployments {
				if matched == p {
					continue
				}

				e := b.getOrCreateEdge(matched, p, true)
				e.ports = append(e.ports, portDescs...)
			}
		}
	}
}

func (b *graphBuilder) AddEdgesForNetworkPolicies(netPols []*storage.NetworkPolicy) {
	b.forEachNetworkPolicy(netPols, b.addEdgesForNetworkPolicy)
}

func (b *graphBuilder) GetApplyingPolicies(allNetPols []*storage.NetworkPolicy) []*storage.NetworkPolicy {
	var applyingPolicies []*storage.NetworkPolicy
	b.forEachNetworkPolicy(allNetPols, func(netPol *storage.NetworkPolicy, _ *storage.NamespaceMetadata, matchedDeployments []*node) {
		if len(matchedDeployments) > 0 {
			applyingPolicies = append(applyingPolicies, netPol)
		}
	})
	return applyingPolicies
}

func (b *graphBuilder) forEachNetworkPolicy(netPols []*storage.NetworkPolicy, do func(*storage.NetworkPolicy, *storage.NamespaceMetadata, []*node)) {
	for _, netPol := range netPols {
		currNS := b.namespacesByName[netPol.GetNamespace()]
		if currNS == nil {
			log.Infof("Unknown namespace for netpol %s, %+v", netPol.GetNamespace(), b.namespacesByName)
			continue // unknown namespace
		}

		deploymentsInNS := b.deploymentsByNS[currNS]
		if len(deploymentsInNS) == 0 {
			continue
		}

		podSelector, err := labels.CompileSelector(netPol.GetSpec().GetPodSelector())
		if err != nil {
			log.Errorf("Network policy %s/%s contains invalid pod selector, ignoring: %v", netPol.GetNamespace(), netPol.GetName(), err)
			continue
		}

		matchedDeployments := matchDeployments(deploymentsInNS, podSelector)
		if len(matchedDeployments) == 0 {
			continue
		}

		do(netPol, currNS, matchedDeployments)
	}
}

func (b *graphBuilder) PostProcess() {
	for _, d := range b.allDeployments {
		sort.Strings(d.applyingPoliciesIDs)
		for _, e := range d.ingressEdges {
			e.ports.normalizeInPlace()
		}
		for _, e := range d.egressEdges {
			e.ports.normalizeInPlace()
		}
	}
}

func bundleForPorts(ports portDescs, includePorts bool) *v1.NetworkEdgePropertiesBundle {
	bundle := &v1.NetworkEdgePropertiesBundle{}
	if includePorts {
		bundle.Properties = ports.ToProto()
	}

	return bundle
}

func (b *graphBuilder) ToProto(includePorts bool) []*v1.NetworkNode {
	// This stores external sources including INTERNET. These nodes are added to final graph only if they exists a connection with them.
	externalNodes := make(map[*node]*v1.NetworkNode)
	// Add INTERNET
	allExternalNodes := make([]*node, 0, len(b.knownExtSrcs)+1)
	allExternalNodes = append(allExternalNodes, b.knownExtSrcs...)
	allExternalNodes = append(allExternalNodes, b.internetSrc)
	for _, n := range allExternalNodes {
		node := &v1.NetworkNode{
			Entity:             n.extSrc,
			InternetAccess:     true,
			NonIsolatedEgress:  true,
			NonIsolatedIngress: true,
			OutEdges:           make(map[int32]*v1.NetworkEdgePropertiesBundle),
		}
		externalNodes[n] = node
	}

	// anyNonIsolatedDeps tracks if there are non-isolated deployments in the cluster. This is a helper that
	// enables adding external sources in cases, such as, no network policies are not defined for a namespace.
	anyNonIsolatedDeps := false

	nodeMap := make(map[*node]int)
	allNodes := make([]*v1.NetworkNode, 0, len(b.allDeployments)+len(b.knownExtSrcs)+1)
	for _, d := range b.allDeployments {
		if !d.isEgressIsolated || !d.isIngressIsolated {
			anyNonIsolatedDeps = true
		}

		node := &v1.NetworkNode{
			Entity:             d.toEntityProto(),
			InternetAccess:     d.internetAccess || !d.isEgressIsolated,
			NonIsolatedEgress:  !d.isEgressIsolated,
			NonIsolatedIngress: !d.isIngressIsolated,
			OutEdges:           make(map[int32]*v1.NetworkEdgePropertiesBundle),
			PolicyIds:          d.applyingPoliciesIDs,
		}
		nodeMap[d] = len(allNodes)
		allNodes = append(allNodes, node)
	}

	for _, src := range b.allDeployments {
		srcIdx := nodeMap[src]
		srcNode := allNodes[srcIdx]
		for tgt := range src.adjacentNodes {
			if !src.isEgressIsolated && !tgt.isIngressIsolated {
				continue
			}

			var portsForEdge portDescs
			if src.isEgressIsolated && tgt.isIngressIsolated {
				portsForEdge = intersectNormalized(src.egressEdges[tgt].getPorts(), tgt.ingressEdges[src].getPorts())
			} else if src.isEgressIsolated { // && !tgt.isIngressIsolated
				portsForEdge = src.egressEdges[tgt].getPorts()
			} else if tgt.isIngressIsolated { // && !src.isEgressIsolated
				portsForEdge = tgt.ingressEdges[src].getPorts()
			}

			if len(portsForEdge) == 0 {
				continue
			}

			tgtIdx, ok := nodeMap[tgt]
			// This means that target is an external source that has not been added to graph yet.
			if !ok {
				tgtNode, ok := externalNodes[tgt]
				if !ok {
					continue
				}
				tgtIdx = len(allNodes)
				nodeMap[tgt] = tgtIdx
				allNodes = append(allNodes, tgtNode)
			}

			if srcNode.OutEdges == nil {
				srcNode.OutEdges = make(map[int32]*v1.NetworkEdgePropertiesBundle)
			}
			srcNode.OutEdges[int32(tgtIdx)] = bundleForPorts(portsForEdge, includePorts)
		}
	}

	if !features.NetworkGraphExternalSrcs.Enabled() {
		return allNodes
	}

	for _, src := range allExternalNodes {
		var srcNode *v1.NetworkNode
		if srcIdx, ok := nodeMap[src]; ok {
			srcNode = allNodes[srcIdx]
		} else {
			srcNode = externalNodes[src]
		}

		for tgt := range src.adjacentNodes {
			// If the target has non-isolated ingress, skip adding edges since it is obvious.
			if !tgt.isIngressIsolated {
				continue
			}

			portsForEdge := tgt.ingressEdges[src].getPorts()
			if len(portsForEdge) == 0 {
				continue
			}

			tgtIdx := nodeMap[tgt]
			if srcNode.OutEdges == nil {
				srcNode.OutEdges = make(map[int32]*v1.NetworkEdgePropertiesBundle)
			}
			srcNode.OutEdges[int32(tgtIdx)] = bundleForPorts(portsForEdge, includePorts)
		}

		// If the external source has outgoing edges but has not been added to graph, add it.
		if _, ok := nodeMap[src]; !ok && len(srcNode.OutEdges) > 0 {
			nodeMap[src] = len(allNodes)
			allNodes = append(allNodes, srcNode)
		}
	}

	if _, ok := nodeMap[b.internetSrc]; !ok && anyNonIsolatedDeps {
		allNodes = append(allNodes, externalNodes[b.internetSrc])
	}
	return allNodes
}
