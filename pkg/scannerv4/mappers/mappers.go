package mappers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	nvdschema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/facebookincubator/nvdtools/cvss2"
	"github.com/facebookincubator/nvdtools/cvss3"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/rhel/vex"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/fixedby"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/nvd"
	"github.com/stackrox/rox/pkg/scannerv4/updater/manual"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	nvdCVEURLPrefix    = "https://nvd.nist.gov/vuln/detail/"
	osvCVEURLPrefix    = "https://osv.dev/vulnerability/"
	redhatCVEURLPrefix = "https://access.redhat.com/security/cve/"
	// TODO(ROX-26672): Remove this when we stop tracking RHSAs as the vuln name.
	redhatErrataURLPrefix = "https://access.redhat.com/errata/"
)

var (
	severityMapping = map[claircore.Severity]v4.VulnerabilityReport_Vulnerability_Severity{
		claircore.Unknown:    v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED,
		claircore.Negligible: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
		claircore.Low:        v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
		claircore.Medium:     v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
		claircore.High:       v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
		claircore.Critical:   v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
	}

	// Updater patterns are used to determine the security updater the
	// vulnerability was detected.

	awsUpdaterPrefix = `aws-`
	osvUpdaterPrefix = `osv/`
	// RedHatUpdaterName is the name of the Red Hat VEX updater.
	RedHatUpdaterName = (*vex.Updater)(nil).Name()

	// Name patterns are regexes to match against vulnerability fields to
	// extract their name according to their updater.

	// alasIDPattern captures Amazon Linux Security Advisories.
	alasIDPattern = regexp.MustCompile(`ALAS\d*-\d{4}-\d+`)
	// cveIDPattern captures CVEs.
	cveIDPattern = regexp.MustCompile(`CVE-\d{4}-\d+`)
	// RedHatAdvisoryPattern captures known Red Hat advisory patterns.
	// TODO(ROX-26672): Remove this and show CVE as the vulnerability name.
	RedHatAdvisoryPattern = regexp.MustCompile(`(RHSA|RHBA|RHEA)-\d{4}:\d+`)

	// vulnNamePatterns is a default prioritized list of regexes to match
	// vulnerability names.
	vulnNamePatterns = []*regexp.Regexp{
		// CVE
		cveIDPattern,
		// GHSA, see: https://github.com/github/advisory-database#ghsa-ids
		regexp.MustCompile(`GHSA(-[2-9cfghjmpqrvwx]{4}){3}`),
		// Catchall
		regexp.MustCompile(`[A-Z]+-\d{4}[-:]\d+`),
	}
)

// ToProtoV4IndexReport maps claircore.IndexReport to v4.IndexReport.
func ToProtoV4IndexReport(r *claircore.IndexReport) (*v4.IndexReport, error) {
	if r == nil {
		return nil, nil
	}
	contents, err := toProtoV4Contents(r.Packages, r.Distributions, r.Repositories, r.Environments, nil)
	if err != nil {
		return nil, err
	}
	return &v4.IndexReport{
		State:    r.State,
		Success:  r.Success,
		Err:      r.Err,
		Contents: contents,
	}, nil
}

// ToProtoV4VulnerabilityReport maps claircore.VulnerabilityReport to v4.VulnerabilityReport.
func ToProtoV4VulnerabilityReport(ctx context.Context, r *claircore.VulnerabilityReport) (*v4.VulnerabilityReport, error) {
	if r == nil {
		return nil, nil
	}
	filterPackages(r)
	filterVulnerabilities(r)
	nvdVulns, err := nvdVulnerabilities(r.Enrichments)
	if err != nil {
		return nil, fmt.Errorf("internal error: parsing nvd vulns: %w", err)
	}
	epssItems, err := cveEPSS(ctx, r.Enrichments)
	if err != nil {
		return nil, fmt.Errorf("internal error: parsing EPSS items: %w", err)
	}
	// TODO(ROX-26672): Remove this line.
	// The CSAF advisories are currently a temporary solution
	// until we start showing CVEs for fixed vulnerabilities affecting
	// Red Hat products.
	csafAdvisories, err := redhatCSAFAdvisories(ctx, r.Enrichments)
	if err != nil {
		return nil, fmt.Errorf("internal error: parsing Red Hat CSAF advisories: %w", err)
	}
	vulnerabilities, err := toProtoV4VulnerabilitiesMap(ctx, r.Vulnerabilities, nvdVulns, epssItems, csafAdvisories)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	pkgFixedBy, err := pkgFixedBy(r.Enrichments)
	if err != nil {
		return nil, fmt.Errorf("internal error: parsing package-level fixedbys: %w", err)
	}
	contents, err := toProtoV4Contents(r.Packages, r.Distributions, r.Repositories, r.Environments, pkgFixedBy)
	if err != nil {
		return nil, err
	}
	return &v4.VulnerabilityReport{
		Vulnerabilities:        vulnerabilities,
		PackageVulnerabilities: toProtoV4PackageVulnerabilitiesMap(r.PackageVulnerabilities, r.Vulnerabilities, vulnerabilities),
		Contents:               contents,
	}, nil
}

