package repomappings

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP_Responses(t *testing.T) {
	type args struct {
		writer  *mock.ResponseWriter
		request *http.Request
		methods []string
	}
	tests := []struct {
		name             string
		args             args
		responseBody     string
		statusCode       int
		centralReachable bool
	}{
		{
			name:             "when central is not reachable then return internal error",
			statusCode:       http.StatusServiceUnavailable,
			responseBody:     "{\"code\":14,\"message\":\"central not reachable\"}",
			centralReachable: false,
		},
		{
			name:             "when central replies 200 with content then writer matches",
			statusCode:       http.StatusOK,
			responseBody:     "the foobar body.",
			centralReachable: true,
		},
		{
			name:             "when central replies 304 then writer matches",
			statusCode:       http.StatusNotModified,
			centralReachable: true,
		},
		{
			name:       "when method is not GET then 405",
			statusCode: http.StatusMethodNotAllowed,
			args: args{
				methods: []string{
					http.MethodHead,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodConnect,
					http.MethodOptions,
					http.MethodTrace,
				},
				request: &http.Request{},
			},
			centralReachable: true,
		},
		{
			name:       "when request contains multiple headers then proxy all of them",
			statusCode: http.StatusOK,
			args: args{
				request: &http.Request{
					URL:    &url.URL{},
					Header: map[string][]string{"Accept-Encoding": {"foo", "bar"}},
				},
			},
			centralReachable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set args defaults.
			if tt.args.writer == nil {
				tt.args.writer = mock.NewResponseWriter()
			}
			if tt.args.methods == nil {
				// Defaults to GET.
				tt.args.methods = []string{http.MethodGet}
			}
			// Perform one test per HTTP method.
			for _, method := range tt.args.methods {
				if tt.args.request == nil {
					tt.args.request = &http.Request{
						Method: method,
						URL:    &url.URL{RawQuery: ""},
						Header: map[string][]string{"If-Modified-Since": {"1209"}, "Accept-Encoding": {""}},
					}
				} else {
					tt.args.request.Method = method
				}

				h := &Handler{
					centralClient: &http.Client{
						Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
							assert.Equal(t, tt.args.request.URL.RawQuery, req.URL.RawQuery)
							for _, header := range headersToProxy.AsSlice() {
								assert.Equal(t, tt.args.request.Header.Values(header), req.Header.Values(header))
							}
							return &http.Response{
								StatusCode: tt.statusCode,
								Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
							}, nil
						}),
					},
				}
				h.centralReachable.Store(tt.centralReachable)
				h.ServeHTTP(tt.args.writer, tt.args.request)
				assert.Equal(t, tt.responseBody, tt.args.writer.Data.String())
				assert.Equal(t, tt.statusCode, tt.args.writer.Code)
			}
		})
	}
}
