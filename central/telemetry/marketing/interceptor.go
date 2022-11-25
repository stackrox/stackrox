package marketing

import (
	"context"
	"strings"
	"sync"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	mPkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"google.golang.org/grpc"
)

var (
	ignoredPaths = set.NewFrozenSet("/v1/ping", "/v1/metadata")
	once         sync.Once
	interceptor  grpc.UnaryServerInterceptor
)

func track(ctx context.Context, t mPkg.Telemeter, err error, info *grpc.UnaryServerInfo, trackedPaths set.FrozenSet[string]) {
	userAgent, userID, path, code := getRequestDetails(ctx, err, info)

	// Track the API path and error code of some requests:
	if ignoredPaths.Contains(path) {
		return
	}
	if trackedPaths.Contains("*") || trackedPaths.Contains(path) {
		t.Track("API Call", userID, map[string]any{
			"Path":       path,
			"Code":       code,
			"User-Agent": userAgent,
		})
	}
}

func getRequestDetails(ctx context.Context, err error, info *grpc.UnaryServerInfo) (userAgent string, userID string, method string, code int) {
	ri := requestinfo.FromContext(ctx)
	userAgent = strings.Join(ri.Metadata.Get("User-Agent"), ", ")

	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		userID = "unauthenticated"
		log.Debug("Cannot identify user from context: ", iderr)
	} else {
		userID = id.UID()
	}

	if ri.HTTPRequest != nil && ri.HTTPRequest.URL != nil {
		method = ri.HTTPRequest.URL.Path
		code = grpcError.ErrToHTTPStatus(err)
	} else {
		method = info.FullMethod
		code = int(erroxGRPC.RoxErrorToGRPCCode(err))
	}
	return
}

// Init initializes the periodic telemetry data gatherer and returns an GRPC API
// call inteceptor. Returns nil if telemetry data collection is disabled.
func getInterceptor() grpc.UnaryServerInterceptor {
	config := mPkg.Singleton()
	trackedPaths := set.NewFrozenSet(config.APIPaths...)
	log.Info("Telemetry device ID:", config.ID)
	log.Info("API path telemetry enabled for: ", config.APIPaths)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		go track(ctx, TelemeterSingleton(), err, info, trackedPaths)
		return resp, err
	}
}

func InterceptorSingleton() grpc.UnaryServerInterceptor {
	once.Do(func() {
		interceptor = getInterceptor()
	})
	return interceptor
}
