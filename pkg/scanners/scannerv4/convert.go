package scannerv4

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

// vulnDataSourceDelimiter separates the parts of a vuln's datasource.
// IMPORTANT: This delimiter was chosen because it does not appear in any known
// Claircore or StackRox updater names.
const vulnDataSourceDelimiter = "::"

// digitSegment matches contiguous runs of digits for numeric segment comparisons.
var digitSegment = regexp.MustCompile(`\d+`)

func imageScan(metadata *storage.ImageMetadata, report *v4.VulnerabilityReport, scannerVersion string) *storage.ImageScan {
	scan := &storage.ImageScan{
		ScannerVersion:  scannerVersion,
		ScanTime:        protocompat.TimestampNow(),
		OperatingSystem: os(report),
		Components:      components(metadata, report),
		Notes:           notes(report),
	}

	return scan
}

func components(metadata *storage.ImageMetadata, report *v4.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	layerSHAToIndex := clair.BuildSHAToIndexMap(metadata)

	pkgs := report.GetContents().GetPackages()
	if len(pkgs) == 0 {
		pkgs = make(map[string]*v4.Package, len(report.GetContents().GetPackagesDEPRECATED()))
		// Fallback to the deprecated slice, if needed.
		for _, pkg := range report.GetContents().GetPackagesDEPRECATED() {
			pkgs[pkg.GetId()] = pkg
		}
	}
	// Filter out non-binary packages that should not become user-facing components:
	// - "ancestry" packages carry VEX suppression metadata only.
	// - "source" packages already referenced by a binary are redundant since the
	//   binary's vulnerability findings are a superset of its source's.
	// Unreferenced source packages are kept defensively.
	dedupe := features.ScannerV4Dedupe.Enabled()
	var referencedSourceIDs set.StringSet
	if dedupe {
		referencedSourceIDs = set.NewStringSet()
		for _, pkg := range pkgs {
			if pkg.GetKind() != "binary" {
				continue
			}
			if srcID := pkg.GetSource().GetId(); srcID != "" {
				referencedSourceIDs.Add(srcID)
			}
		}
	}

	components := make([]*storage.EmbeddedImageScanComponent, 0, len(pkgs))
	for id, pkg := range pkgs {
		if dedupe {
			switch pkg.GetKind() {
			case "ancestry":
				continue
			case "source":
				if referencedSourceIDs.Contains(id) {
					continue
				}
			}
		}
		vulnIDs := report.GetPackageVulnerabilities()[id].GetValues()

		var (
			source   storage.SourceType
			location string
			layerIdx *storage.EmbeddedImageScanComponent_LayerIndex
		)
		env := environment(report, id)
		if env != nil {
			source, location = ParsePackageDB(env.GetPackageDb())
			layerIdx = layerIndex(layerSHAToIndex, env)
		}

		component := &storage.EmbeddedImageScanComponent{
			Name:         pkg.GetName(),
			Version:      pkg.GetVersion(),
			Architecture: pkg.GetArch(),
			Vulns:        vulnerabilities(report.GetVulnerabilities(), vulnIDs, envOS(env, report), pkg.GetFixedInVersion()),
			FixedBy:      pkg.GetFixedInVersion(),
			Source:       source,
			Location:     location,
		}
		// DO NOT BLINDLY SET THIS INSIDE THE STRUCT DECLARATION DIRECTLY ABOVE.
		// IF layerIdx IS nil, IT DOES NOT MEAN HasLayerIndex WILL BE THE SAME nil.
		// GO CAN SOMETIMES BE ANNOYING AND nil DOES NOT ALWAYS EQUAL nil.
		// IT IS MUCH SAFER TO ONLY SET HasLayerIndex TO layerIdx WHEN WE KNOW FOR
		// SURE layerIdx != nil!!!
		//
		// See https://go.dev/doc/faq#nil_error for an explanation about this.
		// For this particular use-case, layerIdx is a pointer to a struct
		// which implements the interface. Therefore,
		// even if layerIdx is set to nil, it still of type
		// *storage.EmbeddedImageScanComponent_LayerIndex.
		// HasLayerIndex is an interface type, and, according to the docs,
		// this will only be nil if both the type and value are unset.
		// If we always set HasLayerIndex to layerIdx, then
		// it will never be nil, as layerIdx always has a type.
		if layerIdx != nil {
			component.HasLayerIndex = layerIdx
		}

		components = append(components, component)
	}

	return components
}

