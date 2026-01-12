package centralproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// newProxyForTest creates a test proxy with a custom transport for testing purposes.
func newProxyForTest(t *testing.T, baseURL *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(baseURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			pkghttputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to contact central: %v", err)
		},
	}
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		centralReachable bool
		wantStatusCode   int
		wantError        string
	}{
		{
			name:             "GET is allowed",
			method:           http.MethodGet,
			centralReachable: true,
			wantStatusCode:   0, // validateRequest returns nil on success
		},
		{
			name:             "POST is allowed",
			method:           http.MethodPost,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "OPTIONS is allowed (for CORS preflight)",
			method:           http.MethodOptions,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "HEAD is allowed",
			method:           http.MethodHead,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "PUT returns 405",
			method:           http.MethodPut,
			centralReachable: true,
			wantStatusCode:   http.StatusMethodNotAllowed,
			wantError:        "method PUT not allowed",
		},
		{
			name:             "DELETE returns 405",
			method:           http.MethodDelete,
			centralReachable: true,
			wantStatusCode:   http.StatusMethodNotAllowed,
			wantError:        "method DELETE not allowed",
		},
		{
			name:             "central not reachable returns 503",
			method:           http.MethodGet,
			centralReachable: false,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantError:        "central not reachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{}
			h.centralReachable.Store(tt.centralReachable)

			req := httptest.NewRequest(tt.method, "/v1/alerts", nil)
			err := h.validateRequest(req)

			if tt.wantStatusCode == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				httpErr, ok := err.(pkghttputil.HTTPError)
				require.True(t, ok, "error should be an HTTPError")
				assert.Equal(t, tt.wantStatusCode, httpErr.HTTPStatusCode())
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	t.Run("validation fails, proxy not called", func(t *testing.T) {
		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := &Handler{
			proxy: newProxyForTest(t, baseURL, nil),
		}
		h.centralReachable.Store(false) // Will fail validation

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "central not reachable")
	})

	t.Run("validation passes, request proxied", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := &Handler{
			proxy: newProxyForTest(t, baseURL, mockTransport),
		}
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.True(t, proxyCalled, "proxy should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("proxy error handled by ErrorHandler", func(t *testing.T) {
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection timeout")
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := &Handler{
			proxy: newProxyForTest(t, baseURL, mockTransport),
		}
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to contact central")
		assert.Contains(t, w.Body.String(), "connection timeout")
	})
}

func TestServeHTTP_ConstructsAbsoluteURLs(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		requestPath  string
		requestQuery string
		expectedURL  string
	}{
		{
			name:        "simple path without query",
			baseURL:     "https://central.stackrox.svc:443",
			requestPath: "/v1/alerts",
			expectedURL: "https://central.stackrox.svc:443/v1/alerts",
		},
		{
			name:         "path with query parameters",
			baseURL:      "https://central.stackrox.svc:443",
			requestPath:  "/v1/alerts",
			requestQuery: "limit=10&offset=20",
			expectedURL:  "https://central.stackrox.svc:443/v1/alerts?limit=10&offset=20",
		},
		{
			name:        "graphql endpoint",
			baseURL:     "https://central.stackrox.svc:443",
			requestPath: "/api/graphql",
			expectedURL: "https://central.stackrox.svc:443/api/graphql",
		},
		{
			name:        "endpoint without port",
			baseURL:     "https://central.stackrox.svc",
			requestPath: "/v1/deployments",
			expectedURL: "https://central.stackrox.svc/v1/deployments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseURL)
			assert.NoError(t, err)

			var capturedURL string
			mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Capture the URL that was used for the request
				capturedURL = req.URL.String()

				// Verify it's an absolute URL with scheme and host
				assert.NotEmpty(t, req.URL.Scheme, "URL scheme should not be empty")
				assert.NotEmpty(t, req.URL.Host, "URL host should not be empty")

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("{}")),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})

			h := &Handler{proxy: newProxyForTest(t, baseURL, mockTransport)}
			h.centralReachable.Store(true)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			if tt.requestQuery != "" {
				req.URL.RawQuery = tt.requestQuery
			}
			writer := httptest.NewRecorder()
			h.ServeHTTP(writer, req)

			assert.Equal(t, tt.expectedURL, capturedURL)
		})
	}
}

func TestNotify(t *testing.T) {
	h := &Handler{}

	t.Run("CentralReachable sets centralReachable to true", func(t *testing.T) {
		h.centralReachable.Store(false)
		h.Notify(common.SensorComponentEventCentralReachable)
		assert.True(t, h.centralReachable.Load())
	})

	t.Run("OfflineMode sets centralReachable to false", func(t *testing.T) {
		h.centralReachable.Store(true)
		h.Notify(common.SensorComponentEventOfflineMode)
		assert.False(t, h.centralReachable.Load())
	})
}
