package common

import (
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
)

// Split splits the input node into a set of parts.
func Split(node *storage.Node, withComponents bool) *NodeParts {
	parts := &NodeParts{
		Node: node.CloneVT(),
	}

	if withComponents {
		parts.Children = splitComponents(parts)
	}

	// Clear components in the top level node.
	if parts.Node.GetScan() != nil {
		parts.Node.GetScan().SetComponents(nil)
	}

	return parts
}

func splitComponents(parts *NodeParts) []*ComponentParts {
	os := parts.Node.GetScan().GetOperatingSystem()
	components := parts.Node.GetScan().GetComponents()
	addedComponents := set.NewStringSet()
	ret := make([]*ComponentParts, 0, len(components))
	for _, component := range parts.Node.GetScan().GetComponents() {
		generatedComponent := GenerateNodeComponent(os, component)
		if !addedComponents.Add(generatedComponent.GetId()) {
			continue
		}
		cp := &ComponentParts{
			Component: generatedComponent,
		}
		cp.Edge = generateNodeComponentEdge(parts.Node, cp.Component)
		cp.Children = splitCVEs(os, cp, component)

		ret = append(ret, cp)
	}
	return ret
}

func splitCVEs(os string, component *ComponentParts, embedded *storage.EmbeddedNodeScanComponent) []*CVEParts {
	cves := embedded.GetVulnerabilities()
	addedCVEs := set.NewStringSet()
	ret := make([]*CVEParts, 0, len(cves))
	for _, cve := range cves {
		generatedCVE := utils.NodeVulnerabilityToNodeCVE(os, cve)
		if !addedCVEs.Add(generatedCVE.GetId()) {
			continue
		}
		cp := &CVEParts{
			CVE: generatedCVE,
		}
		cp.Edge = generateComponentCVEEdge(component.Component, cp.CVE, cve)

		ret = append(ret, cp)
	}
	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.NodeComponent, convertedCVE *storage.NodeCVE, embedded *storage.NodeVulnerability) *storage.NodeComponentCVEEdge {
	ret := &storage.NodeComponentCVEEdge{}
	ret.SetId(pgSearch.IDFromPks([]string{convertedComponent.GetId(), convertedCVE.GetId()}))
	ret.SetIsFixable(embedded.GetFixedBy() != "")
	ret.SetNodeCveId(convertedCVE.GetId())
	ret.SetNodeComponentId(convertedComponent.GetId())

	if ret.GetIsFixable() {
		ret.Set_FixedBy(embedded.GetFixedBy())
	}
	return ret
}

// GenerateNodeComponent returns top-level node component from embedded component.
func GenerateNodeComponent(os string, from *storage.EmbeddedNodeScanComponent) *storage.NodeComponent {
	ret := &storage.NodeComponent{}
	ret.SetId(scancomponent.ComponentID(from.GetName(), from.GetVersion(), os))
	ret.SetOperatingSystem(os)
	ret.SetName(from.GetName())
	ret.SetVersion(from.GetVersion())
	ret.SetRiskScore(from.GetRiskScore())
	ret.SetPriority(from.GetPriority())

	if from.GetSetTopCvss() != nil {
		ret.Set_TopCvss(from.GetTopCvss())
	}
	return ret
}

func generateNodeComponentEdge(node *storage.Node, convertedComponent *storage.NodeComponent) *storage.NodeComponentEdge {
	ret := &storage.NodeComponentEdge{}
	ret.SetId(pgSearch.IDFromPks([]string{node.GetId(), convertedComponent.GetId()}))
	ret.SetNodeId(node.GetId())
	ret.SetNodeComponentId(convertedComponent.GetId())
	return ret
}
