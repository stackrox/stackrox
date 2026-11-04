package overlap

import (
	"regexp"
	"slices"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stripped splits a rule's raw paths the way ruleSetOf does, so witness
// cross-checks test the exact patterns the analyzer was built from.
func stripped(r *rules.Rule) (parts struct{ includes, excludes []string }) {
	parts.includes, parts.excludes = r.PathSources()
	return parts
}

// prefixMatches runs the production matcher semantics on raw patterns: a
// match starting at index 0 of the path (compiling with the standard
// regexp package the production matcher compiles with). It cross-checks
// analyzer witnesses.
func prefixMatches(patterns []string, path string) bool {
	if len(path) == 0 {
		// The empty witness is only produced for all-accepting starts.
		return len(patterns) == 0
	}
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if loc := compiled.FindStringIndex(path); loc != nil && loc[0] == 0 {
			return true
		}
	}
	return false
}

// ruleFromPaths builds a minimal rule for set-algebra tests.
func ruleFromPaths(includes, excludes []string) *rules.Rule {
	return &rules.Rule{
		Name:  "test",
		When:  "any-file-matches",
		Paths: append(slices.Clone(includes), excludes...),
	}
}

// assertWitness verifies the exactness contract end to end: when the
// analyzer reports an exact overlap, the witness must match both rules'
// production matchers with the resolver's prefix semantics and dodge both
// exclusion lists.
func assertWitness(t *testing.T, a, b *rules.Rule, wantOverlap bool) (string, bool) {
	t.Helper()
	match := Of(a, b)
	require.Equal(t, wantOverlap, match.Meets, "overlap verdict")
	if !match.Exact || !match.Meets {
		return match.Witness, match.Exact
	}
	assert.True(t, prefixMatches(stripped(a).includes, match.Witness), "witness %q should match rule a", match.Witness)
	assert.True(t, prefixMatches(stripped(b).includes, match.Witness), "witness %q should match rule b", match.Witness)
	if len(match.Witness) > 0 {
		assert.False(t, prefixMatches(stripped(a).excludes, match.Witness), "witness %q should dodge a's exclusions", match.Witness)
		assert.False(t, prefixMatches(stripped(b).excludes, match.Witness), "witness %q should dodge b's exclusions", match.Witness)
	}
	return match.Witness, match.Exact
}

