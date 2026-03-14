package compliance

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/sensor/common/store"
)

// rhcosOSImageRegexp captures: "Red Hat Enterprise Linux CoreOS <OCP_MAJOR>.<OCP_MINOR>[.<RHCOS_VERSION>]"
// Groups:
//   - r[1]: "Red Hat Enterprise Linux"
//   - r[2]: "CoreOS"
//   - r[3]: OCP major version (e.g., "4")
//   - r[4]: OCP minor version (e.g., "19")
//   - r[5]: RHCOS version suffix (optional, e.g., ".0" or empty)
var rhcosOSImageRegexp = regexp.MustCompile(`(Red Hat Enterprise Linux) (CoreOS) ([0-9]+)\.([0-9]+)(\.?[0-9.-]*)`)

// NodeIDMatcher helps finding NodeWrap by name
type NodeIDMatcher interface {
	GetNodeID(nodename string) (string, error)
}

// NodeRHCOSMatcher is used to check whether Node is RHCOS and obtain its version
// GetRHCOSVersion returns:
//   - isRHCOS: true if the node is running RHCOS
//   - ocpVersion: the OCP major.minor version (e.g., "4.19")
//   - rhcosVersion: the RHCOS version to use for matching
//   - err: any error encountered
type NodeRHCOSMatcher interface {
	GetRHCOSVersion(name string) (isRHCOS bool, ocpVersion string, rhcosVersion string, err error)
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

// GetRHCOSVersion returns information about the RHCOS version running on the node.
// It parses the osImage string to extract the OCP version and RHCOS version.
// For RHCOS 4.19+, the osImage format is "Red Hat Enterprise Linux CoreOS 4.19.0"
// and the actual RHCOS version (e.g., "9.6.20260217-1") comes from node inventory.
// For older RHCOS, the osImage contains the OCP-derived version directly (e.g., "418.94.xxx").
func (c *NodeIDMatcherImpl) GetRHCOSVersion(name string) (isRHCOS bool, ocpVersion string, rhcosVersion string, err error) {
	n := c.nodeStore.GetNode(name)
	if n == nil {
		return false, "", "", fmt.Errorf("cannot find node with name %q", name)
	}
	osImageRef := n.GetOsImage()
	if !strings.HasPrefix(osImageRef, rhcosFullName) {
		return false, "", osImageRef, nil
	}
	r := rhcosOSImageRegexp.FindStringSubmatch(osImageRef)
	// r[0] = entire match
	// r[1] = "Red Hat Enterprise Linux"
	// r[2] = "CoreOS"
	// r[3] = OCP major version (e.g., "4")
	// r[4] = OCP minor version (e.g., "19")
	// r[5] = RHCOS version suffix (e.g., ".0" or "")
	if len(r) < 5 {
		return false, "", osImageRef, fmt.Errorf("valid RHCOS prefix found, but cannot parse version from: %s", osImageRef)
	}
	ocpVersion = fmt.Sprintf("%s.%s", r[3], r[4])

	// For the RHCOS version from osImage, we combine what's in the osImage.
	// For older nodes (pre-4.19), this contains the full version like "418.94.xxx".
	// For 4.19+, this is just the OCP version like "4.19.0" - the actual RHCOS
	// version from /etc/os-release (e.g., "9.6.xxx") is provided separately via node inventory.
	rhcosVersionFromImage := r[3] + "." + r[4]
	if len(r) > 5 && r[5] != "" {
		// r[5] already includes the leading dot if present
		rhcosVersionFromImage = r[3] + "." + r[4] + r[5]
	}
	return true, ocpVersion, rhcosVersionFromImage, nil
}
