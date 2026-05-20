package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/quay/claircore"
)

// rhelCPE represents the expected pattern to identify a CPE which indicates a RHEL major version.
// The purpose of this is to identify the major version represented by this CPE.
var rhelCPE = regexp.MustCompile(`^cpe:2\.3:o:redhat:enterprise_linux:(\d+)(?:\.\d+)*:\*:\*:\*:\*:\*:\*:\*$`)

var hummingbirdCPE = regexp.MustCompile(`^cpe:2\.3:a:redhat:hummingbird:\d+(?:\.\d+)*:\*:\*:\*:\*:\*:\*:\*$`)

// Distributions retrieves the currently known distributions from the database.
//
// A distribution is considered known if there exists at least one row in the vuln table which references it.
func (m *matcherStore) Distributions(ctx context.Context) ([]claircore.Distribution, error) {

	// As of ClairCore v1.5.29, all distributions may be identified by dist_id, dist_version_id, and dist_version except for RHEL.
	// As of this version of ClairCore, RHEL vulnerabilities are not associated with a specific RHEL version, but rather just the CPE(s).
	// So, to capture all RHEL distributions, we must also search for rows with column repo_name matching the expected RHEL-major-version-identifying CPE.
	// Hummingbird is a Red Hat product whose vulns also use CPE-based matching with no dist_* fields populated.
	const selectDists = `SELECT DISTINCT dist_id, dist_version_id, dist_version, repo_name FROM vuln WHERE repo_name = '' OR repo_name LIKE 'cpe:2.3:o:redhat:enterprise_linux:%:*:*:*:*:*:*:*' OR repo_name LIKE 'cpe:2.3:a:redhat:hummingbird:%:*:*:*:*:*:*:*'`

	rows, err := m.pool.Query(ctx, selectDists)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uniqueDists := make(map[claircore.Distribution]struct{})
	for rows.Next() {
		var (
			dID       string
			versionID string
			version   string
			repoName  string
		)
		if err := rows.Scan(&dID, &versionID, &version, &repoName); err != nil {
			return nil, err
		}

		dist := claircore.Distribution{
			DID:       dID,
			VersionID: versionID,
			Version:   version,
		}
		if repoName != "" {
			switch {
			case rhelCPE.MatchString(repoName):
				dist, err = rhelDist(repoName)
			case hummingbirdCPE.MatchString(repoName):
				dist = hummingbirdDist()
			default:
				err = fmt.Errorf("unexpected repo name: %s", repoName)
			}
			if err != nil {
				slog.WarnContext(ctx, "failed to parse repo_name; skipping...", "reason", err)
				continue
			}
		}

		uniqueDists[dist] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	dists := make([]claircore.Distribution, 0, len(uniqueDists))
	for dist := range uniqueDists {
		dists = append(dists, dist)
	}

	return dists, nil
}

func hummingbirdDist() claircore.Distribution {
	return claircore.Distribution{
		DID: "hummingbird",
	}
}

func rhelDist(repoName string) (claircore.Distribution, error) {
	m := rhelCPE.FindStringSubmatch(repoName)
	if len(m) != 2 {
		return claircore.Distribution{}, fmt.Errorf("unexpected repo name: %s", repoName)
	}
	return claircore.Distribution{
		DID:       "rhel",
		VersionID: m[1],
		Version:   m[1],
	}, nil
}
