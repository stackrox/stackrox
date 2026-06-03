package mappers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// NVDItem represents NVD vulnerability data with CVSS scores.
type NVDItem struct {
	CVSSv2 *CVSSScore
	CVSSv3 *CVSSScore
}

// CVSSScore represents a CVSS score with its vector string.
type CVSSScore struct {
	BaseScore float64
	Vector    string
}

// EPSSItem represents EPSS (Exploit Prediction Scoring System) data.
type EPSSItem struct {
	ModelVersion string
	Date         string
	Probability  float64
	Percentile   float64
}

// CSAFAdvisory represents a CSAF advisory from RHEL.
type CSAFAdvisory struct {
	Name        string
	Description string
	ReleaseDate time.Time
	Severity    string
	CVSSv3      CVSSScore
	CVSSv2      CVSSScore
}

// ExtractNVDVulnerabilities extracts NVD vulnerability data from enrichments.
// Returns a map of enrichment key -> CVE ID -> NVD data.
func ExtractNVDVulnerabilities(enrichments map[string][]json.RawMessage) (map[string]map[string]*NVDItem, error) {
	result := make(map[string]map[string]*NVDItem)

	for key, messages := range enrichments {
		if !strings.HasPrefix(key, "message/vnd.clair.map.vulnerability") {
			continue
		}

		cveMap := make(map[string]*NVDItem)
		for _, msg := range messages {
			// Parse the message as a map of CVE ID -> array of NVD entries
			var vulnMap map[string][]struct {
				CVE  string `json:"cve"`
				CVSS []struct {
					Version string  `json:"version"`
					Score   float64 `json:"score"`
					Vector  string  `json:"vector"`
				} `json:"cvss"`
			}

			if err := json.Unmarshal(msg, &vulnMap); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal NVD vulnerability data")
			}

			for cveID, entries := range vulnMap {
				if len(entries) == 0 {
					continue
				}

				// Take the first entry for this CVE
				entry := entries[0]
				nvdItem := &NVDItem{}

				// Process CVSS scores
				for _, cvss := range entry.CVSS {
					score := &CVSSScore{
						BaseScore: cvss.Score,
						Vector:    cvss.Vector,
					}

					// Determine version by checking first character
					if len(cvss.Version) > 0 {
						if cvss.Version[0] == '2' {
							nvdItem.CVSSv2 = score
						} else if cvss.Version[0] == '3' {
							nvdItem.CVSSv3 = score
						}
					}
				}

				cveMap[cveID] = nvdItem
			}
		}

		if len(cveMap) > 0 {
			result[key] = cveMap
		}
	}

	return result, nil
}

// ExtractEPSS extracts EPSS data from enrichments.
// Returns a map of enrichment key -> CVE ID -> EPSS data.
func ExtractEPSS(enrichments map[string][]json.RawMessage) (map[string]map[string]*EPSSItem, error) {
	result := make(map[string]map[string]*EPSSItem)

	for key, messages := range enrichments {
		if !strings.HasPrefix(key, "message/vnd.clair.map.enrichment") {
			continue
		}

		cveMap := make(map[string]*EPSSItem)
		for _, msg := range messages {
			// Parse the message as a map of CVE ID -> array of EPSS entries
			var epssMap map[string][]struct {
				CVE  string `json:"cve"`
				EPSS struct {
					ModelVersion string  `json:"model_version"`
					Date         string  `json:"date"`
					Probability  float64 `json:"probability"`
					Percentile   float64 `json:"percentile"`
				} `json:"epss"`
			}

			if err := json.Unmarshal(msg, &epssMap); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal EPSS data")
			}

			for cveID, entries := range epssMap {
				if len(entries) == 0 {
					continue
				}

				// Take the first entry for this CVE
				entry := entries[0]
				epssItem := &EPSSItem{
					ModelVersion: entry.EPSS.ModelVersion,
					Date:         entry.EPSS.Date,
					Probability:  entry.EPSS.Probability,
					Percentile:   entry.EPSS.Percentile,
				}

				cveMap[cveID] = epssItem
			}
		}

		if len(cveMap) > 0 {
			result[key] = cveMap
		}
	}

	return result, nil
}

// ExtractCSAFAdvisories extracts CSAF advisory data from enrichments.
// Returns a map of CVE ID -> CSAF advisory (first advisory per CVE).
func ExtractCSAFAdvisories(enrichments map[string][]json.RawMessage) (map[string]*CSAFAdvisory, error) {
	const csafKey = "message/vnd.stackrox.scannerv4.map.csaf; enricher=stackrox.rhel-csaf"

	result := make(map[string]*CSAFAdvisory)

	messages, exists := enrichments[csafKey]
	if !exists {
		return result, nil
	}

	for _, msg := range messages {
		// Parse the message as a map of CVE ID -> array of advisories
		var advisoryMap map[string][]struct {
			Name        string    `json:"name"`
			Description string    `json:"description"`
			Released    time.Time `json:"released"`
			Severity    string    `json:"severity"`
			CVSSv3      *struct {
				BaseScore float64 `json:"base_score"`
				Vector    string  `json:"vector"`
			} `json:"cvss_v3,omitzero"`
			CVSSv2 *struct {
				BaseScore float64 `json:"base_score"`
				Vector    string  `json:"vector"`
			} `json:"cvss_v2,omitzero"`
		}

		if err := json.Unmarshal(msg, &advisoryMap); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal CSAF advisory data")
		}

		for cveID, advisories := range advisoryMap {
			if len(advisories) == 0 {
				continue
			}

			// Take the first advisory for this CVE
			adv := advisories[0]
			csafAdv := &CSAFAdvisory{
				Name:        adv.Name,
				Description: adv.Description,
				ReleaseDate: adv.Released,
				Severity:    adv.Severity,
			}

			if adv.CVSSv3 != nil {
				csafAdv.CVSSv3 = CVSSScore{
					BaseScore: adv.CVSSv3.BaseScore,
					Vector:    adv.CVSSv3.Vector,
				}
			}

			if adv.CVSSv2 != nil {
				csafAdv.CVSSv2 = CVSSScore{
					BaseScore: adv.CVSSv2.BaseScore,
					Vector:    adv.CVSSv2.Vector,
				}
			}

			result[cveID] = csafAdv
		}
	}

	return result, nil
}

// ExtractPkgFixedBy extracts package fixed-by version mappings from enrichments.
// Returns a map of package ID -> fixed version.
func ExtractPkgFixedBy(enrichments map[string][]json.RawMessage) (map[string]string, error) {
	const fixedByKey = "message/vnd.stackrox.scannerv4.fixedby; enricher=fixedby"

	result := make(map[string]string)

	messages, exists := enrichments[fixedByKey]
	if !exists {
		return result, nil
	}

	for _, msg := range messages {
		// Parse the message as a map of package ID -> fixed version
		var pkgMap map[string]string

		if err := json.Unmarshal(msg, &pkgMap); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal package fixed-by data")
		}

		for pkgID, fixedVersion := range pkgMap {
			result[pkgID] = fixedVersion
		}
	}

	return result, nil
}
