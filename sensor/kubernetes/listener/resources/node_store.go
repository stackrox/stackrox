package resources

import (
	"sort"

	"github.com/stackrox/rox/pkg/net"
	v1 "k8s.io/api/core/v1"
)

type nodeWrap struct {
	*v1.Node
	addresses []net.IPAddress
}

func wrapNode(node *v1.Node) *nodeWrap {
	wrap := &nodeWrap{Node: node}
	for _, nodeAddr := range node.Status.Addresses {
		if nodeAddr.Type != v1.NodeInternalIP && nodeAddr.Type != v1.NodeExternalIP {
			continue
		}
		parsedIP := net.ParseIP(nodeAddr.Address)
		if parsedIP.IsValid() {
			wrap.addresses = append(wrap.addresses, parsedIP)
		}
	}
	sort.Slice(wrap.addresses, func(i, j int) bool {
		return net.IPAddressLess(wrap.addresses[i], wrap.addresses[j])
	})
	return wrap
}

type nodeStore struct {
	nodes map[string]*nodeWrap
}

func newNodeStore() *nodeStore {
	return &nodeStore{
		nodes: make(map[string]*nodeWrap),
	}
}

// addOrUpdateNode upserts a node to the store.
// It returns true if the IP addresses of the node changed as a result.
func (s *nodeStore) addOrUpdateNode(node *nodeWrap) bool {
	oldNode := s.nodes[node.Name]
	s.nodes[node.Name] = node

	if oldNode == nil || len(oldNode.addresses) != len(node.addresses) {
		return true
	}
	for i, oldAddr := range oldNode.addresses {
		if oldAddr != node.addresses[i] {
			return true
		}
	}
	return false
}

func (s *nodeStore) removeNode(node *v1.Node) {
	delete(s.nodes, node.Name)
}

func (s *nodeStore) getNode(nodeName string) *nodeWrap {
	return s.nodes[nodeName]
}

func (s *nodeStore) getNodes() []*nodeWrap {
	result := make([]*nodeWrap, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result
}
