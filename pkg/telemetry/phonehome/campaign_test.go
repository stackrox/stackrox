package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
			APICallCampaignCriterion{},
		}
		rp := &RequestParams{
			Headers: withUserAgent(t, nil, "some test user-agent"),
			Method:  "GeT",
			Path:    "/some/test/path",
			Code:    202,
		}
		assert.True(t, campaign.IsFulfilled(rp))
	})

	t.Run("Single criterion", func(t *testing.T) {
		campaigns := map[string]APICallCampaign{
			"Code": []APICallCampaignCriterion{
				{
					Codes: []int32{202},
				},
			},
			"Codes": []APICallCampaignCriterion{
				{
					Codes: []int32{100, 202, 400},
				},
			},
			"Method": []APICallCampaignCriterion{
				{
					Methods: []string{"get"},
				},
			},
			"Methods": []APICallCampaignCriterion{
				{
					Methods: []string{"post", "get", "put"},
				},
			},
			"PathPattern": []APICallCampaignCriterion{
				{
					Paths: []string{"/some/test*"},
				},
			},
			"PathPatterns": []APICallCampaignCriterion{
				{
					Paths: []string{"/x", "/some/test*", "/y"},
				},
			},
			"UserAgent": []APICallCampaignCriterion{
				{
					Headers: map[string]string{
						"User-Agent": "*test*",
					},
				},
			},
			"UserAgents": []APICallCampaignCriterion{
				{
					Headers: map[string]string{"User-Agent": "*x*"},
				},
				{
					Headers: map[string]string{"User-Agent": "*test*"},
				}, {
					Headers: map[string]string{"User-Agent": "*y*"},
				},
			},
		}

		t.Run("Test fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent(t, nil, "some test user-agent"),
				Method:  "GeT",
				Path:    "/some/test/path",
				Code:    202,
			}
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					assert.True(t, campaign.IsFulfilled(rp))
				})
			}
		})

		t.Run("Test not fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				Headers: withUserAgent(t, nil, "some user-agent"),
				Method:  "delete",
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
				Methods: []string{"get", "post"},
				Paths:   []string{"/v1/test*", "/v2/test*"},
				Headers: map[string]string{"User-Agent": "*test*"},
			},
			{
				Codes:   []int32{200, 400},
				Methods: []string{"get", "post"},
				Paths:   []string{"/v1/test*", "/v2/test*"},
				Headers: map[string]string{"User-Agent": "*toast*"},
			},
			{
				Codes:   []int32{300, 500},
				Methods: []string{"delete", "options"},
				Paths:   []string{"/v3/test*", "/v4/test*"},
				Headers: map[string]string{"User-Agent": "*teeth*"},
			},
			{
				Codes:   []int32{100},
				Methods: []string{"put"},
				Paths:   []string{"/v5/*"},
				Headers: map[string]string{
					"User-Agent": "*another*",
					"header":     "val*",
				},
			},
		}
		t.Run("All pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					Headers: withUserAgent(t, nil, "some test user-agent 1"),
					Method:  "get",
					Path:    "/v1/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent(t, nil, "some toast user-agent 2"),
					Method:  "post",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 3"),
					Method:  "delete",
					Path:    "/v3/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 4"),
					Method:  "options",
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
					Method:  "get",
					Path:    "/v1/test/path",
					Code:    300,
				},
				{
					Headers: withUserAgent(t, nil, "some toast user-agent 2"),
					Method:  "delete",
					Path:    "/v2/test/path",
					Code:    400,
				},
				{
					Headers: withUserAgent(t, nil, "some teeth user-agent 3"),
					Method:  "delete",
					Path:    "/v3/test/path",
					Code:    200,
				},
				{
					Headers: withUserAgent(t, nil, "some tooth user-agent 4"),
					Method:  "get",
					Path:    "/v4/test/path",
					Code:    500,
				},
				{
					Method: "put",
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
