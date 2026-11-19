package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// fakeExtractor is an IdentityExtractor that returns predetermined values and
// records how often it was consulted.
type fakeExtractor struct {
	identity Identity
	err      *ExtractorError
	calls    int
}

func (f *fakeExtractor) IdentityForRequest(_ context.Context, _ requestinfo.RequestInfo) (Identity, *ExtractorError) {
	f.calls++
	return f.identity, f.err
}

var _ IdentityExtractor = (*fakeExtractor)(nil)

// newMockIdentity returns a mock identity exposing the given roles and service.
func newMockIdentity(it *testing.T, roles []permissions.ResolvedRole, service *storage.ServiceIdentity) Identity {
	ctrl := gomock.NewController(it)
	id := mocks.NewMockIdentity(ctrl)
	id.EXPECT().Roles().AnyTimes().Return(roles)
	id.EXPECT().Service().AnyTimes().Return(service)
	return id
}

func TestContextUpdater(t *testing.T) {
	for name, tc := range map[string]struct {
		setup       func(it *testing.T) (Identity, *ExtractorError)
		expectID    bool
		expectedErr error
	}{
		"No identity and no error yields no identity in the context": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				return nil, nil
			},
			expectID:    false,
			expectedErr: errox.NoCredentials,
		},
		"Identity with roles is added to the context": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				role := permissionMocks.NewMockResolvedRole(gomock.NewController(it))
				return newMockIdentity(it, []permissions.ResolvedRole{role}, nil), nil
			},
			expectID: true,
		},
		"Identity with no roles and no service is rejected with ErrNoValidRole": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				return newMockIdentity(it, nil, nil), nil
			},
			expectID:    false,
			expectedErr: auth.ErrNoValidRole,
		},
		"Service identity with no roles is added to the context": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				return newMockIdentity(it, nil, &storage.ServiceIdentity{}), nil
			},
			expectID: true,
		},
		"Service identity with roles is added to the context": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				role := permissionMocks.NewMockResolvedRole(gomock.NewController(it))
				return newMockIdentity(it, []permissions.ResolvedRole{role}, &storage.ServiceIdentity{}), nil
			},
			expectID: true,
		},
		"Extractor error is translated to NoCredentials in the context": {
			setup: func(it *testing.T) (Identity, *ExtractorError) {
				return nil, NewExtractorError("test", "test msg", errors.New("root cause"))
			},
			expectID:    false,
			expectedErr: errox.NoCredentials,
		},
	} {
		t.Run(name, func(it *testing.T) {
			id, extractorErr := tc.setup(it)
			updater := ContextUpdater(&fakeExtractor{identity: id, err: extractorErr})

			type contextKey string
			origCtx := context.WithValue(context.Background(), contextKey("key"), "value")

			newCtx, err := updater(origCtx)
			// Extraction errors are communicated via the context, not the return value.
			assert.NoError(it, err)
			// The updater must not discard existing context values.
			assert.Equal(it, "value", newCtx.Value(contextKey("key")))

			idFromCtx, errFromCtx := IdentityFromContext(newCtx)
			if tc.expectID {
				assert.Same(it, id, idFromCtx)
				assert.NoError(it, errFromCtx)
				assert.Same(it, id, IdentityFromContextOrNil(newCtx))
			} else {
				assert.Nil(it, idFromCtx)
				assert.Nil(it, IdentityFromContextOrNil(newCtx))
			}
			if tc.expectedErr != nil {
				assert.ErrorIs(it, errFromCtx, tc.expectedErr)
			} else {
				assert.NoError(it, errFromCtx)
			}
		})
	}
}

func TestContextUpdaterPreservesExtractorErrorAsCause(t *testing.T) {
	cause := NewExtractorError("test", "test msg", errors.New("root cause"))
	updater := ContextUpdater(&fakeExtractor{err: cause})

	newCtx, err := updater(context.Background())
	assert.NoError(t, err)

	id, ctxErr := IdentityFromContext(newCtx)
	assert.Nil(t, id)
	assert.ErrorIs(t, ctxErr, errox.NoCredentials)
	assert.ErrorIs(t, ctxErr, cause)
}

func TestContextUpdaterWithMultipleExtractors(t *testing.T) {
	role := permissionMocks.NewMockResolvedRole(gomock.NewController(t))

	t.Run("Extractors yielding no identity are skipped until one yields an identity", func(it *testing.T) {
		none := &fakeExtractor{}
		hit := &fakeExtractor{identity: newMockIdentity(it, []permissions.ResolvedRole{role}, nil)}

		newCtx, err := ContextUpdater(none, hit)(context.Background())
		assert.NoError(it, err)

		assert.Equal(it, 1, none.calls)
		assert.Equal(it, 1, hit.calls)
		id, ctxErr := IdentityFromContext(newCtx)
		assert.Same(it, hit.identity, id)
		assert.NoError(it, ctxErr)
	})

	t.Run("The first extractor yielding an identity wins", func(it *testing.T) {
		first := &fakeExtractor{identity: newMockIdentity(it, []permissions.ResolvedRole{role}, nil)}
		second := &fakeExtractor{identity: newMockIdentity(it, []permissions.ResolvedRole{role}, nil)}

		newCtx, err := ContextUpdater(first, second)(context.Background())
		assert.NoError(it, err)

		assert.Equal(it, 1, first.calls)
		assert.Zero(it, second.calls)
		id, ctxErr := IdentityFromContext(newCtx)
		assert.Same(it, first.identity, id)
		assert.NoError(it, ctxErr)
	})

	t.Run("An extractor error takes precedence over later extractors", func(it *testing.T) {
		failing := &fakeExtractor{err: NewExtractorError("test", "test msg", errors.New("root cause"))}
		succeeding := &fakeExtractor{identity: newMockIdentity(it, []permissions.ResolvedRole{role}, nil)}

		newCtx, err := ContextUpdater(failing, succeeding)(context.Background())
		assert.NoError(it, err)

		assert.Equal(it, 1, failing.calls)
		assert.Zero(it, succeeding.calls)
		id, ctxErr := IdentityFromContext(newCtx)
		assert.Nil(it, id)
		assert.ErrorIs(it, ctxErr, errox.NoCredentials)
	})

	t.Run("No extractors yields no identity in the context", func(it *testing.T) {
		newCtx, err := ContextUpdater()(context.Background())
		assert.NoError(it, err)

		id, ctxErr := IdentityFromContext(newCtx)
		assert.Nil(it, id)
		assert.ErrorIs(it, ctxErr, errox.NoCredentials)
	})

	t.Run("User identity without roles is rejected even if a service identity is available later", func(it *testing.T) {
		// Combined extractors stop at the first non-nil result, so a role-less
		// user identity is not rescued by a subsequent service identity.
		roleLess := &fakeExtractor{identity: newMockIdentity(it, nil, nil)}
		service := &fakeExtractor{identity: newMockIdentity(it, nil, &storage.ServiceIdentity{})}

		newCtx, err := ContextUpdater(roleLess, service)(context.Background())
		assert.NoError(it, err)

		assert.Equal(it, 1, roleLess.calls)
		assert.Zero(it, service.calls)
		id, ctxErr := IdentityFromContext(newCtx)
		assert.Nil(it, id)
		assert.ErrorIs(it, ctxErr, auth.ErrNoValidRole)
	})
}
