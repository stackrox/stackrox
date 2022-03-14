package grpc

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestMisdirectedRequests(t *testing.T) {
	cfg := EndpointConfig{
		ListenEndpoint: "127.0.0.1:0",
		ServeGRPC: true,
		ServeHTTP: true,
		NoHTTP2: false,
	}
	grpcSrv := grpc.NewServer()
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = fmt.Fprintln(w, "hello")
		return
	})

	_, servers, err := cfg.instantiate(dummyHandler, grpcSrv)
	require.NoError(t, err)

	serverErrs := make(chan error, len(servers))
	for _, srvAndLis := range servers {
		srvAndLis := srvAndLis
		go func() {
			serverErrs <- srvAndLis.srv.Serve(srvAndLis.listener)
		}()
	}

	for _, srvAndLis := range servers {
		_ = srvAndLis.listener.Close()
	}

	for i := 0; i < len(servers); i++ {
		<-serverErrs
	}
}
