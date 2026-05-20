//go:build scanner_db_integration

package postgres

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributions(t *testing.T) {
	ctx := test.Logging(t)
	pool := testDB(t, ctx, "distributions_test")
	store, err := InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)

	expected := []claircore.Distribution{
		{
			DID:       "rhel",
			VersionID: "9",
			Version:   "9",
		},
		{
			DID:       "rhel",
			VersionID: "10",
			Version:   "10",
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
		{
			DID: "hummingbird",
		},
	}
	const insertDists = `
INSERT INTO vuln (hash_kind, hash, dist_id, dist_version_id, dist_version, repo_name) VALUES
    ('md5', 'fake1', '',       '',      '',              'cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*'),
    ('md5', 'fake2', '',       '',      '',              'cpe:2.3:o:redhat:enterprise_linux:10.0:*:*:*:*:*:*:*'),
    ('md5', 'fake3', 'ubuntu', '22.04', '22.04 (Jammy)', ''),
    ('md5', 'fake4', 'alpine', '',      '3.17',          ''),
    ('md5', 'fake5', 'alpine', '',      '3.18',          ''),
    ('md5', 'fake6', 'debian', '10',    '10 (buster)',   ''),
    ('md5', 'fake7', '',       '',      '',              'cpe:2.3:o:redhat:enterprise_linux:%:*:*:*:*:*:*:*'),
    ('md5', 'fake8', '',       '',      '',              'cpe:2.3:o:redhat:enterprise_linux:10.1:*:*:*:*:*:*:*'),
    ('md5', 'fake9', '',       '',      '',              'cpe:2.3:a:redhat:hummingbird:1:*:*:*:*:*:*:*')`
	_, err = pool.Exec(ctx, insertDists)
	require.NoError(t, err)

	dists, err := store.Distributions(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, dists)
}

func TestHummingbirdDist(t *testing.T) {
	dist := hummingbirdDist()
	assert.Equal(t, claircore.Distribution{DID: "hummingbird"}, dist)
}

func TestHummingbirdCPE(t *testing.T) {
	for _, tc := range []struct {
		repoName string
		match    bool
	}{
		{"cpe:2.3:a:redhat:hummingbird:1:*:*:*:*:*:*:*", true},
		{"cpe:2.3:a:redhat:hummingbird:2:*:*:*:*:*:*:*", true},
		{"cpe:2.3:a:redhat:hummingbird:1.0:*:*:*:*:*:*:*", true},
		{"cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*", false},
		{"cpe:2.3:a:redhat:other_product:1:*:*:*:*:*:*:*", false},
	} {
		t.Run(tc.repoName, func(t *testing.T) {
			assert.Equal(t, tc.match, hummingbirdCPE.MatchString(tc.repoName))
		})
	}
}

func TestRHELDist(t *testing.T) {
	for _, tc := range []struct {
		repoName string
		expected claircore.Distribution
		wantErr  bool
	}{
		{
			repoName: "cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*",
			expected: claircore.Distribution{
				DID:       "rhel",
				VersionID: "8",
				Version:   "8",
			},
			wantErr: false,
		},
		{
			repoName: "cpe:2.3:o:redhat:enterprise_linux:10.0:*:*:*:*:*:*:*",
			expected: claircore.Distribution{
				DID:       "rhel",
				VersionID: "10",
				Version:   "10",
			},
			wantErr: false,
		},
		{
			repoName: "cpe:2.3:o:redhat:enterprise_linux:10.1:*:*:*:*:*:*:*",
			expected: claircore.Distribution{
				DID:       "rhel",
				VersionID: "10",
				Version:   "10",
			},
			wantErr: false,
		},
		{
			repoName: "cpe:2.3:o:redhat:enterprise_linux:2:*:*:*:*:*:*",
			wantErr:  true,
		},
	} {
		t.Run(tc.repoName, func(t *testing.T) {
			dist, err := rhelDist(tc.repoName)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, dist)
		})
	}
}
