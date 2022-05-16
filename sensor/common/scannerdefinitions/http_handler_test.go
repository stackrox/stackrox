package scannerdefinitions

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type responseWriterMock struct {
	bytes.Buffer
	statusCode int
	headers    http.Header
}

func NewMockResponseWriter() *responseWriterMock {
	return &responseWriterMock{
		headers: make(http.Header),
	}
}

func (m *responseWriterMock) Header() http.Header {
	return m.headers
}

func (m *responseWriterMock) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

// transportMockFunc is a transport mock that call itself to implement http.Transport's RoundTrip.
type transportMockFunc func(req *http.Request) (*http.Response, error)

func (f transportMockFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func Test_scannerDefinitionsHandler_ServeHTTP(t *testing.T) {
	type args struct {
		writer  *responseWriterMock
		request *http.Request
		methods []string
	}
	tests := []struct {
		name         string
		args         args
		responseBody string
		statusCode   int
	}{
		{
			name:         "when central replies 200 with content then writer matches",
			statusCode:   http.StatusOK,
			responseBody: "the foobar body.",
		},
		{
			name:       "when central replies 304 then writer matches",
			statusCode: http.StatusNotModified,
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set args defaults.
			if tt.args.writer == nil {
				tt.args.writer = NewMockResponseWriter()
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
						URL:    &url.URL{RawQuery: "bar=1&foo=2"},
						Header: map[string][]string{"If-Modified-Since": {"1209"}, "Accept-Encoding": {""}},
					}
				} else {
					tt.args.request.Method = method
				}
				h := &scannerDefinitionsHandler{
					centralClient: &http.Client{
						Transport: transportMockFunc(func(req *http.Request) (*http.Response, error) {
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
				h.ServeHTTP(tt.args.writer, tt.args.request)
				assert.Equal(t, tt.responseBody, tt.args.writer.String())
				assert.Equal(t, tt.statusCode, tt.args.writer.statusCode)
			}
		})
	}
}
