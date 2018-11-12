package google

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/cvss"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1alpha1"
)

func (c *googleScanner) convertComponentFromPackageManagerOccurrence(occurrence *containeranalysis.Occurrence) (string, *v1.ImageScanComponent) {
	location := occurrence.GetInstallation().GetLocation()[0]
	version := location.GetVersion()
	component := &v1.ImageScanComponent{
		Name:    occurrence.GetInstallation().GetName(),
		Version: version.GetName() + "-" + version.GetRevision(),
	}
	return location.GetCpeUri(), component
}

func (c *googleScanner) convertVulnerabilityFromPackageVulnerabilityOccurrence(occurrence *containeranalysis.Occurrence, note *containeranalysis.Note) (string, string, *v1.Vulnerability) {
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
	summary := c.getSummary(note.GetVulnerabilityType().GetDetails(), affectedLocation.GetCpeUri())

	vuln := &v1.Vulnerability{
		Cve:     note.GetShortDescription(),
		Link:    link,
		Cvss:    occurrence.GetVulnerabilityDetails().GetCvssScore(),
		Summary: summary,
		SetFixedBy: &v1.Vulnerability_FixedBy{
			FixedBy: pkgIssue.GetFixedLocation().GetVersion().GetRevision(),
		},
	}

	if cvssVector, err := cvss.ParseCVSSV2(strings.TrimPrefix(note.LongDescription, "NIST vectors: ")); err == nil {
		vuln.CvssV2 = cvssVector
	}
	return affectedLocation.GetCpeUri(), affectedLocation.GetPackage(), vuln
}

// getSummary searches through the details and returns the summary that matches the cpeURI
func (c googleScanner) getSummary(details []*containeranalysis.VulnerabilityType_Detail, cpeURI string) string {
	for _, detail := range details {
		if detail.CpeUri == cpeURI {
			return detail.Description
		}
	}
	return ""
}