// envOS will return the operating system name and version associated with an
// environment.
func envOS(env *v4.Environment, report *v4.VulnerabilityReport) string {
	if env == nil {
		return ""
	}

	dists := distributions(report)
	dist, ok := dists[env.GetDistributionId()]
	if !ok || dist.GetDid() == "" || dist.GetVersionId() == "" {
		return ""
	}

	return dist.GetDid() + ":" + dist.GetVersionId()
}

func environment(report *v4.VulnerabilityReport, id string) *v4.Environment {
	environments := report.GetContents().GetEnvironments()
	if environments == nil {
		// Fallback to deprecated environments.
		environments = report.GetContents().GetEnvironmentsDEPRECATED()
	}
	envList, ok := environments[id]
	if !ok {
		return nil
	}

	envs := envList.GetEnvironments()
	if len(envs) > 0 {
		// Just use the first environment.
		// It is possible there are multiple environments associated with this package;
		// however, for our purposes, we only need the first one,
		// as the layer index and package DB will always be the same between different environments.
		// Quay does this, too: https://github.com/quay/quay/blob/v3.10.3/data/secscan_model/secscan_v4_model.py#L583
		return envs[0]
	}

	return nil
}

// ParsePackageDB parses the given packageDB into its source type + filepath.
func ParsePackageDB(packageDB string) (storage.SourceType, string) {
	prefix, path, found := strings.Cut(packageDB, ":")
	if !found {
		// All currently know language packages have a prefix, so this must be an OS package.
		return storage.SourceType_OS, packageDB
	}

	switch prefix {
	case "go":
		return storage.SourceType_GO, path
	case "file", "jar", "maven":
		return storage.SourceType_JAVA, path
	case "nodejs":
		return storage.SourceType_NODEJS, path
	case "python":
		return storage.SourceType_PYTHON, path
	case "ruby":
		return storage.SourceType_RUBY, path
	case "bdb", "sqlite", "ndb":
		// RPM databases are prefixed with the DB kind.
		return storage.SourceType_OS, path
	default:
		// ":" is a valid character in a file path.
		// We could not identify a known prefix, so just return the entire path.
		return storage.SourceType_OS, packageDB
	}
}

func layerIndex(layerSHAToIndex map[string]int32, env *v4.Environment) *storage.EmbeddedImageScanComponent_LayerIndex {
	idx, ok := layerSHAToIndex[env.GetIntroducedIn()]
	if !ok {
		return nil
	}

	return &storage.EmbeddedImageScanComponent_LayerIndex{
		LayerIndex: idx,
	}
}

func vulnerabilities(vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability, ids []string, envOS string, pkgFixedByVersion string) []*storage.EmbeddedVulnerability {
	if len(vulnerabilities) == 0 || len(ids) == 0 {
		return nil
	}

	dedupe := features.ScannerV4Dedupe.Enabled()

	vulns := make([]*storage.EmbeddedVulnerability, 0, len(ids))
	uniqueVulns := set.NewStringSet()
	var cveNameToIdx map[string]int
	if dedupe {
		cveNameToIdx = make(map[string]int, len(ids))
	}

	for _, id := range ids {
		if !uniqueVulns.Add(id) {
			// Already saw this vulnerability, so ignore it.
			continue
		}

		ccVuln, ok := vulnerabilities[id]
		if !ok {
			log.Debugf("vuln ID %q from PackageVulnerabilities not found in Vulnerabilities, skipping", id)
			continue
		}

		name := ccVuln.GetName()

		// Multiple Scanner V4 vulns from different sources can share the
		// same CVE identifier. Merge duplicates into a single entry.
		if dedupe && name != "" {
			if idx, exists := cveNameToIdx[name]; exists {
				mergeFixFields(vulns[idx], ccVuln, envOS, pkgFixedByVersion)
				mergeScoringFields(vulns[idx], ccVuln)
				continue
			}
			cveNameToIdx[name] = len(vulns)
		}

		vulns = append(vulns, buildEmbeddedVulnerability(ccVuln, envOS))
	}

	return vulns
}

