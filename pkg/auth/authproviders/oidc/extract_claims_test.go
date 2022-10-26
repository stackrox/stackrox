package oidc

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

type mockClaimExtractor struct {
	claims map[string]interface{}
}

func (e *mockClaimExtractor) Claims(input interface{}) error {
	switch u := input.(type) {
	case *map[string]interface{}:
		for k, v := range e.claims {
			(*u)[k] = v
		}
		return nil
	default:
		return errors.Errorf("unsupported type %T", input)
	}
}

func TestExtractCustomClaims(t *testing.T) {
	claim := &tokens.ExternalUserClaim{
		Attributes: make(map[string][]string, 0),
	}
	rolesList := []interface{}{
		"a",
		"b",
		"c",
	}
	claimExtractor := &mockClaimExtractor{
		claims: map[string]interface{}{
			"realm_access": map[string]interface{}{
				"roles": rolesList,
			},
			"a": map[string]interface{}{
				"b": "a-b-value",
			},
			"is_internal":       true,
			"is_email_verified": false,
		},
	}
	mappings := map[string]string{
		"realm_access.roles": "roles",
		"is_internal":        "internal",
		"is_email_verified":  "email_verified",
		// Non-existent path should be ignored.
		"non.existent.path": "path",
		"a.b":               "c",
	}
	err := extractCustomClaims(claim, mappings, claimExtractor)
	assert.NoError(t, err)
	for i, role := range rolesList {
		assert.Equal(t, role.(string), claim.Attributes["roles"][i])
	}
	assert.Equal(t, []string{"true"}, claim.Attributes["internal"])
	assert.Equal(t, []string{"false"}, claim.Attributes["email_verified"])
	assert.Equal(t, []string{"a-b-value"}, claim.Attributes["c"])
}
