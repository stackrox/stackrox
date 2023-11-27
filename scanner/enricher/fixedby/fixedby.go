// Package fixedby implements the package-level fixed-by enricher.
//
// This implementation may be pretty fragile, as it completely depends on implementation-specific details
// of the ClairCore dependency version.
package fixedby

import (
	"context"
	"encoding/json"
	"net/url"

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
)

const (
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = "message/vnd.stackrox.scannerv4.fixedby; enricher=fixedby"

	// This appears above and must be the same.
	name = "fixedby"
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
)

var (
	_ driver.Enricher = (*Enricher)(nil)
)

type Matcher interface {
	Vulnerable(ctx context.Context, record *claircore.IndexRecord, vuln *claircore.Vulnerability) (bool, error)
}

// Enricher provides the minimum version which fixes all vulnerabilities
// in a package as enrichment to a claircore.VulnerabilityReport.
type Enricher struct{}

func (e Enricher) Name() string { return name }

// Enrich implements driver.Enricher.
//
// Enrich returns a mapping from package ID to the minimum version which fixes all vulnerabilities.
func (e Enricher) Enrich(ctx context.Context, _ driver.EnrichmentGetter, vr *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/fixedby/Enricher.Enrich")

	// package ID -> fixedBy version
	m := make(map[string]string)
	fixedBy := &claircore.IndexRecord{}
	for pkgID, pkg := range vr.Packages {
		// Do not bother if we cannot gather all the information we will need.
		if pkg == nil || len(vr.Environments[pkgID]) == 0 || len(vr.Environments[pkgID][0].RepositoryIDs) == 0 {
			continue
		}
		// If there aren't any vulnerabilities associated with this package, there is no purpose going further.
		if len(vr.PackageVulnerabilities[pkgID]) == 0 {
			continue
		}

		env := vr.Environments[pkgID][0]
		repo := env.RepositoryIDs[0]

		// Copy pkg so we can overwrite the version.
		p := *pkg
		fixedBy.Package = &p
		fixedBy.Distribution = vr.Distributions[env.DistributionID]
		fixedBy.Repository = vr.Repositories[repo]

		var pkgSemVer *semver.Version

		matcher, versionType := matcher(ctx, fixedBy)
		switch versionType {
		case unknownVersionType:
			zlog.Warn(ctx).
				Str("package", pkg.Name).
				Msg("skipping")
			continue
		case semverVersionType:
			var err error
			pkgSemVer, err = semver.NewVersion(pkg.Version)
			if err != nil {
				zlog.Warn(ctx).Str("package", pkg.Name).
					Err(err).
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
			case semverVersionType:
				// The known semver types do not rely on the Vulnerable() function to determine if a package
				// is affected by a given vulnerability. Instead, it relies on Postgres to compare versions.
				//
				// Here, we will just convert the version to semver, and do the comparisons ourselves.

				if v.FixedInVersion == "" {
					continue
				}

				vulnSemVer, err := semver.NewVersion(v.FixedInVersion)
				if err != nil {
					zlog.Warn(ctx).
						Str("package", pkg.Name).
						Str("vulnerability_id", v.ID).
						Str("vulnerability", v.Name).
						Err(err).
						Msg("skipping")
					continue
				}

				if pkgSemVer.LessThan(vulnSemVer) {
					pkgSemVer = vulnSemVer
					fixedBy.Package.Version = pkgSemVer.String()
				}
			case normalVersionType:
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
					zlog.Warn(ctx).Str("package", pkg.Name).Err(err).Msg("skipping")
					continue
				}
				if vulnerable && v.FixedInVersion != "" {
					version := v.FixedInVersion
					if versionType == urlEncodedVersionType {
						version = parseURLEncoding(version)
					}
					fixedBy.Package.Version = version
				}
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
		return Type, nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return Type, nil, err
	}

	return Type, []json.RawMessage{b}, nil
}

func parseURLEncoding(v string) string {
	decoded, err := url.ParseQuery(v)
	if err != nil {
		return v
	}
	return decoded.Get("fixed")
}

// matcher returns the related matcher and versionType to use for the given record.
func matcher(ctx context.Context, record *claircore.IndexRecord) (Matcher, versionType) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/fixedby/matcher")
	switch {
	case (*alpine.Matcher)(nil).Filter(record):
		return &alpine.Matcher{}, normalVersionType
	case (*aws.Matcher)(nil).Filter(record):
		return &aws.Matcher{}, normalVersionType
	case (*debian.Matcher)(nil).Filter(record):
		return &debian.Matcher{}, normalVersionType
	case (*gobin.Matcher)(nil).Filter(record):
		return &gobin.Matcher{}, semverVersionType
	case (*java.Matcher)(nil).Filter(record):
		return &java.Matcher{}, urlEncodedVersionType
	case (*nodejs.Matcher)(nil).Filter(record):
		return &nodejs.Matcher{}, semverVersionType
	case (*oracle.Matcher)(nil).Filter(record):
		return &oracle.Matcher{}, normalVersionType
	case (*photon.Matcher)(nil).Filter(record):
		return &photon.Matcher{}, normalVersionType
	case (*python.Matcher)(nil).Filter(record):
		return &python.Matcher{}, urlEncodedVersionType
	case rhcc.Matcher.Filter(record):
		return rhcc.Matcher, normalVersionType
	case (*rhel.Matcher)(nil).Filter(record):
		return &rhel.Matcher{}, normalVersionType
	case (*ruby.Matcher)(nil).Filter(record):
		return &ruby.Matcher{}, urlEncodedVersionType
	case (*suse.Matcher)(nil).Filter(record):
		return &suse.Matcher{}, normalVersionType
	case (*ubuntu.Matcher)(nil).Filter(record):
		return &ubuntu.Matcher{}, normalVersionType
	default:
		zlog.Error(ctx).Str("package", record.Package.Name).Msg("unable to determine matcher")
		return nil, unknownVersionType
	}
}
