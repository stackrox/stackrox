package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCampaignFulfilled(t *testing.T) {
	t.Run("Empty campaign", func(t *testing.T) {
		campaign := APICallCampaign{}
		rp := &RequestParams{
			UserAgent: "some test user-agent",
			Method:    "GeT",
			Path:      "/some/test/path",
			Code:      202,
		}
		assert.False(t, campaign.IsFulfilled(rp))
	})
	t.Run("Empty criterion", func(t *testing.T) {
		campaign := APICallCampaign{
			APICallCampaignCriterion{},
		}
		rp := &RequestParams{
			UserAgent: "some test user-agent",
			Method:    "GeT",
			Path:      "/some/test/path",
			Code:      202,
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
					PathPatterns: []string{"/some/test*"},
				},
			},
			"PathPatterns": []APICallCampaignCriterion{
				{
					PathPatterns: []string{"/x", "/some/test*", "/y"},
				},
			},
			"UserAgent": []APICallCampaignCriterion{
				{
					UserAgents: []string{"test"},
				},
			},
			"UserAgents": []APICallCampaignCriterion{
				{
					UserAgents: []string{"x", "test", "y"},
				},
			},
		}

		t.Run("Test fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				UserAgent: "some test user-agent",
				Method:    "GeT",
				Path:      "/some/test/path",
				Code:      202,
			}
			for name, campaign := range campaigns {
				t.Run(name, func(t *testing.T) {
					assert.True(t, campaign.IsFulfilled(rp))
				})
			}
		})

		t.Run("Test not fulfilled", func(t *testing.T) {
			rp := &RequestParams{
				UserAgent: "some user-agent",
				Method:    "delete",
				Path:      "/test/path",
				Code:      305,
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
				Codes:        []int32{200, 400},
				Methods:      []string{"get", "post"},
				PathPatterns: []string{"/v1/test*", "/v2/test*"},
				UserAgents:   []string{"test", "toast"},
			},
			{
				Codes:        []int32{300, 500},
				Methods:      []string{"delete", "options"},
				PathPatterns: []string{"/v3/test*", "/v4/test*"},
				UserAgents:   []string{"teeth", "tooth"},
			},
			{
				Codes:        []int32{100},
				Methods:      []string{"put"},
				PathPatterns: []string{"/v5/*"},
				UserAgents:   []string{"another"},
				HeaderPatterns: map[string]string{
					"header": "val.*",
				},
			},
		}
		t.Run("All pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					UserAgent: "some test user-agent 1",
					Method:    "get",
					Path:      "/v1/test/path",
					Code:      200,
				},
				{
					UserAgent: "some toast user-agent 2",
					Method:    "post",
					Path:      "/v2/test/path",
					Code:      400,
				},
				{
					UserAgent: "some teeth user-agent 3",
					Method:    "delete",
					Path:      "/v3/test/path",
					Code:      300,
				},
				{
					UserAgent: "some tooth user-agent 4",
					Method:    "options",
					Path:      "/v4/test/path",
					Code:      500,
				},
				{
					UserAgent: "some another user-agent",
					Method:    "PUT",
					Code:      100,
					Path:      "/v5/test",
					Headers: func(h string) []string {
						return map[string][]string{
							"header": {"value"},
						}[h]
					},
				},
			}
			for _, rp := range rps {
				assert.True(t, campaign.IsFulfilled(&rp), rp.UserAgent)
			}
		})

		t.Run("All not pass", func(t *testing.T) {
			rps := []RequestParams{
				{
					UserAgent: "some test user-agent 1",
					Method:    "get",
					Path:      "/v1/test/path",
					Code:      300,
				},
				{
					UserAgent: "some toast user-agent 2",
					Method:    "delete",
					Path:      "/v2/test/path",
					Code:      400,
				},
				{
					UserAgent: "some teeth user-agent 3",
					Method:    "delete",
					Path:      "/v3/test/path",
					Code:      200,
				},
				{
					UserAgent: "some tooth user-agent 4",
					Method:    "get",
					Path:      "/v4/test/path",
					Code:      500,
				},
				{
					UserAgent: "some another user-agent 5",
					Method:    "put",
					Path:      "/v5/test/path",
					Code:      100,
					Headers:   func(_ string) []string { return []string{"---"} },
				},
			}
			for _, rp := range rps {
				assert.False(t, campaign.IsFulfilled(&rp), rp.UserAgent)
			}
		})
	})
}