func TestIntersectSets(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		a, b        *rules.Rule
		wantOverlap bool
	}{
		"should overlap on the first alternative": {
			ruleFromPaths([]string{"^(ab|cd)$"}, nil),
			ruleFromPaths([]string{"^ab$"}, nil),
			true,
		},
		"should overlap on the second alternative": {
			ruleFromPaths([]string{"^(ab|cd)$"}, nil),
			ruleFromPaths([]string{"^cd$"}, nil),
			true,
		},
		"should overlap on a shared directory": {
			ruleFromPaths([]string{"^central/sac/"}, nil),
			ruleFromPaths([]string{"^central/"}, nil),
			true,
		},
		"should stay disjoint for sibling directories": {
			ruleFromPaths([]string{"^central/"}, nil),
			ruleFromPaths([]string{"^scanner/"}, nil),
			false,
		},
		"should overlap when a documentation rule claims a real input": {
			ruleFromPaths([]string{"^.*\\.txt$"}, nil),
			ruleFromPaths([]string{"^tools/boilerplate\\.go\\.txt$"}, nil),
			true,
		},
		"should overlap where the carve-out does not reach": {
			ruleFromPaths([]string{"^.*\\.txt$"}, nil),
			ruleFromPaths([]string{"^tools/other\\.go\\.txt$"}, nil),
			true,
		},
		"should treat exclusion as set difference": {
			ruleFromPaths([]string{"^.*\\.txt$"}, []string{"!^tools/boilerplate\\.go\\.txt$"}),
			ruleFromPaths([]string{"^tools/boilerplate\\.go\\.txt$"}, nil),
			false,
		},
		"should respect the other side's exclusion": {
			ruleFromPaths([]string{"^tools/boilerplate\\.go\\.txt$"}, nil),
			ruleFromPaths([]string{"^.*\\.txt$"}, []string{"!^tools/boilerplate\\.go\\.txt$"}),
			false,
		},
		"should overlap when exclusions miss the shared files": {
			ruleFromPaths([]string{"^pkg/"}, []string{"!^pkg/x\\.md$"}),
			ruleFromPaths([]string{"^(?s:.*md)"}, []string{"!^pkg/y\\.md$"}),
			true,
		},
		"should intersect root-file and subtree patterns disjointly": {
			ruleFromPaths([]string{"^operator/[^/]+$"}, nil),
			ruleFromPaths([]string{"^operator/(build|cmd)/"}, nil),
			false,
		},
		"should fold case for (?i) documentation extensions": {
			ruleFromPaths([]string{"^(?is:.*\\.(md|txt))$"}, nil),
			ruleFromPaths([]string{"^pkg/NOTES\\.MD"}, nil),
			true,
		},
		"should find the domain README crossing": {
			ruleFromPaths([]string{"^(?is:.*\\.(md|txt))$", "!^(?s:.*/)?(testdata|fixtures)/"}, nil),
			ruleFromPaths([]string{"^central/(sac|search)/", "^central/(postgres|views)/"}, nil),
			true,
		},
		"should respect testdata exclusions from the documentation rule": {
			ruleFromPaths([]string{"^(?is:.*\\.(md|txt))$"}, []string{"!^(?s:.*/)?(testdata|fixtures)/"}),
			ruleFromPaths([]string{"^central/sac/testdata/"}, nil),
			// Only testdata paths are in the second rule's set, all excluded.
			false,
		},
		"should stay disjoint when the end anchor sits inside a group": {
			ruleFromPaths([]string{`^(?is:docs/.*\.md$)`}, nil),
			ruleFromPaths([]string{"^scanner/"}, nil),
			false,
		},
		"should overlap an exact file when the end anchor sits inside a group": {
			ruleFromPaths([]string{`^(?is:.*\.md$)`}, nil),
			ruleFromPaths([]string{`^README\.md$`}, nil),
			true,
		},
		"should keep an escaped dollar as a literal": {
			ruleFromPaths([]string{`^foo\$`}, nil),
			ruleFromPaths([]string{"^bar/"}, nil),
			false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			witness, exact := assertWitness(t, c.a, c.b, c.wantOverlap)
			assert.True(t, exact, "production-shaped patterns should analyze exactly")
			if c.wantOverlap {
				assert.NotEmpty(t, witness, "exact overlaps should carry a witness")
			}
		})
	}
}

func TestIntersectSetsWitnessShapes(t *testing.T) {
	t.Parallel()

	t.Run("the domain crossing should witness a realistic path", func(t *testing.T) {
		t.Parallel()
		a := ruleFromPaths([]string{"^(?is:.*\\.(md|txt))$"}, []string{"!^(?s:.*/)?(testdata|fixtures)/"})
		b := ruleFromPaths([]string{"^central/(sac|search)/"}, nil)
		witness, _ := assertWitness(t, a, b, true)
		assert.Contains(t, witness, "central/sac/")
		assert.Regexp(t, `(?i)\.(md|txt)$`, witness, "witness should end in a documentation extension")
	})

	t.Run("prefix-sibling directories should stay disjoint", func(t *testing.T) {
		t.Parallel()
		a := ruleFromPaths([]string{"^central/sac/deep/"}, nil)
		b := ruleFromPaths([]string{"^central/sac/deeper/"}, nil)
		// No path can start with both prefixes: after "deep" one pattern
		// demands "/" while the other demands "e".
		assertWitness(t, a, b, false)
	})
}

