package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
)

// Merge merges the node parts into a node.
func Merge(parts *NodeParts) *storage.Node {
	ret := parts.Node.Clone()
	mergeComponents(parts, ret)
	return ret
}

func mergeComponents(parts *NodeParts, node *storage.Node) {
	// If the node has a nil scan, there is nothing to fill in.
	if node.Scan == nil {
		return
	}

	// Use the edges to combine into the parent node.
	for _, cp := range parts.Children {
		// Parse the IDs of the edge.
		nodeComponentEdgeID, err := edges.FromString(cp.Edge.GetId())
		if err != nil {
			log.Error(err)
			continue
		}
		if nodeComponentEdgeID.ParentID != node.GetId() {
			log.Error("node to component edge does not match node")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		node.Scan.Components = append(node.Scan.Components, generateEmbeddedComponent(cp))
	}
}

func generateEmbeddedComponent(cp *ComponentParts) *storage.EmbeddedNodeScanComponent {
	if cp.Component == nil || cp.Edge == nil {
		return nil
	}
	ret := &storage.EmbeddedNodeScanComponent{
		Name:      cp.Component.GetName(),
		Version:   cp.Component.GetVersion(),
		RiskScore: cp.Component.GetRiskScore(),
		Priority:  cp.Component.GetPriority(),
	}

	if cp.Component.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: cp.Component.GetTopCvss()}
	}

	ret.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(cp.Children))
	for _, cve := range cp.Children {
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(cve))
	}
	return ret
}

func generateEmbeddedCVE(cp *CVEParts) *storage.EmbeddedVulnerability {
	if cp.CVE == nil || cp.Edge == nil {
		return nil
	}

	ret := utils.ProtoCVEToEmbeddedCVE(cp.CVE)
	if cp.Edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}

	// Only legacy vuln snoozing feature affected node vulns state.
	if ret.GetSuppressed() {
		ret.State = storage.VulnerabilityState_DEFERRED
	}

	// The `Suppressed` field is transferred to `State` field in `converter.ProtoCVEToEmbeddedCVE` and node cve deferral
	// through vuln risk management workflow is not supported, hence, nothing to do here.
	return ret
}
