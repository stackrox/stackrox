package interceptor

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type registeringHandlerBase struct {
	called bool
}

func (b *registeringHandlerBase) Handler(_ context.Context, _ interface{}) (interface{}, error) {
	b.called = true
	return nil, nil
}

type serverWithAuthorizer struct {
	authZError error
}

func (s *serverWithAuthorizer) AuthFuncOverride(ctx context.Context, methodName string) (context.Context, error) {
	return ctx, s.authZError
}

type serverWithoutAuthFuncOverride struct{}

func TestAuthStatusString(t *testing.T) {
	for name, tc := range map[string]struct {
		authZError error
		expected   string
	}{
		"no error yields empty string": {
			authZError: nil,
			expected:   "",
		},
		"error string is propagated (raw sentinel error)": {
			authZError: errox.NotAuthorized,
			expected:   "not authorized",
		},
		"error string is propagated (enriched sentinel error)": {
			authZError: errox.NotAuthorized.New("you shall not pass"),
			expected:   "you shall not pass",
		},
	} {
		t.Run(name, func(it *testing.T) {
			status := &AuthStatus{Error: tc.authZError}
			actual := status.String()
			assert.Equal(it, tc.expected, actual)
		})
	}
}

func TestAuthContextUpdaterInterceptor(t *testing.T) {
	for name, tc := range map[string]struct {
		server     interface{}
		authZError error
	}{
		"service call to server without AuthFuncOverride is denied": {
			server:     &serverWithoutAuthFuncOverride{},
			authZError: errox.NoAuthzConfigured,
		},
		"service call to server with allowing AuthFuncOverride is allowed": {
			server:     &serverWithAuthorizer{authZError: nil},
			authZError: nil,
		},
		"service call to server with rejecting AuthFuncOverride is denied": {
			server:     &serverWithAuthorizer{authZError: errox.NotAuthorized},
			authZError: errox.NotAuthorized,
		},
	} {
		t.Run(name, func(it *testing.T) {
			interceptor := AuthContextUpdaterInterceptor()
			handlerCalled := false
			handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
				handlerCalled = true
				authStatus := GetAuthErrorFromContext(ctx)
				if tc.authZError != nil {
					assert.ErrorIs(it, authStatus.Error, tc.authZError)
				} else {
					assert.NoError(it, authStatus.Error)
				}
				return nil, nil
			}
			serverInfo := &grpc.UnaryServerInfo{
				Server:     tc.server,
				FullMethod: "test method",
			}
			_, err := interceptor(it.Context(), nil, serverInfo, handler)
			assert.NoError(it, err)
			// Even when the service call is denied, the authorization context updater
			// calls the handler. The goal is to be able to audit denied service calls.
			assert.True(it, handlerCalled)
		})
	}
}

func TestAuthCheckerInterceptor(t *testing.T) {
	for name, tc := range map[string]struct {
		authzErr error
		wantErr  bool
	}{
		"no authorization error -> service call processed": {
			authzErr: nil,
			wantErr:  false,
		},
		"authorization error -> service call not processed": {
			authzErr: errors.New("authorization error"),
			wantErr:  true,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := ContextWithAuthStatus(it.Context(), tc.authzErr)
			interceptor := AuthCheckerInterceptor()
			handlerBase := &registeringHandlerBase{called: false}
			_, err := interceptor(ctx, nil, nil, handlerBase.Handler)
			if tc.wantErr {
				assert.ErrorIs(it, err, tc.authzErr)
				// The handler is not called when authorization rejects the service call.
				assert.False(it, handlerBase.called)
			} else {
				assert.NoError(it, err)
				assert.True(it, handlerBase.called)
			}
		})
	}

	t.Run("no authz status", func(it *testing.T) {
		interceptor := AuthCheckerInterceptor()
		handlerBase := &registeringHandlerBase{called: false}
		const expectedErrorText = "authentication status is always required"
		_, err := interceptor(it.Context(), nil, nil, handlerBase.Handler)
		assert.ErrorContains(it, err, expectedErrorText)
		assert.False(it, handlerBase.called)
	})
}
