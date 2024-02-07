package authproviders

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenProviderUpdate(t *testing.T) {
	before, err := timestamp.TimestampProto(time.Now().Add(-1 * time.Hour))
	require.NoError(t, err)
	after, err := timestamp.TimestampProto(time.Now().Add(1 * time.Hour))
	require.NoError(t, err)
	leeway, err := timestamp.TimestampProto(time.Now().Add(5 * time.Second))
	require.NoError(t, err)

	cases := map[string]struct {
		provider *storage.AuthProvider
		claims   *tokens.Claims
		fails    bool
	}{
		"empty timestamp should lead to no error": {
			claims: &tokens.Claims{
				Claims: jwt.Claims{
					IssuedAt: jwt.NewNumericDate(time.Now()),
				},
			},
			provider: &storage.AuthProvider{},
		},
		"non-empty timestamp lower than issued at should lead to no error": {
			claims: &tokens.Claims{
				Claims: jwt.Claims{
					IssuedAt: jwt.NewNumericDate(time.Now()),
				},
			},
			provider: &storage.AuthProvider{
				LastUpdated: before,
			},
		},
		"non-empty timestamp higher than issued at should lead to error": {
			claims: &tokens.Claims{
				Claims: jwt.Claims{
					IssuedAt: jwt.NewNumericDate(time.Now()),
				},
			},
			provider: &storage.AuthProvider{
				LastUpdated: after,
			},
			fails: true,
		},
		"non-empty timestamp higher than issued at but within leeway should lead to no error": {
			claims: &tokens.Claims{
				Claims: jwt.Claims{
					IssuedAt: jwt.NewNumericDate(time.Now()),
				},
			},
			provider: &storage.AuthProvider{
				LastUpdated: leeway,
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateTokenProviderUpdate(c.provider, c.claims)
			if c.fails {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