// ToClairCoreIndexReport converts v4.Contents to a claircore.IndexReport.
func ToClairCoreIndexReport(contents *v4.Contents) (*claircore.IndexReport, error) {
	if contents == nil {
		return nil, errors.New("internal error: empty contents")
	}
	pkgs, err := convertSliceToMap(contents.GetPackages(), toClairCorePackage)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	dists, err := convertSliceToMap(contents.GetDistributions(), toClairCoreDistribution)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	repos, err := convertSliceToMap(contents.GetRepositories(), toClairCoreRepository)
	if err != nil {
		return nil, fmt.Errorf("internal error: %w", err)
	}
	var environments map[string][]*claircore.Environment
	if envs := contents.GetEnvironments(); envs != nil {
		environments = make(map[string][]*claircore.Environment, len(envs))
		for k, v := range envs {
			for _, env := range v.GetEnvironments() {
				ccEnv, err := toClairCoreEnvironment(env)
				if err != nil {
					return nil, err
				}
				environments[k] = append(environments[k], ccEnv)
			}
		}
	}
	return &claircore.IndexReport{
		Packages:      pkgs,
		Distributions: dists,
		Repositories:  repos,
		Environments:  environments,
	}, nil
}

func toProtoV4Contents(
	pkgs map[string]*claircore.Package,
	dists map[string]*claircore.Distribution,
	repos map[string]*claircore.Repository,
	envs map[string][]*claircore.Environment,
	pkgFixedBy map[string]string,
) (*v4.Contents, error) {
	var environments map[string]*v4.Environment_List
	if len(envs) > 0 {
		environments = make(map[string]*v4.Environment_List, len(envs))
	}
	for k, v := range envs {
		l, ok := environments[k]
		if !ok {
			l = &v4.Environment_List{}
			environments[k] = l
		}
		for _, e := range v {
			l.Environments = append(l.Environments, toProtoV4Environment(e))
		}
	}
	var packages []*v4.Package
	for _, ccP := range pkgs {
		pkg, err := toProtoV4Package(ccP)
		if err != nil {
			return nil, err
		}
		pkg.FixedInVersion = pkgFixedBy[pkg.GetId()]
		packages = append(packages, pkg)
	}
	return &v4.Contents{
		Packages:      packages,
		Distributions: convertMapToSlice(toProtoV4Distribution, dists),
		Repositories:  convertMapToSlice(toProtoV4Repository, repos),
		Environments:  environments,
	}, nil
}

func toProtoV4Package(p *claircore.Package) (*v4.Package, error) {
	if p == nil {
		return nil, nil
	}
	if p.Source != nil && p.Source.Source != nil {
		return nil, fmt.Errorf("package %q: invalid source package %q: source specifies source",
			p.ID, p.Source.ID)
	}
	// Conversion function.
	toNormalizedVersion := func(version claircore.Version) *v4.NormalizedVersion {
		return &v4.NormalizedVersion{
			Kind: version.Kind,
			V:    version.V[:],
		}
	}
	srcPkg, err := toProtoV4Package(p.Source)
	if err != nil {
		return nil, err
	}
	return &v4.Package{
		Id:                p.ID,
		Name:              p.Name,
		Version:           p.Version,
		NormalizedVersion: toNormalizedVersion(p.NormalizedVersion),
		Kind:              p.Kind,
		Source:            srcPkg,
		PackageDb:         p.PackageDB,
		RepositoryHint:    p.RepositoryHint,
		Module:            p.Module,
		Arch:              p.Arch,
		Cpe:               toCPEString(p.CPE),
	}, nil
}

// VersionID returns the distribution version ID.
func VersionID(d *claircore.Distribution) string {
	vID := d.VersionID
	if vID == "" {
		switch d.DID {
		// TODO(ROX-21678): `VersionId` is currently not populated for Alpine[1],
		//                  temporarily falling back to the version.
		//
		// [1]: https://github.com/quay/claircore/blob/88ccfbecee88d7b326b9a2fb3ab5b5f4cfa0b610/alpine/distributionscanner.go#L110-L113
		case "alpine":
			vID = d.Version
		}
	}
	return vID
}

func toProtoV4Distribution(d *claircore.Distribution) *v4.Distribution {
	if d == nil {
		return nil
	}
	return &v4.Distribution{
		Id:              d.ID,
		Did:             d.DID,
		Name:            d.Name,
		Version:         d.Version,
		VersionCodeName: d.VersionCodeName,
		VersionId:       VersionID(d),
		Arch:            d.Arch,
		Cpe:             toCPEString(d.CPE),
		PrettyName:      d.PrettyName,
	}
}

