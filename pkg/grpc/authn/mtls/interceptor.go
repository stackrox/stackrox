package mtls

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.LoggerForModule()
)

// UnaryInterceptor applies authentication to unary gRPC server calls.
// It is intended for use in a chain of interceptors.
func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return authUnary
}

// StreamInterceptor applies authentication to streaming gRPC server calls.
// It is intended for use in a chain of interceptors.
func StreamInterceptor() grpc.StreamServerInterceptor {
	return authStream
}

func authUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	newCtx := doAuth(ctx)
	return handler(newCtx, req)
}

func authStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	newCtx := doAuth(stream.Context())
	newStream := &authn.StreamWithContext{
		ServerStream:    stream,
		ContextOverride: newCtx,
	}
	return handler(srv, newStream)
}

func doAuth(ctx context.Context) (newCtx context.Context) {
	newCtx, err := authTLS(ctx)
	if err != nil {
		logger.Debugf("Request failed TLS validation: %v", err)
		return ctx
	}
	return newCtx
}

func authTLS(ctx context.Context) (newCtx context.Context, err error) {
	client, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "Could not access authentication information")
	}
	tls, ok := client.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "Could not get TLS information from peer")
	}
	l := len(tls.State.VerifiedChains)
	switch {
	case l == 0:
		return ctx, status.Error(codes.Unauthenticated, "No verified certificate chains were presented")
	case l > 1:
		return ctx, status.Error(codes.Unauthenticated, "Providing multiple verified chains is not supported")
	}
	chain := tls.State.VerifiedChains[0]
	leaf := chain[0]
	cn := mtls.CommonNameFromString(leaf.Subject.CommonName)
	return authn.NewTLSContext(ctx, authn.TLSIdentity{
		Identity: mtls.Identity{
			Name:   cn,
			Serial: leaf.SerialNumber,
		},
		Expiration: leaf.NotAfter,
	}), nil
}
