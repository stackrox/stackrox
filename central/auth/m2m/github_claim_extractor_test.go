package m2m

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func Test_githubClaimExtractor(t *testing.T) {
	githubToken := &IDToken{
		Subject:  "test-subject",
		Audience: []string{"repoA", "repoB"},
		Claims: func(v any) error {
			*v.(*githubActionClaims) = githubActionClaims{
				Actor:           "my-github-user",
				ActorID:         "123456789",
				Environment:     "production",
				EventName:       "PullRequest",
				GitRef:          "sha256824",
				Repository:      "sample-repo",
				RepositoryOwner: "sample-org",
			}
			return nil
		},
	}

	expectedRoxClaims := tokens.RoxClaims{
		ExternalUser: &tokens.ExternalUserClaim{
			UserID:   "123456789|my-github-user",
			FullName: "my-github-user",
			Email:    "",
			Attributes: map[string][]string{
				"actor":            {"my-github-user"},
				"actor_id":         {"123456789"},
				"repository":       {"sample-repo"},
				"repository_owner": {"sample-org"},
				"environment":      {"production"},
				"event_name":       {"PullRequest"},
				"ref":              {"sha256824"},
				"sub":              {"test-subject"},
				"aud":              {"repoA", "repoB"},

				"base_ref":              {""},
				"head_ref":              {""},
				"job_workflow_ref":      {""},
				"job_workflow_sha":      {""},
				"ref_type":              {""},
				"repository_id":         {""},
				"repository_owner_id":   {""},
				"repository_visibility": {""},
				"run_id":                {""},
				"run_number":            {""},
				"run_attempt":           {""},
				"runner_environment":    {""},
				"workflow":              {""},
				"workflow_ref":          {""},
				"workflow_sha":          {""},
			},
		},
		Name: "123456789|my-github-user",
	}

	e := &githubClaimExtractor{}
	claims, err := e.ExtractClaims(githubToken)
	assert.NoError(t, err)
	roxClaims, err := e.ExtractRoxClaims(claims)
	assert.NoError(t, err)
	assert.Equal(t, expectedRoxClaims, roxClaims)
}
