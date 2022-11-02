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

func TestMapCustomClaims(t *testing.T) {
	for _, testCase := range []struct {
		desc               string
		claims             map[string]interface{}
		mappings           map[string]string
		expectedAttributes map[string][]string
		wantErr            bool
	}{
		{
			desc: "ignore non-existent, map arrays and string/bool primitives",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{
						"a", "b", "c",
					},
				},
				"a": map[string]interface{}{
					"b": "a-b-value",
				},
				"is_internal":       true,
				"is_email_verified": false,
			},
			mappings: map[string]string{
				"realm_access.roles": "roles",
				"is_internal":        "internal",
				"is_email_verified":  "email_verified",
				// Non-existent path should be ignored.
				"non.existent.path": "path",
				"a.b":               "c",
			},
			expectedAttributes: map[string][]string{
				"internal":       {"true"},
				"email_verified": {"false"},
				"c":              {"a-b-value"},
				"roles":          {"a", "b", "c"},
			},
			wantErr: false,
		},
		{
			desc: "mapping struct claims should result in an error",
			claims: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "c",
				},
			},
			mappings: map[string]string{
				"a": "a",
			},
			expectedAttributes: map[string][]string{},
			wantErr:            true,
		},
		{
			desc: "ignore mapping if can't follow the path",
			claims: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "c",
				},
			},
			mappings: map[string]string{
				"a.b.c": "a",
			},
			expectedAttributes: map[string][]string{},
			wantErr:            false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			claim := &tokens.ExternalUserClaim{
				Attributes: make(map[string][]string, 0),
			}
			claimExtractor := &mockClaimExtractor{
				claims: c.claims,
			}
			if err := mapCustomClaims(claim, c.mappings, claimExtractor); (err == nil) == c.wantErr {
				t.Errorf("mapCustomClaims() error = %v, wantErr %v", err, c.wantErr)
				return
			}
			for k, values := range c.expectedAttributes {
				assert.Equal(t, values, claim.Attributes[k])
			}
		})
	}
}