// buildEmbeddedVulnerability converts a single v4 vulnerability into its
// storage representation, populating all fields from the v4 source.
func buildEmbeddedVulnerability(ccVuln *v4.VulnerabilityReport_Vulnerability, envOS string) *storage.EmbeddedVulnerability {
	// TODO(ROX-20355): Populate last modified once the API is available.
	vuln := &storage.EmbeddedVulnerability{
		Cve:      ccVuln.GetName(),
		Advisory: advisory(ccVuln.GetAdvisory()),
		Summary:  ccVuln.GetDescription(),
		// TODO(ROX-26547)
		// The link field will be overwritten if preferred CVSS source is available
		Link:        link(ccVuln.GetLink()),
		PublishedOn: ccVuln.GetIssued(),
		// LastModified: ,
		VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		Severity:              normalizedSeverity(ccVuln.GetNormalizedSeverity()),
		Epss:                  epss(ccVuln.GetEpssMetrics()),
		FixAvailableTimestamp: ccVuln.GetFixedDate(),
		Datasource:            vulnDataSource(ccVuln, envOS),
	}
	if err := setScoresAndScoreVersions(vuln, ccVuln.GetCvssMetrics()); err != nil {
		utils.Should(err)
	}
	maybeOverwriteSeverity(vuln)
	if ccVuln.GetFixedInVersion() != "" {
		vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: ccVuln.GetFixedInVersion(),
		}
	}
	return vuln
}

// vulnDataSource builds a string that uniquely identifies a vulnerability's datasource.
// The datasource represents CVE uniqueness and can be used to associate a CVE with
// other data, such as fixed date.
//
// IMPORTANT: The datasource value MUST be treated as an opaque string because it contains
// the Claircore updater which is an 'internal' field with no guarantee it will be stable
// between releases - do not parse or extract components from it. It should only be used for:
//   - Equality comparisons
//   - Storage/retrieval as a database key
//
// For Red Hat vulns the product (repo, cpe, etc.) is also needed to uniquely represent the
// vuln which is NOT included in the returned datasource.
//
// Examples:
//   - OS vulnerabilities: "updater::os" (e.g., "debian-bookworm-updater::debian:12")
//   - Language vulnerabilities: "updater" (e.g., "osv/go", "nvd")
//
// When this format changes or ClairCore updater names change, a database migration may be required.
func vulnDataSource(ccVuln *v4.VulnerabilityReport_Vulnerability, os string) string {
	if ccVuln.GetUpdater() == "" {
		return ""
	}

	if os == "" {
		// ie: "osv/go", "nvd"
		return ccVuln.GetUpdater()
	}

	// ie: "debian/updater::debian:12", "ubuntu/updater/focal::ubuntu:20.04"
	return strings.Join([]string{
		ccVuln.GetUpdater(),
		os,
	}, vulnDataSourceDelimiter)
}

func advisory(advisory *v4.VulnerabilityReport_Advisory) *storage.Advisory {
	if advisory == nil {
		return nil
	}
	return &storage.Advisory{
		Name: advisory.GetName(),
		Link: advisory.GetLink(),
	}
}

func epss(epssDetail *v4.VulnerabilityReport_Vulnerability_EPSS) *storage.EPSS {
	if epssDetail == nil {
		return nil
	}
	return &storage.EPSS{
		EpssProbability: epssDetail.GetProbability(),
		EpssPercentile:  epssDetail.GetPercentile(),
	}
}

