package image_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = tracker.LazyLabelGetters[*finding]{
	"Cluster":            func(f *finding) string { return f.deployment.GetClusterName() },
	"Namespace":          func(f *finding) string { return f.deployment.GetNamespace() },
	"Deployment":         func(f *finding) string { return f.deployment.GetName() },
	"IsPlatformWorkload": isPlatformWorkload,

	"ImageID":          func(f *finding) string { return f.image.GetId() },
	"ImageRegistry":    func(f *finding) string { return f.name.GetRegistry() },
	"ImageRemote":      func(f *finding) string { return f.name.GetRemote() },
	"ImageTag":         func(f *finding) string { return f.name.GetTag() },
	"Component":        func(f *finding) string { return f.component.GetName() },
	"ComponentVersion": func(f *finding) string { return f.component.GetVersion() },
	"OperatingSystem":  func(f *finding) string { return f.image.GetScan().GetOperatingSystem() },

	"CVE":      func(f *finding) string { return f.vuln.GetCve() },
	"CVSS":     func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) },
	"Severity": func(f *finding) string { return f.vuln.GetSeverity().String() },
	"EPSSProbability": func(f *finding) string {
		return strconv.FormatFloat(float64(f.vuln.GetEpss().GetEpssProbability()), 'f', 1, 32)
	},
	"EPSSPercentile": func(f *finding) string {
		return strconv.FormatFloat(float64(f.vuln.GetEpss().GetEpssPercentile()), 'f', 1, 32)
	},
	"IsFixable": func(f *finding) string { return strconv.FormatBool(f.vuln.GetFixedBy() != "") },
}

// finding holds all information for computing any label in this category.
// The aggregator calls the lazy label's Getter function with every finding to
// compute the values for the list of defined labels.
type finding struct {
	deployment *storage.Deployment
	image      *storage.Image
	name       *storage.ImageName
	component  *storage.EmbeddedImageScanComponent
	vuln       *storage.EmbeddedVulnerability
}

func isPlatformWorkload(f *finding) string {
	p, _ := matcher.Singleton().MatchDeployment(f.deployment)
	return strconv.FormatBool(p)
}
