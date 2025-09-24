package image_vulnerabilities

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/generated/storage"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	{Label: "Namespace", Getter: func(f *finding) string { return f.deployment.GetNamespace() }},
	{Label: "Deployment", Getter: func(f *finding) string { return f.deployment.GetName() }},
	{Label: "IsPlatformWorkload", Getter: isPlatformWorkload},

	{Label: "ImageID", Getter: func(f *finding) string { return f.image.GetId() }},
	{Label: "ImageRegistry", Getter: func(f *finding) string { return f.name.GetRegistry() }},
	{Label: "ImageRemote", Getter: func(f *finding) string { return f.name.GetRemote() }},
	{Label: "ImageTag", Getter: func(f *finding) string { return f.name.GetTag() }},
	{Label: "Component", Getter: func(f *finding) string { return f.component.GetName() }},
	{Label: "ComponentVersion", Getter: func(f *finding) string { return f.component.GetVersion() }},
	{Label: "OperatingSystem", Getter: func(f *finding) string { return f.image.GetScan().GetOperatingSystem() }},

	{Label: "CVE", Getter: func(f *finding) string { return f.vuln.GetCve() }},
	{Label: "CVSS", Getter: func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) }},
	{Label: "Severity", Getter: func(f *finding) string { return f.vuln.GetSeverity().String() }},
	{Label: "SeverityV2", Getter: func(f *finding) string { return f.vuln.GetCvssV2().GetSeverity().String() }},
	{Label: "SeverityV3", Getter: func(f *finding) string { return f.vuln.GetCvssV3().GetSeverity().String() }},
	{Label: "IsFixable", Getter: func(f *finding) string { return strconv.FormatBool(f.vuln.GetFixedBy() != "") }},
}

// finding holds all information for computing any label in this category.
// The aggregator calls the lazy label's Getter function with every finding to
// compute the values for the list of defined labels.
type finding struct {
	err        error
	deployment *storage.Deployment
	image      *storage.Image
	name       *storage.ImageName
	component  *storage.EmbeddedImageScanComponent
	vuln       *storage.EmbeddedVulnerability
}

func (f *finding) GetError() error { return f.err }

func isPlatformWorkload(f *finding) string {
	p, _ := matcher.Singleton().MatchDeployment(f.deployment)
	return strconv.FormatBool(p)
}
