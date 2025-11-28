package image_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var lazyLabels = tracker.LazyLabelGetters[*finding]{
	"Cluster":    func(f *finding) string { return f.ClusterName },
	"Namespace":  func(f *finding) string { return f.Namespace },
	"Deployment": func(f *finding) string { return f.DeploymentName },
	// Note: IsPlatformWorkload removed - requires full deployment object which isn't available
	// in the flattened SQL result. Could be added if needed by joining deployment table.

	"ImageID":          func(f *finding) string { return f.ImageID },
	"ImageRegistry":    func(f *finding) string { return f.ImageRegistry },
	"ImageRemote":      func(f *finding) string { return f.ImageRemote },
	"ImageTag":         func(f *finding) string { return f.ImageTag },
	"Component":        func(f *finding) string { return f.ComponentName },
	"ComponentVersion": func(f *finding) string { return f.ComponentVersion },
	"OperatingSystem":  func(f *finding) string { return f.OperatingSystem },

	"CVE":             func(f *finding) string { return f.CVE },
	"CVSS":            func(f *finding) string { return strconv.FormatFloat(float64(f.CVSS), 'f', 1, 32) },
	"Severity":        func(f *finding) string { return f.Severity.String() },
	"EPSSProbability": func(f *finding) string { return strconv.FormatFloat(float64(f.EPSSProbability), 'f', 1, 32) },
	// Note: EPSSPercentile not available in denormalized image_cves_v2 table
	"IsFixable": func(f *finding) string { return strconv.FormatBool(f.FixedBy != "") },
}

type finding = store.DeploymentVulnFinding
