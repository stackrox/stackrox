package authproviders

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testUserPKIType = "test-userpki"

// TestBackendLifecycle runs the backend lifecycle test suite
func TestBackendLifecycle(t *testing.T) {
	suite.Run(t, new(BackendLifecycleTestSuite))
}

// BackendLifecycleTestSuite tests that backend lifecycle hooks (OnEnable/OnDisable)
// fire correctly when backends are created, swapped, or destroyed.
type BackendLifecycleTestSuite struct {
	suite.Suite

	registry  *registryImpl
	callbacks *recordingCallbacks
	ctx       context.Context
}

func (s *BackendLifecycleTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.callbacks = newRecordingCallbacks()

	s.registry = NewStoreBackedRegistry(
		"lifecycletest",
		"testredirect",
		&testStoreWithProviders{}, // empty: avoids log noise from unknown provider types
		&tstTokenIssuerFactory{},
		&tstRoleMapperFactory{},
		nil, // ClusterResolver not needed for these tests
	).(*registryImpl)

	// Register a test userpki-like factory
	factoryFunc := newTestUserPKIFactoryFunc(s.callbacks)
	err := s.registry.RegisterBackendFactory(s.ctx, testUserPKIType, factoryFunc)
	s.Require().NoError(err, "test userpki factory registration should succeed")

	err = s.registry.Init()
	s.Require().NoError(err, "registry initialization should succeed")
}

func (s *BackendLifecycleTestSuite) TearDownTest() {
	s.callbacks.reset()
}

// TestUpdateProviderConfigChange verifies that UpdateProvider with a config change
// fires OnDisable for the old backend, builds a fresh backend, then fires OnEnable
// with the updated config rather than re-registering the stale backend.
func (s *BackendLifecycleTestSuite) TestUpdateProviderConfigChange() {
	// Generate two different CA certificates
	ca1, ca1Pem := generateTestCA(s.T())
	ca2, ca2Pem := generateTestCA(s.T())
	ca1Fingerprint := cryptoutils.CertFingerprint(ca1)
	ca2Fingerprint := cryptoutils.CertFingerprint(ca2)

	// Create provider with CA1
	provider, err := s.registry.CreateProvider(s.ctx,
		WithStorageView(&storage.AuthProvider{
			Id:      "test-provider-1",
			Name:    "Test Provider 1",
			Type:    testUserPKIType,
			Enabled: true,
			Config: map[string]string{
				"keys": ca1Pem,
			},
		}),
	)
	s.Require().NoError(err, "CreateProvider should succeed")
	s.Require().NotNil(provider, "provider should not be nil")

	// Verify CA1 is registered
	s.callbacks.mu.Lock()
	registeredCerts, found := s.callbacks.registered[provider.ID()]
	s.callbacks.mu.Unlock()
	s.Require().True(found, "provider should be registered")
	s.Require().Len(registeredCerts, 1, "should have one CA registered")
	s.Equal(ca1Fingerprint, cryptoutils.CertFingerprint(registeredCerts[0]),
		"registered CA should be CA1")

	// Clear the recording to track only the update
	s.callbacks.reset()

	// Update provider config to use CA2
	updated, err := s.registry.UpdateProvider(s.ctx, provider.ID(),
		WithConfig(map[string]string{
			"keys": ca2Pem,
		}),
	)
	s.Require().NoError(err, "UpdateProvider should succeed")
	s.Require().NotNil(updated, "updated provider should not be nil")

	s.callbacks.mu.Lock()
	defer s.callbacks.mu.Unlock()

	s.Assert().Contains(s.callbacks.unregistered, provider.ID(),
		"UpdateProvider should unregister old backend")

	newRegisteredCerts, registered := s.callbacks.registered[provider.ID()]
	s.Assert().True(registered, "provider should be re-registered")
	if s.Assert().Len(newRegisteredCerts, 1, "should have one CA registered") {
		s.Assert().Equal(ca2Fingerprint, cryptoutils.CertFingerprint(newRegisteredCerts[0]),
			"registered CA should be CA2")
	}
}

