package mappers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	nvdschema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/rhel/vex"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/fixedby"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/nvd"
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
	filterPackages(r.Packages, r.Environments, r.PackageVulnerabilities)
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
		advisory, advisoryExists := csafAdvisories[v.ID]

		normalizedSeverity := toProtoV4VulnerabilitySeverity(ctx, v.NormalizedSeverity)
		if advisoryExists {
			// Replace the normalized severity for the CVE with the severity of the related Red Hat advisory.
			normalizedSeverity = toProtoV4VulnerabilitySeverityFromString(ctx, advisory.Severity)
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
		metrics, err := cvssMetrics(ctx, v, name, &nvdVuln, advisory)
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
		if advisoryExists {
			// Replace the description for the CVE with the description of the related Red Hat advisory.
			description = advisory.Description
		}
		if description == "" {
			// No description provided, so fall back to NVD.
			if len(nvdVuln.Descriptions) > 0 {
				description = nvdVuln.Descriptions[0].Value
			}
		}

		vulnPublished := v.Issued
		if advisoryExists {
			// Replace the published date for the CVE with the published date of the related Red Hat advisory.
			vulnPublished = advisory.ReleaseDate
		}
		issued := issuedTime(vulnPublished, nvdVuln.Published)
		if issued == nil {
			zlog.Warn(ctx).
				Str("vuln_id", v.ID).
				Str("vuln_name", v.Name).
				Str("vuln_updater", v.Updater).
				Bool("advisory_exists", advisoryExists).
				// Use Str instead of Time because the latter will format the time into
				// RFC3339 form, which may not be valid for this.
				Str("advisory_release_date", advisory.ReleaseDate.String()).
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

// filterPackages filters out packages from the given map.
func filterPackages(packages map[string]*claircore.Package, environments map[string][]*claircore.Environment, packageVulns map[string][]string) {
	// We only filter out Node.js packages with no known vulnerabilities (if configured to do so) at this time.
	if !env.ScannerV4PartialNodeJSSupport.BooleanSetting() {
		return
	}
	for pkgID := range packages {
		envs := environments[pkgID]
		// This is unexpected, but check here to be safe.
		if len(envs) == 0 {
			continue
		}
		if srcType, _ := scannerv4.ParsePackageDB(envs[0].PackageDB); srcType != storage.SourceType_NODEJS {
			continue
		}
		if len(packageVulns[pkgID]) == 0 {
			delete(packages, pkgID)
			delete(environments, pkgID)
			delete(packageVulns, pkgID)
		}
	}
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

// cvssValues contains CVSS-related data which are parsed from a ClairCore vulnerability report.
type cvssValues struct {
	v2Vector string
	v2Score  float32
	v3Vector string
	v3Score  float32
	source   v4.VulnerabilityReport_Vulnerability_CVSS_Source
	url      string
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
