package vuln

import (
	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types"
	"github.com/stackrox/rox/pkg/env"
)

// skipNotAffectedFilter, when true, drops "known not affected" vulnerability
// records that Scanner V4 never consults, before they are written to the
// database. See [ignoreVulnerability].
var skipNotAffectedFilter = env.RegisterBooleanSetting("ROX_SCANNER_V4_SKIP_UNUSED_NOT_AFFECTED", true)

// ignoreVulnerability reports whether a vulnerability record can be dropped on
// import because it is never used for matching.
//
// The Red Hat VEX feed emits a "known not affected" (Invert) record for every
// product/package a CVE does not affect. For RPM content this is the large
// majority of the feed — on a recent bundle ~4.9M of rhel-vex's ~8.9M records —
// yet ACS only ever consults not-affected assertions for *Ancestry* (RHCC /
// container) packages: the only matcher that reads Invert records is the RHCC
// matcher, and it matches on Ancestry packages against container repositories.
// A not-affected record for a non-Ancestry package is therefore dead weight: it
// is written and indexed but can never match, so importing it only costs load
// time, database size, and memory.
//
// This mirrors the exporter-side filter in claircore's rhel/vex parser (which
// only ingests OCI/RHCC known-not-affected data). Applying it here as well means
// ACS benefits immediately, even while consuming bundles produced by an exporter
// that has not yet adopted that filter.
func ignoreVulnerability(v *claircore.Vulnerability) bool {
	if v == nil {
		return false
	}
	if !skipNotAffectedFilter.BooleanSetting() {
		return false
	}
	return v.Invert && (v.Package == nil || v.Package.Kind != types.AncestryPackage)
}
