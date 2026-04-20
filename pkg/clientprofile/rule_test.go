package clientprofile

import (
	"maps"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/grpc/common/requestinterceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const userAgentHeaderKey = "User-Agent"

func withUserAgent(ua string) http.Header {
	return http.Header{userAgentHeaderKey: {ua}}
}

// RequestParams holds intercepted call parameters.
type RequestParams = requestinterceptor.RequestParams

func TestRuleSet_CountMatched(t *testing.T) {
	doNothing := func(*Rule, Headers) {}
	captureFulfilled := func(fcc *RuleSet) func(*Rule, Headers) {
		return func(cc *Rule, _ Headers) {
			*fcc = append(*fcc, cc)
		}
	}
	t.Run("Empty rule set", func(t *testing.T) {
		ruleset := RuleSet{}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.Zero(t, ruleset.CountMatched(rp, doNothing))
	})
	t.Run("Empty criterion", func(t *testing.T) {
		ruleset := RuleSet{
			&Rule{},
		}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		var fulfilled RuleSet
		if assert.Equal(t, 1, ruleset.CountMatched(rp, captureFulfilled(&fulfilled))) {
			assert.Same(t, ruleset[0], fulfilled[0])
		}
	})
	t.Run("Nil criterion", func(t *testing.T) {
		ruleset := RuleSet{
			nil,
		}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GET",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.Zero(t, ruleset.CountMatched(rp, doNothing))
	})

	t.Run("Single rule", func(t *testing.T) {
		rulesets := map[string]RuleSet{
			"Code":         {(Codes(202))},
			"Codes":        {Codes(100, 202, 400)},
			"Method":       {MethodPattern("GET")},
			"Methods":      {MethodPattern("{POST,GET,PUT}")},
			"PathPattern":  {PathPattern("/some/test*")},
			"PathPatterns": {PathPattern("{/x,/some/test*,/y}")},
			"UserAgent":    {HeaderPattern("User-Agent", "*test*")},
			"UserAgents": {
				HeaderPattern("User-Agent", "*x*"),
				HeaderPattern("User-Agent", "*test*"),
				HeaderPattern("User-Agent", "*y*")},
		}

		t.Run("Test fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent("some test user-agent"),
				Method:  "GET",
				Path:    "/some/test/path",
				Code:    202,
			}
			for name, ruleset := range rulesets {
				t.Run(name, func(t *testing.T) {
					require.NoError(t, ruleset.Compile())
					assert.Equal(t, 1, ruleset.CountMatched(rp, doNothing))
				})
			}
		})

		t.Run("Test not fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent("some user-agent"),
				Method:  "DELETE",
				Path:    "/test/path",
				Code:    305,
			}
			for name, ruleset := range rulesets {
				t.Run(name, func(t *testing.T) {
					assert.Zero(t, ruleset.CountMatched(rp, doNothing))
				})
			}
		})
	})

	t.Run("Test the list of fulfilled criteria", func(t *testing.T) {
		rp := &RequestParams{
			Headers: http.Header{
				userAgentHeaderKey: {"some test user-agent"},
				"X-Header-1":       {"value 1", "value 2"},
				"X-Header-2":       {"value 1", "value 2"},
				"X-Header-3":       {"value 3", "value 4"},
			},
			Method: "GET",
			Path:   "/some/test/path",
			Code:   202,
		}
		ruleset := RuleSet{
			HeaderPattern(userAgentHeaderKey, "CI"),     // 0 - no
			HeaderPattern(userAgentHeaderKey, "*test*"), // 1 - yes
			MethodPattern("GET"),                        // 2 - yes
			MethodPattern("POST"),                       // 3 - no
			{ // 4 - no
				Codes:   []int32{203},
				Headers: GlobMap{"X-Header-1": "value ?"},
			},
			{ // 5 - yes
				Codes: []int32{404, 202},
				Headers: GlobMap{
					"X-Header-1": "value 2",
					"X-Header-2": NoHeaderOrAnyValue,
				},
			},
			{
				Codes:   []int32{500}, // 6 - no
				Headers: GlobMap{"X-Header-3": NoHeaderOrAnyValue},
			},
		}
		expected := RuleSet{ruleset[1], ruleset[2], ruleset[5]}
		expectedHeaders := Headers{
			userAgentHeaderKey: rp.Headers[userAgentHeaderKey],
			"X-Header-1":       {"value 2"}, // "value 1" is not matched
			"X-Header-2":       {"value 1", "value 2"},
		}

		var fulfilled RuleSet
		matchedHeaders := Headers{}

		assert.Equal(t, len(expected), ruleset.CountMatched(rp,
			func(cc *Rule, h Headers) {
				fulfilled = append(fulfilled, cc)
				maps.Copy(matchedHeaders, h)
			}))
		assert.Equal(t, expected, fulfilled)
		assert.Equal(t, expectedHeaders, matchedHeaders)
	})

	t.Run("Missing headers", func(t *testing.T) {
		ruleset := RuleSet{
			HeaderPattern("X-Header", NoHeaderOrAnyValue),
		}
		rp := &RequestParams{
			Headers: withUserAgent("some user-agent"),
			Method:  "DELETE",
			Path:    "/test/path",
			Code:    305,
		}
		assert.Equal(t, 1, ruleset.CountMatched(rp, doNothing))
	})

	t.Run("Test mutiple fulfilled", func(t *testing.T) {
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GET",
			Path:    "/v1/test/path",
			Code:    202,
		}
		ruleset := RuleSet{
			{
				Path: glob.Pattern("/v1/test*").Ptr(),
			},
			{
				Method: glob.Pattern("GET").Ptr(),
			},
		}
		require.NoError(t, ruleset.Compile())
		var fulfilled RuleSet
		if assert.Equal(t, 2, ruleset.CountMatched(rp, captureFulfilled(&fulfilled))) {
			assert.Same(t, ruleset[0], fulfilled[0])
			assert.Same(t, ruleset[1], fulfilled[1])
		}
	})
	t.Run("Header name and value globs", func(t *testing.T) {
		headers := http.Header{
			"X-Custom-One": {"alpha"},
			"X-Custom-Two": {"beta"},
			"X-Other":      {"gamma"},
		}
		rp := &RequestParams{
			Headers: headers,
			Method:  "GET",
			Path:    "/test",
			Code:    200,
		}

		t.Run("Glob header name matches", func(t *testing.T) {
			campaign := RuleSet{
				HeaderPattern("X-Custom-*", "*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountMatched(rp, doNothing))
		})

		t.Run("Glob header name and value match", func(t *testing.T) {
			campaign := RuleSet{
				HeaderPattern("X-Custom-*", "al*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountMatched(rp, doNothing))
		})

		t.Run("Glob header name matches but value does not", func(t *testing.T) {
			campaign := RuleSet{
				HeaderPattern("X-Custom-*", "zzz*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Zero(t, campaign.CountMatched(rp, doNothing))
		})

		t.Run("Glob header name does not match", func(t *testing.T) {
			campaign := RuleSet{
				HeaderPattern("X-Missing-*", "*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Zero(t, campaign.CountMatched(rp, doNothing))
		})

		t.Run("Multiple glob header criteria", func(t *testing.T) {
			campaign := RuleSet{
				{
					Headers: GlobMap{
						"X-Custom-*": "al*",
						"X-Other":    "gam*",
					},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountMatched(rp, doNothing))
		})

		t.Run("Method match captures all X- headers", func(t *testing.T) {
			campaign := RuleSet{
				{
					Method:  glob.Pattern("GET").Ptr(),
					Headers: GlobMap{"X-*": "*"},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountMatched(rp, doNothing))

			rpPost := &RequestParams{
				Headers: headers,
				Method:  "POST",
				Path:    "/test",
				Code:    200,
			}
			assert.Zero(t, campaign.CountMatched(rpPost, doNothing))
		})

		t.Run("Capture by method with missing glob header pattern", func(t *testing.T) {
			campaign := RuleSet{
				{
					Method:  glob.Pattern("GET").Ptr(),
					Headers: GlobMap{"X-*": NoHeaderOrAnyValue},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountMatched(rp, doNothing))

			campaign[0].Method = glob.Pattern("POST").Ptr()
			assert.Zero(t, campaign.CountMatched(rp, doNothing))
		})
	})

	t.Run("All matchers", func(t *testing.T) {
		ruleset := RuleSet{
			{
				Codes:   []int32{200, 400},
				Method:  glob.Pattern("{GET,POST}").Ptr(),
				Path:    glob.Pattern("{/v1/test*,/v2/test*}").Ptr(),
				Headers: GlobMap{"User-Agent": "*test*"},
			},
			{
				Codes:   []int32{200, 400},
				Method:  glob.Pattern("{GET,POST}").Ptr(),
				Path:    glob.Pattern("{/v1/test*,/v2/test*}").Ptr(),
				Headers: GlobMap{"User-Agent": "*toast*"},
			},
			{
				Codes:   []int32{300, 500},
				Method:  glob.Pattern("{DELETE,OPTIONS}").Ptr(),
				Path:    glob.Pattern("{/v3/test*,/v4/test*}").Ptr(),
				Headers: GlobMap{"User-Agent": "{*tooth*,*teeth*}"},
			},
			{
				Codes:  []int32{100},
				Method: glob.Pattern("PUT").Ptr(),
				Path:   glob.Pattern("/v5/*").Ptr(),
				Headers: GlobMap{
					"User-Agent": "*another*",
					"Header":     "val*",
				},
			},
		}
		require.NoError(t, ruleset.Compile())
		t.Run("All pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					Headers: withUserAgent("some test user-agent 1"),
					Method:  "GET",
					Path:    "/v1/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent("some toast user-agent 2"),
					Method:  "POST",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent("some teeth user-agent 3"),
					Method:  "DELETE",
					Path:    "/v3/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent("some teeth user-agent 4"),
					Method:  "OPTIONS",
					Path:    "/v4/test/path",
					Code:    500,
				},
				{
					Method: "PUT",
					Code:   100,
					Path:   "/v5/test",
					Headers: http.Header{
						userAgentHeaderKey: {"some another user-agent"},
						"Header":           {"value"},
					},
				},
			}
			for _, rp := range rps {
				assert.Equal(t, 1, ruleset.CountMatched(&rp, doNothing), rp.Headers.Get(userAgentHeaderKey))
			}
		})

		t.Run("All not pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					Headers: withUserAgent("some test user-agent 1"),
					Method:  "GET",
					Path:    "/v1/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent("some toast user-agent 2"),
					Method:  "DELETE",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent("some teeth user-agent 3"),
					Method:  "DELETE",
					Path:    "/v3/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent("some tooth user-agent 4"),
					Method:  "GET",
					Path:    "/v4/test/path",
					Code:    500,
				},
				{
					Method: "PUT",
					Path:   "/v5/test/path",
					Code:   100,
					Headers: http.Header{
						userAgentHeaderKey: {"some another user-agent 5"},
						"h":                {"---"},
					},
				},
			}
			for _, rp := range rps {
				assert.Zero(t, ruleset.CountMatched(&rp, doNothing), rp.Headers.Get(userAgentHeaderKey))
			}
		})
	})
}

func TestCompile(t *testing.T) {
	cases := []struct {
		rule         Rule
		errorMessage string
	}{
		{
			rule:         Rule{},
			errorMessage: "",
		},
		{
			rule:         *PathPattern("[b-a]"),
			errorMessage: `error parsing path pattern: failed to compile "[b-a]": hi character 'a' should be greater than lo 'b'`,
		},
	}

	for _, test := range cases {
		t.Run(test.errorMessage, func(t *testing.T) {
			err := test.rule.Compile()
			if err == nil {
				assert.Empty(t, test.errorMessage)
			} else {
				assert.Equal(t, test.errorMessage, err.Error())
			}
		})
	}

	t.Run("nil ruleset", func(t *testing.T) {
		var ruleset RuleSet
		assert.NoError(t, ruleset.Compile())
	})
}
