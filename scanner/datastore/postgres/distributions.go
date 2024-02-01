package postgres

import (
	"context"

	"github.com/quay/claircore"
)

// Distributions retrieves the currently known distributions from the database.
//
// A distribution is considered known if there exists at least one row in the vuln table which references it.
func (m *matcherStore) Distributions(ctx context.Context) ([]claircore.Distribution, error) {
	const selectDists = `SELECT DISTINCT dist_id, dist_version_id, dist_version FROM vuln`

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
		)
		if err := rows.Scan(&dID, &versionID, &version); err != nil {
			return nil, err
		}
		if dID == "" {
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