func TestCovers(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		skip, run *rules.Rule
		covers    bool
		exact     bool
	}{
		"should cover a file inside a directory": {
			skip: ruleFromPaths([]string{"^docs/"}, nil), run: ruleFromPaths([]string{`^docs/readme\.md$`}, nil),
			covers: true, exact: true,
		},
		"should not cover a directory from one file": {
			skip: ruleFromPaths([]string{`^docs/readme\.md$`}, nil), run: ruleFromPaths([]string{"^docs/"}, nil),
			covers: false, exact: true,
		},
		"should not cover files the skip carves out": {
			skip: ruleFromPaths([]string{"^docs/"}, []string{"!^docs/internal/"}), run: ruleFromPaths([]string{"^docs/"}, nil),
			covers: false, exact: true,
		},
		"should not prove coverage for an unmodeled caret": {
			skip: ruleFromPaths([]string{"^foo^bar"}, nil), run: ruleFromPaths([]string{"^foo"}, nil),
			covers: false, exact: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cover := Covers(c.skip, c.run)
			assert.Equal(t, c.covers, cover.Covers)
			assert.Equal(t, c.exact, cover.Exact)
		})
	}
}

func TestIntersectSetsConservative(t *testing.T) {
	t.Parallel()
	t.Run("unmodeled constructs should over-approximate without exactness", func(t *testing.T) {
		t.Parallel()
		a := ruleFromPaths([]string{"^a\\b"}, nil) // word boundary
		b := ruleFromPaths([]string{"^zzz/"}, nil)
		match := Of(a, b)
		assert.True(t, match.Meets, "conservative sets should report overlap")
		assert.False(t, match.Exact)
		assert.Empty(t, match.Witness)
	})
	t.Run("a multiline end anchor should over-approximate", func(t *testing.T) {
		t.Parallel()
		// '(?m)' makes '$' match before a newline, so foo\nX matches.
		// That is not a full-match dollar. An exact miss would be a false proof.
		end := compiledRule(t, `^(?m:foo$)`)
		line := compiledRule(t, "^foo\n")
		const path = "foo\nX"
		require.True(t, end.Matches(path))
		require.True(t, line.Matches(path))
		match := Of(end, line)
		assert.True(t, match.Meets)
		assert.False(t, match.Exact)
		cover := Covers(end, line)
		assert.False(t, cover.Covers)
		assert.False(t, cover.Exact)
	})
	t.Run("a caret after the start should over-approximate", func(t *testing.T) {
		t.Parallel()
		a := ruleFromPaths([]string{"^foo^bar"}, nil)
		b := ruleFromPaths([]string{"^scanner/"}, nil)
		match := Of(a, b)
		assert.True(t, match.Meets, "an unmodeled caret overlaps rather than ruling the pair out")
		assert.False(t, match.Exact)
		assert.Empty(t, match.Witness)
	})
}

// compiledRule is a one-pattern rule whose Matches agrees with the resolver.
func compiledRule(t *testing.T, pattern string) *rules.Rule {
	t.Helper()
	rule := ruleFromPaths([]string{pattern}, nil)
	require.NoError(t, rule.Compile())
	return rule
}

// TestAnchorScopeDoesNotProveAMiss pins patterns whose '$' or suffix does
// not apply to every alternative. The counterexample matches both rules.
func TestAnchorScopeDoesNotProveAMiss(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		a, b, path string
	}{
		"a suffix stays outside an ungrouped alternative": {
			a: `^foo|bar`, b: `^fooX$`, path: "fooX",
		},
		"a dollar on one alternative does not end the other": {
			a: `^foo|bar$`, b: `^(foo|bar$)`, path: "fooX",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a, b := compiledRule(t, c.a), compiledRule(t, c.b)
			require.True(t, a.Matches(c.path), "counterexample should match %s", c.a)
			require.True(t, b.Matches(c.path), "counterexample should match %s", c.b)
			match := Of(a, b)
			assert.True(t, match.Meets)
			assert.True(t, match.Exact)
		})
	}
}

// TestUngroupedDollarDoesNotProveCoverage pins a skip of exactly foo or bar
// against a run pattern that also matches fooX. Coverage would be a false proof.
func TestUngroupedDollarDoesNotProveCoverage(t *testing.T) {
	t.Parallel()
	skip := compiledRule(t, `^(foo|bar)$`)
	run := compiledRule(t, `^foo|bar$`)
	require.True(t, run.Matches("fooX"))
	require.False(t, skip.Matches("fooX"))
	cover := Covers(skip, run)
	assert.True(t, cover.Exact)
	assert.False(t, cover.Covers)
}
