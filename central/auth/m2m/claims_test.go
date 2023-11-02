package m2m

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func TestCreateRoxClaimsFromGenericClaims(t *testing.T) {
	testCases := []struct {
		subject      string
		unstructured map[string]interface{}
		roxClaims    tokens.RoxClaims
	}{
		{
			subject: "test-subject-email",
			unstructured: map[string]interface{}{
				"email":     "test@something.com",
				"something": "else",
				"another":   []string{"value", "of", "things"},
			},
			roxClaims: tokens.RoxClaims{
				ExternalUser: &tokens.ExternalUserClaim{
					UserID:   "test-subject-email|test@something.com",
					FullName: "test@something.com",
					Attributes: map[string][]string{
						"email":     {"test@something.com"},
						"something": {"else"},
						"another":   {"value", "of", "things"},
					},
				},
				Name: "test-subject-email|test@something.com",
			},
		},
		{
			subject: "test-subject-preferred_username",
			unstructured: map[string]interface{}{
				"preferred_username": "test-user",
				"something":          "else",
				"another":            []string{"value", "of", "things"},
			},
			roxClaims: tokens.RoxClaims{
				ExternalUser: &tokens.ExternalUserClaim{
					UserID:   "test-subject-preferred_username|test-user",
					FullName: "test-user",
					Attributes: map[string][]string{
						"preferred_username": {"test-user"},
						"something":          {"else"},
						"another":            {"value", "of", "things"},
					},
				},
				Name: "test-subject-preferred_username|test-user",
			},
		},
		{
			subject: "test-subject-full_name",
			unstructured: map[string]interface{}{
				"full_name": "i am the test user",
				"something": "else",
				"another":   []string{"value", "of", "things"},
			},
			roxClaims: tokens.RoxClaims{
				ExternalUser: &tokens.ExternalUserClaim{
					UserID:   "test-subject-full_name|i am the test user",
					FullName: "i am the test user",
					Attributes: map[string][]string{
						"full_name": {"i am the test user"},
						"something": {"else"},
						"another":   {"value", "of", "things"},
					},
				},
				Name: "test-subject-full_name|i am the test user",
			},
		},
		{
			subject: "test-subject-empty",
			unstructured: map[string]interface{}{
				"something": "else",
				"another":   []string{"value", "of", "things"},
			},
			roxClaims: tokens.RoxClaims{
				ExternalUser: &tokens.ExternalUserClaim{
					UserID:   "test-subject-empty",
					FullName: "test-subject-empty",
					Attributes: map[string][]string{
						"something": {"else"},
						"another":   {"value", "of", "things"},
					},
				},
				Name: "test-subject-empty",
			},
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			roxClaims := createRoxClaimsFromGenericClaims(testCase.subject, testCase.unstructured)
			assert.Equal(t, testCase.roxClaims, roxClaims)
		})
	}
}

func TestCreateRoxClaimsFromGitHubClaims(t *testing.T) {
	claims := githubActionClaims{
		Actor:           "my-github-user",
		ActorID:         "123456789",
		Environment:     "production",
		EventName:       "PullRequest",
		GitRef:          "sha256824",
		Repository:      "sample-repo",
		RepositoryOwner: "sample-org",
	}
	subject := "test-subject"
	audiences := []string{"repoA", "repoB"}
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
			},
		},
		Name: "123456789|my-github-user",
	}

	roxClaims := createRoxClaimsFromGitHubClaims(subject, audiences, claims)
	assert.Equal(t, expectedRoxClaims, roxClaims)
}