// TestLateFactoryRegistration verifies that registering a backend factory after Init()
// fires OnEnable for existing enabled providers once the backend is created.
func (s *BackendLifecycleTestSuite) TestLateFactoryRegistration() {
	// Generate a test CA for the pre-populated provider
	ca, caPem := generateTestCA(s.T())
	caFingerprint := cryptoutils.CertFingerprint(ca)

	// Create a provider that will be pre-populated in the store
	// This simulates a provider that exists in the database from a previous Central run
	prePopulatedProvider := &storage.AuthProvider{
		Id:      "late-factory-provider",
		Name:    "Late Factory Provider",
		Type:    testUserPKIType,
		Enabled: true,
		Config: map[string]string{
			"keys": caPem,
		},
		Active: true,
	}

	// Create a store with the pre-populated provider
	// This simulates loading providers from the database at startup
	storeWithProvider := &testStoreWithProviders{
		providers: []*storage.AuthProvider{prePopulatedProvider},
	}

	// Create fresh registry and callbacks
	freshCallbacks := newRecordingCallbacks()
	freshRegistry := NewStoreBackedRegistry(
		"latefactorytest",
		"testredirect",
		storeWithProvider,
		&tstTokenIssuerFactory{},
		&tstRoleMapperFactory{},
		nil, // ClusterResolver not needed
	).(*registryImpl)

	// Initialize registry WITHOUT registering the factory first
	// This will load the provider from the store but can't create a backend (no factory)
	err := freshRegistry.Init()
	s.Require().NoError(err, "Init should succeed even without factory")

	// Get the provider that was loaded from the store
	provider := freshRegistry.GetProvider(prePopulatedProvider.GetId())
	s.Require().NotNil(provider, "provider should exist after Init")

	// Verify no CA is registered yet (no backend created during Init)
	freshCallbacks.mu.Lock()
	_, registered := freshCallbacks.registered[provider.ID()]
	freshCallbacks.mu.Unlock()
	s.Require().False(registered, "no CA should be registered yet (no backend)")

	freshCallbacks.reset()

	// Now register the factory late (simulates late factory registration)
	// In production, this happens when a factory module loads after Init()
	factoryFunc := newTestUserPKIFactoryFunc(freshCallbacks)
	err = freshRegistry.RegisterBackendFactory(s.ctx, testUserPKIType, factoryFunc)
	s.Require().NoError(err, "late factory registration should succeed")

	// The late registration triggers a goroutine (registry_impl.go:171) that creates backends
	// for existing providers using WithBackendFromFactory. We need to wait for it to complete.
	// Poll until the backend is created or timeout.
	var backend Backend
	s.Require().Eventually(func() bool {
		backend = provider.Backend()
		return backend != nil
	}, 5*time.Second, 10*time.Millisecond, "backend should be created by late-registration goroutine")

	freshCallbacks.mu.Lock()
	defer freshCallbacks.mu.Unlock()

	s.Assert().Contains(freshCallbacks.registered, provider.ID(),
		"CA should be registered after backend creation via late factory registration")

	if registeredCerts, found := freshCallbacks.registered[provider.ID()]; found {
		if s.Assert().Len(registeredCerts, 1, "should have one CA registered") {
			s.Assert().Equal(caFingerprint, cryptoutils.CertFingerprint(registeredCerts[0]),
				"registered CA should match the configured CA")
		}
	}
}

