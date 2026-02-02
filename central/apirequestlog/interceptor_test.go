package apirequestlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	// Create a test handler that returns success
	successHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// Create a test handler that returns an error
	errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}

	tests := []struct {
		name        string
		handler     grpc.UnaryHandler
		fullMethod  string
		expectError bool
	}{
		{
			name:        "successful request",
			handler:     successHandler,
			fullMethod:  "/v1.ClusterService/GetClusters",
			expectError: false,
		},
		{
			name:        "failed request",
			handler:     errorHandler,
			fullMethod:  "/v1.PolicyService/GetPolicy",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			info := &grpc.UnaryServerInfo{
				FullMethod: tt.fullMethod,
			}

			// The interceptor will use requestinfo.FromContext which returns zero value
			resp, err := interceptor(ctx, nil, info, tt.handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "success", resp)
			}
		})
	}
}

func TestHTTPInterceptor(t *testing.T) {
	interceptor := HTTPInterceptor()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wrappedHandler := interceptor(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")

	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "OK", recorder.Body.String())
}
