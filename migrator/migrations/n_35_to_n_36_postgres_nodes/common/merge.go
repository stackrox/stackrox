package common

import (
	"github.com/stackrox/rox/generated/storage"
	converter "github.com/stackrox/rox/migrator/migrations/cvehelper"
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

	os := node.GetScan().GetOperatingSystem()
	// Use the edges to combine into the parent node.
	for _, cp := range parts.Children {
		// Parse the IDs of the edge.
		nodeComponentEdgeID, err := edges.FromString(cp.Edge.GetId())
		if err != nil {
			log.WriteToStderrf("%v", err)
			continue
		}
		if nodeComponentEdgeID.ParentID != node.GetId() {
			log.WriteToStderr("node to component edge does not match node")
			continue
		}

		// Generate an embedded component for the edge and non-embedded version.
		node.Scan.Components = append(node.Scan.Components, generateEmbeddedComponent(os, cp))
	}
}

func generateEmbeddedComponent(os string, cp *ComponentParts) *storage.EmbeddedNodeScanComponent {
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
		ret.Vulns = append(ret.Vulns, generateEmbeddedCVE(os, cve))
	}
	return ret
}

func generateEmbeddedCVE(os string, cp *CVEParts) *storage.EmbeddedVulnerability {
	if cp.CVE == nil || cp.Edge == nil {
		return nil
	}

	ret := converter.ProtoCVEToEmbeddedCVE(cp.CVE)
	if cp.Edge.IsFixable {
		ret.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: cp.Edge.GetFixedBy(),
		}
	}

	// Only legacy vuln snoozing feature affected node vulns state.
	if ret.GetSuppressed() {
		ret.State = storage.VulnerabilityState_DEFERRED
	}

	if distroSpecifics, ok := cp.CVE.GetDistroSpecifics()[os]; ok {
		ret.Severity = distroSpecifics.GetSeverity()
		ret.Cvss = distroSpecifics.GetCvss()
		ret.CvssV2 = distroSpecifics.GetCvssV2()
		ret.CvssV3 = distroSpecifics.GetCvssV3()
		ret.ScoreVersion = converter.CVEScoreVersionToEmbeddedScoreVersion(distroSpecifics.GetScoreVersion())
	}

	// The `Suppressed` field is transferred to `State` field in `converter.ProtoCVEToEmbeddedCVE` and node cve deferral
	// through vuln risk management workflow is not supported, hence, nothing to do here.
	return ret
}
