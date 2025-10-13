package m2m

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
)

var (
	_ authproviders.Provider = (*provider)(nil)
)

// newProviderFromConfig returns a provider that can be used to extract tokens issued based on machine to machine
// configs.
func newProviderFromConfig(config *storage.AuthMachineToMachineConfig, rm permissions.RoleMapper) authproviders.Provider {
	return &provider{config: config, rm: rm}
}

// provider implements the authproviders.Provider interface and is used as the token source for machine to machine
// tokens.
type provider struct {
	config *storage.AuthMachineToMachineConfig
	rm     permissions.RoleMapper
}

func (s *provider) ID() string {
	return s.config.GetId()
}

func (s *provider) Name() string {
	return fmt.Sprintf("%s-%s", s.config.GetType().String(), s.config.GetId())
}

func (s *provider) Type() string {
	return s.config.GetType().String()
}

func (s *provider) Enabled() bool {
	// A provider needs to be enabled, otherwise it will fail during the identity extraction.
	// See pkg/grpc/authn/tokenbased/extractor.go for more details.
	return true
}

func (s *provider) RoleMapper() permissions.RoleMapper {
	return s.rm
}

// Unimplemented methods to satisfy authrproviders.provider.

func (s *provider) Validate(_ context.Context, _ *tokens.Claims) error         { return nil }
func (s *provider) MergeConfigInto(newCfg map[string]string) map[string]string { return newCfg }
func (s *provider) StorageView() *storage.AuthProvider                         { return nil }
func (s *provider) BackendFactory() authproviders.BackendFactory               { return nil }
func (s *provider) Backend() authproviders.Backend                             { return nil }
func (s *provider) GetOrCreateBackend(_ context.Context) (authproviders.Backend, error) {
	return nil, nil
}
func (s *provider) Issuer() tokens.Issuer                                { return nil }
func (s *provider) AttributeVerifier() user.AttributeVerifier            { return nil }
func (s *provider) ApplyOptions(_ ...authproviders.ProviderOption) error { return nil }
func (s *provider) Active() bool                                         { return true }
func (s *provider) MarkAsActive() error                                  { return nil }
