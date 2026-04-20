package clientprofile

import (
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestHeaders(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value 1")
	h.Add("key", "value 2")
	assert.Equal(t, []string{"value 1", "value 2"}, Headers(h).Get("key"))

	h = make(http.Header)
	Headers(h).Set("key", "value1", "value2")
	assert.Equal(t, "value1", h.Get("key"))
	assert.Equal(t, []string{"value1", "value2"}, Headers(h).Get("key"))
}

func TestKeyCase(t *testing.T) {
	const keyCase1 = "TEST-key"
	const keyCase2 = "test-KEY"
	const goodValue = "good"

	testKeys := func(t *testing.T, getter interface{ Get(string) []string }) {
		assert.Equal(t, []string{goodValue}, getter.Get(keyCase1))
		assert.Equal(t, []string{goodValue}, getter.Get(keyCase2))
	}

	t.Run("test metadata.MD", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(keyCase1, goodValue)
		testKeys(t, md)
	})

	t.Run("test http.Header", func(t *testing.T) {
		// keys are canonicalized in http.Header.
		h := make(http.Header)
		h.Add(keyCase1, goodValue)
		testKeys(t, Headers(h))
	})
}

func TestGetMatchingValues(t *testing.T) {
	cases := map[string]struct {
		headers  http.Header
		key      string
		pattern  glob.Pattern
		expected []string
	}{
		"nil": {
			headers:  nil,
			key:      "Missing",
			pattern:  NoHeaderOrAnyValue,
			expected: []string{},
		},
		"absent key returns empty on NoHeaderOrAnyValue": {
			headers:  http.Header{},
			key:      "Missing",
			pattern:  NoHeaderOrAnyValue,
			expected: []string{},
		},
		"key with no values, matching pattern returns empty slice": {
			headers:  http.Header{"Key": {}},
			key:      "Key",
			pattern:  NoHeaderOrAnyValue,
			expected: []string{},
		},
		"key with no values, non-matching pattern returns nil": {
			headers:  http.Header{"Key": {}},
			key:      "Key",
			pattern:  "specific",
			expected: nil,
		},
		"single value matches pattern": {
			headers:  http.Header{"Key": {"val1"}},
			key:      "Key",
			pattern:  "val*",
			expected: []string{"val1"},
		},
		"single value does not match pattern": {
			headers:  http.Header{"Key": {"val1"}},
			key:      "Key",
			pattern:  "other*",
			expected: nil,
		},
		"multiple values, all match NoHeaderOrAnyValue": {
			headers:  http.Header{"Key": {"a", "b", "c"}},
			key:      "Key",
			pattern:  NoHeaderOrAnyValue,
			expected: []string{"a", "b", "c"},
		},
		"multiple values, pattern filters subset": {
			headers:  http.Header{"Key": {"alpha", "beta", "almond"}},
			key:      "Key",
			pattern:  "al*",
			expected: []string{"alpha", "almond"},
		},
		"multiple values, none match": {
			headers:  http.Header{"Key": {"alpha", "beta"}},
			key:      "Key",
			pattern:  "z*",
			expected: nil,
		},
		"key lookup is case-insensitive": {
			headers:  http.Header{"Content-Type": {"text/html"}},
			key:      "content-type",
			pattern:  "text/*",
			expected: []string{"text/html"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := Headers(tc.headers).getMatchingValues(tc.key, tc.pattern)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMatching_withKeyPattern(t *testing.T) {
	h := make(http.Header)
	h.Add("key-1", "value 1")
	h.Add("key-2", "value 2")
	h.Add("key-2", "value 1")
	h.Add("something-else", "value 2")
	h.Add("something-else", "value 3")

	headers := Headers(h)
	matching := headers.GetMatching("Key-*", "value 1")
	assert.Equal(t, map[string][]string{"Key-1": {"value 1"}, "Key-2": {"value 1"}}, matching)

	matching = headers.GetMatching("nope", "value 1")
	assert.Nil(t, matching)

	matching = headers.GetMatching("Key-1", "nope")
	assert.Nil(t, matching)

	matching = headers.GetMatching("Key-[1-]", "nope")
	assert.Nil(t, matching, "nil as bad pattern")

	matching = headers.GetMatching("Key-1", "value [1-]")
	assert.Nil(t, matching, "nil as bad pattern")

	matching = headers.GetMatching("Key-??", NoHeaderOrAnyValue)
	assert.Equal(t, map[string][]string{}, matching)

	matching = headers.GetMatching("*", "value [2-3]")
	assert.Equal(t, map[string][]string{"Something-Else": {"value 2", "value 3"}, "Key-2": {"value 2"}}, matching)
}

func TestHeaders_Match(t *testing.T) {

	t.Run("empty request", func(t *testing.T) {
		var h Headers
		ok, matched := h.Match(nil)
		assert.True(t, ok)
		assert.Equal(t, Headers{}, matched)

		ok, matched = h.Match(GlobMap{"header": "value"})
		assert.False(t, ok)
		assert.Nil(t, matched)

		ok, matched = h.Match(GlobMap{"header": NoHeaderOrAnyValue})
		assert.True(t, ok)
		assert.Equal(t, Headers{}, matched)
	})

	headers := Headers{
		"Empty": {},
		"One":   {"one"},
		"Two":   {"one", "two"},
	}

	tests := map[string]struct {
		patterns GlobMap
		expected Headers
	}{
		"empty": {
			expected: Headers{},
		},
		"empty not matching": {
			patterns: GlobMap{
				"Empty": "with value",
			},
			expected: nil,
		},
		"empty matching": {
			patterns: GlobMap{
				"Empty": NoHeaderOrAnyValue,
			},
			expected: Headers{"Empty": {}},
		},
		"unknown empty": {
			patterns: GlobMap{
				"Third": NoHeaderOrAnyValue,
			},
			expected: Headers{},
		},
		"one": {
			patterns: GlobMap{
				"One": "on?",
			},
			expected: Headers{"One": {"one"}},
		},
		"one-two": {
			patterns: GlobMap{
				"Two": "two",
			},
			expected: Headers{"Two": {"two"}},
		},
		"no match": {
			patterns: GlobMap{
				"Three": "x*",
			},
			expected: nil,
		},
		"one of multiple match": {
			patterns: GlobMap{
				"One": "on?",
				"Two": "x",
			},
			expected: nil,
		},
		"all of multiple match": {
			patterns: GlobMap{
				"One": "on?",
				"Two": "two",
			},
			expected: Headers{"One": {"one"}, "Two": {"two"}},
		},
		"one of multiple doesn't exist": {
			patterns: GlobMap{
				"One":   "on?",
				"Two":   "two",
				"Three": "th*",
			},
			expected: nil,
		},
		"multiple matching": {
			patterns: GlobMap{
				"Tw?": "one",
				"?wo": "two",
			},
			expected: Headers{"Two": {"one", "two"}},
		},
	}
	for name, test := range tests {
		require.NoError(t, (&Rule{Headers: test.patterns}).Compile())

		t.Run(name, func(t *testing.T) {
			ok, h := headers.Match(test.patterns)
			assert.Equal(t, test.expected != nil, ok)
			// The order of the values joined from different matching keys may differ due to the map access order.
			if assert.ElementsMatch(t, slices.Collect(maps.Keys(test.expected)), slices.Collect(maps.Keys(h))) {
				for k, values := range test.expected {
					assert.ElementsMatch(t, values, h[k])
				}
			}
		})
	}
}
