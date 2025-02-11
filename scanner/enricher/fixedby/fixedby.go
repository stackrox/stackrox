// Package fixedby implements the package-level fixed-by enricher.
//
// This implementation may be pretty fragile, as it completely depends on implementation-specific details
// of the ClairCore dependency version.
package fixedby

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/quay/claircore"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/aws"
	"github.com/quay/claircore/debian"
	"github.com/quay/claircore/gobin"
	"github.com/quay/claircore/java"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/oracle"
	"github.com/quay/claircore/photon"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/ruby"
	"github.com/quay/claircore/suse"
	"github.com/quay/claircore/ubuntu"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/fixedby"
)

// versionType represents the type of the versions associated with a matcher.
//
//go:generate stringer -type=versionType
type versionType int

const (
	unknownVersionType versionType = iota
	normalVersionType
	urlEncodedVersionType
	semverVersionType
	goSemverVersionType
)

var (
	_ driver.Enricher = (*Enricher)(nil)
)

// matcher is essentially a driver.Matcher but it only exposes the functions necessary for the enricher.
type matcher interface {
	Name() string
	Vulnerable(ctx context.Context, record *claircore.IndexRecord, vuln *claircore.Vulnerability) (bool, error)
}

// Enricher provides the minimum version which fixes all vulnerabilities
// in a package as enrichment to a claircore.VulnerabilityReport.
type Enricher struct{}

// Name implements driver.Enricher and driver.EnrichmentUpdater.
func (e Enricher) Name() string { return fixedby.Name }

// Enrich returns a mapping from package ID to the minimum version which fixes all vulnerabilities.
func (e Enricher) Enrich(ctx context.Context, _ driver.EnrichmentGetter, vr *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/fixedby/Enricher.Enrich")

	// package ID -> fixedBy version
	m := make(map[string]string)
	fixedBy := &claircore.IndexRecord{}
	for pkgID, pkg := range vr.Packages {
		// Do not bother if we cannot gather all the information we will need.
		if pkg == nil || len(vr.Environments[pkgID]) == 0 {
			continue
		}
		// If there aren't any vulnerabilities associated with this package, there is no purpose going further.
		if len(vr.PackageVulnerabilities[pkgID]) == 0 {
			continue
		}

		env := vr.Environments[pkgID][0]

		// Copy pkg so we can overwrite the version.
		p := *pkg
		fixedBy.Package = &p

		// Set the Distribution.
		// If we cannot identify the distribution, then use a dummy one,
		// as we still want to support language-level packages.
		fixedBy.Distribution = &claircore.Distribution{}
		for _, d := range vr.Distributions {
			// Just use the first one, as we only support single-distribution images.
			fixedBy.Distribution = d
			break
		}

		// Set the Repository.
		// Instead of creating an index record per repository
		// and then calling Vulnerable for each IndexRecord,
		// just use the first one, if it exists.
		if len(env.RepositoryIDs) > 0 {
			repo, ok := vr.Repositories[env.RepositoryIDs[0]]
			if !ok {
				zlog.Warn(ctx).
					Str("package", pkg.Name).
					Msg("invalid repo id, skipping")
				// We could try again, but this is unexpected and indicates some other kind of issue.
				// Just continue.
				continue
			}
			fixedBy.Repository = repo
		}

		var pkgSemver *semver.Version

		matcher, versionType := findMatcher(ctx, fixedBy)
		switch versionType {
		case unknownVersionType:
			zlog.Warn(ctx).
				Str("package", pkg.Name).
				Msg("unknown matcher, skipping")
			continue
		case semverVersionType, goSemverVersionType:
			pkgVersion := pkg.Version
			// If this is the "stdlib" package, remove the "go" prefix.
			// This is what ClairCore does for the version used in the PostgreSQL range checks
			// https://github.com/quay/claircore/blob/v1.5.28/gobin/exe.go#L57.
			if versionType == goSemverVersionType && pkg.Name == "stdlib" {
				pkgVersion = strings.TrimPrefix(pkgVersion, "go")
			}

			var err error
			pkgSemver, err = semver.NewVersion(pkgVersion)
			if err != nil {
				zlog.Warn(ctx).
					Err(err).
					Str("package", pkg.Name).
					Str("version", pkg.Version).
					Msg("skipping")
				continue
			}
		default:
		}

		vulnIDs := vr.PackageVulnerabilities[pkgID]
		// Determine the highest FixedInVersion for each vulnerability which affects the package.
		// By the end of this loop, fixedBy.Package.Version will store this version.
		for _, vulnID := range vulnIDs {
			v := vr.Vulnerabilities[vulnID]
			if v == nil {
				continue
			}

			switch versionType {
			case semverVersionType, goSemverVersionType:
				// The known semver types do not rely on the Vulnerable() function to determine if a package
				// is affected by a given vulnerability. Instead, it relies on Postgres to compare versions.
				//
				// Here, we will just convert the version to semver, and do the comparisons ourselves.

				if v.FixedInVersion == "" {
					continue
				}
				vulnSemver, err := semver.NewVersion(v.FixedInVersion)
				if err != nil {
					zlog.Warn(ctx).
						Err(err).
						Str("package", pkg.Name).
						Str("vulnerability_id", v.ID).
						Str("vulnerability", v.Name).
						Str("fixed_in_version", v.FixedInVersion).
						Msg("skipping")
					continue
				}
				if pkgSemver.LessThan(vulnSemver) {
					pkgSemver = vulnSemver
					fixedBy.Package.Version = v.FixedInVersion
				}
			case normalVersionType, urlEncodedVersionType:
				// Vulnerable indicates if fixedBy.Package
				// is vulnerable to v.
				//
				// We abuse this functionality here to tell us
				// if v.FixedInVersion is higher than the current
				// fixedBy.Package.Version.
				// If so, then we replace the current fixedBy.Package.Version
				// with v.FixedInVersion.
				vulnerable, err := matcher.Vulnerable(ctx, fixedBy, v)
				if err != nil {
					zlog.Warn(ctx).
						Err(err).
						Str("package", pkg.Name).
						Str("vulnerability_id", v.ID).
						Str("vulnerability", v.Name).
						Str("fixed_in_version", v.FixedInVersion).
						Msg("skipping")
					continue
				}
				if !vulnerable {
					continue
				}

				// Unfixed.
				if v.FixedInVersion == "" {
					continue
				}
				// Special case Ubuntu. ClairCore claims a FixedInVersion of 0 always affects
				// the package. This should not be treated like a real FixedInVersion.
				if matcher.Name() == (*ubuntu.Matcher)(nil).Name() && v.FixedInVersion == "0" {
					continue
				}

				version := v.FixedInVersion
				if versionType == urlEncodedVersionType {
					version = parseURLEncoding(version)
					if version == "" {
						continue
					}
				}
				fixedBy.Package.Version = version
			default:
				// Just log and skip.
				zlog.Warn(ctx).
					Str("version_type", versionType.String()).
					Msg("unexpected version type, skipping")
				continue
			}
		}

		// Default to assuming the package has no fixed vulnerabilities.
		m[pkgID] = ""
		// If the fixedBy version differs from the package's version,
		// then the package is affected by at least one fixable vulnerability.
		if pkg.Version != fixedBy.Package.Version {
			m[pkgID] = fixedBy.Package.Version
		}

		// Reset fixedBy.
		fixedBy.Package = nil
		fixedBy.Distribution = nil
		fixedBy.Repository = nil
	}

	if len(m) == 0 {
		return fixedby.Type, nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return fixedby.Type, nil, err
	}

	return fixedby.Type, []json.RawMessage{b}, nil
}

