package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withUserAgent(_ *testing.T, headers map[string][]string, ua string) func(string) []string {
	return func(s string) []string {
		if http.CanonicalHeaderKey(s) == userAgentHeaderKey {
			return []string{ua}
		}
		return headers[s]
	}
}

func TestCampaignFulfilled(t *testing.T) {
	t.Run("Empty campaign", func(t *testing.T) {
		campaign := APICallCampaign{}
		rp := &RequestParams{
			Headers: withUserAgent(t, nil, "some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.False(t, campaign.IsFulfilled(rp))
	})
	t.Run("Empty criterion", func(t *testing.T) {
		campaign := APICallCampaign{
			&APICallCampaignCriterion{},
		}
		rp := &RequestParams{
			Headers: withUserAgent(t, nil, "some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.True(t, campaign.IsFulfilled(rp))
	})
	t.Run("Nil criterion", func(t *testing.T) {
		campaign := APICallCampaign{
			nil,
		}
		rp := &RequestParams{
			Headers: withUserAgent(t, nil, "some test user-agent"),
			Method:  "GET",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.False(t, campaign.IsFulfilled(rp))
	})

	t.Run("Single criterion", func(t *testing.T) {
		campaigns := map[string]APICallCampaign{
			"Code": []*APICallCampaignCriterion{
				{
					Codes: []int32{202},
				},
			},
			"Codes": []*APICallCampaignCriterion{
				{
					Codes: []int32{100, 202, 400},
				},
			},
			"Method": []*APICallCampaignCriterion{
				{
					Method: Pattern("GET").Ptr(),
				},
			},
			"Methods": []*APICallCampaignCriterion{
				{
					Method: Pattern("{POST,GET,PUT}").Ptr(),
				},
			},
			"PathPattern": []*APICallCampaignCriterion{
				{
					Path: Pattern("/some/test*").Ptr(),
				},
			},
			"PathPatterns": []*APICallCampaignCriterion{
				{
					Path: Pattern("{/x,/some/test*,/y}").Ptr(),
				},
			},
			"UserAgent": []*APICallCampaignCriterion{
				{
					Headers: map[string]Pattern{
						"User-Agent": "*test*",
					},
				},
			},
			"UserAgents": []*APICallCampaignCriterion{
				{
					Headers: map[string]Pattern{"User-Agent": "*x*"},
				},
				{
					Headers: map[string]Pattern{"User-Agent": "*test*"},
				}, {
					Headers: map[string]Pattern{"User-Agent": "*y*"},
				},
			},
		}

		t.Run("Test fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent(t, nil, "some test user-agent"),
				Method:  "GET",
				Path:    "/some/test/path",
				Code:    202,
			}
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					require.NoError(t, campaign.Compile())
					assert.True(t, campaign.IsFulfilled(rp))
				})
			}
		})

		t.Run("Test not fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent(t, nil, "some user-agent"),
				Method:  "DELETE",
				Path:    "/test/path",
				Code:    305,
			}
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					assert.False(t, campaign.IsFulfilled(rp))
				})
			}
		})
	})

	t.Run("All criteria", func(t *testing.T) {
		campaign := APICallCampaign{
			{
				Codes:   []int32{200, 400},
				Method:  Pattern("{GET,POST}").Ptr(),
				Path:    Pattern("{/v1/test*,/v2/test*}").Ptr(),
				Headers: map[string]Pattern{"User-Agent": "*test*"},
			},
			{
				Codes:   []int32{200, 400},
				Method:  Pattern("{GET,POST}").Ptr(),
				Path:    Pattern("{/v1/test*,/v2/test*}").Ptr(),
				Headers: map[string]Pattern{"User-Agent": "*toast*"},
			},
			{
				Codes:   []int32{300, 500},
				Method:  Pattern("{DELETE,OPTIONS}").Ptr(),
				Path:    Pattern("{/v3/test*,/v4/test*}").Ptr(),
				Headers: map[string]Pattern{"User-Agent": "{*tooth*,*teeth*}"},
			},
			{
				Codes:  []int32{100},
				Method: Pattern("PUT").Ptr(),
				Path:   Pattern("/v5/*").Ptr(),
				Headers: map[string]Pattern{
					"User-Agent": "*another*",
					"header":     "val*",
				},
			},
		}
		require.NoError(t, campaign.Compile())
		t.Run("All pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					Headers: withUserAgent(t, nil, "some test user-agent 1"),
					Method:  "GET",
					Path:    "/v1/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent(t, nil, "some toast user-agent 2"),
					Method:  "POST",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 3"),
					Method:  "DELETE",
					Path:    "/v3/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 4"),
					Method:  "OPTIONS",
					Path:    "/v4/test/path",
					Code:    500,
				},
				{
					Method: "PUT",
					Code:   100,
					Path:   "/v5/test",
					Headers: func(h string) []string {
						return map[string][]string{
							userAgentHeaderKey: {"some another user-agent"},
							"header":           {"value"},
						}[h]
					},
				},
			}
			for _, rp := range rps {
				assert.True(t, campaign.IsFulfilled(&rp), rp.Headers(userAgentHeaderKey))
			}
		})

		t.Run("All not pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					Headers: withUserAgent(t, nil, "some test user-agent 1"),
					Method:  "GET",
					Path:    "/v1/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent(t, nil, "some toast user-agent 2"),
					Method:  "DELETE",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 3"),
					Method:  "DELETE",
					Path:    "/v3/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent(t, nil, "some tooth user-agent 4"),
					Method:  "GET",
					Path:    "/v4/test/path",
					Code:    500,
				},
				{
					Method: "PUT",
					Path:   "/v5/test/path",
					Code:   100,
					Headers: withUserAgent(t,
						map[string][]string{"h": {"---"}},
						"some another user-agent 5"),
				},
			}
			for _, rp := range rps {
				assert.False(t, campaign.IsFulfilled(&rp), rp.Headers(userAgentHeaderKey))
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
			criterion: APICallCampaignCriterion{
				Path: Pattern("[b-a]").Ptr(),
			},
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
}
