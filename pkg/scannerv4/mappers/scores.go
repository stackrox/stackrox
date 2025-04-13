package mappers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	nvdschema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/facebookincubator/nvdtools/cvss2"
	"github.com/facebookincubator/nvdtools/cvss3"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stackrox/rox/pkg/scannerv4/updater/manual"
)

// sortByNVDCVSS sorts the vulnerability IDs in decreasing NVD CVSS order.
func sortByNVDCVSS(ids []string, vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability) {
	slices.SortStableFunc(ids, func(idA, idB string) int {
		vulnA := vulnerabilities[idA]
		vulnB := vulnerabilities[idB]

		var (
			vulnANVDMetrics *v4.VulnerabilityReport_Vulnerability_CVSS
			vulnBNVDMetrics *v4.VulnerabilityReport_Vulnerability_CVSS
		)
		for _, metrics := range vulnA.GetCvssMetrics() {
			if metrics.GetSource() == v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
				vulnANVDMetrics = metrics
				break
			}
		}
		for _, metrics := range vulnB.GetCvssMetrics() {
			if metrics.GetSource() == v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
				vulnBNVDMetrics = metrics
				break
			}
		}

		// Handle nil NVD metrics explicitly: nil is considered lower.
		if vulnANVDMetrics == nil && vulnBNVDMetrics == nil {
			return 0 // keep the original order
		}
		if vulnANVDMetrics == nil {
			return +1 // vulnBNVDMetrics non-nil, so prefer vulnB
		}
		if vulnBNVDMetrics == nil {
			return -1 // vulnANVDMetrics non-nil, so prefer vulnA
		}

		// Determine the base scores and indicate the vuln with the higher score goes in front.
		vulnAScore := baseScore([]*v4.VulnerabilityReport_Vulnerability_CVSS{vulnANVDMetrics})
		vulnBScore := baseScore([]*v4.VulnerabilityReport_Vulnerability_CVSS{vulnBNVDMetrics})
		if vulnAScore > vulnBScore {
			return -1
		}
		if vulnAScore < vulnBScore {
			return +1
		}
		return 0
	})
}

// vulnCVSS returns CVSS metrics based on the given vulnerability and its source.
func vulnCVSS(vuln *claircore.Vulnerability, source v4.VulnerabilityReport_Vulnerability_CVSS_Source) (*v4.VulnerabilityReport_Vulnerability_CVSS, error) {
	// It is assumed the Severity stores a CVSS vector.
	cvssVector := vuln.Severity
	if cvssVector == "" {
		return nil, errors.New("severity is empty")
	}

	values := cvssValues{
		source: source,
	}

	// TODO(ROX-26462): add CVSS v4 support.
	switch {
	case strings.HasPrefix(cvssVector, `CVSS:3.0`), strings.HasPrefix(cvssVector, `CVSS:3.1`):
		v, err := cvss3.VectorFromString(cvssVector)
		if err != nil {
			return nil, fmt.Errorf("parsing CVSS v3 vector %q: %w", cvssVector, err)
		}
		values.v3Vector = cvssVector
		values.v3Score = float32(v.BaseScore())
	default:
		// Fallback to CVSS 2.0
		v, err := cvss2.VectorFromString(cvssVector)
		if err != nil {
			return nil, fmt.Errorf("parsing (potential) CVSS v2 vector %q: %w", cvssVector, err)
		}
		values.v2Vector = cvssVector
		values.v2Score = float32(v.BaseScore())
	}

	switch source {
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD:
		values.url = nvdCVEURLPrefix + vuln.Name
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV:
		values.url = osvCVEURLPrefix + vuln.Name
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT:
		values.url = redhatCVEURLPrefix + vuln.Name
	default:
		values.url = vuln.Links
	}

	cvss := toCVSS(values)
	return cvss, nil
}

// nvdCVSS returns cvssValues based on the given vulnerability and the associated NVD item.
func nvdCVSS(v *nvdschema.CVEAPIJSON20CVEItem) (*v4.VulnerabilityReport_Vulnerability_CVSS, error) {
	// Sanity check the NVD data.
	if v.Metrics == nil || (v.Metrics.CvssMetricV31 == nil && v.Metrics.CvssMetricV30 == nil && v.Metrics.CvssMetricV2 == nil) {
		return nil, errors.New("no NVD CVSS metrics")
	}

	values := cvssValues{
		source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
		url:    nvdCVEURLPrefix + v.ID,
	}

	if len(v.Metrics.CvssMetricV30) > 0 {
		if cvssv30 := v.Metrics.CvssMetricV30[0]; cvssv30 != nil && cvssv30.CvssData != nil {
			values.v3Score = float32(cvssv30.CvssData.BaseScore)
			values.v3Vector = cvssv30.CvssData.VectorString
		}
	}
	// If there is both CVSS 3.0 and 3.1 data, use 3.1.
	if len(v.Metrics.CvssMetricV31) > 0 {
		if cvssv31 := v.Metrics.CvssMetricV31[0]; cvssv31 != nil && cvssv31.CvssData != nil {
			values.v3Score = float32(cvssv31.CvssData.BaseScore)
			values.v3Vector = cvssv31.CvssData.VectorString
		}
	}
	if len(v.Metrics.CvssMetricV2) > 0 {
		if cvssv2 := v.Metrics.CvssMetricV2[0]; cvssv2 != nil && cvssv2.CvssData != nil {
			values.v2Score = float32(cvssv2.CvssData.BaseScore)
			values.v2Vector = cvssv2.CvssData.VectorString
		}
	}

	cvss := toCVSS(values)
	return cvss, nil
}

