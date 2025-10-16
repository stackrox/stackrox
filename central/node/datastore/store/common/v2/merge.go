package common

import (
	"sort"

	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// Merge merges the node parts into a node.
func Merge(parts *NodeParts) *storage.Node {
	mergeComponents(parts, parts.Node)
	return parts.Node
}

func mergeComponents(parts *NodeParts, node *storage.Node) {
	// If the node has a nil scan, there is nothing to fill in.
	if node.GetScan() == nil {
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
		node.GetScan().SetComponents(append(node.GetScan().GetComponents(), generateEmbeddedComponent(cp)))
	}

	components := node.GetScan().GetComponents()
	sort.SliceStable(components, func(i, j int) bool {
		if components[i].GetName() == components[j].GetName() {
			return components[i].GetVersion() < components[j].GetVersion()
		}
		return components[i].GetName() < components[j].GetName()
	})
	for _, comp := range components {
		sort.SliceStable(comp.GetVulnerabilities(), func(i, j int) bool {
			return comp.GetVulnerabilities()[i].GetCveBaseInfo().GetCve() < comp.GetVulnerabilities()[j].GetCveBaseInfo().GetCve()
		})
	}
}

func generateEmbeddedComponent(cp *ComponentParts) *storage.EmbeddedNodeScanComponent {
	ret := &storage.EmbeddedNodeScanComponent{}
	ret.SetName(cp.Component.GetName())
	ret.SetVersion(cp.Component.GetVersion())
	ret.SetRiskScore(cp.Component.GetRiskScore())
	ret.SetPriority(cp.Component.GetPriority())

	if cp.Component.GetSetTopCvss() != nil {
		ret.Set_TopCvss(cp.Component.GetTopCvss())
	}

	ret.SetVulnerabilities(make([]*storage.NodeVulnerability, 0, len(cp.Children)))
	for _, cve := range cp.Children {
		if cve.CVE == nil {
			log.Errorf("UNEXPECTED: nil CVE when adding vulns for component %q", cp.Component.GetId())
			continue
		}
		ret.SetVulnerabilities(append(ret.GetVulnerabilities(), generateEmbeddedCVE(cve)))
	}
	return ret
}

func generateEmbeddedCVE(cp *CVEParts) *storage.NodeVulnerability {
	ret := utils.NodeCVEToNodeVulnerability(cp.CVE)
	if cp.Edge.GetFixedBy() != "" {
		ret.Set_FixedBy(cp.Edge.GetFixedBy())
	}
	return ret
}
