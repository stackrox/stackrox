package tokenbased

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
)

var (
	_ authproviders.Provider = (*noopProvider)(nil)
)

type noopProvider struct{}

func (*noopProvider) ID() string                                                 { return "" }
func (*noopProvider) Name() string                                               { return "" }
func (*noopProvider) Type() string                                               { return "" }
func (*noopProvider) Enabled() bool                                              { return false }
func (*noopProvider) Active() bool                                               { return false }
func (*noopProvider) RoleMapper() permissions.RoleMapper                         { return nil }
func (*noopProvider) MergeConfigInto(newCfg map[string]string) map[string]string { return newCfg }
func (*noopProvider) Validate(_ context.Context, _ *tokens.Claims) error         { return nil }
func (*noopProvider) StorageView() *storage.AuthProvider                         { return nil }
func (*noopProvider) BackendFactory() authproviders.BackendFactory               { return nil }
func (*noopProvider) Backend() authproviders.Backend                             { return nil }
func (*noopProvider) GetOrCreateBackend(_ context.Context) (authproviders.Backend, error) {
	return nil, nil
}
func (*noopProvider) Issuer() tokens.Issuer                                { return nil }
func (*noopProvider) AttributeVerifier() user.AttributeVerifier            { return nil }
func (*noopProvider) ApplyOptions(_ ...authproviders.ProviderOption) error { return nil }
func (*noopProvider) MarkAsActive() error                                  { return nil }
