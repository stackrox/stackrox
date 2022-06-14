package google

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	v2 "github.com/stackrox/rox/pkg/cvss/cvssv2"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1beta1/grafeas"
)

const (
	noteNamePrefix = "projects/goog-vulnz/notes/"
)

func (c *googleScanner) convertComponentFromPackageAndVersion(pv packageAndVersion) *storage.EmbeddedImageScanComponent {
	component := &storage.EmbeddedImageScanComponent{
		Name:    pv.name,
		Version: pv.version,
	}
	return component
}

func (c *googleScanner) processOccurrences(o *grafeas.Occurrence, convertChan chan *storage.EmbeddedVulnerability) {
	convertChan <- c.convertVulnsFromOccurrence(o)
}

func (c *googleScanner) convertVulnsFromOccurrences(occurrences []*grafeas.Occurrence) []*storage.EmbeddedVulnerability {
	// Parallelize this as it makes a bunch of calls to the API
	convertChan := make(chan *storage.EmbeddedVulnerability)
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(occurrences))
	for _, o := range occurrences {
		go c.processOccurrences(o, convertChan)
	}
	for range occurrences {
		if vuln := <-convertChan; vuln != nil {
			vulns = append(vulns, vuln)
		}
	}
	return vulns
}

func (c *googleScanner) getSummary(occurrence *grafeas.Occurrence) string {
	ctx, cancel := grpcContext()
	defer cancel()
	note, err := c.betaClient.GetOccurrenceNote(ctx, &grafeas.GetOccurrenceNoteRequest{Name: occurrence.GetName()})
	if err != nil {
		return "Unknown CVE description"
	}
	for _, l := range note.GetVulnerability().GetDetails() {
		if l.CpeUri == occurrence.GetVulnerability().GetPackageIssue()[0].GetAffectedLocation().GetCpeUri() {
			return l.Description
		}
	}
	return "Unknown CVE description"
}

func getCVEName(occ *grafeas.Occurrence) string {
	return strings.TrimPrefix(occ.GetNoteName(), noteNamePrefix)
}

func (c *googleScanner) convertVulnsFromOccurrence(occurrence *grafeas.Occurrence) *storage.EmbeddedVulnerability {
	vulnerability := occurrence.GetVulnerability()

	packageIssues := vulnerability.GetPackageIssue()
	if len(packageIssues) == 0 {
		return nil
	}
	pkgIssue := packageIssues[0]

	cveName := getCVEName(occurrence)
	if cveName == "" {
		return nil
	}

	var link string
	for _, url := range vulnerability.RelatedUrls {
		if url.Url != "" {
			link = url.Url
			break
		}
	}
	if link == "" {
		link = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveName)
	}

	vuln := &storage.EmbeddedVulnerability{
		Cve:     cveName,
		Link:    link,
		Cvss:    vulnerability.GetCvssScore(),
		Summary: c.getSummary(occurrence),
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: pkgIssue.GetFixedLocation().GetVersion().GetRevision(),
		},
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		// On looking up cvss for a cve (CVE-2015-5186 in image gcr.io/ultra-current-825/srox/jump-host:latest) on nvd,
		// it is concluded that score version is v2.
		ScoreVersion: storage.EmbeddedVulnerability_V2,
	}

	vuln.CvssV2 = &storage.CVSSV2{}

	if cvssVector, err := v2.ParseCVSSV2(strings.TrimPrefix(vulnerability.LongDescription, "NIST vectors: ")); err == nil {
		vuln.CvssV2 = cvssVector
	}

	vuln.CvssV2.Severity = v2.Severity(vulnerability.GetCvssScore())

	return vuln
}
