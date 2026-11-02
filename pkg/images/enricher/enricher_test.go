package enricher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchOption_forceReindex(t *testing.T) {
	cases := map[string]struct {
		option   FetchOption
		expected bool
	}{
		"ForceRefetchScansOnly forces reindex": {
			option:   ForceRefetchScansOnly,
			expected: true,
		},
		"UseImageNamesRefetchCachedValues forces reindex": {
			// This is the FetchOption actually used by ScanImage's force:true, the
			// SBOM generation handler's scanner-mismatch retry, and the detection
			// service's force handling. All three already bypass Central's own
			// cached scan and re-ask the configured scanner (fetchFromDatabase
			// treats the existing image as not found for this option) -- this
			// signal lets a scanner also discard its own internal cache.
			option:   UseImageNamesRefetchCachedValues,
			expected: true,
		},
		"UseCachesIfPossible does not force reindex": {
			option:   UseCachesIfPossible,
			expected: false,
		},
		"ForceRefetch forces reindex": {
			// Used by ScanImageInternal and Central's one-time startup reprocessing pass,
			// both of which already bypass Central's cached scan via the same
			// fetchFromDatabase short-circuit as UseImageNamesRefetchCachedValues.
			option:   ForceRefetch,
			expected: true,
		},
		"ForceRefetchCachedValuesOnly does not force reindex": {
			option:   ForceRefetchCachedValuesOnly,
			expected: false,
		},
		"ForceRefetchSignaturesOnly does not force reindex": {
			option:   ForceRefetchSignaturesOnly,
			expected: false,
		},
		"ForceRefetchMetadataOnly does not force reindex": {
			option:   ForceRefetchMetadataOnly,
			expected: false,
		},
		"NoExternalMetadata does not force reindex": {
			option:   NoExternalMetadata,
			expected: false,
		},
		"IgnoreExistingImages does not force reindex": {
			option:   IgnoreExistingImages,
			expected: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, c.option.forceReindex())
		})
	}
}
