package compliance

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/sensor/common/store"
)

var rhcosOSImageRegexp = regexp.MustCompile(`(Red Hat Enterprise Linux) (CoreOS) ([0-9.-]+)`)

// NodeIDMatcher helps finding NodeWrap by name
type NodeIDMatcher interface {
	GetNodeID(nodename string) (string, error)
}

// NodeRHCOSMatcher is used to check whether Node is RHCOS and obtain its version
type NodeRHCOSMatcher interface {
	GetRHCOSVersion(name string) (bool, string, error)
}

// NodeIDMatcherImpl finds Node by name within NodeStore
type NodeIDMatcherImpl struct {
	nodeStore store.NodeStore
}

// NewNodeIDMatcher creates a NodeIDMatcherImpl
func NewNodeIDMatcher(store store.NodeStore) *NodeIDMatcherImpl {
	return &NodeIDMatcherImpl{
		nodeStore: store,
	}
}

// GetNodeID returns NodeID if a Node with matching name has been found
func (c *NodeIDMatcherImpl) GetNodeID(nodename string) (string, error) {
	if node := c.nodeStore.GetNode(nodename); node != nil {
		return node.GetId(), nil
	}
	return "", fmt.Errorf("cannot find node with name '%s'", nodename)
}

// GetRHCOSVersion returns bool=true if node is running RHCOS, and it's version reported
// by the orchestrator.
func (c *NodeIDMatcherImpl) GetRHCOSVersion(name string) (bool, string, error) {
	n := c.nodeStore.GetNode(name)
	if n == nil {
		return false, "", fmt.Errorf("cannot find node with name %q", name)
	}
	osImageRef := n.GetOsImage()
	if !strings.HasPrefix(osImageRef, rhcosFullName) {
		return false, osImageRef, nil
	}
	r := rhcosOSImageRegexp.FindStringSubmatch(osImageRef)
	// r[0] contains the entire osImageRef
	if len(r) < 4 {
		return false, osImageRef, fmt.Errorf("valid RHCOS prefix found, but cannot parse version from: %s", osImageRef)
	}
	return true, r[3], nil
}
