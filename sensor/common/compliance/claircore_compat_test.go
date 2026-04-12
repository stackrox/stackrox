package compliance

import (
	"testing"

	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/pkg/rhctag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIndexFinishedMatchesClaircore verifies that the inlined "IndexFinished"
// string literal matches the actual claircore constant. This guards against
// upstream changes to the constant's string representation.
func TestIndexFinishedMatchesClaircore(t *testing.T) {
	assert.Equal(t, "IndexFinished", controller.IndexFinished.String(),
		"inlined IndexFinished string must match claircore constant")
}

// TestNormalizeVersionMatchesRhctag verifies that the inlined normalizeVersion
// function produces the same result as claircore's rhctag.Parse for the
// major.minor components we use.
func TestNormalizeVersionMatchesRhctag(t *testing.T) {
	cases := []string{
		"4.12",
		"4.14.0-0.nightly-2024-01-01-000000",
		"4.9",
		"5.0.1",
	}

	for _, version := range cases {
		t.Run(version, func(t *testing.T) {
			inlined := normalizeVersion(version)

			rhctagVersion, err := rhctag.Parse(version)
			require.NoError(t, err)
			m := rhctagVersion.MinorStart()
			v := m.Version(true).V

			assert.Equal(t, v[0], inlined[0], "major version mismatch for %s", version)
			assert.Equal(t, v[1], inlined[1], "minor version mismatch for %s", version)
		})
	}
}
