package common

import (
	"github.com/stackrox/rox/generated/storage"
	converter "github.com/stackrox/rox/migrator/migrations/cvehelper"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/set"
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
	os := parts.Node.GetScan().GetOperatingSystem()
	components := parts.Node.GetScan().GetComponents()
	addedComponents := set.NewStringSet()
	ret := make([]*ComponentParts, 0, len(components))
	for _, component := range parts.Node.GetScan().GetComponents() {
		generatedComponent := generateNodeComponent(os, component)
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
	cves := embedded.GetVulns()
	addedCVEs := set.NewStringSet()
	ret := make([]*CVEParts, 0, len(cves))
	for _, cve := range cves {
		generatedCVE := converter.EmbeddedCVEToProtoCVE(os, cve, false)
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

func generateComponentCVEEdge(convertedComponent *storage.ImageComponent, convertedCVE *storage.CVE, embedded *storage.EmbeddedVulnerability) *storage.ComponentCVEEdge {
	ret := &storage.ComponentCVEEdge{
		Id:               edges.EdgeID{ParentID: convertedComponent.GetId(), ChildID: convertedCVE.GetId()}.ToString(),
		IsFixable:        embedded.GetFixedBy() != "",
		ImageCveId:       convertedCVE.GetId(),
		ImageComponentId: convertedComponent.GetId(),
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ComponentCVEEdge_FixedBy{
			FixedBy: embedded.GetFixedBy(),
		}
	}
	return ret
}

func generateNodeComponent(os string, from *storage.EmbeddedNodeScanComponent) *storage.ImageComponent {
	ret := &storage.ImageComponent{
		Id:        scancomponent.ComponentID(from.GetName(), from.GetVersion(), os),
		Name:      from.GetName(),
		Version:   from.GetVersion(),
		Source:    storage.SourceType_INFRASTRUCTURE,
		RiskScore: from.GetRiskScore(),
		Priority:  from.GetPriority(),
	}

	if from.GetSetTopCvss() != nil {
		ret.SetTopCvss = &storage.ImageComponent_TopCvss{TopCvss: from.GetTopCvss()}
	}
	return ret
}

func generateNodeComponentEdge(node *storage.Node, convertedComponent *storage.ImageComponent) *storage.NodeComponentEdge {
	ret := &storage.NodeComponentEdge{
		Id:              edges.EdgeID{ParentID: node.GetId(), ChildID: convertedComponent.GetId()}.ToString(),
		NodeId:          node.GetId(),
		NodeComponentId: convertedComponent.GetId(),
	}
	return ret
}
