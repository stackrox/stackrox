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

	{Label: "CVE", Getter: func(f *finding) string { return f.vuln.GetCve() }},
	{Label: "CVSS", Getter: func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) }},
	{Label: "Severity", Getter: func(f *finding) string { return f.vuln.GetSeverity().String() }},
	{Label: "SeverityV2", Getter: func(f *finding) string { return f.vuln.GetCvssV2().GetSeverity().String() }},
	{Label: "SeverityV3", Getter: func(f *finding) string { return f.vuln.GetCvssV3().GetSeverity().String() }},
	{Label: "IsFixable", Getter: func(f *finding) string { return strconv.FormatBool(f.vuln.GetFixedBy() != "") }},
	{Label: "IsSuppressed", Getter: func(f *finding) string { return strconv.FormatBool(f.vuln.GetSuppressed()) }},
}

type finding struct {
	err       error
	node      *storage.Node
	component *storage.EmbeddedNodeScanComponent
	vuln      *storage.EmbeddedVulnerability
}

func (f *finding) GetError() error { return f.err }