// TestHealthyPath_CreateAndDisableProvider verifies that the healthy path works correctly:
// creating a provider fires OnEnable, disabling fires OnDisable.
func (s *BackendLifecycleTestSuite) TestHealthyPath_CreateAndDisableProvider() {
	ca, caPem := generateTestCA(s.T())
	caFingerprint := cryptoutils.CertFingerprint(ca)

	// Create enabled provider
	provider, err := s.registry.CreateProvider(s.ctx,
		WithStorageView(&storage.AuthProvider{
			Id:      "healthy-provider",
			Name:    "Healthy Provider",
			Type:    testUserPKIType,
			Enabled: true,
			Config: map[string]string{
				"keys": caPem,
			},
		}),
	)
	s.Require().NoError(err)
	s.Require().NotNil(provider)

	// Verify OnEnable was called
	s.callbacks.mu.Lock()
	registeredCerts, found := s.callbacks.registered[provider.ID()]
	s.callbacks.mu.Unlock()
	s.Require().True(found, "provider should be registered")
	s.Require().Len(registeredCerts, 1)
	s.Equal(caFingerprint, cryptoutils.CertFingerprint(registeredCerts[0]))

	s.callbacks.reset()

	// Disable the provider
	updated, err := s.registry.UpdateProvider(s.ctx, provider.ID(), WithEnabled(false))
	s.Require().NoError(err)
	s.Require().False(updated.Enabled())

	// Verify OnDisable was called
	s.callbacks.mu.Lock()
	defer s.callbacks.mu.Unlock()
	s.Assert().Contains(s.callbacks.unregistered, provider.ID(),
		"disabling provider should call OnDisable")
}

// recordingCallbacks is a spy implementation of ProviderCallbacks that records
// all RegisterAuthProvider and UnregisterAuthProvider calls for test assertions.
type recordingCallbacks struct {
	mu           sync.Mutex
	registered   map[string][]*x509.Certificate // providerID -> certs
	unregistered []string                       // providerIDs
}

func newRecordingCallbacks() *recordingCallbacks {
	return &recordingCallbacks{
		registered:   make(map[string][]*x509.Certificate),
		unregistered: make([]string, 0),
	}
}

func (r *recordingCallbacks) RegisterAuthProvider(provider Provider, certs []*x509.Certificate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Store a copy of the certs to avoid aliasing issues
	certsCopy := make([]*x509.Certificate, len(certs))
	copy(certsCopy, certs)
	r.registered[provider.ID()] = certsCopy
}

func (r *recordingCallbacks) UnregisterAuthProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unregistered = append(r.unregistered, provider.ID())
	// Also remove from registered map to simulate real behavior
	delete(r.registered, provider.ID())
}

func (r *recordingCallbacks) GetProviderForFingerprint(_ string) Provider {
	// Not needed for these tests
	return nil
}

func (r *recordingCallbacks) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registered = make(map[string][]*x509.Certificate)
	r.unregistered = make([]string, 0)
}

// testUserPKIBackend mimics the userpki backend for testing lifecycle hooks
type testUserPKIBackend struct {
	callbacks ProviderCallbacks
	certs     []*x509.Certificate
	config    map[string]string
}

// ProviderCallbacks interface that the backend needs
type ProviderCallbacks interface {
	RegisterAuthProvider(provider Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider Provider)
	GetProviderForFingerprint(fingerprint string) Provider
}

func (b *testUserPKIBackend) Config() map[string]string {
	return b.config
}

func (b *testUserPKIBackend) OnEnable(provider Provider) {
	b.callbacks.RegisterAuthProvider(provider, b.certs)
}

func (b *testUserPKIBackend) OnDisable(provider Provider) {
	b.callbacks.UnregisterAuthProvider(provider)
}

func (b *testUserPKIBackend) LoginURL(_ string, _ *requestinfo.RequestInfo) (string, error) {
	return "", nil
}

func (b *testUserPKIBackend) RefreshURL() string {
	return ""
}

func (b *testUserPKIBackend) ProcessHTTPRequest(_ http.ResponseWriter, _ *http.Request) (*AuthResponse, error) {
	return nil, nil
}

func (b *testUserPKIBackend) ExchangeToken(_ context.Context, _, _ string) (*AuthResponse, string, error) {
	return nil, "", nil
}

