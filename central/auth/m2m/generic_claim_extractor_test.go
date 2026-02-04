package m2m

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func Test_genericClaimExtractor(t *testing.T) {
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
		{
			subject: "test-k8s",
			unstructured: map[string]interface{}{
				"aud": []string{"https://example.com"},
				"exp": 1763119831,
				"iat": 1763116231,
				"iss": "https://example.com",
				"jti": "6a5e8681-3b2a-44f2-9462-ecf16f52c779",
				"kubernetes.io": map[string]interface{}{
					"namespace": "stackrox",
					"serviceaccount": map[string]interface{}{
						"name": "config-controller",
						"uid":  "3cd68f8a-7e72-44e7-af17-b283e7027980",
					},
				},
				"nbf": 1763116231,
				"sub": "system:serviceaccount:stackrox:config-controller",
			},
			roxClaims: tokens.RoxClaims{
				ExternalUser: &tokens.ExternalUserClaim{
					UserID:   "system:serviceaccount:stackrox:config-controller",
					FullName: "system:serviceaccount:stackrox:config-controller",
					Attributes: map[string][]string{
						"sub": {"system:serviceaccount:stackrox:config-controller"},
						"aud": {"https://example.com"},
						"iss": {"https://example.com"},
						"jti": {"6a5e8681-3b2a-44f2-9462-ecf16f52c779"},

						"kubernetes.io.namespace":           {"stackrox"},
						"kubernetes.io.serviceaccount.name": {"config-controller"},
						"kubernetes.io.serviceaccount.uid":  {"3cd68f8a-7e72-44e7-af17-b283e7027980"},
					},
				},
				Name: "system:serviceaccount:stackrox:config-controller",
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
