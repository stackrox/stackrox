package postgres

import (
	"context"
	"fmt"
	"regexp"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
)

var rhelCPE = regexp.MustCompile(`cpe:2\.3:o:redhat:enterprise_linux:(\d+):\*:\*:\*:\*:\*:\*:\*`)

// Distributions retrieves the currently known distributions from the database.
//
// A distribution is considered known if there exists at least one row in the vuln table which references it.
func (m *matcherStore) Distributions(ctx context.Context) ([]claircore.Distribution, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/distributions/Distributions")

	const selectDists = `SELECT DISTINCT dist_id, dist_version_id, dist_version, repo_name FROM vuln WHERE repo_name = '' OR repo_name LIKE 'cpe:2.3:o:redhat:enterprise_linux:%:*:*:*:*:*:*:*'`

	rows, err := m.pool.Query(ctx, selectDists)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dists []claircore.Distribution
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
		if repoName != "" {
			dist, err := rhelDist(repoName)
			if err != nil {
				zlog.Warn(ctx).Err(err).Msg("failed to fetch distribution; skipping...")
				continue
			}
			dists = append(dists, dist)
			continue
		}

		dists = append(dists, claircore.Distribution{
			DID:       dID,
			VersionID: versionID,
			Version:   version,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dists, nil
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
