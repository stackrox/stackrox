package ingestion

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stackrox/rox/central/scandata/datastore"
	"github.com/stackrox/rox/central/scandata/types"
)

// Ingestor converts a v4.VulnerabilityReport into ScanData and stores it
type Ingestor struct {
	scanDataStore datastore.DataStore
}

// NewIngestor creates an Ingestor instance
func NewIngestor(scanDataStore datastore.DataStore) *Ingestor {
	return &Ingestor{scanDataStore: scanDataStore}
}

// IngestScan converts scanner v4 output and stores it in the new scan tables
func (i *Ingestor) IngestScan(ctx context.Context, imageID string, metadata *storage.ImageMetadata, report *v4.VulnerabilityReport) error {
	scanID := uuid.NewString()

	// Extract scan metadata
	scan := &storage.ImageScanV2{
		Id:             scanID,
		ImageId:        imageID,
		ScanTime:       timestamppb.Now(),
		ScannerVersion: report.GetScannerVersion(),
		BundleVersion:  report.GetBundleVersion(),
		DataSources:    report.GetDataSources(),
		// Notes will be stored as jsonb - convert to string representation
		Notes: notesToJSON(report.GetNotes()),
	}

	// Convert packages to ScanComponents and vulnerabilities to ScanFindings
	components, findings, err := convertPackagesAndVulns(scanID, imageID, metadata, report)
	if err != nil {
		return fmt.Errorf("converting packages and vulnerabilities: %w", err)
	}

	// Store via datastore
	return i.scanDataStore.UpsertScanData(ctx, &types.ScanData{
		Scan:       scan,
		Components: components,
		Findings:   findings,
	})
}

func convertPackagesAndVulns(scanID, imageID string, metadata *storage.ImageMetadata, report *v4.VulnerabilityReport) ([]*storage.ScanComponent, []*storage.ScanFinding, error) {
	layerSHAToIndex := clair.BuildSHAToIndexMap(metadata)

	pkgs := report.GetContents().GetPackages()
	if len(pkgs) == 0 {
		// Fallback to deprecated slice
		pkgs = make(map[string]*v4.Package, len(report.GetContents().GetPackagesDEPRECATED()))
		for _, pkg := range report.GetContents().GetPackagesDEPRECATED() {
			pkgs[pkg.GetId()] = pkg
		}
	}

	components := make([]*storage.ScanComponent, 0, len(pkgs))
	findings := []*storage.ScanFinding{}

	// Estimate total findings count
	totalVulns := 0
	for _, vulnIDs := range report.GetPackageVulnerabilities() {
		totalVulns += len(vulnIDs.GetValues())
	}
	findings = make([]*storage.ScanFinding, 0, totalVulns)

	for pkgID, pkg := range pkgs {
		// Create component
		componentID := componentIDFromPackage(pkgID, scanID)
		component := &storage.ScanComponent{
			Id:              componentID,
			ScanId:          scanID,
			ImageId:         imageID,
			Name:            pkg.GetName(),
			Version:         pkg.GetVersion(),
			FixedBy:         pkg.GetFixedInVersion(),
			OperatingSystem: osForPackage(report, pkgID),
			Arch:            pkg.GetArch(),
			Module:          pkg.GetModule(),
			Cpe:             pkg.GetCpe(),
			Kind:            pkg.GetKind(),
			RepositoryHint:  pkg.GetRepositoryHint(),
		}

		// Extract source package info from nested Package
		if source := pkg.GetSource(); source != nil {
			component.SourcePackageName = source.GetName()
			component.SourcePackageVersion = source.GetVersion()
		}

		// Get environment info for this package
		env := getEnvironment(report, pkgID)
		if env != nil {
			source, location := parsePackageDB(env.GetPackageDb())
			component.Source = source
			component.Location = location

			// Set layer index if available
			if layerIdx, ok := layerSHAToIndex[env.GetIntroducedIn()]; ok {
				component.HasLayerIndex = &storage.ScanComponent_LayerIndex{LayerIndex: layerIdx}
			}

			// For prototype: always mark as APPLICATION layer type
			// In production, this would check against base image layer index
			component.LayerType = storage.LayerType_APPLICATION
		}

		components = append(components, component)

		// Create findings for this component
		vulnIDs := report.GetPackageVulnerabilities()[pkgID].GetValues()
		componentFindings := convertVulnerabilities(
			scanID,
			imageID,
			componentID,
			report.GetVulnerabilities(),
			vulnIDs,
			osForPackage(report, pkgID),
		)
		findings = append(findings, componentFindings...)
	}

	return components, findings, nil
}

