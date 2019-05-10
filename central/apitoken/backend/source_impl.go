package backend

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	id = `https://stackrox.io/jwt-sources#api-tokens`
)

type sourceImpl struct {
	revocationLayer tokens.RevocationLayer
}

func (s *sourceImpl) initFromStore(ctx context.Context, apiTokens datastore.DataStore) error {
	revokedTokenReq := &v1.GetAPITokensRequest{
		RevokedOneof: &v1.GetAPITokensRequest_Revoked{
			Revoked: true,
		},
	}
	existingTokens, err := apiTokens.GetTokens(ctx, revokedTokenReq)
	if err != nil {
		return err
	}

	for _, token := range existingTokens {
		expiry := protoconv.ConvertTimestampToTimeOrDefault(token.GetExpiration(), timeutil.Max)
		s.revocationLayer.Revoke(token.GetId(), expiry)
	}

	return nil
}

func (s *sourceImpl) Validate(claims *tokens.Claims) error {
	return s.revocationLayer.Validate(claims)
}

func (s *sourceImpl) Revoke(tokenID string, expiry time.Time) {
	s.revocationLayer.Revoke(tokenID, expiry)
}

func (s *sourceImpl) ID() string {
	return id
}
