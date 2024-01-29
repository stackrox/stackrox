package graph

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

type graphBuilder struct {
	namespacesByName    map[string]*storage.NamespaceMetadata
	allDeployments      []*node
	extSrcs             []*node
	extSrcIDs           set.StringSet
	internetSrc         *node
	networkTree         tree.ReadOnlyNetworkTree
	deploymentsByNS     map[*storage.NamespaceMetadata][]*node
	deploymentPredicate func(string) bool
}

func newGraphBuilder(queryDeploymentIDs set.StringSet, deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, namespacesByID map[string]*storage.NamespaceMetadata) *graphBuilder {
	b := &graphBuilder{}
	b.init(queryDeploymentIDs, deployments, networkTree, namespacesByID)
	return b
}

func (b *graphBuilder) init(queryDeploymentIDs set.StringSet, deployments []*storage.Deployment, networkTree tree.ReadOnlyNetworkTree, namespacesByID map[string]*storage.NamespaceMetadata) {
	b.allDeployments = make([]*node, 0, len(deployments))
	b.extSrcs = make([]*node, 0)
	b.extSrcIDs = set.NewStringSet()
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

	if queryDeploymentIDs == nil {
		b.deploymentPredicate = func(string) bool { return true }
	} else {
		b.deploymentPredicate = func(id string) bool { return queryDeploymentIDs.Contains(id) }
	}

	if networkTree == nil {
		networkTree = tree.NewMultiNetworkTree(tree.NewDefaultNetworkTreeWrapper())
	}

	b.networkTree = networkTree
	b.internetSrc = b.getOrCreateExtSrcNode(networkgraph.InternetEntity().ToProto())
}

