package auth

import (
	"context"
	"crypto/x509/pkix"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.New("grpc/auth")
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
	newCtx, err := auth(ctx)
	if err != nil {
		return nil, err
	}
	return handler(newCtx, req)
}

func authStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	newCtx, err := auth(stream.Context())
	if err != nil {
		return err
	}
	newStream := &streamWithContext{
		ServerStream:    stream,
		ContextOverride: newCtx,
	}
	return handler(srv, newStream)
}

func auth(ctx context.Context) (newCtx context.Context, err error) {
	newCtx, ok, err := authTLS(ctx)
	if ok {
		return newCtx, nil
	}
	logger.Debugf("Request failed TLS validation: %v", err)

	newCtx, ok = authToken(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "No client certificate or authentication provided")
	}
	return newCtx, nil
}

func authTLS(ctx context.Context) (newCtx context.Context, ok bool, err error) {
	client, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, false, status.Error(codes.Unauthenticated, "Could not access authentication information")
	}
	tls, ok := client.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ctx, false, status.Error(codes.Unauthenticated, "Could not get TLS information from peer")
	}
	l := len(tls.State.VerifiedChains)
	switch {
	case l == 0:
		return ctx, false, status.Error(codes.Unauthenticated, "No verified certificate chains were presented")
	case l > 1:
		return ctx, false, status.Error(codes.Unauthenticated, "Providing multiple verified chains is not supported")
	}
	chain := tls.State.VerifiedChains[0]
	leaf := chain[0]
	return newContext(ctx, Identity{
		User:         leaf.Subject.CommonName,
		IdentityType: IdentityType{ServiceType: ouType(leaf.Subject)},
		Serial:       leaf.SerialNumber,
	}), true, nil
}

func ouType(subject pkix.Name) v1.ServiceType {
	if len(subject.OrganizationalUnit) > 0 {
		return v1.ServiceType(v1.ServiceType_value[subject.OrganizationalUnit[0]])
	}
	return v1.ServiceType_UNKNOWN_SERVICE
}

func authToken(ctx context.Context) (newCtx context.Context, ok bool) {
	// TODO(cg): This handler obviously isn't secure. Replace it when user auth is implemented.

	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// TODO(cg): When auth is mandatory, return Unauthenticated status.
		return ctx, true
	}
	var username string
	if len(meta["username"]) > 0 {
		username = meta["username"][0]
	}
	return newContext(ctx, Identity{
		User:         username,
		IdentityType: IdentityType{EndUser: true},
	}), true
}