func setScoresAndScoreVersions(vuln *storage.EmbeddedVulnerability, CVSSMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS) error {
	if len(CVSSMetrics) == 0 {
		return nil
	}

	errList := errorhelpers.NewErrorList("failed to get CVSS Metrics")
	var scores []*storage.CVSSScore
	for _, cvss := range CVSSMetrics {
		score := &storage.CVSSScore{
			Source: CVSSSource(cvss.GetSource()),
			Url:    cvss.GetUrl(),
		}
		if cvss.GetV2() != nil {
			baseScore, cvssV2, v2Err := toCVSSV2Scores(cvss, vuln.GetCve())
			if v2Err == nil && cvssV2 != nil {
				score.CvssScore = &storage.CVSSScore_Cvssv2{Cvssv2: cvssV2}
				// CVSS metrics has maximum two entries, one from NVD, one from updater if available
				if len(CVSSMetrics) == 1 || (len(CVSSMetrics) > 1 && cvss.GetSource() != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD) {
					vuln.CvssV2 = cvssV2.CloneVT()
					vuln.ScoreVersion = storage.EmbeddedVulnerability_V2
					vuln.Cvss = baseScore
					vuln.Link = cvss.GetUrl()
				}
			} else {
				errList.AddError(v2Err)
			}
		}
		if cvss.GetV3() != nil {
			baseScore, cvssV3, v3Err := toCVSSV3Scores(cvss, vuln.GetCve())
			if v3Err == nil && cvssV3 != nil {
				// overwrite if v3 available
				score.CvssScore = &storage.CVSSScore_Cvssv3{Cvssv3: cvssV3}
				// CVSS metrics has maximum two entries, one from NVD, one from Rox updater if available
				if len(CVSSMetrics) == 1 || (len(CVSSMetrics) > 1 && cvss.GetSource() != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD) {
					vuln.CvssV3 = cvssV3.CloneVT()
					// overwrite if v3 available
					vuln.ScoreVersion = storage.EmbeddedVulnerability_V3
					vuln.Cvss = baseScore
					vuln.Link = cvss.GetUrl()
				}
			} else {
				errList.AddError(v3Err)
			}
		}
		if score.CvssScore != nil {
			scores = append(scores, score)
		}
	}

	if len(scores) > 0 {
		vuln.CvssMetrics = scores
		if errList.Empty() {
			return nil
		}
	}

	return errList.ToError()
}

func CVSSSource(source v4.VulnerabilityReport_Vulnerability_CVSS_Source) storage.Source {
	switch source {
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD:
		return storage.Source_SOURCE_NVD
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV:
		return storage.Source_SOURCE_OSV
	case v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT:
		return storage.Source_SOURCE_RED_HAT
	default:
		return storage.Source_SOURCE_UNKNOWN
	}
}

func toCVSSV2Scores(vulnCVSS *v4.VulnerabilityReport_Vulnerability_CVSS, cve string) (float32, *storage.CVSSV2, error) {
	v2 := vulnCVSS.GetV2()
	c, err := cvssv2.ParseCVSSV2(v2.GetVector())
	if err == nil {
		err = cvssv2.CalculateScores(c)
		if err != nil {
			return 0, nil, fmt.Errorf("calculating CVSS v2 scores: %w", err)
		}
		// Use the report's score if it exists.
		if baseScore := v2.GetBaseScore(); baseScore != 0.0 && baseScore != c.GetScore() {
			log.Debugf("Calculated CVSSv2 score does not match given base score (%f != %f) for %s. Using given score...", c.GetScore(), baseScore, cve)
			c.Score = baseScore
		}
		c.Severity = cvssv2.Severity(c.GetScore())
		return c.GetScore(), c, nil
	}
	return 0, nil, fmt.Errorf("parsing CVSS v2 vector: %w", err)
}

