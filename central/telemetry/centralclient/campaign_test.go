package centralclient

import (
	"testing"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
)

func TestCampaignFulfilled(t *testing.T) {
	campaigns := map[string]Campaign{
		"Code": []CampaignCriterium{
			{
				Codes: []int{202},
			},
		},
		"Codes": []CampaignCriterium{
			{
				Codes: []int{100, 202, 400},
			},
		},
		"Method": []CampaignCriterium{
			{
				Methods: []string{"get"},
			},
		},
		"Methods": []CampaignCriterium{
			{
				Methods: []string{"post", "get", "put"},
			},
		},
		"PathPattern": []CampaignCriterium{
			{
				PathPatterns: []string{"/some/test*"},
			},
		},
		"PathPatterns": []CampaignCriterium{
			{
				PathPatterns: []string{"/x", "/some/test*", "/y"},
			},
		},
		"UserAgent": []CampaignCriterium{
			{
				UserAgents: []string{"test"},
			},
		},
		"UserAgents": []CampaignCriterium{
			{
				UserAgents: []string{"x", "test", "y"},
			},
		},
	}

	t.Run("Test fulfilled", func(t *testing.T) {
		rp := &phonehome.RequestParams{
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
		rp := &phonehome.RequestParams{
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

	t.Run("All criteria", func(t *testing.T) {
		campaign := Campaign{
			{
				Codes:        []int{200, 400},
				Methods:      []string{"get", "post"},
				PathPatterns: []string{"/v1/test*", "/v2/test*"},
				UserAgents:   []string{"test", "toast"},
			},
			{
				Codes:        []int{300, 500},
				Methods:      []string{"delete", "options"},
				PathPatterns: []string{"/v3/test*", "/v4/test*"},
				UserAgents:   []string{"teeth", "tooth"},
			},
		}
		t.Run("All pass", func(t *testing.T) {
			rps := []phonehome.RequestParams{
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
			}
			for _, rp := range rps {
				assert.True(t, campaign.IsFulfilled(&rp), rp.UserAgent)
			}
		})

		t.Run("All not pass", func(t *testing.T) {
			rps := []phonehome.RequestParams{
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
			}
			for _, rp := range rps {
				assert.False(t, campaign.IsFulfilled(&rp), rp.UserAgent)
			}
		})
	})
}