func toProtoV4Repository(r *claircore.Repository) *v4.Repository {
	if r == nil {
		return nil
	}
	return &v4.Repository{
		Id:   r.ID,
		Name: r.Name,
		Key:  r.Key,
		Uri:  r.URI,
		Cpe:  toCPEString(r.CPE),
	}
}

func toProtoV4Environment(e *claircore.Environment) *v4.Environment {
	if e == nil {
		return nil
	}
	return &v4.Environment{
		PackageDb:      e.PackageDB,
		IntroducedIn:   toDigestString(e.IntroducedIn),
		DistributionId: e.DistributionID,
		RepositoryIds:  append([]string(nil), e.RepositoryIDs...),
	}
}

func toProtoV4PackageVulnerabilitiesMap(ccPkgVulnerabilities map[string][]string, ccVulnerabilities map[string]*claircore.Vulnerability, vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability) map[string]*v4.StringList {
	if ccPkgVulnerabilities == nil {
		return nil
	}
	var pkgVulns map[string]*v4.StringList
	if len(ccPkgVulnerabilities) > 0 {
		pkgVulns = make(map[string]*v4.StringList, len(ccPkgVulnerabilities))
	}
	for id, vulnIDs := range ccPkgVulnerabilities {
		if vulnIDs == nil {
			continue
		}
		// First, deduplicate any vulnerabilities which Claircore may repeat.
		// This may happen, for example, when we match the same vulnerability to multiple CPEs
		// in Red Hat's OVAL or VEX data.
		vulnIDs = dedupeVulns(vulnIDs, ccVulnerabilities)
		// Only do the following if we want to use the CSAF enrichment data.
		if features.ScannerV4RedHatCSAF.Enabled() {
			// Next, sort by NVD CVSS score.
			sortByNVDCVSS(vulnIDs, vulnerabilities)
			// Next, deduplicate and vulnerabilities with the same Red Hat advisory name.
			// We just take the first one here, which is why we sorted by NVD CVSS score beforehand.
			// We will take the version of the advisory associated with the highest NVD CVSS score.
			vulnIDs = dedupeAdvisories(vulnIDs, vulnerabilities)
		}
		// Lastly, sort by severity in case we may still have any duplications we missed previously.
		sortBySeverity(vulnIDs, vulnerabilities)
		pkgVulns[id] = &v4.StringList{
			Values: vulnIDs,
		}
	}
	return pkgVulns
}

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