func (b *testUserPKIBackend) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

// testUserPKIBackendFactory creates test userpki backends
type testUserPKIBackendFactory struct {
	callbacks ProviderCallbacks
}

func (f *testUserPKIBackendFactory) CreateBackend(_ context.Context, _ string, _ []string, config map[string]string, _ map[string]string) (Backend, error) {
	pemData := config["keys"]
	if pemData == "" {
		return nil, fmt.Errorf("keys parameter is required")
	}

	certs, err := helpers.ParseCertificatesPEM([]byte(pemData))
	if err != nil {
		return nil, err
	}

	return &testUserPKIBackend{
		callbacks: f.callbacks,
		certs:     certs,
		config:    config,
	}, nil
}

func (f *testUserPKIBackendFactory) ProcessHTTPRequest(_ http.ResponseWriter, _ *http.Request) (string, string, error) {
	return "", "", nil
}

func (f *testUserPKIBackendFactory) ResolveProviderAndClientState(_ string) (string, string, error) {
	return "", "", nil
}

func (f *testUserPKIBackendFactory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *testUserPKIBackendFactory) MergeConfig(newCfg, _ map[string]string) map[string]string {
	return newCfg
}

func (f *testUserPKIBackendFactory) GetSuggestedAttributes() []string {
	return []string{}
}

func newTestUserPKIFactoryFunc(callbacks ProviderCallbacks) func(string) BackendFactory {
	return func(_ string) BackendFactory {
		return &testUserPKIBackendFactory{callbacks: callbacks}
	}
}

// generateTestCA creates a test CA certificate and returns the parsed cert
// and a PEM-encoded string suitable for userpki config.
func generateTestCA(t *testing.T) (*x509.Certificate, string) {
	t.Helper()

	// Generate a CA using certgen
	ca, err := certgen.GenerateCA()
	require.NoError(t, err, "generating test CA should succeed")

	caPEM := ca.CertPEM()
	require.NotEmpty(t, caPEM, "CA PEM should not be empty")

	// Parse the PEM to get the *x509.Certificate
	block, _ := pem.Decode(caPEM)
	require.NotNil(t, block, "PEM decode should succeed")

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err, "parsing certificate should succeed")

	return cert, string(caPEM)
}

type testStoreWithProviders struct {
	providers []*storage.AuthProvider
}

func (s *testStoreWithProviders) GetAuthProvider(_ context.Context, id string) (*storage.AuthProvider, bool, error) {
	for _, p := range s.providers {
		if p.GetId() == id {
			return p, true, nil
		}
	}
	return nil, false, nil
}

func (s *testStoreWithProviders) ForEachAuthProvider(_ context.Context, fn func(obj *storage.AuthProvider) error) error {
	for _, p := range s.providers {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *testStoreWithProviders) GetAuthProvidersFiltered(_ context.Context, filter func(provider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error) {
	var result []*storage.AuthProvider
	for _, p := range s.providers {
		if filter == nil || filter(p) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *testStoreWithProviders) AuthProviderExistsWithName(_ context.Context, name string) (bool, error) {
	for _, p := range s.providers {
		if p.GetName() == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *testStoreWithProviders) AddAuthProvider(_ context.Context, provider *storage.AuthProvider) error {
	s.providers = append(s.providers, provider)
	return nil
}

func (s *testStoreWithProviders) UpdateAuthProvider(_ context.Context, provider *storage.AuthProvider) error {
	for i, p := range s.providers {
		if p.GetId() == provider.GetId() {
			s.providers[i] = provider
			return nil
		}
	}
	return fmt.Errorf("provider %s not found", provider.GetId())
}

func (s *testStoreWithProviders) RemoveAuthProvider(_ context.Context, id string, _ bool) error {
	for i, p := range s.providers {
		if p.GetId() == id {
			s.providers = append(s.providers[:i], s.providers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("provider %s not found", id)
}
