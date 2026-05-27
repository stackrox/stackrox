package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntersectionNonEmpty(t *testing.T) {
	tests := map[string]struct {
		pattern1 string
		pattern2 string
		overlap  bool
	}{
		// overlap: true
		"identical literal": {
			pattern1: "/etc/passwd",
			pattern2: "/etc/passwd",
			overlap:  true,
		},
		"literal contained in glob": {
			pattern1: "/etc/passwd",
			pattern2: "/etc/**",
			overlap:  true,
		},
		"overlapping globs": {
			pattern1: "/etc/**",
			pattern2: "/**",
			overlap:  true,
		},
		"star vs double star": {
			pattern1: "/etc/*",
			pattern2: "/etc/**",
			overlap:  true,
		},
		"brace expansion overlap": {
			pattern1: "/etc/{passwd,shadow}",
			pattern2: "/etc/passwd",
			overlap:  true,
		},
		"character class overlap": {
			pattern1: "/etc/[a-m]*",
			pattern2: "/etc/[k-z]*",
			overlap:  true,
		},
		"double star slash overlap": {
			pattern1: "/a/**/c",
			pattern2: "/a/b/c",
			overlap:  true,
		},
		"both empty patterns": {
			pattern1: "",
			pattern2: "",
			overlap:  true,
		},
		"root glob vs anything": {
			pattern1: "/**",
			pattern2: "/tmp/foo/bar",
			overlap:  true,
		},
		"question mark overlap": {
			pattern1: "/etc/??",
			pattern2: "/etc/ab",
			overlap:  true,
		},
		// overlap: false
		"disjoint literals": {
			pattern1: "/etc/passwd",
			pattern2: "/etc/shadow",
			overlap:  false,
		},
		"disjoint globs": {
			pattern1: "/etc/**",
			pattern2: "/tmp/**",
			overlap:  false,
		},
		"brace expansion no overlap": {
			pattern1: "/etc/{passwd,shadow}",
			pattern2: "/etc/group",
			overlap:  false,
		},
		"character class disjoint": {
			pattern1: "/etc/[a-c]*",
			pattern2: "/etc/[x-z]*",
			overlap:  false,
		},
		"double star slash no overlap": {
			pattern1: "/a/**/c",
			pattern2: "/b/x/c",
			overlap:  false,
		},
		"empty vs non-empty": {
			pattern1: "",
			pattern2: "/etc",
			overlap:  false,
		},
		"question mark no overlap": {
			pattern1: "/etc/??",
			pattern2: "/etc/abc",
			overlap:  false,
		},
		"negated class vs positive class disjoint": {
			pattern1: "/etc/[!a-z]*",
			pattern2: "/etc/[a-z]*",
			overlap:  false,
		},
		"single char negated vs positive disjoint": {
			pattern1: "/etc/[!a]",
			pattern2: "/etc/[a]",
			overlap:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			n1, err := buildNFA(tc.pattern1)
			require.NoError(t, err)
			n2, err := buildNFA(tc.pattern2)
			require.NoError(t, err)

			result := n1.intersects(n2)
			assert.Equal(t, tc.overlap, result,
				"PatternsOverlap(%q, %q) = %v, want %v",
				tc.pattern1, tc.pattern2, result, tc.overlap)
		})
	}
}
