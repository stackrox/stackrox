package common

import (
	"sort"

	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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
		parts := pgSearch.IDToParts(cp.Edge.GetId())
		if len(parts) == 0 {
			log.Error("node to component edge does not have primary keys")
			continue
		}
		nodeID := parts[0]
		if nodeID != node.GetId() {
			log.Error("node to component edge does not match node")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		if cp.Component == nil {
			log.Errorf("UNEXPECTED: nil component when retrieving components for node %q", nodeID)
			continue
		}
		node.Scan.Components = append(node.Scan.Components, generateEmbeddedComponent(cp))
	}

	components := node.GetScan().GetComponents()
	sort.SliceStable(components, func(i, j int) bool {
		if components[i].GetName() == components[j].GetName() {
			return components[i].GetVersion() < components[j].GetVersion()
		}
		return components[i].GetName() < components[j].GetName()
	})
	for _, comp := range components {
		sort.SliceStable(comp.Vulnerabilities, func(i, j int) bool {
			return comp.Vulnerabilities[i].GetCveBaseInfo().GetCve() < comp.Vulnerabilities[j].GetCveBaseInfo().GetCve()
		})
	}
}

func generateEmbeddedComponent(cp *ComponentParts) *storage.EmbeddedNodeScanComponent {
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
		if cve.CVE == nil {
			log.Errorf("UNEXPECTED: nil CVE when adding vulns for component %q", cp.Component.GetId())
			continue
		}
		ret.Vulnerabilities = append(ret.Vulnerabilities, generateEmbeddedCVE(cve))
	}
	return ret
}

func generateEmbeddedCVE(cp *CVEParts) *storage.NodeVulnerability {
	ret := utils.NodeCVEToNodeVulnerability(cp.CVE)
	if cp.Edge.GetFixedBy() != "" {
		ret.SetFixedBy = &storage.NodeVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}
	return ret
}
