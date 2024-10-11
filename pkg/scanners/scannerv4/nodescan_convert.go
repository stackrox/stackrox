package scannerv4

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/protocompat"
)

func ToNodeScan(r *v4.VulnerabilityReport) *storage.NodeScan {
	return &storage.NodeScan{
		ScanTime:       protocompat.TimestampNow(),
		Components:     toStorageComponents(r),
		Notes:          toStorageNotes(r.Notes),
		ScannerVersion: storage.NodeScan_SCANNER_V4,
	}
}

func toStorageComponents(r *v4.VulnerabilityReport) []*storage.EmbeddedNodeScanComponent {
	result := make([]*storage.EmbeddedNodeScanComponent, 0)
	packages := r.GetContents().GetPackages()

	for _, pkg := range packages {
		vulns := getPackageVulns(pkg.GetId(), r)
		result = append(result, createEmbeddedComponent(pkg, vulns))
	}
	return result
}

func getPackageVulns(packageID string, r *v4.VulnerabilityReport) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0)
	mapping, ok := r.GetPackageVulnerabilities()[packageID]
	if !ok {
		// No vulnerabilities for this package, skip
		return vulns
	}
	for _, vulnID := range mapping.GetValues() {
		vulnerability, ok := r.Vulnerabilities[vulnID]
		if !ok {
			log.Warnf("Unable to find vulnerability %s in report - skipping this vulnerability", vulnID)
			continue
		}
		vulns = append(vulns, convertVulnerability(vulnerability))
	}
	return vulns
}

func convertVulnerability(v *v4.VulnerabilityReport_Vulnerability) *storage.EmbeddedVulnerability {
	converted := &storage.EmbeddedVulnerability{
		Cve:               v.GetName(),
		Summary:           v.GetDescription(),
		SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: v.GetFixedInVersion()},
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          clair.SeverityToStorageSeverity(v.GetSeverity()),
	}

	for _, c := range v.GetCvssMetrics() {
		if c.GetV3() == nil || c.GetV3().GetVector() == "" {
			log.Debugf("Skipping metrics as v3 information is unavailable/incomplete")
			continue
		}

		// As EmbeddedVulnerability can only track one URL, we'll pick and keep the first one encountered
		if c.GetUrl() != "" && converted.GetLink() == "" {
			converted.Link = c.GetUrl()
		}

		if cvssV3, err := cvssv3.ParseCVSSV3(c.GetV3().GetVector()); err == nil {
			cvssV3.Score = c.GetV3().GetBaseScore()

			converted.CvssV3 = cvssV3
			converted.Cvss = cvssV3.GetScore()
			converted.ScoreVersion = storage.EmbeddedVulnerability_V3
			converted.CvssV3.Severity = cvssv3.Severity(converted.GetCvss())
		} else {
			log.Errorf("converting v4.VulnerabilityReport CVSSv3: %v", err)
		}
	}

	return converted
}

func createEmbeddedComponent(pkg *v4.Package, vulns []*storage.EmbeddedVulnerability) *storage.EmbeddedNodeScanComponent {
	return &storage.EmbeddedNodeScanComponent{
		Name:    pkg.GetName(),
		Version: pkg.GetVersion(),
		Vulns:   vulns,
	}
}

func toStorageNotes(notes []v4.VulnerabilityReport_Note) []storage.NodeScan_Note {
	if notes == nil {
		return nil
	}
	convertedNotes := make([]storage.NodeScan_Note, 0, len(notes))
	for _, n := range notes {
		switch n {
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN:
			convertedNotes = append(convertedNotes, storage.NodeScan_UNSUPPORTED)
		case v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			convertedNotes = append(convertedNotes, storage.NodeScan_UNSUPPORTED)
		case v4.VulnerabilityReport_NOTE_UNSPECIFIED:
			convertedNotes = append(convertedNotes, storage.NodeScan_UNSET)
		default:
			log.Warnf("encountered unknown Vulnerability Report Note type while converting: %s", n.String())
		}
	}
	return convertedNotes
}