func parseURLEncoding(v string) string {
	decoded, err := url.ParseQuery(v)
	if err != nil {
		return ""
	}
	return decoded.Get("fixed")
}

// findMatcher returns the related matcher and versionType to use for the given record.
// If the matcher cannot be determined, this returns unknownVersionType.
//
// Note: Even though it'd be nice to use the omnimatcher provided by ClairCore,
// version comparisons became tricky without knowing more specifics about the matcher and expected
// version formatting. Doing it this way is easier for us.
func findMatcher(ctx context.Context, record *claircore.IndexRecord) (matcher, versionType) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/fixedby/matcher")
	// THE ORDERING IS SPECIFICALLY CHOSEN TO ENSURE WE CHOOSE
	// THE CORRECT MATCHER.
	// CHANGE IT AT YOUR OWN RISK.
	// Try to match language/application-related matchers first,
	// then distribution-related matcher.
	switch {
	case (*gobin.Matcher)(nil).Filter(record):
		// Note: Go versions are not always supported by the semver library.
		// For example: v1.2.2023071210521689159162
		// TODO(ROX-22533): Once ClairCore converges on a version library,
		// use that.
		return &gobin.Matcher{}, goSemverVersionType
	case (*java.Matcher)(nil).Filter(record):
		return &java.Matcher{}, urlEncodedVersionType
	case (*nodejs.Matcher)(nil).Filter(record):
		return &nodejs.Matcher{}, semverVersionType
	case (*python.Matcher)(nil).Filter(record):
		return &python.Matcher{}, urlEncodedVersionType
	case (*ruby.Matcher)(nil).Filter(record):
		return &ruby.Matcher{}, urlEncodedVersionType
	case (*alpine.Matcher)(nil).Filter(record):
		return &alpine.Matcher{}, normalVersionType
	case (*aws.Matcher)(nil).Filter(record):
		return &aws.Matcher{}, normalVersionType
	case (*debian.Matcher)(nil).Filter(record):
		return &debian.Matcher{}, normalVersionType
	case (*oracle.Matcher)(nil).Filter(record):
		return &oracle.Matcher{}, normalVersionType
	case (*photon.Matcher)(nil).Filter(record):
		return &photon.Matcher{}, normalVersionType
	case rhcc.Matcher.Filter(record):
		return rhcc.Matcher, normalVersionType
	case (*rhel.Matcher)(nil).Filter(record):
		return &rhel.Matcher{}, normalVersionType
	case (*suse.Matcher)(nil).Filter(record):
		return &suse.Matcher{}, normalVersionType
	case (*ubuntu.Matcher)(nil).Filter(record):
		return &ubuntu.Matcher{}, normalVersionType
	default:
		zlog.Error(ctx).Str("package", record.Package.Name).Msg("unable to determine matcher")
		return nil, unknownVersionType
	}
}