func convertVulnerabilities(scanID, imageID, componentID string, allVulns map[string]*v4.VulnerabilityReport_Vulnerability, vulnIDs []string, envOS string) []*storage.ScanFinding {
	if len(allVulns) == 0 || len(vulnIDs) == 0 {
		return nil
	}

	findings := make([]*storage.ScanFinding, 0, len(vulnIDs))
	uniqueAdvisories := set.NewStringSet()

	for _, vulnID := range vulnIDs {
		ccVuln, ok := allVulns[vulnID]
		if !ok {
			continue
		}

		// Use CveName from scanner, fallback to Name
		cveName := ccVuln.GetCveName()
		if cveName == "" {
			cveName = ccVuln.GetName()
		}

		advisoryID := ccVuln.GetAdvisoryId()
		if advisoryID == "" {
			// Fallback: use vuln ID if no advisory ID
			advisoryID = vulnID
		}

		// Deduplicate by advisory ID since multiple vulnIDs can map to the same advisory
		if !uniqueAdvisories.Add(advisoryID) {
			continue // Already saw this advisory for this component
		}

		findingID := findingIDFromAdvisory(advisoryID, componentID, scanID)

		finding := &storage.ScanFinding{
			Id:            findingID,
			AdvisoryId:    advisoryID,
			CveName:       cveName,
			ComponentId:   componentID,
			ScanId:        scanID,
			ImageId:       imageID,
			Severity:      normalizedSeverity(ccVuln.GetNormalizedSeverity()),
			Description:   ccVuln.GetDescription(),
			PublishedDate: ccVuln.GetIssued(),
			FixedDate:     ccVuln.GetFixedDate(),
			DataSource:    vulnDataSource(ccVuln, envOS),
			SourceName:    ccVuln.GetSourceName(),
			State:         storage.VulnerabilityState_OBSERVED,
		}

		// Set first-seen timestamps to current scan time.
		// This is an approximation; the true value should be the earliest time
		// this CVE was ever observed, but the prototype ingestion path does not
		// call the enricher that maintains that state.
		finding.FirstImageOccurrence = timestamppb.Now()
		finding.FirstSystemOccurrence = timestamppb.Now()

		// Set CVSS scores
		if err := setScoresFromMetrics(finding, ccVuln.GetCvssMetrics(), cveName); err != nil {
			// Log error but continue
			continue
		}

		// Set EPSS
		if epssMetrics := ccVuln.GetEpssMetrics(); epssMetrics != nil {
			finding.EpssProbability = epssMetrics.GetProbability()
			finding.EpssPercentile = epssMetrics.GetPercentile()
		}

		// Set fixable status
		if ccVuln.GetFixedInVersion() != "" {
			finding.IsFixable = true
			finding.FixedBy = ccVuln.GetFixedInVersion()
		}

		// Set links - parse space-separated links
		if links := ccVuln.GetLink(); links != "" {
			finding.Links = strings.Fields(links)
		}

		findings = append(findings, finding)

		// If this vuln has an Advisory (RHSA), create a second finding for it.
		// The Advisory field is populated by Scanner V4's mappers from CSAF enrichment.
		if adv := ccVuln.GetAdvisory(); adv != nil && adv.GetName() != "" && uniqueAdvisories.Add(adv.GetName()) {
			rhsaFinding := *finding // shallow copy
			rhsaFinding.AdvisoryId = adv.GetName()
			rhsaFinding.Id = findingIDFromAdvisory(adv.GetName(), componentID, scanID)
			rhsaFinding.SourceName = "Red Hat Advisory"
			rhsaFinding.Links = []string{adv.GetLink()}
			findings = append(findings, &rhsaFinding)
		}
	}

	return findings
}

func setScoresFromMetrics(finding *storage.ScanFinding, cvssMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS, cveName string) error {
	if len(cvssMetrics) == 0 {
		return nil
	}

	// Prefer non-NVD source (e.g., vendor-specific like Red Hat)
	// If multiple entries, NVD is fallback
	for _, metric := range cvssMetrics {
		if metric.GetV3() != nil {
			v3 := metric.GetV3()
			cvssV3, err := cvssv3.ParseCVSSV3(v3.GetVector())
			if err != nil {
				continue
			}

			if err := cvssv3.CalculateScores(cvssV3); err != nil {
				continue
			}

			baseScore := v3.GetBaseScore()
			if baseScore == 0.0 {
				baseScore = cvssV3.GetScore()
			}

			// Set primary CVSS if this is preferred source (non-NVD) or first entry
			if len(cvssMetrics) == 1 || metric.GetSource() != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
				finding.Cvss = baseScore
				finding.CvssVersion = storage.CvssScoreVersion_V3
			}

			// Set NVD CVSS separately if from NVD
			if metric.GetSource() == v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
				finding.NvdCvss = baseScore
				finding.NvdCvssVersion = storage.CvssScoreVersion_V3
			}
		} else if metric.GetV2() != nil {
			v2 := metric.GetV2()
			cvssV2, err := cvssv2.ParseCVSSV2(v2.GetVector())
			if err != nil {
				continue
			}

			if err := cvssv2.CalculateScores(cvssV2); err != nil {
				continue
			}

			baseScore := v2.GetBaseScore()
			if baseScore == 0.0 {
				baseScore = cvssV2.GetScore()
			}

			// Set primary CVSS if no V3 and this is preferred source
			if finding.CvssVersion == storage.CvssScoreVersion_UNKNOWN_VERSION {
				if len(cvssMetrics) == 1 || metric.GetSource() != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
					finding.Cvss = baseScore
					finding.CvssVersion = storage.CvssScoreVersion_V2
				}
			}

			// Set NVD CVSS separately if from NVD
			if metric.GetSource() == v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD {
				if finding.NvdCvssVersion == storage.CvssScoreVersion_UNKNOWN_VERSION {
					finding.NvdCvss = baseScore
					finding.NvdCvssVersion = storage.CvssScoreVersion_V2
				}
			}
		}
	}

	// If severity is unknown, derive it from CVSS
	if finding.Severity == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY && finding.Cvss > 0 {
		finding.Severity = severityFromCVSS(finding.Cvss, finding.CvssVersion)
	}

	return nil
}