func toProtoV4VulnerabilitiesMap(
	ctx context.Context,
	vulns map[string]*claircore.Vulnerability,
	nvdVulns map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem,
	epssItems map[string]map[string]*epss.EPSSItem,
	csafAdvisories map[string]csaf.Advisory,
) (map[string]*v4.VulnerabilityReport_Vulnerability, error) {
	if vulns == nil {
		return nil, nil
	}
	var vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability
	var rhelEPSSDetails map[string]epss.EPSSItem
	if !features.ScannerV4RedHatCVEs.Enabled() {
		rhelEPSSDetails = rhelVulnsEPSS(vulns, epssItems)
	}
	for k, v := range vulns {
		if v == nil {
			continue
		}
		var pkgID string
		if v.Package != nil {
			pkgID = v.Package.ID
		}
		var distID string
		if v.Dist != nil {
			distID = v.Dist.ID
		}
		var repoID string
		if v.Repo != nil {
			repoID = v.Repo.ID
		}

		name := vulnerabilityName(v)
		// TODO(ROX-26672): Remove this line.
		csafAdvisory, csafAdvisoryExists := csafAdvisories[v.ID]

		normalizedSeverity := toProtoV4VulnerabilitySeverity(ctx, v.NormalizedSeverity)
		if csafAdvisoryExists {
			// Replace the normalized severity for the CVE with the severity of the related Red Hat advisory.
			normalizedSeverity = toProtoV4VulnerabilitySeverityFromString(ctx, csafAdvisory.Severity)
		}

		// Determine the related CVE for this vulnerability. This is necessary, as NVD and EPSS are CVE-based.
		cve, foundCVE := FindName(v, cveIDPattern)
		// Find the related NVD vuln for this vulnerability name, let it be empty if no
		// NVD vuln for that name was found.
		var nvdVuln nvdschema.CVEAPIJSON20CVEItem
		if nvdCVEs, ok := nvdVulns[v.ID]; ok {
			if v, ok := nvdCVEs[cve]; foundCVE && ok {
				nvdVuln = *v
			}
		}
		metrics, err := cvssMetrics(ctx, v, name, &nvdVuln, csafAdvisory)
		if err != nil {
			zlog.Debug(ctx).
				Err(err).
				Str("vuln_id", v.ID).
				Str("vuln_name", v.Name).
				Str("vuln_updater", v.Updater).
				Str("severity", v.Severity).
				Msg("missing severity and/or CVSS score(s): proceeding with partial values")
		}
		var preferredCVSS *v4.VulnerabilityReport_Vulnerability_CVSS
		if len(metrics) > 0 {
			// The preferred CVSS metrics will always be stored at the first index.
			preferredCVSS = metrics[0]
		}

		description := v.Description
		if csafAdvisoryExists {
			// Replace the description for the CVE with the description of the related Red Hat advisory.
			description = csafAdvisory.Description
		}
		if description == "" {
			// No description provided, so fall back to NVD.
			if len(nvdVuln.Descriptions) > 0 {
				description = nvdVuln.Descriptions[0].Value
			}
		}

		vulnPublished := v.Issued
		if csafAdvisoryExists {
			// Replace the published date for the CVE with the published date of the related Red Hat advisory.
			vulnPublished = csafAdvisory.ReleaseDate
		}
		issued := issuedTime(vulnPublished, nvdVuln.Published)
		if issued == nil {
			zlog.Warn(ctx).
				Str("vuln_id", v.ID).
				Str("vuln_name", v.Name).
				Str("vuln_updater", v.Updater).
				Bool("csaf_advisory_exists", csafAdvisoryExists).
				// Use Str instead of Time because the latter will format the time into
				// RFC3339 form, which may not be valid for this.
				Str("csaf_advisory_release_date", csafAdvisory.ReleaseDate.String()).
				Str("claircore_issued", v.Issued.String()).
				Str("nvd_published", nvdVuln.Published).
				Msg("issued time invalid: leaving empty")
		}
		var vulnEPSS *epss.EPSSItem
		if epssVulnItem, ok := epssItems[v.ID]; ok {
			if v, ok := epssVulnItem[cve]; foundCVE && ok {
				vulnEPSS = v
			}
		}
		// overwrite with RHSA EPSS score if it exists
		if rhelEPSS, ok := rhelEPSSDetails[name]; ok {
			vulnEPSS = &rhelEPSS
		}

		if vulnerabilities == nil {
			vulnerabilities = make(map[string]*v4.VulnerabilityReport_Vulnerability, len(vulns))
		}
		vulnerabilities[k] = &v4.VulnerabilityReport_Vulnerability{
			Id:                 v.ID,
			Name:               name,
			Advisory:           advisory(v),
			Description:        description,
			Issued:             issued,
			Link:               v.Links,
			Severity:           v.Severity,
			NormalizedSeverity: normalizedSeverity,
			PackageId:          pkgID,
			DistributionId:     distID,
			RepositoryId:       repoID,
			FixedInVersion:     fixedInVersion(v),
			Cvss:               preferredCVSS,
			CvssMetrics:        metrics,
		}
		if vulnEPSS != nil {
			vulnerabilities[k].EpssMetrics = &v4.VulnerabilityReport_Vulnerability_EPSS{
				ModelVersion: vulnEPSS.ModelVersion,
				Date:         vulnEPSS.Date,
				Probability:  float32(vulnEPSS.EPSS),
				Percentile:   float32(vulnEPSS.Percentile),
			}
		}
	}
	return vulnerabilities, nil
}

// issuedTime attempts to return the issued time for the vulnerability.
// If issued is non-zero, that time is preferred. Otherwise, if the nvdIssued is populated, then use that.
// Otherwise, return nil.
func issuedTime(issued time.Time, nvdIssued string) *timestamppb.Timestamp {
	if !issued.IsZero() {
		return protocompat.ConvertTimeToTimestampOrNil(&issued)
	}
	if nvdIssued != "" {
		return protoconv.ConvertTimeString(nvdIssued)
	}

	return nil
}

func toProtoV4VulnerabilitySeverity(ctx context.Context, ccSeverity claircore.Severity) v4.VulnerabilityReport_Vulnerability_Severity {
	if mappedSeverity, ok := severityMapping[ccSeverity]; ok {
		return mappedSeverity
	}
	zlog.Warn(ctx).
		Str("claircore_severity", ccSeverity.String()).
		Msgf("unknown ClairCore severity, mapping to %s", v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED.String())
	return v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED
}

// TODO(ROX-26672): Remove this.
// This is currently used to map the CSAF enrichment's severity to the equivalent proto severity.
func toProtoV4VulnerabilitySeverityFromString(ctx context.Context, severity string) v4.VulnerabilityReport_Vulnerability_Severity {
	switch {
	case strings.EqualFold("low", severity):
		return v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW
	case strings.EqualFold("moderate", severity):
		return v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE
	case strings.EqualFold("important", severity):
		return v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT
	case strings.EqualFold("critical", severity):
		return v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL
	default:
		zlog.Warn(ctx).
			Str("severity_string", severity).
			Msgf("unknown severity, mapping to %s", v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED.String())
		return v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED
	}
}

func toCPEString(c cpe.WFN) string {
	return c.BindFS()
}

func toDigestString(digest claircore.Digest) string {
	return digest.String()
}

