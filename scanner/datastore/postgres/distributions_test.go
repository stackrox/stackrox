//go:build scanner_db_integration

package postgres

import (
	"context"
	"testing"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributions(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t, ctx, "distributions_test")
	store, err := InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)

	expected := []claircore.Distribution{
		{
			DID:       "rhel",
			VersionID: "8",
			Version:   "8",
		},
		{
			DID:       "rhel",
			VersionID: "9",
			Version:   "9",
		},
		{
			DID:       "ubuntu",
			VersionID: "22.04",
			Version:   "22.04 (Jammy)",
		},
		{
			DID:       "debian",
			VersionID: "10",
			Version:   "10 (buster)",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.17",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.18",
		},
	}
	const insertDists = `
INSERT INTO vuln (hash_kind, hash, dist_id, dist_version_id, dist_version) VALUES
    ('md5', 'fake1', 'rhel', '8', '8'),
    ('md5', 'fake2', 'rhel', '9', '9'),
    ('md5', 'fake3', 'ubuntu', '22.04', '22.04 (Jammy)'),
    ('md5', 'fake4', 'alpine', '', '3.17'),
    ('md5', 'fake5', 'alpine', '', '3.18'),
    ('md5', 'fake6', 'debian', '10', '10 (buster)')`
	_, err = pool.Exec(ctx, insertDists)
	require.NoError(t, err)

	dists, err := store.Distributions(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, dists)
}
