package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// recordingAuthorizer records every Authorized invocation and returns the
// configured error (nil means the request is authorized).
type recordingAuthorizer struct {
	returnErr error

	calls []struct {
		ctx            context.Context
		fullMethodName string
	}
}

func (a *recordingAuthorizer) Authorized(ctx context.Context, fullMethodName string) error {
	a.calls = append(a.calls, struct {
		ctx            context.Context
		fullMethodName string
	}{ctx: ctx, fullMethodName: fullMethodName})
	return a.returnErr
}

func TestCustomRouteHandler(t *testing.T) {
	const route = "/api/test/route"

	testCases := map[string]struct {
		authorizer authz.Authorizer

		expectStatus            int
		expectAuthorizerCalls   int
		expectInterceptorCalls  int
		expectServerHandlerCall bool
	}{
		"authorized request passes through the post-auth interceptor to the server handler": {
			authorizer:              &recordingAuthorizer{},
			expectStatus:            http.StatusOK,
			expectAuthorizerCalls:   1,
			expectInterceptorCalls:  1,
			expectServerHandlerCall: true,
		},
		"unauthorized request is rejected without invoking the post-auth interceptor": {
			authorizer:              &recordingAuthorizer{returnErr: errox.NotAuthorized},
			expectStatus:            http.StatusForbidden,
			expectAuthorizerCalls:   1,
			expectInterceptorCalls:  0,
			expectServerHandlerCall: false,
		},
		// A nil authorizer falls back to deny.Everyone(), which returns
		// errox.NoAuthzConfigured; that maps to codes.Unimplemented (HTTP 501).
		"nil authorizer denies everyone": {
			authorizer:              nil,
			expectStatus:            http.StatusNotImplemented,
			expectAuthorizerCalls:   0,
			expectInterceptorCalls:  0,
			expectServerHandlerCall: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			serverHandlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				serverHandlerCalled = true
				_, _ = w.Write([]byte("server handler response"))
			})

			interceptorCalls := 0
			postAuthInterceptor := func(h http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					interceptorCalls++
					// Mark the response so we can assert the server handler ran
					// wrapped by the interceptor, not beside it.
					w.Header().Set("X-Post-Auth-Interceptor", "invoked")
					h.ServeHTTP(w, req)
				})
			}

			customRoute := CustomRoute{
				Route:         route,
				Authorizer:    tc.authorizer,
				ServerHandler: handler,
			}

			req := httptest.NewRequest(http.MethodGet, route, nil)
			recorder := httptest.NewRecorder()
			reqCtx := req.Context()

			customRoute.Handler(postAuthInterceptor).ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectStatus, recorder.Code)
			assert.Equal(t, tc.expectInterceptorCalls, interceptorCalls)
			assert.Equal(t, tc.expectServerHandlerCall, serverHandlerCalled)

			// The comma-ok form guards the "nil authorizer" case, where the
			// authorizer falls back to deny.Everyone and is never recorded.
			if authorizer, ok := tc.authorizer.(*recordingAuthorizer); ok {
				require.Len(t, authorizer.calls, tc.expectAuthorizerCalls)
				if tc.expectAuthorizerCalls > 0 {
					// The authorizer must be exercised with the HTTP-equivalent
					// RPC name and the context of the incoming request.
					assert.Equal(t, RPCNameForHTTP(route, http.MethodGet), authorizer.calls[0].fullMethodName)
					assert.Equal(t, reqCtx, authorizer.calls[0].ctx)
				}
			}

			if tc.expectServerHandlerCall {
				assert.Equal(t, "server handler response", recorder.Body.String())
				assert.Equal(t, "invoked", recorder.Header().Get("X-Post-Auth-Interceptor"))
			}
		})
	}
}

func TestRPCNameForHTTP(t *testing.T) {
	testCases := map[string]struct {
		route    string
		method   string
		wantName string
	}{
		"empty method yields the bare route": {
			route:    "/api/test/route",
			method:   "",
			wantName: "/api/test/route",
		},
		"method and route are combined": {
			route:    "/api/test/route",
			method:   http.MethodPost,
			wantName: "POST /api/test/route",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.wantName, RPCNameForHTTP(tc.route, tc.method))
		})
	}
}
