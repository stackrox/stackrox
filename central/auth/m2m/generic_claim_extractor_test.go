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
						"sub":       {"test-subject-email"},
						"aud":       nil,
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
						"sub":                {"test-subject-preferred_username"},
						"aud":                nil,
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
						"sub":       {"test-subject-full_name"},
						"aud":       nil,
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
						"sub":       {"test-subject-empty"},
						"aud":       nil,
						"something": {"else"},
						"another":   {"value", "of", "things"},
					},
				},
				Name: "test-subject-empty",
			},
		},
	}

	for i, testCase := range testCases {
		e := &genericClaimExtractor{}
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			token := &IDToken{
				Subject: testCase.subject,
				Claims: func(v any) error {
					*v.(*map[string]any) = testCase.unstructured
					return nil
				},
			}
			claims, err := e.ExtractClaims(token)
			assert.NoError(t, err)
			roxClaims, err := e.ExtractRoxClaims(claims)
			assert.NoError(t, err)
			assert.Equal(t, testCase.roxClaims, roxClaims)
		})
	}
}
