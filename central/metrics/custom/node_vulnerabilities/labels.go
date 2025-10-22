package node_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = tracker.LazyLabelGetters[*finding]{
	"Cluster":          func(f *finding) string { return f.node.GetClusterName() },
	"Node":             func(f *finding) string { return f.node.GetName() },
	"Kernel":           func(f *finding) string { return f.node.GetKernelVersion() },
	"OperatingSystem":  func(f *finding) string { return f.node.GetScan().GetOperatingSystem() },
	"OSImage":          func(f *finding) string { return f.node.GetOsImage() },
	"Component":        func(f *finding) string { return f.component.GetName() },
	"ComponentVersion": func(f *finding) string { return f.component.GetVersion() },

	"CVE":       func(f *finding) string { return f.vulnerability.GetCveBaseInfo().GetCve() },
	"CVSS":      func(f *finding) string { return strconv.FormatFloat(float64(f.vulnerability.GetCvss()), 'f', 1, 32) },
	"Severity":  func(f *finding) string { return f.vulnerability.GetSeverity().String() },
	"IsFixable": func(f *finding) string { return strconv.FormatBool(f.vulnerability.GetFixedBy() != "") },
	"IsSnoozed": func(f *finding) string { return strconv.FormatBool(f.vulnerability.GetSnoozed()) },
	"EPSSPercentile": func(f *finding) string {
		return strconv.FormatFloat(float64(f.vulnerability.GetCveBaseInfo().GetEpss().GetEpssPercentile()), 'f', 1, 32)
	},
	"EPSSProbability": func(f *finding) string {
		return strconv.FormatFloat(float64(f.vulnerability.GetCveBaseInfo().GetEpss().GetEpssProbability()), 'f', 1, 32)
	},
}

type finding struct {
	node          *storage.Node
	component     *storage.EmbeddedNodeScanComponent
	vulnerability *storage.NodeVulnerability
}
