package phonehome

import (
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
		assert.Equal(t, 1, campaign.CountFulfilled(rp, doNothing))
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
		assert.Equal(t, 2, campaign.CountFulfilled(rp, doNothing))
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