func componentIDFromPackage(pkgID, scanID string) string {
	return pgSearch.IDFromPks([]string{pkgID, scanID})
}

func findingIDFromAdvisory(advisoryID, componentID, scanID string) string {
	return pgSearch.IDFromPks([]string{advisoryID, componentID, scanID})
}

func osForPackage(report *v4.VulnerabilityReport, pkgID string) string {
	env := getEnvironment(report, pkgID)
	if env == nil {
		return ""
	}

	dists := getDistributions(report)
	dist, ok := dists[env.GetDistributionId()]
	if !ok || dist.GetDid() == "" || dist.GetVersionId() == "" {
		return ""
	}

	return dist.GetDid() + ":" + dist.GetVersionId()
}

func getEnvironment(report *v4.VulnerabilityReport, pkgID string) *v4.Environment {
	environments := report.GetContents().GetEnvironments()
	if environments == nil {
		environments = report.GetContents().GetEnvironmentsDEPRECATED()
	}

	envList, ok := environments[pkgID]
	if !ok {
		return nil
	}

	envs := envList.GetEnvironments()
	if len(envs) > 0 {
		return envs[0]
	}

	return nil
}

func getDistributions(report *v4.VulnerabilityReport) map[string]*v4.Distribution {
	dists := report.GetContents().GetDistributions()
	if len(dists) == 0 {
		dists = make(map[string]*v4.Distribution, len(report.GetContents().GetDistributionsDEPRECATED()))
		for _, dist := range report.GetContents().GetDistributionsDEPRECATED() {
			dists[dist.GetId()] = dist
		}
	}
	return dists
}

func parsePackageDB(packageDB string) (storage.SourceType, string) {
	prefix, path, found := strings.Cut(packageDB, ":")
	if !found {
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
		return storage.SourceType_OS, path
	default:
		return storage.SourceType_OS, packageDB
	}
}

func vulnDataSource(ccVuln *v4.VulnerabilityReport_Vulnerability, os string) string {
	if ccVuln.GetUpdater() == "" {
		return ""
	}

	if os == "" {
		return ccVuln.GetUpdater()
	}

	return strings.Join([]string{ccVuln.GetUpdater(), os}, "::")
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

func severityFromCVSS(cvss float32, version storage.CvssScoreVersion) storage.VulnerabilitySeverity {
	switch version {
	case storage.CvssScoreVersion_V2:
		return cvssv2SeverityFromScore(cvss)
	case storage.CvssScoreVersion_V3:
		return cvssv3SeverityFromScore(cvss)
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

func cvssv2SeverityFromScore(score float32) storage.VulnerabilitySeverity {
	// CVSSv2: 0.0-3.9=Low, 4.0-6.9=Medium, 7.0-10.0=High
	switch {
	case score >= 7.0:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case score >= 4.0:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case score > 0:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

func cvssv3SeverityFromScore(score float32) storage.VulnerabilitySeverity {
	// CVSSv3: 0.1-3.9=Low, 4.0-6.9=Medium, 7.0-8.9=High, 9.0-10.0=Critical
	switch {
	case score >= 9.0:
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	case score >= 7.0:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case score >= 4.0:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case score > 0:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

func notesToJSON(notes []v4.VulnerabilityReport_Note) string {
	if len(notes) == 0 {
		return "[]"
	}

	// Simple JSON array of note values
	var noteStrings []string
	for _, note := range notes {
		switch note {
		case v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			noteStrings = append(noteStrings, `"OS_CVES_UNAVAILABLE"`)
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN:
			noteStrings = append(noteStrings, `"OS_UNAVAILABLE"`)
		}
	}

	if len(noteStrings) > 0 {
		noteStrings = append(noteStrings, `"PARTIAL_SCAN_DATA"`)
	}

	return "[" + strings.Join(noteStrings, ",") + "]"
}
