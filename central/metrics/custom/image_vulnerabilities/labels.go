package image_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/central/views/deploymentcve"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = tracker.LazyLabelGetters[*finding]{
	"Cluster":            func(f *finding) string { return f.ClusterName },
	"Namespace":          func(f *finding) string { return f.Namespace },
	"Deployment":         func(f *finding) string { return f.DeploymentName },
	"IsPlatformWorkload": isPlatformWorkload,

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

func isPlatformWorkload(f *finding) string {
	// MatchDeployment only needs the Namespace field, which we have in the
	// flattened finding.
	isPlatform, _ := platformmatcher.Singleton().MatchDeployment(&storage.Deployment{
		Namespace: f.Namespace,
	})
	return strconv.FormatBool(isPlatform)
}

type finding = deploymentcve.VulnFinding
