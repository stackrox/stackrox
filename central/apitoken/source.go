package apitoken

import (
	"github.com/stackrox/rox/central/apitoken/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	id = `https://stackrox.io/jwt-sources#api-tokens`
)

type source struct {
	store.Store
	tokens.RevocationLayer
}

func newSource(store store.Store) (*source, error) {
	src := &source{
		Store:           store,
		RevocationLayer: tokens.NewRevocationLayer(),
	}
	if err := src.initFromStore(); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *source) initFromStore() error {
	revokedTokenReq := &v1.GetAPITokensRequest{
		RevokedOneof: &v1.GetAPITokensRequest_Revoked{
			Revoked: true,
		},
	}
	existingTokens, err := s.Store.GetTokens(revokedTokenReq)
	if err != nil {
		return err
	}

	for _, token := range existingTokens {
		expiry := protoconv.ConvertTimestampToTimeOrDefault(token.GetExpiration(), timeutil.Max)
		s.RevocationLayer.Revoke(token.GetId(), expiry)
	}

	return nil
}

func (s *source) RevokeToken(tokenID string) (bool, error) {
	t, err := s.Store.GetTokenOrNil(tokenID)
	if err != nil {
		return false, err
	}
	if t == nil {
		return false, nil
	}
	if t.Revoked {
		return true, nil
	}
	expiry := protoconv.ConvertTimestampToTimeOrDefault(t.GetExpiration(), timeutil.Max)
	s.RevocationLayer.Revoke(tokenID, expiry)
	return s.Store.RevokeToken(tokenID)
}

func (s *source) ID() string {
	return id
}
