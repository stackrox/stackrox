package mtls

import (
	"context"

	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.LoggerForModule()
)

// UnaryInterceptor applies authentication to unary gRPC server calls.
// It is intended for use in a chain of interceptors.
func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return contextutil.UnaryServerInterceptor(doAuth)
}

// StreamInterceptor applies authentication to streaming gRPC server calls.
// It is intended for use in a chain of interceptors.
func StreamInterceptor() grpc.StreamServerInterceptor {
	return contextutil.StreamServerInterceptor(doAuth)
}

func doAuth(ctx context.Context) (context.Context, error) {
	newCtx, err := authTLS(ctx)
	if err != nil {
		logger.Debugf("Request failed TLS validation: %v", err)
		return ctx, nil
	}
	return newCtx, nil
}

func authTLS(ctx context.Context) (newCtx context.Context, err error) {
	ri := requestinfo.FromContext(ctx)
	l := len(ri.VerifiedChains)
	switch {
	case l == 0:
		return ctx, status.Error(codes.Unauthenticated, "No verified certificate chains were presented")
	case l > 1:
		return ctx, status.Error(codes.Unauthenticated, "Providing multiple verified chains is not supported")
	}
	leaf := ri.VerifiedChains[0][0]
	cn := mtls.SubjectFromCommonName(leaf.Subject.CommonName)
	return authn.NewTLSContext(ctx, authn.TLSIdentity{
		Identity: mtls.Identity{
			Subject: cn,
			Serial:  leaf.SerialNumber,
		},
		Expiration: leaf.NotAfter,
	}), nil
}
