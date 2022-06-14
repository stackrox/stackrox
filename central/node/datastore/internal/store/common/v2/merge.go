package common

import (
	"sort"

	"github.com/stackrox/stackrox/central/cve/converter"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/search/postgres"
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
		var nodeID string
		if features.PostgresDatastore.Enabled() {
			parts := postgres.IDToParts(cp.Edge.GetId())
			if len(parts) == 0 {
				log.Error("node to component edge does not have primary keys")
				continue
			}
			nodeID = parts[0]
		} else {
			// Parse the IDs of the edge.
			imageComponentEdgeID, err := edges.FromString(cp.Edge.GetId())
			if err != nil {
				log.Error(err)
				continue
			}
			nodeID = imageComponentEdgeID.ParentID
		}
		if nodeID != node.GetId() {
			log.Error("node to component edge does not match node")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		node.Scan.Components = append(node.Scan.Components, generateEmbeddedComponent(cp))
	}

	sort.SliceStable(node.GetScan().GetComponents(), func(i, j int) bool {
		if node.GetScan().GetComponents()[i].GetName() == node.GetScan().GetComponents()[j].GetName() {
			return node.GetScan().GetComponents()[i].GetVersion() < node.GetScan().GetComponents()[j].GetVersion()
		}
		return node.GetScan().GetComponents()[i].GetName() < node.GetScan().GetComponents()[j].GetName()
	})
	for _, comp := range node.GetScan().GetComponents() {
		sort.SliceStable(comp.Vulnerabilities, func(i, j int) bool {
			return comp.Vulnerabilities[i].GetCveBaseInfo().GetCve() < comp.Vulnerabilities[j].GetCveBaseInfo().GetCve()
		})
	}
}

func generateEmbeddedComponent(cp *ComponentParts) *storage.EmbeddedNodeScanComponent {
	if cp.Component == nil {
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

	ret.Vulnerabilities = make([]*storage.NodeVulnerability, 0, len(cp.Children))
	for _, cve := range cp.Children {
		ret.Vulnerabilities = append(ret.Vulnerabilities, generateEmbeddedCVE(cve))
	}
	return ret
}

func generateEmbeddedCVE(cp *CVEParts) *storage.NodeVulnerability {
	if cp.CVE == nil {
		return nil
	}
	ret := converter.NodeCVEToNodeVulnerability(cp.CVE)
	if cp.Edge.GetFixedBy() != "" {
		ret.SetFixedBy = &storage.NodeVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}
	return ret
}