func toClairCoreCPE(s string) (cpe.WFN, error) {
	c, err := cpe.Unbind(s)
	if err != nil {
		return c, fmt.Errorf("%q: %s", s, strings.TrimPrefix(err.Error(), "cpe: "))
	}
	return c, nil
}

func toClairCorePackage(p *v4.Package) (string, *claircore.Package, error) {
	if p == nil {
		return "", nil, nil
	}
	// Conversion function.
	toNormalizedVersion := func(v *v4.NormalizedVersion) (ccV claircore.Version) {
		ccV.Kind = v.GetKind()
		copy(ccV.V[:], v.GetV())
		return
	}
	// Fields that might fail.
	ccCPE, err := toClairCoreCPE(p.GetCpe())
	if err != nil {
		return "", nil, fmt.Errorf("package %q: %w", p.GetId(), err)
	}
	if p.GetSource().GetSource() != nil {
		return "", nil, fmt.Errorf("package %q: invalid source package %q: source specifies source",
			p.GetId(), p.GetSource().GetId())
	}
	_, src, err := toClairCorePackage(p.GetSource())
	if err != nil {
		return "", nil, err
	}
	return p.GetId(), &claircore.Package{
		ID:                p.GetId(),
		Name:              p.GetName(),
		Version:           p.GetVersion(),
		Kind:              p.GetKind(),
		Source:            src,
		PackageDB:         p.GetPackageDb(),
		RepositoryHint:    p.GetRepositoryHint(),
		NormalizedVersion: toNormalizedVersion(p.GetNormalizedVersion()),
		Module:            p.GetModule(),
		Arch:              p.GetArch(),
		CPE:               ccCPE,
	}, nil
}

func toClairCoreDistribution(d *v4.Distribution) (string, *claircore.Distribution, error) {
	if d == nil {
		return "", nil, nil
	}
	ccCPE, err := toClairCoreCPE(d.GetCpe())
	if err != nil {
		return "", nil, fmt.Errorf("distribution %q: %w", d.GetId(), err)
	}
	return d.GetId(), &claircore.Distribution{
		ID:              d.GetId(),
		DID:             d.GetDid(),
		Name:            d.GetName(),
		Version:         d.GetVersion(),
		VersionCodeName: d.GetVersionCodeName(),
		VersionID:       d.GetVersionId(),
		Arch:            d.GetArch(),
		CPE:             ccCPE,
		PrettyName:      d.GetPrettyName(),
	}, nil
}

func toClairCoreRepository(r *v4.Repository) (string, *claircore.Repository, error) {
	if r == nil {
		return "", nil, nil
	}
	ccCPE, err := toClairCoreCPE(r.GetCpe())
	if err != nil {
		return "", nil, fmt.Errorf("repository %q: %w", r.GetId(), err)
	}
	return r.GetId(), &claircore.Repository{
		ID:   r.Id,
		Name: r.Name,
		Key:  r.Key,
		URI:  r.Uri,
		CPE:  ccCPE,
	}, nil
}

func toClairCoreEnvironment(env *v4.Environment) (*claircore.Environment, error) {
	introducedIn, err := claircore.ParseDigest(env.GetIntroducedIn())
	if err != nil {
		return nil, err
	}
	return &claircore.Environment{
		PackageDB:      env.GetPackageDb(),
		IntroducedIn:   introducedIn,
		DistributionID: env.GetDistributionId(),
		RepositoryIDs:  env.GetRepositoryIds(),
	}, nil
}

// convertSliceToMap converts a slice of pointers of a generic type to a map
// based on the returned value of a conversion function that returns a string
// key, the pointer to the converted value, or error if the conversion failed.
// Nils in the slice are ignored.
func convertSliceToMap[IN any, OUT any](in []*IN, convF func(*IN) (string, *OUT, error)) (map[string]*OUT, error) {
	if len(in) == 0 {
		return nil, nil
	}
	m := make(map[string]*OUT, len(in))
	for _, v := range in {
		if v == nil {
			continue
		}
		k, ccV, err := convF(v)
		if err != nil {
			return nil, err
		}
		if ccV == nil {
			continue
		}
		m[k] = ccV
	}
	return m, nil
}

// convertMapToSlice converts generic maps keyed by strings to a slice using a
// provided conversion function.
func convertMapToSlice[IN any, OUT any](convF func(*IN) *OUT, in map[string]*IN) (out []*OUT) {
	for _, i := range in {
		out = append(out, convF(i))
	}
	return out
}

// fixedInVersion returns the fixed in string, typically provided the report's
// `FixedInVersion` as a plain string, but, in some OSV updaters, it can be an
// urlencoded string.
func fixedInVersion(v *claircore.Vulnerability) string {
	fixedIn := v.FixedInVersion
	// Try to parse url encoded params; if expected values are not found leave it.
	if q, err := url.ParseQuery(fixedIn); err == nil && q.Has("fixed") {
		fixedIn = q.Get("fixed")
	}
	return fixedIn
}

