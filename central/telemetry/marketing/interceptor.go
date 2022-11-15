package marketing

import (
	"context"
	"strings"

	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"google.golang.org/grpc"
)

var ignoredPaths = set.NewFrozenSet("/v1/ping", "/v1/metadata")

func track(ctx context.Context, t mpkg.Telemeter, err error, info *grpc.UnaryServerInfo, trackedPaths set.FrozenSet[string]) {
	userAgent, userID, path, code := getRequestDetails(ctx, err, info)

	// Track the API path and error code of some requests:
	if !ignoredPaths.Contains(path) && (trackedPaths.Contains("*") || trackedPaths.Contains(path)) {
		t.TrackProps("API Call", userID, map[string]any{
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
		userID = "unknown"
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
