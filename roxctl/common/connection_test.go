package common

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCShouldRetry(t *testing.T) {
	ctx := context.Background()
	called := 0
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Header().Set("content-type", "application/grpc+proto")
		if called == 1 {
			w.WriteHeader(http.StatusInternalServerError) // retry
		} else {
			w.WriteHeader(http.StatusOK) // do not retry
		}
	})
	srv, host, serverName, opts := runServer(handlerFunc)
	defer srv.Close()

	config := grpcConfig{
		opts:       opts,
		serverName: serverName,
		endpoint:   host,
	}
	conn, err := createGRPCConn(config)
	assert.NoError(t, err)

	conn.Connect()

	err = conn.Invoke(ctx, "/test", nil, nil)
	assert.Error(t, err)
	assert.Equal(t, 2, called)
}

func runServer(handlerFunc http.HandlerFunc) (*httptest.Server, string, string, clientconn.Options) {
	noopServer := httptest.NewUnstartedServer(handlerFunc)
	noopServer.EnableHTTP2 = true
	noopServer.StartTLS()
	u, _ := url.Parse(noopServer.URL)
	certPool := x509.NewCertPool()
	certPool.AddCert(noopServer.Certificate())
	serverName := noopServer.Certificate().DNSNames[0]
	opts := clientconn.Options{TLS: clientconn.TLSConfigOptions{ServerName: serverName, RootCAs: certPool}}
	return noopServer, u.Host, serverName, opts
}

func Test_shouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Context Deadline Exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "Context Canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "Network Timeout Error",
			err:      &net.DNSError{IsTimeout: true},
			expected: true,
		},
		{
			name:     "Certificate Error",
			err:      errors.New("x509: certificate signed by unknown authority"),
			expected: false,
		},
		{
			name:     "GRPC Unavailable Error",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: true,
		},
		{
			name:     "GRPC Resource Exhausted Error",
			err:      status.Error(codes.ResourceExhausted, "resource exhausted"),
			expected: true,
		},
		{
			name:     "GRPC Unauthenticated Error",
			err:      status.Error(codes.Unauthenticated, "unauthenticated"),
			expected: false,
		},
		{
			name:     "GRPC Permission Denied Error",
			err:      status.Error(codes.PermissionDenied, "permission denied"),
			expected: false,
		},
		{
			name:     "Unknown Error",
			err:      errors.New("unknown error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
