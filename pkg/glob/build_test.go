package glob

import (
	"testing"

	globstar "github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNFA(t *testing.T) {
	tests := map[string]struct {
		pattern string
		matches []string
		rejects []string
	}{
		"literal": {
			pattern: "/etc/passwd",
			matches: []string{"/etc/passwd"},
			rejects: []string{"/etc/shadow", "/etc", "", "/etc/passwd/extra"},
		},
		"single star": {
			pattern: "/etc/*",
			matches: []string{"/etc/passwd", "/etc/shadow", "/etc/x", "/etc/"},
			rejects: []string{"/etc/a/b", "/tmp/passwd", "/etc"},
		},
		"double star": {
			pattern: "/etc/**",
			matches: []string{"/etc/", "/etc/passwd", "/etc/a/b/c", "/etc"},
			rejects: []string{"/tmp/passwd"},
		},
		"question mark": {
			pattern: "/etc/??",
			matches: []string{"/etc/ab"},
			rejects: []string{"/etc/a", "/etc/abc", "/etc/a/"},
		},
		"character class": {
			pattern: "/etc/[abc]",
			matches: []string{"/etc/a", "/etc/b", "/etc/c"},
			rejects: []string{"/etc/d", "/etc/", "/etc/ab"},
		},
		"negated character class": {
			pattern: "/etc/[!abc]",
			matches: []string{"/etc/d", "/etc/x"},
			rejects: []string{"/etc/a", "/etc/b", "/etc/c", "/etc/ab"},
		},
		"character range": {
			pattern: "/etc/[a-c]",
			matches: []string{"/etc/a", "/etc/b", "/etc/c"},
			rejects: []string{"/etc/d", "/etc/z"},
		},
		"brace expansion": {
			pattern: "/etc/{passwd,shadow}",
			matches: []string{"/etc/passwd", "/etc/shadow"},
			rejects: []string{"/etc/group", "/etc/"},
		},
		"brace expansion with globs": {
			pattern: "/etc/{*.conf,*.cfg}",
			matches: []string{"/etc/foo.conf", "/etc/bar.cfg"},
			rejects: []string{"/etc/foo.txt", "/etc/foo.conf.bak"},
		},
		"double star slash": {
			pattern: "/a/**/b",
			matches: []string{"/a/b", "/a/x/b", "/a/x/y/z/b"},
			rejects: []string{"/a/", "/a/b/c", "/b"},
		},
		"double star at end": {
			pattern: "/a/**",
			matches: []string{"/a", "/a/", "/a/b", "/a/b/c/d"},
			rejects: []string{"/b/a"},
		},
		"double star at start": {
			pattern: "**/*.conf",
			matches: []string{"/etc/foo.conf", "foo.conf", "/a/b/c.conf"},
			rejects: []string{"/etc/foo.txt"},
		},
		"escaped star": {
			pattern: `/etc/\*`,
			matches: []string{"/etc/*"},
			rejects: []string{"/etc/passwd", "/etc/a"},
		},
		"empty pattern": {
			pattern: "",
			matches: []string{""},
			rejects: []string{"a", "/"},
		},
		"root glob": {
			pattern: "/**",
			matches: []string{"", "/", "/a", "/a/b/c"},
			rejects: []string{},
		},
		"unicode literal": {
			pattern: "/tmp/données",
			matches: []string{"/tmp/données"},
			rejects: []string{"/tmp/donnees", "/tmp/donn"},
		},
		"unicode star": {
			pattern: "/tmp/café*",
			matches: []string{"/tmp/café", "/tmp/cafébar"},
			rejects: []string{"/tmp/cafe", "/tmp/café/sub"},
		},
		"unicode char class": {
			pattern: "/tmp/[à-ÿ]",
			matches: []string{"/tmp/é", "/tmp/ü"},
			rejects: []string{"/tmp/a", "/tmp/z"},
		},
		"unicode escaped": {
			pattern: `/tmp/\é`,
			matches: []string{"/tmp/é"},
			rejects: []string{"/tmp/e"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			n, err := buildNFA(tc.pattern)
			require.NoError(t, err, "failed to build NFA for %q", tc.pattern)

			for _, s := range tc.matches {
				assert.True(t, n.accepts(s), "NFA for %q should accept %q", tc.pattern, s)
			}
			for _, s := range tc.rejects {
				assert.False(t, n.accepts(s), "NFA for %q should reject %q", tc.pattern, s)
			}
		})
	}
}

// TestBuildNFAMatchesDoublestar verifies that our NFA matches the same
// strings as the doublestar library for a set of pattern/path combinations.
func TestBuildNFAMatchesDoublestar(t *testing.T) {
	patterns := []string{
		"/etc/**",
		"/etc/*",
		"/etc/passwd",
		"/a/**/b",
		"/etc/{passwd,shadow}",
		"/etc/[a-z]*",
		"/**",
		"/etc/**/*.conf",
	}

	paths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/a/b/c",
		"/a/b",
		"/a/x/y/b",
		"/a/b/c",
		"/etc/",
		"/etc",
		"/tmp/foo",
		"/etc/foo.conf",
		"/etc/sub/bar.conf",
		"/",
	}

	for _, pattern := range patterns {
		n, err := buildNFA(pattern)
		require.NoError(t, err, "failed to build NFA for %q", pattern)

		for _, path := range paths {
			dsMatch, dsErr := globstar.Match(pattern, path)
			if dsErr != nil {
				continue
			}
			nfaMatch := n.accepts(path)
			assert.Equal(t, dsMatch, nfaMatch,
				"mismatch for pattern %q, path %q: doublestar=%v, nfa=%v",
				pattern, path, dsMatch, nfaMatch)
		}
	}
}

func TestBuildNFAErrors(t *testing.T) {
	tests := map[string]string{
		"unclosed bracket":   "/etc/[abc",
		"trailing backslash": `/etc/foo\`,
		"unclosed brace":     "/etc/{a,b",
		"nested unclosed":    "/etc/{a,[b}",
	}

	for name, pattern := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := buildNFA(pattern)
			assert.Error(t, err, "expected error for pattern %q", pattern)
		})
	}
}
