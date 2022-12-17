package common

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stretchr/testify/assert"
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
