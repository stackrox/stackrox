package google

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/genproto/googleapis/devtools/containeranalysis/v1beta1/grafeas"
)

const (
	noteNamePrefix = "projects/goog-vulnz/notes/"
)

func (c *googleScanner) convertComponentFromPackageAndVersion(pv packageAndVersion) *storage.ImageScanComponent {
	component := &storage.ImageScanComponent{
		Name:    pv.name,
		Version: pv.version,
	}
	return component
}

func (c *googleScanner) processOccurrences(o *grafeas.Occurrence, convertChan chan *storage.Vulnerability) {
	convertChan <- c.convertVulnsFromOccurrence(o)
}

func (c *googleScanner) convertVulnsFromOccurrences(occurrences []*grafeas.Occurrence) []*storage.Vulnerability {
	// Parallelize this as it makes a bunch of calls to the API
	convertChan := make(chan *storage.Vulnerability)
	vulns := make([]*storage.Vulnerability, 0, len(occurrences))
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

func (c *googleScanner) convertVulnsFromOccurrence(occurrence *grafeas.Occurrence) *storage.Vulnerability {
	vulnerability := occurrence.GetVulnerability()

	packageIssues := vulnerability.GetPackageIssue()
	if len(packageIssues) == 0 {
		return nil
	}
	pkgIssue := packageIssues[0]

	var link string
	if len(vulnerability.RelatedUrls) != 0 {
		link = vulnerability.GetRelatedUrls()[0].GetUrl()
	}

	cveName := getCVEName(occurrence)
	if cveName == "" {
		return nil
	}

	vuln := &storage.Vulnerability{
		Cve:     cveName,
		Link:    link,
		Cvss:    vulnerability.GetCvssScore(),
		Summary: stringutils.Truncate(c.getSummary(occurrence), 64, stringutils.WordOriented{}),
		SetFixedBy: &storage.Vulnerability_FixedBy{
			FixedBy: pkgIssue.GetFixedLocation().GetVersion().GetRevision(),
		},
	}

	if cvssVector, err := cvss.ParseCVSSV2(strings.TrimPrefix(vulnerability.LongDescription, "NIST vectors: ")); err == nil {
		vuln.CvssV2 = cvssVector
	}
	return vuln
}
