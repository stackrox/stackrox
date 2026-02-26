package scannerv4

import (
	"fmt"
	"regexp"
	"strings"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	rhcosOSImageRegexp = regexp.MustCompile(`(Red Hat Enterprise Linux) (CoreOS) ([\d])([\d]+)`)
)

const (
	rhcosFullName = "Red Hat Enterprise Linux CoreOS"
)

func toNodeScan(r *v4.VulnerabilityReport, osImageRef string) *storage.NodeScan {
	// TODO(ROX-26593): Instead of fixing notes here, add RHCOS DistributionScanner to ClairCore
	fixedNotes := fixNotes(toStorageNotes(r.GetNotes()), osImageRef)

	convertedOS := toOperatingSystem(osImageRef)
	if convertedOS == "" {
		log.Warnf("Could not determine operating system from OSimage ref %s", osImageRef)
	}

	return &storage.NodeScan{
		ScanTime:        protocompat.TimestampNow(),
		Components:      toStorageComponents(r),
		Notes:           fixedNotes,
		ScannerVersion:  storage.NodeScan_SCANNER_V4,
		OperatingSystem: convertedOS,
	}
}

func toOperatingSystem(ref string) string {
	r := rhcosOSImageRegexp.FindStringSubmatch(ref)
	if len(r) != 5 {
		return ""
	}
	return fmt.Sprintf("rhcos:%s.%s", r[3], r[4])
}

func toStorageComponents(r *v4.VulnerabilityReport) []*storage.EmbeddedNodeScanComponent {
	packages := r.GetContents().GetPackages()
	if len(packages) == 0 {
		packages = make(map[string]*v4.Package, len(r.GetContents().GetPackagesDEPRECATED()))
		// Fallback to the deprecated slice, if needed.
		for _, pkg := range r.GetContents().GetPackagesDEPRECATED() {
			packages[pkg.GetId()] = pkg
		}
	}
	result := make([]*storage.EmbeddedNodeScanComponent, 0, len(packages))

	for id, pkg := range packages {
		vulns := getPackageVulns(id, r)
		result = append(result, createEmbeddedComponent(pkg, vulns))
	}
	return result
}

func getPackageVulns(packageID string, r *v4.VulnerabilityReport) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0)
	mapping, ok := r.GetPackageVulnerabilities()[packageID]
	if !ok {
		// No vulnerabilities for this package, skip.
		return vulns
	}
	processedVulns := set.NewStringSet()
	for _, vulnID := range mapping.GetValues() {
		if !processedVulns.Add(vulnID) {
			// Already processed this vulnerability, skip it.
			continue
		}
		vulnerability, ok := r.GetVulnerabilities()[vulnID]
		if !ok {
			log.Debugf("Mapping for package %s contains a vulnerability with unknown ID %s. This vulnerability won't be stored", packageID, vulnID)
			continue
		}
		vulns = append(vulns, convertVulnerability(vulnerability))
	}
	return dedupeNodeVulns(vulns)
}

// dedupeNodeVulns deduplicates node vulnerabilities by CVE name.
// When multiple records share the same CVE name — which can happen when
// ClairCore matches both "known_affected" (no FixedInVersion) and "fixed"
// (with FixedInVersion) VEX records due to broad CPE matching on nodes
// (ROX-26593) — this function prefers records that have a FixedBy version set.
// This prevents false positives where a patched package is incorrectly reported
// as vulnerable due to a "known_affected" record from a different CPE stream.
func dedupeNodeVulns(vulns []*storage.EmbeddedVulnerability) []*storage.EmbeddedVulnerability {
	if len(vulns) <= 1 {
		return vulns
	}

	// Group vulnerabilities by CVE name and track the best record for each.
	type vulnEntry struct {
		vuln  *storage.EmbeddedVulnerability
		index int
	}
	bestByCVE := make(map[string]vulnEntry, len(vulns))
	for i, vuln := range vulns {
		name := vuln.GetCve()
		existing, found := bestByCVE[name]
		if !found {
			bestByCVE[name] = vulnEntry{vuln: vuln, index: i}
			continue
		}
		// Prefer the record that has a FixedBy version set, as it provides
		// actionable fix information and indicates an accurate stream match.
		existingHasFix := existing.vuln.GetFixedBy() != ""
		newHasFix := vuln.GetFixedBy() != ""
		if !existingHasFix && newHasFix {
			bestByCVE[name] = vulnEntry{vuln: vuln, index: i}
		}
	}

	// If no duplicates were found, return the original slice unchanged.
	if len(bestByCVE) == len(vulns) {
		return vulns
	}

	// Rebuild the slice preserving the original ordering of surviving entries.
	kept := set.NewIntSet()
	for _, entry := range bestByCVE {
		kept.Add(entry.index)
	}
	deduped := make([]*storage.EmbeddedVulnerability, 0, len(bestByCVE))
	for i, vuln := range vulns {
		if kept.Contains(i) {
			deduped = append(deduped, vuln)
		}
	}
	return deduped
}

func convertVulnerability(v *v4.VulnerabilityReport_Vulnerability) *storage.EmbeddedVulnerability {
	converted := &storage.EmbeddedVulnerability{
		Cve:               v.GetName(),
		Summary:           v.GetDescription(),
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		Severity:          normalizedSeverity(v.GetNormalizedSeverity()),
		Link:              link(v.GetLink()),
		PublishedOn:       v.GetIssued(),
	}

	if err := setScoresAndScoreVersions(converted, v.GetCvssMetrics()); err != nil {
		utils.Should(err)
	}
	maybeOverwriteSeverity(converted)
	if v.GetFixedInVersion() != "" {
		converted.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: v.GetFixedInVersion(),
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

// TODO(ROX-26593): Instead of fixing notes here, add RHCOS DistributionScanner to ClairCore
// All nodes currently get the note UNSUPPORTED assigned to them because the IndexReport does not contain
// Distribution information. To include it there, a specialized RHCOS DistributionScanner needs to be added
// to ClairCore and then called in Compliances' IndexNode function where the IndexReport is created.
func fixNotes(notes []storage.NodeScan_Note, osImageRef string) []storage.NodeScan_Note {
	if !strings.HasPrefix(osImageRef, rhcosFullName) {
		// Keep notes as they are for nodes other than RHCOS
		return notes
	}
	fixedNotes := make([]storage.NodeScan_Note, 0)
	for _, note := range notes {
		switch note {
		case storage.NodeScan_UNSUPPORTED:
		default:
			fixedNotes = append(fixedNotes, note)
		}
	}
	return fixedNotes
}
