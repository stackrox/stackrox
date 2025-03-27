package authproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func TestValidateName(t *testing.T) {
	ctx := context.Background()
	t.Run("provider with no name", func(it *testing.T) {
		provider := &providerImpl{}
		store := &mockStore{}
		err := validateName(ctx, provider, store)
		assert.ErrorIs(it, err, errNoProviderName)
	})
	t.Run("provider with valid name, store error", func(it *testing.T) {
		provider := &providerImpl{
			storedInfo: &storage.AuthProvider{
				Name: "Test AuthProvider",
			},
		}
		store := &mockStore{
			expectedFound: false,
			expectedErr:   errox.NotAuthorized,
		}
		err := validateName(ctx, provider, store)
		assert.ErrorIs(it, err, errox.NotAuthorized)
	})
	t.Run("provider with name already in use", func(it *testing.T) {
		provider := &providerImpl{
			storedInfo: &storage.AuthProvider{
				Name: "Test AuthProvider",
			},
		}
		store := &mockStore{
			expectedFound: true,
			expectedErr:   nil,
		}
		err := validateName(ctx, provider, store)
		assert.ErrorIs(it, err, errDuplicateProviderName)
	})
	t.Run("provider with name not in use", func(it *testing.T) {
		provider := &providerImpl{
			storedInfo: &storage.AuthProvider{
				Name: "Test AuthProvider",
			},
		}
		store := &mockStore{
			expectedFound: false,
			expectedErr:   nil,
		}
		err := validateName(ctx, provider, store)
		assert.NoError(it, err)
	})
}

// region test helpers

type mockStore struct {
	expectedProvider  *storage.AuthProvider
	expectedProviders []*storage.AuthProvider
	expectedFound     bool
	expectedErr       error
}

func (s *mockStore) GetAuthProvider(_ context.Context, _ string) (*storage.AuthProvider, bool, error) {
	return s.expectedProvider, s.expectedFound, s.expectedErr
}

func (s *mockStore) ProcessAuthProviders(_ context.Context, fn func(obj *storage.AuthProvider) error) error {
	for _, p := range s.expectedProviders {
		err := fn(p)
		if err != nil {
			return err
		}
	}
	return s.expectedErr
}

func (s *mockStore) GetAuthProvidersFiltered(_ context.Context, _ func(authProvider *storage.AuthProvider) bool) ([]*storage.AuthProvider, error) {
	return s.expectedProviders, s.expectedErr
}

func (s *mockStore) AuthProviderExistsWithName(_ context.Context, _ string) (bool, error) {
	return s.expectedFound, s.expectedErr
}

func (s *mockStore) AddAuthProvider(_ context.Context, _ *storage.AuthProvider) error {
	return s.expectedErr
}

func (s *mockStore) UpdateAuthProvider(_ context.Context, _ *storage.AuthProvider) error {
	return s.expectedErr
}

func (s *mockStore) RemoveAuthProvider(_ context.Context, _ string, _ bool) error {
	return s.expectedErr
}

// endregion test helpers
