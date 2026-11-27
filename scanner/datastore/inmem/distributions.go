package inmem

import (
	"regexp"

	"github.com/quay/claircore"
)

// rhelCPE and hummingbirdCPE mirror scanner/datastore/postgres/distributions.go
// so KnownDistributions matches the Postgres matcher store for CPE-based
// distributions whose vulnerabilities do not populate dist_* fields.
var (
	rhelCPE        = regexp.MustCompile(`^cpe:2\.3:o:redhat:enterprise_linux:(\d+)(?:\.\d+)*:\*:\*:\*:\*:\*:\*:\*$`)
	hummingbirdCPE = regexp.MustCompile(`^cpe:2\.3:a:redhat:hummingbird:\d+(?:\.\d+)*:\*:\*:\*:\*:\*:\*:\*$`)
)

// extractKnownDist returns the known-distribution key contributed by a
// vulnerability row, replicating scanner/datastore/postgres/distributions.go.
func extractKnownDist(d *claircore.Distribution, repoName string) (claircore.Distribution, bool) {
	if repoName == "" {
		if d == nil || (d.DID == "" && d.VersionID == "" && d.Version == "") {
			return claircore.Distribution{}, false
		}
		return claircore.Distribution{
			DID:       d.DID,
			VersionID: d.VersionID,
			Version:   d.Version,
		}, true
	}
	if m := rhelCPE.FindStringSubmatch(repoName); len(m) == 2 {
		return claircore.Distribution{
			DID:       "rhel",
			VersionID: m[1],
			Version:   m[1],
		}, true
	}
	if hummingbirdCPE.MatchString(repoName) {
		return claircore.Distribution{
			DID: "hummingbird",
		}, true
	}
	return claircore.Distribution{}, false
}
