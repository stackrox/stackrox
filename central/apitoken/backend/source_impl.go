package backend

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	id       = `https://stackrox.io/jwt-sources#api-tokens`
	apiToken = "api-token"
)

var _ authproviders.Provider = (*sourceImpl)(nil)

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

func (s *sourceImpl) Validate(ctx context.Context, claims *tokens.Claims) error {
	return s.revocationLayer.Validate(ctx, claims)
}

func (s *sourceImpl) Revoke(tokenID string, expiry time.Time) {
	s.revocationLayer.Revoke(tokenID, expiry)
}

func (s *sourceImpl) ID() string {
	return id
}

func (s *sourceImpl) Name() string {
	return apiToken
}

func (s *sourceImpl) Type() string {
	return apiToken
}

func (s *sourceImpl) Enabled() bool {
	return true
}

func (s *sourceImpl) StorageView() *storage.AuthProvider {
	// API token sources have no storage view.
	return nil
}

func (s *sourceImpl) BackendFactory() authproviders.BackendFactory {
	// API token sources have no Backend factory
	return nil
}

func (s *sourceImpl) MergeConfigInto(newCfg map[string]string) map[string]string {
	return newCfg
}

func (s *sourceImpl) Backend() authproviders.Backend {
	// API token sources have no Backend
	return nil
}

func (s *sourceImpl) GetOrCreateBackend(_ context.Context) (authproviders.Backend, error) {
	return nil, nil
}

func (s *sourceImpl) RoleMapper() permissions.RoleMapper {
	// API token sources have no RoleMapper
	return nil
}

func (s *sourceImpl) Issuer() tokens.Issuer {
	// API token sources have an Issuer but it isn't accessed from here
	return nil
}

func (s *sourceImpl) ApplyOptions(_ ...authproviders.ProviderOption) error {
	// API token sources are not modified through Options methods as they aren't in the registry
	return nil
}

func (s *sourceImpl) Active() bool {
	return true
}

func (s *sourceImpl) MarkAsActive() error {
	return nil
}

func (s *sourceImpl) AttributeVerifier() user.AttributeVerifier {
	return nil
}
