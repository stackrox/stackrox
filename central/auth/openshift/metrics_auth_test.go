package openshift

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	providerMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testCAPEM = "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n"

func setupMocks(t *testing.T) (*providerMocks.MockRegistry, *roleMocks.MockDataStore) {
	ctrl := gomock.NewController(t)
	return providerMocks.NewMockRegistry(ctrl),
		roleMocks.NewMockDataStore(ctrl)
}

func TestSeed_FirstBoot_CreatesProviderAndPermissionSet(t *testing.T) {
	registry, roleDS := setupMocks(t)
	ctx := context.Background()

	roleDS.EXPECT().PermissionSetExists(gomock.Any(), permissionSetID).Return(false, nil)
	roleDS.EXPECT().AddPermissionSet(gomock.Any(), gomock.Any()).Return(nil)
	registry.EXPECT().GetProvider(authProviderID).Return(nil)
	registry.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).Return(nil, nil)

	ensurePermissionSet(ctx, roleDS)
	ensureAuthProvider(ctx, registry, testCAPEM)
}

func TestSeed_SubsequentBoot_CAUnchanged_NoUpdate(t *testing.T) {
	registry, _ := setupMocks(t)
	ctx := context.Background()

	existing := providerMocks.NewMockProvider(gomock.NewController(t))
	existing.EXPECT().StorageView().Return(&storage.AuthProvider{
		Config: map[string]string{userpki.ConfigKeys: testCAPEM},
	})
	registry.EXPECT().GetProvider(authProviderID).Return(existing)

	ensureAuthProvider(ctx, registry, testCAPEM)
}

func TestSeed_ExistingPermissionSetIsPreserved(t *testing.T) {
	registry, roleDS := setupMocks(t)
	ctx := context.Background()

	roleDS.EXPECT().PermissionSetExists(gomock.Any(), permissionSetID).Return(true, nil)
	registry.EXPECT().GetProvider(authProviderID).Return(nil)
	registry.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).Return(nil, nil)

	ensurePermissionSet(ctx, roleDS)
	ensureAuthProvider(ctx, registry, testCAPEM)
}

// TestSeed_CARotation_RegistersNewCA runs against a real auth provider registry because
// the interesting part of a rotation happens past the registry: a mocked registry only
// shows that UpdateProvider was called, not whether the new CA reached the client CA
// manager. Persisting the bundle without also rebuilding the provider backend leaves
// Central advertising the superseded CA until restart, breaking certificate authentication
// when OpenShift rotates its CSR signer.
func TestSeed_CARotation_RegistersNewCA(t *testing.T) {
	ctx := context.Background()
	caManager := &recordingCAManager{}
	store := &inMemoryAuthProviderStore{providers: map[string]*storage.AuthProvider{}}

	registry := authproviders.NewStoreBackedRegistry("/sso/", "/auth/response/generic",
		store, noopIssuerFactory{}, noopRoleMapperFactory{}, nil)
	require.NoError(t, registry.RegisterBackendFactory(ctx, userpki.TypeName,
		userpki.NewFactoryFactory(caManager)))
	require.NoError(t, registry.Init())

	oldCAPEM, oldCA := generateCA(t)
	newCAPEM, newCA := generateCA(t)

	ensureAuthProvider(ctx, registry, oldCAPEM)
	require.Equal(t, []string{oldCA.Subject.String()}, registeredSubjects(caManager.certs))

	ensureAuthProvider(ctx, registry, newCAPEM)

	provider := registry.GetProvider(authProviderID)
	require.NotNil(t, provider)
	assert.Equal(t, storage.Traits_DEFAULT, provider.StorageView().GetTraits().GetOrigin())
	assert.Equal(t, newCAPEM, provider.StorageView().GetConfig()[userpki.ConfigKeys],
		"rotated CA should be persisted in the auth provider config")
	assert.Equal(t, []string{newCA.Subject.String()}, registeredSubjects(caManager.certs),
		"rotated CA should also be registered with the client CA manager")
}

func generateCA(t *testing.T) (string, *x509.Certificate) {
	ca, err := certgen.GenerateCA()
	require.NoError(t, err)
	return string(ca.CertPEM()), ca.Certificate()
}

func registeredSubjects(certs []*x509.Certificate) []string {
	subjects := make([]string, 0, len(certs))
	for _, cert := range certs {
		subjects = append(subjects, cert.Subject.String())
	}
	return subjects
}

// recordingCAManager stands in for central/tlsconfig.Manager, which is what Central wires
// into the userpki backend factory. It records the certificates the backend hands over,
// i.e. what ends up in the client CA pool advertised during the TLS handshake.
type recordingCAManager struct {
	certs []*x509.Certificate
}

func (c *recordingCAManager) RegisterAuthProvider(_ authproviders.Provider, certs []*x509.Certificate) {
	c.certs = certs
}

func (c *recordingCAManager) UnregisterAuthProvider(authproviders.Provider) {
	c.certs = nil
}

func (c *recordingCAManager) GetProviderForFingerprint(string) authproviders.Provider {
	return nil
}

type inMemoryAuthProviderStore struct {
	providers map[string]*storage.AuthProvider
}

func (s *inMemoryAuthProviderStore) GetAuthProvider(_ context.Context, id string) (*storage.AuthProvider, bool, error) {
	provider, found := s.providers[id]
	return provider, found, nil
}

func (s *inMemoryAuthProviderStore) ForEachAuthProvider(_ context.Context, fn func(*storage.AuthProvider) error) error {
	for _, provider := range s.providers {
		if err := fn(provider); err != nil {
			return err
		}
	}
	return nil
}

func (s *inMemoryAuthProviderStore) GetAuthProvidersFiltered(_ context.Context,
	filter func(*storage.AuthProvider) bool) ([]*storage.AuthProvider, error) {
	var filtered []*storage.AuthProvider
	for _, provider := range s.providers {
		if filter(provider) {
			filtered = append(filtered, provider)
		}
	}
	return filtered, nil
}

func (s *inMemoryAuthProviderStore) AuthProviderExistsWithName(_ context.Context, name string) (bool, error) {
	for _, provider := range s.providers {
		if provider.GetName() == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *inMemoryAuthProviderStore) AddAuthProvider(ctx context.Context, provider *storage.AuthProvider) error {
	if !declarativeconfig.CanModifyResource(ctx, provider) {
		return errors.New("cannot create provider with this origin context")
	}
	s.providers[provider.GetId()] = provider.CloneVT()
	return nil
}

func (s *inMemoryAuthProviderStore) UpdateAuthProvider(ctx context.Context, provider *storage.AuthProvider) error {
	if !declarativeconfig.CanModifyResource(ctx, s.providers[provider.GetId()]) ||
		!declarativeconfig.CanModifyResource(ctx, provider) {
		return errors.New("cannot update provider with this origin context")
	}
	s.providers[provider.GetId()] = provider.CloneVT()
	return nil
}

func (s *inMemoryAuthProviderStore) RemoveAuthProvider(_ context.Context, id string, _ bool) error {
	delete(s.providers, id)
	return nil
}

type noopIssuerFactory struct{}

func (noopIssuerFactory) CreateIssuer(tokens.Source, ...tokens.Option) (tokens.Issuer, error) {
	return nil, nil
}

func (noopIssuerFactory) UnregisterSource(tokens.Source) error { return nil }

type noopRoleMapperFactory struct{}

func (noopRoleMapperFactory) GetRoleMapper(string) permissions.RoleMapper { return nil }
