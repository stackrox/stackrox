package node_vulnerabilities

import (
	"context"
	"iter"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	cveDS "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var getters = []common.LabelGetter[*finding]{
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
	common.OneOrMore
	node      *storage.Node
	component *storage.EmbeddedNodeScanComponent
	vuln      *storage.EmbeddedVulnerability
}

type datastores struct {
	nDS   nodeDS.DataStore
	cveDS cveDS.DataStore
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*finding] {
	tc := common.MakeTrackerConfig(
		"node vulnerabilities",
		"aggregated node CVEs",
		getters,
		common.Bind4th(trackVulnerabilityMetrics, datastores{nodeDS.Singleton(), cveDS.Singleton()}),
		gauge)
	return tc
}

func trackVulnerabilityMetrics(ctx context.Context, query *v1.Query, mcfg common.MetricsConfiguration, ds datastores) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.nDS.WalkByQuery(ctx, query, func(node *storage.Node) error {
			f.node = node
			if !forEachFinding(yield, &f) {
				return common.ErrStopIterator
			}
			return nil
		})
	}
}

func forEachFinding(yield func(*finding) bool, f *finding) bool {
	for _, f.component = range f.node.GetScan().GetComponents() {
		for _, f.vuln = range f.component.GetVulns() {
			if !yield(f) {
				return false
			}
		}
	}
	return true
}