// cvssMetrics processes the CVSS metrics and severity for a given vulnerability.
// This function gathers CVSS metrics data from multiple sources and
// returns a slice of CVSS metrics collected from different sources (e.g., RHEL, NVD, OSV).
// When not empty, the first entry is the "preferred" metric.
// An error is returned when there is a failure to collect CVSS metrics from all sources;
// however, the returned slice of metrics will still be populated with any successfully gathered metrics.
// It is up to the caller to ensure the returned slice is populated prior to using it.
//
// TODO(ROX-26672): Remove vulnName and advisory parameters.
// They are part of a temporary patch until we stop making RHSAs the top-level vulnerability.
func cvssMetrics(_ context.Context, vuln *claircore.Vulnerability, vulnName string, nvdVuln *nvdschema.CVEAPIJSON20CVEItem, advisory csaf.Advisory) ([]*v4.VulnerabilityReport_Vulnerability_CVSS, error) {
	var metrics []*v4.VulnerabilityReport_Vulnerability_CVSS

	var preferredCVSS *v4.VulnerabilityReport_Vulnerability_CVSS
	var preferredErr error
	switch {
	case strings.EqualFold(vuln.Updater, RedHatUpdaterName):
		// If the Name is empty, then the whole advisory is.
		if advisory.Name == "" {
			preferredCVSS, preferredErr = vulnCVSS(vuln, v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT)
		} else {
			// Set the preferred CVSS metrics to the ones provided by the related Red Hat advisory.
			// TODO(ROX-26462): add CVSS v4 support.
			preferredCVSS = toCVSS(cvssValues{
				v2Vector: advisory.CVSSv2.Vector,
				v2Score:  advisory.CVSSv2.Score,
				v3Vector: advisory.CVSSv3.Vector,
				v3Score:  advisory.CVSSv3.Score,
				source:   v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
			})
		}
		// TODO(ROX-26672): Remove this
		// Note: Do NOT use the advisory data here, as it's possible CSAF enrichment is disabled while [features.ScannerV4RedHatCVEs]
		// is also disabled.
		if !features.ScannerV4RedHatCVEs.Enabled() && preferredCVSS != nil && RedHatAdvisoryPattern.MatchString(vulnName) {
			preferredCVSS.Url = redhatErrataURLPrefix + vulnName
		}
	case strings.HasPrefix(vuln.Updater, osvUpdaterPrefix) && !isOSVDBSpecificSeverity(vuln.Severity):
		preferredCVSS, preferredErr = vulnCVSS(vuln, v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV)
	case strings.EqualFold(vuln.Updater, manual.UpdaterName):
		// It is expected manually added vulnerabilities only have a single link.
		preferredCVSS, preferredErr = vulnCVSS(vuln, sourceFromLinks(vuln.Links))
	}
	if preferredCVSS != nil {
		metrics = append(metrics, preferredCVSS)
	}

	var nvdErr error
	// Manually added vulnerabilities may have its data sourced from NVD.
	// In that scenario, there is no need to add yet another NVD entry,
	// especially since there is a reason the manual entry exists in the first place.
	if preferredCVSS.GetSource() != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
		var cvss *v4.VulnerabilityReport_Vulnerability_CVSS
		cvss, nvdErr = nvdCVSS(nvdVuln)
		if cvss != nil {
			metrics = append(metrics, cvss)
		}
	}

	return metrics, errors.Join(preferredErr, nvdErr)
}

// toCVSS converts the given CVSS values into CVSS metrics.
// It is assumed there is data for at least one CVSS version.
// TODO(ROX-26462): Add CVSS v4 support.
func toCVSS(vals cvssValues) *v4.VulnerabilityReport_Vulnerability_CVSS {
	hasV2, hasV3 := vals.v2Vector != "", vals.v3Vector != ""
	cvss := &v4.VulnerabilityReport_Vulnerability_CVSS{
		Source: vals.source,
		Url:    vals.url,
	}
	if hasV2 {
		cvss.V2 = &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
			BaseScore: vals.v2Score,
			Vector:    vals.v2Vector,
		}
	}
	if hasV3 {
		cvss.V3 = &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
			BaseScore: vals.v3Score,
			Vector:    vals.v3Vector,
		}
	}
	return cvss
}

