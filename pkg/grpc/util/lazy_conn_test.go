package util

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/examples/helloworld/helloworld"
)

type helloServer struct {
	helloworld.UnimplementedGreeterServer
}

func (s *helloServer) SayHello(_ context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func TestLazyConn(t *testing.T) {
	serverSignal := concurrency.Signal{}
	server := grpc.NewServer()
	defer server.Stop()
	helloworld.RegisterGreeterServer(server, &helloServer{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		defer serverSignal.Signal()
		err = server.Serve(listener)
		require.NoError(t, err)
	}()

	lazyConn := NewLazyClientConn()

	// Test if gRPC connection times out
	failSignal := concurrency.NewSignal()
	failCtx, failCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer failCancel()
	go func(ctx context.Context) {
		defer failSignal.Signal()
		client := helloworld.NewGreeterClient(lazyConn)
		_, err := client.SayHello(ctx, &helloworld.HelloRequest{Name: "Lazy Hazard"})
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "Error types did not match")
	}(failCtx)

	// Test if gRPC connection will be active after lazyConn receives an active gRPC connection
	successSignal := concurrency.NewSignal()
	successCtx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()
	go func(ctx context.Context) {
		client := helloworld.NewGreeterClient(lazyConn)
		resp, err := client.SayHello(ctx, &helloworld.HelloRequest{Name: "Lazy Hazard"})

		require.NoError(t, err)
		assert.Equal(t, "Hello Lazy Hazard", resp.Message)
		successSignal.Signal()
	}(successCtx)

	// Wait until the call in the first goroutine returns, or the context expires,
	// and ensure that even the former case can only manifest after the context
	// has expired.
	select {
	case <-failCtx.Done():
	case <-failSignal.WaitC():
	}
	require.Error(t, failCtx.Err())
	assert.True(t, errors.Is(failCtx.Err(), context.DeadlineExceeded), "Error types did not match")

	// connect to gRPC server
	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)
	defer utils.IgnoreError(conn.Close)

	lazyConn.Set(conn)
	select {
	case <-successSignal.WaitC():
		break
	case <-successCtx.Done():
	}
	assert.NoError(t, successCtx.Err(), "Timeout for gRPC request.")

	serverSignal.Wait()
}
