package google

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1alpha1"
)

func (g *googleScanner) convertComponentFromPackageManagerOccurrence(occurrence *containeranalysis.Occurrence) (string, *storage.ImageScanComponent) {
	location := occurrence.GetInstallation().GetLocation()[0]
	version := location.GetVersion()
	component := &storage.ImageScanComponent{
		Name:    occurrence.GetInstallation().GetName(),
		Version: version.GetName() + "-" + version.GetRevision(),
	}
	return location.GetCpeUri(), component
}

func (g *googleScanner) convertVulnerabilityFromPackageVulnerabilityOccurrence(occurrence *containeranalysis.Occurrence, note *containeranalysis.Note) (string, string, *storage.Vulnerability) {
	packageIssues := occurrence.GetVulnerabilityDetails().GetPackageIssue()
	if len(packageIssues) == 0 {
		return "", "", nil
	}

	pkgIssue := packageIssues[0]
	affectedLocation := pkgIssue.GetAffectedLocation()
	var link string
	if len(note.GetRelatedUrl()) != 0 {
		link = note.GetRelatedUrl()[0].GetUrl()
	}
	summary := g.getSummary(note.GetVulnerabilityType().GetDetails(), affectedLocation.GetCpeUri())

	vuln := &storage.Vulnerability{
		Cve:     note.GetShortDescription(),
		Link:    link,
		Cvss:    occurrence.GetVulnerabilityDetails().GetCvssScore(),
		Summary: summary,
		SetFixedBy: &storage.Vulnerability_FixedBy{
			FixedBy: pkgIssue.GetFixedLocation().GetVersion().GetRevision(),
		},
	}

	if cvssVector, err := cvss.ParseCVSSV2(strings.TrimPrefix(note.LongDescription, "NIST vectors: ")); err == nil {
		vuln.CvssV2 = cvssVector
	}
	return affectedLocation.GetCpeUri(), affectedLocation.GetPackage(), vuln
}

// getSummary searches through the details and returns the summary that matches the cpeURI
func (g googleScanner) getSummary(details []*containeranalysis.VulnerabilityType_Detail, cpeURI string) string {
	for _, detail := range details {
		if detail.CpeUri == cpeURI {
			return detail.Description
		}
	}
	return ""
}