// nvdVulnerabilities look for NVD CVSS in the vulnerability report enrichments and
// returns a map of CVEs.
func nvdVulnerabilities(enrichments map[string][]json.RawMessage) (map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem, error) {
	enrichmentsList := enrichments[nvd.Type]
	if len(enrichmentsList) == 0 {
		return nil, nil
	}
	var items map[string][]nvdschema.CVEAPIJSON20CVEItem
	// The CVSS enrichment always contains only one element.
	err := json.Unmarshal(enrichmentsList[0], &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	// Returns a map of maps keyed by CVE ID due to enrichment matching on multiple
	// vulnerability fields, potentially including unrelated records--we assume the
	// caller will know how to filter what is relevant.
	ret := make(map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem)
	for ccVulnID, list := range items {
		if len(list) > 0 {
			m := make(map[string]*nvdschema.CVEAPIJSON20CVEItem)
			for idx := range list {
				vulnData := list[idx]
				m[vulnData.ID] = &vulnData
			}
			ret[ccVulnID] = m
		}
	}
	return ret, nil
}

// TODO(ROX-26672): Remove this function when we no longer require reading advisory data.
func redhatCSAFAdvisories(ctx context.Context, enrichments map[string][]json.RawMessage) (map[string]csaf.Advisory, error) {
	// Do not read CSAF data if it's not enabled.
	if !features.ScannerV4RedHatCSAF.Enabled() {
		return nil, nil
	}
	// No reason to read CSAF data when we want to only show CVEs.
	if features.ScannerV4RedHatCVEs.Enabled() {
		return nil, nil
	}
	enrichmentsList := enrichments[csaf.Type]
	if len(enrichmentsList) == 0 {
		return nil, nil
	}
	var items map[string][]csaf.Advisory
	// The CSAF enrichment always contains only one element.
	err := json.Unmarshal(enrichmentsList[0], &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	// There is only one record per ID, so remove the slice.
	ret := make(map[string]csaf.Advisory)
	for id, records := range items {
		if len(records) != 1 {
			zlog.Warn(ctx).Str("vuln_id", id).Msgf("unexpected number of CSAF enrichment records than expected (%d != 1)", len(records))
		}
		if len(records) == 0 {
			// Unexpected, but ok... Ignore this.
			continue
		}
		record := records[0]
		if record.Name == "" {
			// Unexpected, but ok... Ignore this.
			zlog.Warn(ctx).Str("vuln_id", id).Msg("advisory incomplete")
			continue
		}
		ret[id] = records[0]
	}
	return ret, nil
}

// filterPackages filters out packages from the given vulnerability report.
func filterPackages(report *claircore.VulnerabilityReport) {
	// We only filter out Node.js packages with no known vulnerabilities (if configured to do so) at this time.
	if !features.ScannerV4PartialNodeJSSupport.Enabled() {
		return
	}
	for pkgID := range report.Packages {
		envs := report.Environments[pkgID]
		// This is unexpected, but check here to be safe.
		if len(envs) == 0 {
			continue
		}
		if srcType, _ := scannerv4.ParsePackageDB(envs[0].PackageDB); srcType != storage.SourceType_NODEJS {
			continue
		}
		if len(report.PackageVulnerabilities[pkgID]) == 0 {
			delete(report.Packages, pkgID)
			delete(report.Environments, pkgID)
			delete(report.PackageVulnerabilities, pkgID)
		}
	}
}

// filterVulnerabilities filters out vulnerabilities from the given vulnerability report.
// Note: This function only modifies the report's PackageVulnerabilities map.
// It is non-trivial to remove all unreferenced vulnerabilities from the report's
// Vulnerabilities map, so we leave it untouched and accept we may send Scanner V4 clients
// extra vulnerabilities. It is up to the client to handle this.
func filterVulnerabilities(report *claircore.VulnerabilityReport) {
	// We only filter out non-Red Hat vulnerabilities found in Red Hat layers (if configured to do so) at this time.
	if !features.ScannerV4RedHatLayers.Enabled() {
		return
	}

	redhatLayers := rhccLayers(report)
	if redhatLayers.IsEmpty() {
		// This image is neither an official Red Hat image nor based on one, so nothing to do here.
		return
	}
	for pkgID, pkg := range report.Packages {
		// Sanity check.
		if pkg == nil {
			continue
		}

		envs, found := report.Environments[pkgID]
		if !found {
			// Did not find a related environment, so we cannot determine the layer.
			continue
		}

		// Just use the first environment.
		// It is possible there are multiple environments associated with this package;
		// however, for our purposes, we only need the first one,
		// as the layer index will always be the same between different environments.
		// Quay does this, too: https://github.com/quay/quay/blob/v3.10.3/data/secscan_model/secscan_v4_model.py#L583.
		// We also do this in our own client code.
		env := envs[0]
		// If this package was introduced in a non-Red Hat layer, skip it.
		if !redhatLayers.Contains(env.IntroducedIn.String()) {
			continue
		}

		var n int
		for _, vulnID := range report.PackageVulnerabilities[pkgID] {
			vuln := report.Vulnerabilities[vulnID]
			// Sanity check.
			if vuln == nil {
				continue
			}

			// If the vulnerability did not come from Red Hat's VEX data, then skip it.
			if vuln.Updater != vexUpdater {
				continue
			}

			report.PackageVulnerabilities[pkgID][n] = vulnID
			n++
		}

		// Hint to the GC the filtered vulnerabilities are no longer needed.
		clear(report.PackageVulnerabilities[pkgID][n:])
		report.PackageVulnerabilities[pkgID] = report.PackageVulnerabilities[pkgID][:n:n]

		// If there aren't any vulnerabilities from Red Hat's VEX data,
		// then delete the whole entry.
		if n == 0 {
			delete(report.PackageVulnerabilities, pkgID)
		}
	}
}

var (
	// vexUpdater is the name of the Red Hat VEX updater used by Claircore.
	vexUpdater = (*vex.Updater)(nil).Name()
	// rhccRepoName is the name of the "Gold Repository".
	rhccRepoName = rhcc.GoldRepo.Name
	// rhccRepoURI is the URI of the "Gold Repository".
	rhccRepoURI = rhcc.GoldRepo.URI
)

// rhccLayers returns a set of SHAs for the layers in official Red Hat images.
// TODO(ROX-29158): account for images built via Konflux, as they do not contain the same identifiers
// as the old build system.
func rhccLayers(report *claircore.VulnerabilityReport) set.FrozenStringSet {
	layers := set.NewStringSet()

	var rhccID string
	for id, repo := range report.Repositories {
		if repo.Name == rhccRepoName && repo.URI == rhccRepoURI {
			rhccID = id
			break
		}
	}
	if rhccID == "" {
		// Not an official Red Hat image nor based on one.
		return layers.Freeze()
	}

	for _, envs := range report.Environments {
		for _, env := range envs {
			for _, repoID := range env.RepositoryIDs {
				if repoID == rhccID {
					layers.Add(env.IntroducedIn.String())
				}
			}
		}
	}

	return layers.Freeze()
}

// pkgFixedBy unmarshals and returns the package-fixed-by enrichment, if it exists.
func pkgFixedBy(enrichments map[string][]json.RawMessage) (map[string]string, error) {
	enrichmentsList := enrichments[fixedby.Type]
	if len(enrichmentsList) == 0 {
		return nil, nil
	}
	var pkgFixedBys map[string]string
	// The fixedby enrichment always contains only one element.
	err := json.Unmarshal(enrichmentsList[0], &pkgFixedBys)
	if err != nil {
		return nil, err
	}
	if len(pkgFixedBys) == 0 {
		return nil, nil
	}
	return pkgFixedBys, nil
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

// cvssValues contains CVSS-related data which are parsed from a ClairCore vulnerability report.
type cvssValues struct {
	v2Vector string
	v2Score  float32
	v3Vector string
	v3Score  float32
	source   v4.VulnerabilityReport_Vulnerability_CVSS_Source
	url      string
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

// isOSVDBSpecificSeverity determines if the given severity is a valid severity
// from a database_specific object in the OSV data.
// See https://github.com/quay/claircore/blob/v1.5.30/updater/osv/osv.go#L686 for more information.
func isOSVDBSpecificSeverity(severity string) bool {
	switch strings.ToLower(severity) {
	case "unknown", "negligible", "low", "moderate", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

// sourceFromLinks parses the CVSS source from the vulnerability's link(s).
func sourceFromLinks(links string) v4.VulnerabilityReport_Vulnerability_CVSS_Source {
	switch {
	case strings.HasPrefix(links, nvdCVEURLPrefix):
		return v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD
	case strings.HasPrefix(links, redhatCVEURLPrefix):
		return v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT
	case strings.HasPrefix(links, osvCVEURLPrefix):
		return v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV
	default:
		return v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_UNKNOWN
	}
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

// vulnerabilityName searches the best known candidate for the vulnerability name
// in the vulnerability details. It works by matching data against well-known
// name patterns, and defaults to the original name if nothing is found.
func vulnerabilityName(vuln *claircore.Vulnerability) string {
	// Attempt per-updater patterns.
	switch {
	case strings.HasPrefix(vuln.Updater, awsUpdaterPrefix):
		if v, ok := FindName(vuln, alasIDPattern); ok {
			return v
		}
	// TODO(ROX-26672): Remove this to show CVE as the vuln name.
	case strings.EqualFold(vuln.Updater, RedHatUpdaterName):
		if !features.ScannerV4RedHatCVEs.Enabled() {
			if v, ok := FindName(vuln, RedHatAdvisoryPattern); ok {
				return v
			}
		}
	}
	// Default patterns.
	for _, p := range vulnNamePatterns {
		if v, ok := FindName(vuln, p); ok {
			return v
		}
	}
	return vuln.Name
}

// FindName searches for a vulnerability name using the specified regex in
// pre-determined fields of the vulnerability, returning the name if found.
func FindName(vuln *claircore.Vulnerability, p *regexp.Regexp) (string, bool) {
	v := p.FindString(vuln.Name)
	if v != "" {
		return v, true
	}
	v = p.FindString(vuln.Links)
	if v != "" {
		return v, true
	}
	return "", false
}

// advisory returns the vulnerability's related advisory.
//
// Only Red Hat advisories (RHSA/RHBA/RHEA) are supported at this time.
func advisory(vuln *claircore.Vulnerability) string {
	// Do not return an advisory if we do not want to separate
	// CVEs and Red Hat advisories.
	if !features.ScannerV4RedHatCVEs.Enabled() {
		return ""
	}

	// If the vulnerability is not from Red Hat's VEX data,
	// then it's definitely not an advisory we support at this time.
	if !strings.EqualFold(vuln.Updater, RedHatUpdaterName) {
		return ""
	}

	// The advisory name will be found in the vulnerability's links,
	// if it exists, so just return what we get when looking for
	// valid Red Hat advisory patterns in the links.
	return RedHatAdvisoryPattern.FindString(vuln.Links)
}

// dedupeVulns deduplicates repeat vulnerabilities out of vulnIDs and returns the result.
// This function does not guarantee ordering is preserved.
func dedupeVulns(vulnIDs []string, ccVulnerabilities map[string]*claircore.Vulnerability) []string {
	// Group each vulnerability by name.
	// This maps each name to a slice of vulnerabilities to protect against the possibility
	// Claircore finds multiple vulnerabilities with the same name for this package from different vulnerability streams.
	// In that situation, it is not clear which one may be the single, correct option to choose, so just allow for both.
	vulnsByName := make(map[string][]*claircore.Vulnerability)
OUTER:
	for _, vulnID := range vulnIDs {
		vuln := ccVulnerabilities[vulnID]
		if vuln == nil {
			continue
		}

		// Find the currently tracked vulnerabilities with the same name.
		// If this entry matches any of those, then ignore this one.
		matching := vulnsByName[vuln.Name]
		for _, match := range matching {
			if vulnsEqual(match, vuln) {
				continue OUTER
			}
		}

		// Add the unique entry to the map.
		vulnsByName[vuln.Name] = append(vulnsByName[vuln.Name], vuln)
	}

	filtered := make([]string, 0, len(vulnIDs))
	for _, vulns := range vulnsByName {
		for _, vuln := range vulns {
			filtered = append(filtered, vuln.ID)
		}
	}
	return filtered
}

// vulnsEqual determines if the vulnerabilities are essentially equal.
// That is, this function does not check all fields of the vulnerability struct,
// to prevent consumers from seeing two seemingly identical vulnerabilities
// for the same package in the same image.
//
// For example: Claircore currently returns CVE-2019-12900 twice for the bzip2-libs package
// in one particular image. The two versions of the CVE are exactly the same
// except for the repository name (cpe:/a:redhat:enterprise_linux:8::appstream vs cpe:/o:redhat:enterprise_linux:8::baseos).
// The entry for this vulnerability as it matched this package in this image may be found in
// https://security.access.redhat.com/data/oval/v2/RHEL8/rhel-8-including-unpatched.oval.xml.bz2.
// After reading the entry in this file, it is clear Claircore matched this vulnerability to this stream's
// CVE-2019-12900 entry twice (once per matching repository).
//
// The goal of this function is to make it clear those two CVE-2019-12900 findings are exactly the same.
func vulnsEqual(a, b *claircore.Vulnerability) bool {
	return a.Name == b.Name &&
		a.Description == b.Description &&
		a.Issued == b.Issued &&
		a.Links == b.Links &&
		a.Severity == b.Severity &&
		a.NormalizedSeverity == b.NormalizedSeverity &&
		a.FixedInVersion == b.FixedInVersion
}

// dedupeAdvisories deduplicates repeat advisories out of vulnIDs and returns the result.
// This function will only filter if ROX_SCANNER_V4_RED_HAT_CSAF is enabled; otherwise,
// it'll just return the original slice of vulnIDs.
// This function does not guarantee order is preserved.
func dedupeAdvisories(vulnIDs []string, protoVulns map[string]*v4.VulnerabilityReport_Vulnerability) []string {
	filtered := make([]string, 0, len(vulnIDs))
	// advisories tracks the unique advisories.
	advisories := set.NewStringSet()
	for _, vulnID := range vulnIDs {
		vuln := protoVulns[vulnID]
		if vuln == nil {
			continue
		}

		name := vuln.GetName()
		if RedHatAdvisoryPattern.MatchString(name) && !advisories.Add(name) {
			continue
		}

		filtered = append(filtered, vulnID)
	}

	return filtered
}
