package node_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.node.GetClusterName() }},
	{Label: "Node", Getter: func(f *finding) string { return f.node.GetName() }},
	{Label: "Kernel", Getter: func(f *finding) string { return f.node.GetKernelVersion() }},
	{Label: "OperatingSystem", Getter: func(f *finding) string { return f.node.GetScan().GetOperatingSystem() }},
	{Label: "OSImage", Getter: func(f *finding) string { return f.node.GetOsImage() }},
	{Label: "Component", Getter: func(f *finding) string { return f.component.GetName() }},
	{Label: "ComponentVersion", Getter: func(f *finding) string { return f.component.GetVersion() }},

	{Label: "CVE", Getter: func(f *finding) string { return f.vulnerability.GetCveBaseInfo().GetCve() }},
	{Label: "CVSS", Getter: func(f *finding) string { return strconv.FormatFloat(float64(f.vulnerability.GetCvss()), 'f', 1, 32) }},
	{Label: "Severity", Getter: func(f *finding) string { return f.vulnerability.GetSeverity().String() }},
	{Label: "IsFixable", Getter: func(f *finding) string { return strconv.FormatBool(f.vulnerability.GetFixedBy() != "") }},
	{Label: "IsSnoozed", Getter: func(f *finding) string { return strconv.FormatBool(f.vulnerability.GetSnoozed()) }},
	{Label: "EPSSPercentile", Getter: func(f *finding) string {
		return strconv.FormatFloat(float64(f.vulnerability.GetCveBaseInfo().GetEpss().GetEpssPercentile()), 'f', 1, 32)
	}},
	{Label: "EPSSProbability", Getter: func(f *finding) string {
		return strconv.FormatFloat(float64(f.vulnerability.GetCveBaseInfo().GetEpss().GetEpssProbability()), 'f', 1, 32)
	}},
}

type finding struct {
	tracker.CommonFinding
	err           error
	node          *storage.Node
	component     *storage.EmbeddedNodeScanComponent
	vulnerability *storage.NodeVulnerability
}

func (f *finding) GetError() error { return f.err }
