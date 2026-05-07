package phonehome

import (
	"maps"
	"testing"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withUserAgent(ua string) Headers {
	return Headers{userAgentHeaderKey: {ua}}
}

func TestCampaignFulfilled(t *testing.T) {
	doNothing := func(*APICallCampaignCriterion, Headers) {}
	captureFulfilled := func(fcc *APICallCampaign) func(*APICallCampaignCriterion, Headers) {
		return func(cc *APICallCampaignCriterion, _ Headers) {
			*fcc = append(*fcc, cc)
		}
	}
	t.Run("Empty campaign", func(t *testing.T) {
		campaign := APICallCampaign{}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
	})
	t.Run("Empty criterion", func(t *testing.T) {
		campaign := APICallCampaign{
			&APICallCampaignCriterion{},
		}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		var fulfilled APICallCampaign
		if assert.Equal(t, 1, campaign.CountFulfilled(rp, captureFulfilled(&fulfilled))) {
			assert.Same(t, campaign[0], fulfilled[0])
		}
	})
	t.Run("Nil criterion", func(t *testing.T) {
		campaign := APICallCampaign{
			nil,
		}
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GET",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
	})

	t.Run("Single criterion", func(t *testing.T) {
		campaigns := map[string]APICallCampaign{
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
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					require.NoError(t, campaign.Compile())
					assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
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
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
				})
			}
		})
	})

	t.Run("Test the list of fulfilled criteria", func(t *testing.T) {
		rp := &RequestParams{
			Headers: Headers{
				userAgentHeaderKey: {"some test user-agent"},
				"X-Header-1":       {"value 1", "value 2"},
				"X-Header-2":       {"value 1", "value 2"},
				"X-Header-3":       {"value 3", "value 4"},
			},
			Method: "GET",
			Path:   "/some/test/path",
			Code:   202,
		}
		campaign := APICallCampaign{
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
		expected := APICallCampaign{campaign[1], campaign[2], campaign[5]}
		expectedHeaders := Headers{
			userAgentHeaderKey: rp.Headers[userAgentHeaderKey],
			"X-Header-1":       {"value 2"}, // "value 1" is not matched
			"X-Header-2":       {"value 1", "value 2"},
		}

		var fulfilled APICallCampaign
		matchedHeaders := Headers{}

		assert.Equal(t, len(expected), campaign.CountFulfilled(rp,
			func(cc *APICallCampaignCriterion, h Headers) {
				fulfilled = append(fulfilled, cc)
				maps.Copy(matchedHeaders, h)
			}))
		assert.Equal(t, expected, fulfilled)
		assert.Equal(t, expectedHeaders, matchedHeaders)
	})

	t.Run("Missing headers", func(t *testing.T) {
		campaign := APICallCampaign{
			HeaderPattern("X-Header", NoHeaderOrAnyValue),
		}
		rp := &RequestParams{
			Headers: withUserAgent("some user-agent"),
			Method:  "DELETE",
			Path:    "/test/path",
			Code:    305,
		}
		assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
	})

	t.Run("Test mutiple fulfilled", func(t *testing.T) {
		rp := &RequestParams{
			Headers: withUserAgent("some test user-agent"),
			Method:  "GET",
			Path:    "/v1/test/path",
			Code:    202,
		}
		campaign := APICallCampaign{
			{
				Path: glob.Pattern("/v1/test*").Ptr(),
			},
			{
				Method: glob.Pattern("GET").Ptr(),
			},
		}
		require.NoError(t, campaign.Compile())
		var fulfilled APICallCampaign
		if assert.Equal(t, 2, campaign.CountFulfilled(rp, captureFulfilled(&fulfilled))) {
			assert.Same(t, campaign[0], fulfilled[0])
			assert.Same(t, campaign[1], fulfilled[1])
		}
	})
	t.Run("Header name and value globs", func(t *testing.T) {
		headers := Headers{
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
			campaign := APICallCampaign{
				HeaderPattern("X-Custom-*", "*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
		})

		t.Run("Glob header name and value match", func(t *testing.T) {
			campaign := APICallCampaign{
				HeaderPattern("X-Custom-*", "al*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
		})

		t.Run("Glob header name matches but value does not", func(t *testing.T) {
			campaign := APICallCampaign{
				HeaderPattern("X-Custom-*", "zzz*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
		})

		t.Run("Glob header name does not match", func(t *testing.T) {
			campaign := APICallCampaign{
				HeaderPattern("X-Missing-*", "*"),
			}
			require.NoError(t, campaign.Compile())
			assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
		})

		t.Run("Multiple glob header criteria", func(t *testing.T) {
			campaign := APICallCampaign{
				{
					Headers: GlobMap{
						"X-Custom-*": "al*",
						"X-Other":    "gam*",
					},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
		})

		t.Run("Method match captures all X- headers", func(t *testing.T) {
			campaign := APICallCampaign{
				{
					Method:  glob.Pattern("GET").Ptr(),
					Headers: GlobMap{"X-*": "*"},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))

			rpPost := &RequestParams{
				Headers: headers,
				Method:  "POST",
				Path:    "/test",
				Code:    200,
			}
			assert.Zero(t, campaign.CountFulfilled(rpPost, doNothing))
		})

		t.Run("Capture by method with missing glob header pattern", func(t *testing.T) {
			campaign := APICallCampaign{
				{
					Method:  glob.Pattern("GET").Ptr(),
					Headers: GlobMap{"X-*": NoHeaderOrAnyValue},
				},
			}
			require.NoError(t, campaign.Compile())
			assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))

			campaign[0].Method = glob.Pattern("POST").Ptr()
			assert.Zero(t, campaign.CountFulfilled(rp, doNothing))
		})
	})

	t.Run("All criteria", func(t *testing.T) {
		campaign := APICallCampaign{
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
		require.NoError(t, campaign.Compile())
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
					Headers: Headers{
						userAgentHeaderKey: {"some another user-agent"},
						"Header":           {"value"},
					},
				},
			}
			for _, rp := range rps {
				assert.Equal(t, 1, campaign.CountFulfilled(&rp, doNothing), rp.Headers.Get(userAgentHeaderKey))
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
					Headers: Headers{
						userAgentHeaderKey: {"some another user-agent 5"},
						"h":                {"---"},
					},
				},
			}
			for _, rp := range rps {
				assert.Zero(t, campaign.CountFulfilled(&rp, doNothing), rp.Headers.Get(userAgentHeaderKey))
			}
		})
	})
}

func TestCompile(t *testing.T) {
	cases := []struct {
		criterion    APICallCampaignCriterion
		errorMessage string
	}{
		{
			criterion:    APICallCampaignCriterion{},
			errorMessage: "",
		},
		{
			criterion:    *PathPattern("[b-a]"),
			errorMessage: `error parsing path pattern: failed to compile "[b-a]": hi character 'a' should be greater than lo 'b'`,
		},
	}

	for _, test := range cases {
		t.Run(test.errorMessage, func(t *testing.T) {
			err := test.criterion.Compile()
			if err == nil {
				assert.Empty(t, test.errorMessage)
			} else {
				assert.Equal(t, test.errorMessage, err.Error())
			}
		})
	}

	t.Run("nil campaign", func(t *testing.T) {
		var campaign APICallCampaign
		assert.NoError(t, campaign.Compile())
	})
}