func (b *graphBuilder) evaluatePeers(currentNS *storage.NamespaceMetadata, peers []*storage.NetworkPolicyPeer) ([]*node, bool) {
	if len(peers) == 0 {
		// An empty peers list means all possible peers are allowed. We skip adding any known external sources, since
		// there could be many. Instead only all INTERNET node, which abstracts all the external sources.
		allNodes := make([]*node, 0, len(b.allDeployments)+1)
		allNodes = append(allNodes, b.allDeployments...)
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
		if len(allPeers) == len(b.allDeployments)+b.networkTree.Cardinality()+1 {
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
		var allNodes []*node
		_, ipNet, err := net.ParseCIDR(peer.GetIpBlock().GetCidr())
		if err != nil {
			log.Warnf("Failed to parse CIDR block: %s", err)
			return allNodes
		}

		// If the IP is in the private range, we add edges to other deployments in the cluster.
		// This is still not the perfect approach, but it solves user issues where edges were being
		// shown on *every* deployment with a Network Policy that uses CIDR block matchers. By
		// limiting to private ranges only, the set of edges in the graph is likely more accurate, but
		// still not the reality. There is an epic created to tackle this further and come up with a
		// more elaborate logic for this: ROX-12120
		if netutil.IsIPNetOverlapingPrivateRange(ipNet) {
			allNodes = append(allNodes, b.allDeployments...)
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

func (b *graphBuilder) getOrCreateExtSrcNode(extSrc *storage.NetworkEntityInfo) *node {
	if b.extSrcIDs.Add(extSrc.GetId()) {
		n := newExternalSrcNode(extSrc)
		b.extSrcs = append(b.extSrcs, n)
		return n
	}
	for _, existingExtSrcNode := range b.extSrcs {
		if existingExtSrcNode.extSrc.Id == extSrc.GetId() {
			return existingExtSrcNode
		}
	}
	// Should never get here since we keep the set and the list in sync.
	// Note that utils.Should here is useless since we will panic elsewhere anyway,
	// if such a fundamental assertion is broken.
	panic(fmt.Sprintf("UNEXPECTED: %v id found in extSrcIDs but not in extSrcs", extSrc))
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
		n := b.getOrCreateExtSrcNode(extSrc)
		return []*node{n}
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
			n := b.getOrCreateExtSrcNode(extSrc)
			peers = append(peers, n)
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

func (b *graphBuilder) GetApplyingPoliciesPerDeployment(allNetPols []*storage.NetworkPolicy) map[string][]*storage.NetworkPolicy {
	deploymentsToNetPols := make(map[string][]*storage.NetworkPolicy)
	b.forEachNetworkPolicy(allNetPols, func(netPol *storage.NetworkPolicy, _ *storage.NamespaceMetadata, matchedDeployments []*node) {
		for _, node := range matchedDeployments {
			if id := node.deployment.GetId(); id != "" {
				deploymentsToNetPols[id] = append(deploymentsToNetPols[id], netPol)
			}
		}
	})
	return deploymentsToNetPols
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
	nodeMap, allNodes := b.getRelevantNodes()
	for src, srcIdx := range nodeMap {
		srcQueried := b.deploymentPredicate(src.deployment.GetId())
		for tgt := range src.adjacentNodes {
			if tgt == nil {
				utils.Should(errors.New("network policy graph peer node is nil"))
				continue
			}

			// Add an edge between nodes iff one endpoint was queried.
			if !srcQueried && !b.deploymentPredicate(tgt.deployment.GetId()) {
				continue
			}

			// If the target has non-isolated ingress, skip adding edges since it is obvious.
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

			srcNode := allNodes[srcIdx]
			if srcNode.OutEdges == nil {
				srcNode.OutEdges = make(map[int32]*v1.NetworkEdgePropertiesBundle)
			}

			tgtIdx, ok := nodeMap[tgt]
			if !ok {
				if tgt.deployment != nil {
					utils.Should(errors.Errorf("deployment node %s not found in network node map", tgt.deployment.GetId()))
				} else if tgt.extSrc != nil {
					utils.Should(errors.Errorf("external node %s not found in network node map", tgt.extSrc.GetId()))
				}
				continue
			}

			srcNode.OutEdges[int32(tgtIdx)] = bundleForPorts(portsForEdge, includePorts)
		}
	}
	return allNodes
}

func (b *graphBuilder) getRelevantNodes() (map[*node]int, []*v1.NetworkNode) {
	filteredNodeIDs := b.getRelevantNodeIDs()
	// Sort nodes to keep the node order across multiple graph builds consistent.
	sort.Slice(b.allDeployments, func(i, j int) bool {
		return strings.Compare(b.allDeployments[i].deployment.GetId(), b.allDeployments[j].deployment.GetId()) < 0
	})

	sort.Slice(b.extSrcs, func(i, j int) bool {
		return strings.Compare(b.extSrcs[i].extSrc.GetId(), b.extSrcs[j].extSrc.GetId()) < 0
	})

	nodeMap := make(map[*node]int)
	allNodes := make([]*v1.NetworkNode, 0, len(filteredNodeIDs))
	for _, node := range b.allDeployments {
		if !filteredNodeIDs.Contains(node.deployment.GetId()) {
			continue
		}

		nodeMap[node] = len(allNodes)
		allNodes = append(allNodes, &v1.NetworkNode{
			Entity:             node.toEntityProto(),
			InternetAccess:     node.internetAccess || !node.isEgressIsolated,
			NonIsolatedEgress:  !node.isEgressIsolated,
			NonIsolatedIngress: !node.isIngressIsolated,
			OutEdges:           make(map[int32]*v1.NetworkEdgePropertiesBundle),
			PolicyIds:          node.applyingPoliciesIDs,
			QueryMatch:         b.deploymentPredicate(node.deployment.GetId()),
		})
	}

	for _, node := range b.extSrcs {
		if !filteredNodeIDs.Contains(node.extSrc.GetId()) {
			continue
		}

		nodeMap[node] = len(allNodes)
		allNodes = append(allNodes, &v1.NetworkNode{
			Entity:             node.extSrc,
			InternetAccess:     true,
			NonIsolatedEgress:  true,
			NonIsolatedIngress: true,
			OutEdges:           make(map[int32]*v1.NetworkEdgePropertiesBundle),
		})
	}
	return nodeMap, allNodes
}

func (b *graphBuilder) getRelevantNodeIDs() set.StringSet {
	var anyQueryDepWithNonIsolatedIngress, anyQueryDepWithNonIsolatedEgress bool
	// First, determine if any queried deployments are non-isolated.
	for _, d := range b.allDeployments {
		if !d.isIngressIsolated {
			if b.deploymentPredicate(d.deployment.GetId()) {
				anyQueryDepWithNonIsolatedIngress = true
			}
		}
		if !d.isEgressIsolated {
			if b.deploymentPredicate(d.deployment.GetId()) {
				anyQueryDepWithNonIsolatedEgress = true
			}
		}
	}

	filteredNodeIDs := set.NewStringSet()
	for _, currNode := range b.allDeployments {
		var srcQueried, anyValidConns bool
		if b.deploymentPredicate(currNode.deployment.GetId()) {
			srcQueried = true
		}

		for adjNode := range currNode.adjacentNodes {
			if adjNode == nil || (adjNode.deployment == nil && adjNode.extSrc == nil) {
				utils.Should(errors.New("network policy graph peer node is nil"))
				continue
			}

			dstQueried := b.deploymentPredicate(adjNode.deployment.GetId())
			// If neither endpoint was queried, skip it for now. If this `adjNode` is relevant for some other node, it will be added eventually.
			if !srcQueried && !dstQueried {
				continue
			}

			// If isolated, check if any out edges actually exist to the target. If no edges exists, skip it.
			var numEdges int
			if currNode.isEgressIsolated && adjNode.isIngressIsolated {
				numEdges = len(intersectNormalized(currNode.egressEdges[adjNode].getPorts(), adjNode.ingressEdges[currNode].getPorts()))
			} else if currNode.isEgressIsolated {
				numEdges = len(currNode.egressEdges[adjNode].getPorts())
			} else if adjNode.isIngressIsolated {
				numEdges = len(adjNode.ingressEdges[currNode].getPorts())
			} else {
				numEdges = 1 // non-isolated case-!currNode.isEgressIsolated && !adjNode.isIngressIsolated
			}

			if numEdges == 0 {
				continue
			}

			anyValidConns = true

			if adjNode.deployment != nil {
				filteredNodeIDs.Add(adjNode.deployment.GetId())
			} else if adjNode.extSrc != nil {
				filteredNodeIDs.Add(adjNode.extSrc.GetId())
			}
		}

		// Current node must exist in the graph if it satisfies any of this following conditions, else, skip it for now.
		// - Either current node or any of its peers were queried, or
		// - Current node has non-isolated egress and there exists at least one queried currNode that has non-isolated ingress, or
		// - Current node has non-isolated ingress and there exists at least one queried currNode that has non-isolated egress

		var relevantNode bool
		if srcQueried || anyValidConns {
			relevantNode = true
		} else if anyQueryDepWithNonIsolatedIngress && !currNode.isEgressIsolated {
			relevantNode = true
		} else if anyQueryDepWithNonIsolatedEgress && !currNode.isIngressIsolated {
			relevantNode = true
		}

		if relevantNode {
			filteredNodeIDs.Add(currNode.deployment.GetId())
		}
	}

	// Now determine relevant external entities. An external node should exists iff its peer was queried.
	for _, node := range b.extSrcs {
		for adjNode := range node.adjacentNodes {
			if adjNode == nil {
				utils.Should(errors.New("network policy graph peer node is nil"))
				continue
			}

			if b.deploymentPredicate(adjNode.deployment.GetId()) {
				filteredNodeIDs.Add(node.extSrc.GetId())
				break
			}
		}
	}

	// If any queried node is non-isolated, INTERNET node must be added to the graph.
	if anyQueryDepWithNonIsolatedIngress || anyQueryDepWithNonIsolatedEgress {
		filteredNodeIDs.Add(b.internetSrc.extSrc.GetId())
	}
	return filteredNodeIDs
}