func toCVSSV3Scores(vulnCVSS *v4.VulnerabilityReport_Vulnerability_CVSS, cve string) (float32, *storage.CVSSV3, error) {
	v3 := vulnCVSS.GetV3()
	c, err := cvssv3.ParseCVSSV3(v3.GetVector())
	if err == nil {
		if err := cvssv3.CalculateScores(c); err != nil {
			return 0, nil, fmt.Errorf("calculating CVSS v3 scores: %w", err)
		}
		// Use the report's score if it exists and differs from the calculated score
		if baseScore := v3.GetBaseScore(); baseScore != 0.0 && baseScore != c.GetScore() {
			log.Debugf("Calculated CVSSv3 score does not match given base score (calculated: %f, given: %f) for %s. Using given score...", c.GetScore(), baseScore, cve)
			c.Score = baseScore
		}
		c.Severity = cvssv3.Severity(c.GetScore())
		return c.GetScore(), c, nil
	}
	return 0, nil, fmt.Errorf("parsing CVSS v3 vector: %w", err)
}

// link returns the first link from space separated list of links (which is how ClairCore provides links).
// The ACS UI will fail to show a vulnerability's link if it is an invalid URL.
func link(links string) string {
	link, _, _ := strings.Cut(links, " ")
	return link
}

// maybeOverwriteSeverity overwrites vuln.Severity with one derived from the CVSS scores
// if vuln.Severity == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY.
func maybeOverwriteSeverity(vuln *storage.EmbeddedVulnerability) {
	vuln.Severity = cvss.VulnToSeverity(cvss.NewFromEmbeddedVulnerability(vuln))
}