// rhelVulnsEPSS gets highest EPSS score for each Red Hat advisory name
// TODO(ROX-27729): get the highest EPSS score for an RHSA across all CVEs associated with that RHSA
func rhelVulnsEPSS(vulns map[string]*claircore.Vulnerability, epssItems map[string]map[string]*epss.EPSSItem) map[string]epss.EPSSItem {
	if vulns == nil || epssItems == nil {
		return nil
	}
	rhsaEPSS := make(map[string]epss.EPSSItem)
	for _, v := range vulns {
		if v == nil {
			continue
		}
		cve, foundCVE := FindName(v, cveIDPattern)
		if !foundCVE {
			continue // continue if it's not a CVE
		}
		rhelName, foundRHEL := FindName(v, RedHatAdvisoryPattern)
		if !foundRHEL {
			continue // continue if it's not a RHSA
		}
		vulnEPSSItems, ok := epssItems[v.ID]
		if !ok {
			continue // no epss items related to current vuln id
		}
		epssItem, ok := vulnEPSSItems[cve]
		if !ok {
			continue // no epss score
		}

		// if both CVE and rhsa names exist
		rhelEPSS, ok := rhsaEPSS[rhelName]
		if !ok {
			rhsaEPSS[rhelName] = *epssItem
		} else {
			if epssItem.EPSS > rhelEPSS.EPSS {
				rhsaEPSS[rhelName] = *epssItem
			}
		}
	}

	return rhsaEPSS
}

// baseScore returns the CVSS base score, prioritizing V3 over V2.
func baseScore(cvssMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS) float32 {
	var metric *v4.VulnerabilityReport_Vulnerability_CVSS
	if len(cvssMetrics) == 0 {
		return 0.0
	}
	metric = cvssMetrics[0] // first one is guaranteed to be the preferred
	if v3 := metric.GetV3(); v3 != nil {
		return v3.GetBaseScore()
	} else if v2 := metric.GetV2(); v2 != nil {
		return v2.GetBaseScore()
	}
	return 0.0
}

// cveEPSS unmarshals and returns the EPSS enrichment, if it exists.
func cveEPSS(ctx context.Context, enrichments map[string][]json.RawMessage) (map[string]map[string]*epss.EPSSItem, error) {
	if !features.EPSSScore.Enabled() {
		return nil, nil
	}
	enrichmentList := enrichments[epss.Type]
	if len(enrichmentList) == 0 {
		zlog.Warn(ctx).
			Str("enrichments", epss.Type).
			Msg("No EPSS enrichments found. Verify that the vulnerability enrichment data is available and complete.")
		return nil, nil
	}

	var epssItems map[string][]epss.EPSSItem
	// The EPSS enrichment always contains only one element.
	err := json.Unmarshal(enrichmentList[0], &epssItems)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling EPSS enrichment: %w", err)
	}

	if len(epssItems) == 0 {
		zlog.Warn(ctx).
			Str("enrichments", epss.Type).
			Msg("No EPSS enrichments found. Verify that the vulnerability enrichment data is available and complete.")
		return nil, nil
	}

	ret := make(map[string]map[string]*epss.EPSSItem)
	for ccVulnID, list := range epssItems {
		if len(list) > 0 {
			m := make(map[string]*epss.EPSSItem)
			for idx := range list {
				epssData := list[idx]
				m[epssData.CVE] = &epssData
			}
			ret[ccVulnID] = m
		}
	}
	return ret, nil
}

// sortBySeverity sorts the vulnerability IDs based on normalized severity and,
// if equal, by the highest CVSS base score, decreasing.
func sortBySeverity(ids []string, vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability) {
	sort.SliceStable(ids, func(i, j int) bool {
		vulnI := vulnerabilities[ids[i]]
		vulnJ := vulnerabilities[ids[j]]

		// Handle nil vulnerabilities explicitly: nil is considered lower
		if vulnI == nil && vulnJ == nil {
			return false // keep the original order
		}
		if vulnI == nil {
			return false // vulnJ non-nil, higher
		}
		if vulnJ == nil {
			return true // vulnI non-nil, higher
		}

		// Compare by normalized severity (higher severity first).
		if vulnI.GetNormalizedSeverity() != vulnJ.GetNormalizedSeverity() {
			return vulnI.GetNormalizedSeverity() > vulnJ.GetNormalizedSeverity()
		}

		// If severities are equal, compare by the highest CVSS base score.
		scoreI := baseScore(vulnI.GetCvssMetrics())
		scoreJ := baseScore(vulnJ.GetCvssMetrics())

		return scoreI > scoreJ
	})
}
