package csaf

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/klauspost/compress/snappy"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types/csaf"
	pkgcsaf "github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
)

// ParseEnrichment implements driver.EnrichmentUpdater.
// The contents should be a line-delimited list of CSAF data, all of which is Snappy-compressed.
// This method parses out the data the enricher cares about and marshals the result into JSON.
func (e *Enricher) ParseEnrichment(_ context.Context, contents io.ReadCloser) ([]driver.EnrichmentRecord, error) {
	records := make(map[string]pkgcsaf.Advisory)

	r := bufio.NewReader(snappy.NewReader(contents))
	for b, err := r.ReadBytes('\n'); err == nil; b, err = r.ReadBytes('\n') {
		c, err := csaf.Parse(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("error parsing CSAF: %w", err)
		}
		if c.Document.Tracking.Status == "deleted" {
			continue
		}

		record := pkgcsaf.Advisory{
			// This would be the RHSA/RHBA/RHEA ID.
			Name: c.Document.Tracking.ID,
			// Use the title as the advisory's description.
			// The current csaf type from Claircore as of toolkit/v1.2.4 does not enable us
			// to fetch c.Document.Notes which would have more in-depth descriptions.
			// The title is a decent compromise for now.
			Description: c.Document.Title,
			// The initial_release_date is the date when the advisory was published.
			// The current_release_date is the last-updated date, so
			// we use initial_release_date here.
			ReleaseDate: c.Document.Tracking.InitialReleaseDate,
			// Obtain the aggregate severity rating of all related CVEs for this advisory,
			// which tends to be the highest severity of all related CVEs.
			// This matches the severity we'd obtain from OVAL.
			Severity: c.Document.AggregateSeverity.Text,
		}

		// Back in the Red Hat OVAL days, we would assign an advisory the highest-related CVSS scores.
		// We mimic that functionality here.
		//
		// Note: it is very possible a CVE has two different CVSS scores, depending on the product.
		// For example: https://access.redhat.com/security/cve/CVE-2023-3899 is scored 7.8, in general,
		// but 6.1 for subscription-manager in RHEL 7.
		// For this case, the OVAL v2 entry in https://security.access.redhat.com/data/oval/v2/RHEL7/rhel-7-including-unpatched.oval.xml.bz2
		// for the associated RHSA, RHSA-2023:4701, actually has the general CVSS score (7.8) instead of the true score (6.1).
		// Meanwhile, the CSAF entry in https://security.access.redhat.com/data/csaf/v2/advisories/2023/rhsa-2023_4701.json
		// lists the true, correct score of 6.1.
		// So, the output we get here will differ from the previous OVAL v2-based output, but it will be more accurate
		// (though we acknowledge Red Hat advisories should not really be assigned CVSS scores in the first place).
		var cvss2, cvss3 struct {
			score  float64
			vector string
		}
		for _, v := range c.Vulnerabilities {
			// Loop through each vulnerability's scores, but it is not expected for there to be more than one,
			// as Red Hat advisories are per-product, and each product should only have a single CVSS v2/v3 score.
			for _, score := range v.Scores {
				if score.CVSSV3 != nil && score.CVSSV3.BaseScore > cvss3.score {
					cvss3.score = score.CVSSV3.BaseScore
					cvss3.vector = score.CVSSV3.VectorString
				}
				if score.CVSSV2 != nil && score.CVSSV2.BaseScore > cvss2.score {
					cvss2.score = score.CVSSV2.BaseScore
					cvss2.vector = score.CVSSV2.VectorString
				}
			}
		}

		if cvss3.vector != "" {
			record.CVSSv3.Score = float32(cvss3.score)
			record.CVSSv3.Vector = cvss3.vector
		}
		if cvss2.vector != "" {
			record.CVSSv2.Score = float32(cvss2.score)
			record.CVSSv2.Vector = cvss2.vector
		}

		records[record.Name] = record
	}

	out := make([]driver.EnrichmentRecord, 0, len(records))
	for _, record := range records {
		b, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		out = append(out, driver.EnrichmentRecord{
			Tags:       []string{record.Name},
			Enrichment: b,
		})
	}

	return out, nil
}