func normalizedSeverity(severity v4.VulnerabilityReport_Vulnerability_Severity) storage.VulnerabilitySeverity {
	switch severity {
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL:
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

// os retrieves the OS name:version for the image represented by the given
// vulnerability report.
// If there are zero known distributions for the image or if there are multiple distributions,
// return "unknown", as StackRox only supports a single base-OS at this time.
func os(report *v4.VulnerabilityReport) string {
	dists := distributions(report)
	if len(dists) != 1 {
		return "unknown"
	}

	var dist *v4.Distribution
	for _, d := range dists {
		dist = d
		break
	}
	return dist.GetDid() + ":" + dist.GetVersionId()
}

func distributions(report *v4.VulnerabilityReport) map[string]*v4.Distribution {
	dists := report.GetContents().GetDistributions()
	if len(dists) == 0 {
		// Fallback to the deprecated slice, if needed.
		dists = make(map[string]*v4.Distribution, len(report.GetContents().GetDistributionsDEPRECATED()))
		for _, dist := range report.GetContents().GetDistributionsDEPRECATED() {
			dists[dist.GetId()] = dist
		}
	}

	return dists
}

func notes(report *v4.VulnerabilityReport) []storage.ImageScan_Note {
	notes := make([]storage.ImageScan_Note, 0, len(v4.VulnerabilityReport_Note_value))

	for _, note := range report.GetNotes() {
		switch note {
		case v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			notes = append(notes, storage.ImageScan_OS_CVES_UNAVAILABLE)
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN:
			notes = append(notes, storage.ImageScan_OS_UNAVAILABLE)
		default:
			// Ignore unknown/unsupported note.
		}
	}

	if len(notes) > 0 {
		notes = append(notes, storage.ImageScan_PARTIAL_SCAN_DATA)
	}

	return notes
}

// mergeFixFields overwrites fix-related fields on dst when src has more
// recent or more complete fix data. Priority: later advisory, has fix over
// doesn't, matches package-level fix version, higher version by numeric
// comparison.
func mergeFixFields(dst *storage.EmbeddedVulnerability, src *v4.VulnerabilityReport_Vulnerability, envOS, pkgFixedByVersion string) {
	srcAdv := advisory(src.GetAdvisory())
	c := cmp.Or(
		compareAdvisories(srcAdv, dst.GetAdvisory()),
		compareFixVersions(src.GetFixedInVersion(), dst.GetFixedBy(), pkgFixedByVersion),
	)
	if c > 0 {
		applyFixFields(dst, src, srcAdv, envOS)
	}
}

// compareFixVersions returns positive when a represents a more complete or
// higher fix version than b. Priority: having a fix over not, matching
// pkgFixedBy, higher version by numeric comparison.
func compareFixVersions(a, b, pkgFixedBy string) int {
	aHasFix, bHasFix := a != "", b != ""
	if aHasFix != bHasFix {
		if aHasFix {
			return 1
		}
		return -1
	}
	if !aHasFix {
		return 0
	}
	if pkgFixedBy != "" && (a == pkgFixedBy) != (b == pkgFixedBy) {
		if a == pkgFixedBy {
			return 1
		}
		return -1
	}
	// Reaching here means both have a fix, neither matches pkgFixedBy (or it
	// is empty), and the versions disagree. This is rare — it requires two
	// sources to report different fix versions for the same CVE. Use a
	// deterministic numeric comparison so the result is stable across runs.
	if a != b {
		return compareNumericSegments(a, b)
	}
	return 0
}

// applyFixFields overwrites fix-related fields on dst from src.
func applyFixFields(dst *storage.EmbeddedVulnerability, src *v4.VulnerabilityReport_Vulnerability, srcAdv *storage.Advisory, envOS string) {
	dst.Advisory = srcAdv
	dst.Datasource = vulnDataSource(src, envOS)
	dst.FixAvailableTimestamp = src.GetFixedDate()
	dst.SetFixedBy = nil
	if fix := src.GetFixedInVersion(); fix != "" {
		dst.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: fix,
		}
	}
}

// compareNumericSegments compares two strings by extracting their numeric
// segments and comparing left-to-right, falling back to lexicographic order.
func compareNumericSegments(a, b string) int {
	if c := slices.Compare(splitVersionNumbers(a), splitVersionNumbers(b)); c != 0 {
		return c
	}
	return cmp.Compare(a, b)
}

func splitVersionNumbers(v string) []int {
	matches := digitSegment.FindAllString(v, -1)
	nums := make([]int, 0, len(matches))
	for _, m := range matches {
		n, _ := strconv.Atoi(m)
		nums = append(nums, n)
	}
	return nums
}

// mergeScoringFields overwrites scoring-related fields on dst when src has more
// complete or higher-severity scoring data. Priority: more CVSS metrics, higher
// severity, higher CVSS base score.
func mergeScoringFields(dst *storage.EmbeddedVulnerability, src *v4.VulnerabilityReport_Vulnerability) {
	c := cmp.Or(
		cmp.Compare(len(src.GetCvssMetrics()), len(dst.GetCvssMetrics())),
		cmp.Compare(normalizedSeverity(src.GetNormalizedSeverity()), dst.GetSeverity()),
		cmp.Compare(v4BaseScore(src.GetCvssMetrics()), dst.GetCvss()),
	)
	if c <= 0 {
		return
	}

	dst.Summary = src.GetDescription()
	dst.Severity = normalizedSeverity(src.GetNormalizedSeverity())
	dst.CvssV2 = nil
	dst.CvssV3 = nil
	dst.Cvss = 0
	dst.ScoreVersion = 0
	dst.CvssMetrics = nil
	dst.NvdCvss = 0
	dst.Link = link(src.GetLink())
	dst.PublishedOn = src.GetIssued()
	if err := setScoresAndScoreVersions(dst, src.GetCvssMetrics()); err != nil {
		utils.Should(err)
	}
	maybeOverwriteSeverity(dst)
	dst.Epss = epss(src.GetEpssMetrics())
}

// compareAdvisories compares two advisories by their numeric segments.
// Nil is less than non-nil.
func compareAdvisories(a, b *storage.Advisory) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return compareNumericSegments(a.GetName(), b.GetName())
}

// v4BaseScore returns the base score from the preferred CVSS metric entry.
// The preferred entry is always at index 0 — this ordering is guaranteed by
// the mapper that builds the v4 proto (see baseScore in mappers.go).
func v4BaseScore(metrics []*v4.VulnerabilityReport_Vulnerability_CVSS) float32 {
	if len(metrics) == 0 {
		return 0
	}
	m := metrics[0]
	if v3 := m.GetV3(); v3 != nil {
		return v3.GetBaseScore()
	}
	if v2 := m.GetV2(); v2 != nil {
		return v2.GetBaseScore()
	}
	return 0
}
