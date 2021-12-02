package dackbox

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
)

// Merge merges the node parts into a node.
func Merge(parts *NodeParts) *storage.Node {
	ret := parts.node.Clone()
	mergeComponents(parts, ret)
	return ret
}

func mergeComponents(parts *NodeParts, node *storage.Node) {
	// If the node has a nil scan, there is nothing to fill in.
	if node.Scan == nil {
		return
	}

	// Use the edges to combine into the parent node.
	for _, cp := range parts.children {
		// Parse the IDs of the edge.
		nodeComponentEdgeIDs, err := edges.FromString(cp.edge.GetId())
		if err != nil {
			log.Error(err)
			continue
		}
		if nodeComponentEdgeIDs.ParentID != node.GetId() {
			log.Error("node to component edge does not match node")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		node.Scan.Components = append(node.Scan.Components, generateEmbeddedComponent(cp, parts.nodeCVEEdges))
	}
}

func generateEmbeddedComponent(cp *ComponentParts, nodeCVEEdges map[string]*storage.NodeCVEEdge) *storage.EmbeddedNodeScanComponent {
	if cp.component == nil || cp.edge == nil {
		return nil
	}
	ret := &storage.EmbeddedNodeScanComponent{
		Name:      cp.component.GetName(),
		Version:   cp.component.GetVersion(),
		RiskScore: cp.component.GetRiskScore(),
	}

	if cp.component.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: cp.component.GetTopCvss()}
	}

	ret.Vulns = make([]*storage.EmbeddedVulnerability, 0, len(cp.children))
	for _, cve := range cp.children {
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(cve, nodeCVEEdges[cve.cve.GetId()]))
	}
	return ret
}

func generateEmbeddedCVE(cp *CVEParts, nodeCVEEdge *storage.NodeCVEEdge) *storage.EmbeddedVulnerability {
	if cp.cve == nil || cp.edge == nil {
		return nil
	}

	ret := converter.ProtoCVEToEmbeddedCVE(cp.cve)
	if cp.edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.edge.GetFixedBy(),
		}
	}
	// The `Suppressed` field is transferred to `State` field in `converter.ProtoCVEToEmbeddedCVE` and node cve deferral
	// through vuln risk management workflow is not supported, hence, nothing to do here.
	ret.FirstNodeOccurrence = nodeCVEEdge.GetFirstNodeOccurrence()
	return ret
}
