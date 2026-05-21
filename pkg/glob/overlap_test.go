package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternsOverlap(t *testing.T) {
	tests := map[string]struct {
		pattern1 string
		pattern2 string
		overlap  bool
	}{
		// AIDE-style: negating /etc/passwd when monitoring /etc/**
		"negate passwd under etc glob": {
			pattern1: "/etc/passwd",
			pattern2: "/etc/**",
			overlap:  true,
		},
		// AIDE-style: negating /var/** when only monitoring /etc/**
		"negate var when monitoring etc": {
			pattern1: "/var/**",
			pattern2: "/etc/**",
			overlap:  false,
		},
		// AIDE-style: negating specific files under monitored tree
		"negate shadow under etc glob": {
			pattern1: "/etc/shadow",
			pattern2: "/etc/**",
			overlap:  true,
		},
		// AIDE-style: brace expansion overlap
		"negate passwd via brace": {
			pattern1: "/etc/{passwd,shadow}",
			pattern2: "/etc/**",
			overlap:  true,
		},
		// Negating a subtree under a monitored tree
		"negate subtree": {
			pattern1: "/etc/systemd/**",
			pattern2: "/etc/**",
			overlap:  true,
		},
		// Completely disjoint trees
		"disjoint trees": {
			pattern1: "/home/**",
			pattern2: "/var/**",
			overlap:  false,
		},
		// Root glob overlaps with everything
		"root glob vs specific": {
			pattern1: "/**",
			pattern2: "/tmp/foo/bar/baz",
			overlap:  true,
		},
		// Identical patterns
		"identical globs": {
			pattern1: "/etc/**/*.conf",
			pattern2: "/etc/**/*.conf",
			overlap:  true,
		},
		// Wildcard extension overlap
		"conf vs cfg no overlap": {
			pattern1: "/etc/**/*.conf",
			pattern2: "/etc/**/*.cfg",
			overlap:  false,
		},
		// Deep path with double star
		"deep path overlap": {
			pattern1: "/a/**/z",
			pattern2: "/a/b/c/d/z",
			overlap:  true,
		},
		// Character class used in negation
		"char class negate": {
			pattern1: "/etc/[a-m]*",
			pattern2: "/etc/**",
			overlap:  true,
		},
		"unicode paths overlap": {
			pattern1: "/tmp/données/**",
			pattern2: "/tmp/données/résumé",
			overlap:  true,
		},
		"unicode paths disjoint": {
			pattern1: "/tmp/données/**",
			pattern2: "/tmp/café/**",
			overlap:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := PatternsOverlap(tc.pattern1, tc.pattern2)
			require.NoError(t, err)
			assert.Equal(t, tc.overlap, result,
				"PatternsOverlap(%q, %q) = %v, want %v",
				tc.pattern1, tc.pattern2, result, tc.overlap)
		})
	}
}

func TestPatternsOverlapErrors(t *testing.T) {
	tests := map[string]struct {
		pattern1 string
		pattern2 string
	}{
		"bad pattern1": {
			pattern1: "/etc/[abc",
			pattern2: "/etc/**",
		},
		"bad pattern2": {
			pattern1: "/etc/**",
			pattern2: "/etc/{a,b",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := PatternsOverlap(tc.pattern1, tc.pattern2)
			assert.Error(t, err)
		})
	}
}

func BenchmarkPatternsOverlap(b *testing.B) {
	benchmarks := map[string][2]string{
		"literal_vs_literal":     {"/etc/passwd", "/etc/shadow"},
		"literal_vs_glob":        {"/etc/passwd", "/etc/**"},
		"glob_vs_glob":           {"/etc/**", "/var/**"},
		"doublestar_vs_deep":     {"/a/**/z", "/a/b/c/d/e/f/g/z"},
		"root_glob_vs_literal":   {"/**", "/tmp/foo/bar/baz"},
		"brace_vs_glob":          {"/etc/{passwd,shadow,group}", "/etc/**"},
		"charclass_vs_glob":      {"/etc/[a-m]*", "/etc/**"},
		"complex_vs_complex":     {"/etc/**/*.conf", "/etc/systemd/**/*.conf"},
		"many_braces":            {"/etc/{a,b,c,d,e,f}", "/etc/{d,e,f,g,h,i}"},
		"deep_doublestar_vs_dbl": {"/a/**/b/**/c", "/a/**/c"},
	}

	for name, pair := range benchmarks {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				_, _ = PatternsOverlap(pair[0], pair[1])
			}
		})
	}
}

func TestPatternsOverlapSymmetric(t *testing.T) {
	pairs := [][2]string{
		{"/etc/**", "/etc/passwd"},
		{"/var/**", "/etc/**"},
		{"/**", "/tmp/foo"},
		{"/etc/[a-m]*", "/etc/[n-z]*"},
	}

	for _, pair := range pairs {
		ab, err1 := PatternsOverlap(pair[0], pair[1])
		ba, err2 := PatternsOverlap(pair[1], pair[0])
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, ab, ba,
			"PatternsOverlap(%q, %q) = %v but PatternsOverlap(%q, %q) = %v",
			pair[0], pair[1], ab, pair[1], pair[0], ba)
	}
}
