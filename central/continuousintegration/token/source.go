package token

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ tokens.Source = (*continuousIntegrationSource)(nil)
	// The requirement to satisfy the authproviders.Provider interface lies within the tokenbased extractor.
	// It only accepts tokens which have an authproviders.Provider as source, and rejects all others.
	// While this may be something we want to change in the future (API tokens are currently also not really coming from
	// an authentication provider), this is the way it's handled for now.
	// See: https://github.com/stackrox/stackrox/blob/master/pkg/grpc/authn/tokenbased/extractor.go#L58-L61
	_ authproviders.Provider = (*continuousIntegrationSource)(nil)

	onceSources sync.Once
	sources     map[storage.ContinuousIntegrationType]tokens.Source
)

type continuousIntegrationSource struct {
	integrationType storage.ContinuousIntegrationType
}

// SingletonSourceForContinuousIntegration returns a singleton tokens.Source for the specific
// storage.ContinuousIntegrationType.
func SingletonSourceForContinuousIntegration(integrationType storage.ContinuousIntegrationType) tokens.Source {
	onceSources.Do(func() {
		gitHubSource := &continuousIntegrationSource{integrationType: storage.ContinuousIntegrationType_GITHUB_ACTIONS}
		sources = map[storage.ContinuousIntegrationType]tokens.Source{
			storage.ContinuousIntegrationType_GITHUB_ACTIONS: gitHubSource,
		}
	})
	return sources[integrationType]
}

func (c *continuousIntegrationSource) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

func (c *continuousIntegrationSource) ID() string {
	return c.integrationType.String()
}

func (c *continuousIntegrationSource) Name() string {
	return c.integrationType.String()
}

func (c *continuousIntegrationSource) Type() string {
	return c.integrationType.String()
}

func (c *continuousIntegrationSource) Enabled() bool {
	// Auth provider needs to be enabled, otherwise the identity verification within the tokenbased extractor fails.
	// See: https://github.com/stackrox/stackrox/blob/master/pkg/grpc/authn/tokenbased/extractor.go#L62
	return true
}

func (c *continuousIntegrationSource) MergeConfigInto(newCfg map[string]string) map[string]string {
	return newCfg
}

func (c *continuousIntegrationSource) StorageView() *storage.AuthProvider {
	return nil
}

func (c *continuousIntegrationSource) BackendFactory() authproviders.BackendFactory {
	return nil
}

func (c *continuousIntegrationSource) Backend() authproviders.Backend {
	return nil
}

func (c *continuousIntegrationSource) GetOrCreateBackend(_ context.Context) (authproviders.Backend, error) {
	return nil, nil
}

func (c *continuousIntegrationSource) RoleMapper() permissions.RoleMapper {
	return newRoleMapper(c.integrationType)
}

func (c *continuousIntegrationSource) Issuer() tokens.Issuer {
	return nil
}

func (c *continuousIntegrationSource) AttributeVerifier() user.AttributeVerifier {
	return nil
}

func (c *continuousIntegrationSource) ApplyOptions(_ ...authproviders.ProviderOption) error {
	return nil
}

func (c *continuousIntegrationSource) Active() bool {
	return true
}

func (c *continuousIntegrationSource) MarkAsActive() error {
	return nil
}
