package common

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// Split splits the input node into a set of parts.
func Split(node *storage.Node, withComponents bool) *NodeParts {
	parts := &NodeParts{
		Node: node.Clone(),
	}

	if withComponents {
		parts.Children = splitComponents(parts)
	}

	// Clear components in the top level node.
	if parts.Node.GetScan() != nil {
		parts.Node.Scan.Components = nil
	}

	return parts
}

func splitComponents(parts *NodeParts) []*ComponentParts {
	components := parts.Node.GetScan().GetComponents()
	ret := make([]*ComponentParts, 0, len(components))
	for _, component := range components {
		cp := &ComponentParts{}
		cp.Component = generateNodeComponent(component)
		cp.Edge = generateNodeComponentEdge(parts.Node, cp.Component)
		cp.Children = splitCVEs(parts.Node.GetScan().GetOperatingSystem(), cp, component)

		ret = append(ret, cp)
	}

	return ret
}

func splitCVEs(os string, component *ComponentParts, embedded *storage.EmbeddedNodeScanComponent) []*CVEParts {
	cves := embedded.GetVulns()
	ret := make([]*CVEParts, 0, len(cves))
	for _, cve := range cves {
		cp := &CVEParts{}
		cp.CVE = converter.EmbeddedCVEToProtoCVE(os, cve)
		cp.Edge = generateComponentCVEEdge(component.Component, cp.CVE, cve)

		ret = append(ret, cp)
	}

	return ret
}

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{
		Id:        edges.EdgeID{ParentID: convertedComponent.GetId(), ChildID: convertedCVE.GetId()}.ToString(),
		IsFixable: embedded.GetFixedBy() != "",
	}
	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

func generateNodeComponent(from *storage.EmbeddedNodeScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:        scancomponent.ComponentID(from.GetName(), from.GetVersion(), ""),
		Name:      from.GetName(),
		Version:   from.GetVersion(),
		Source:    storage.SourceType_INFRASTRUCTURE,
		RiskScore: from.GetRiskScore(),
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateNodeComponentEdge(node *storage.Node, converted *storage.ImageComponent) *storage.NodeComponentEdge {
	return &storage.NodeComponentEdge{
		Id: edges.EdgeID{ParentID: node.GetId(), ChildID: converted.GetId()}.ToString(),
	}
}
